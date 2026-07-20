package douyin

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// RefundPullParams 是抖店售后/退款拉取参数。
type RefundPullParams struct {
	StartTime     string
	EndTime       string
	StartPage     int
	Concurrency   int
	UseUpdateTime bool
}

// RefundPullResult 记录售后/退款拉取结果。
type RefundPullResult struct {
	StoreCount   int                     `json:"storeCount"`
	FailedStores int                     `json:"failedStores"`
	FailedPages  int                     `json:"failedPages"`
	TotalOrders  int                     `json:"totalOrders"`
	TotalSaved   int                     `json:"totalSaved"`
	DurationMs   int                     `json:"durationMs"`
	StoreResults []RefundStorePullResult `json:"storeResults"`
}

// RefundStorePullResult 是单店铺售后/退款拉取结果。
type RefundStorePullResult struct {
	StoreID     int    `json:"storeId"`
	StoreName   string `json:"storeName"`
	TotalOrders int    `json:"totalOrders"`
	TotalSaved  int    `json:"totalSaved"`
	FailedPages []int  `json:"failedPages,omitempty"`
	Error       string `json:"error,omitempty"`
}

// RefundPullService 拉取并存储抖店售后/退款订单。
type RefundPullService struct{}

func NewRefundPullService() *RefundPullService {
	return &RefundPullService{}
}

// Pull 拉取所有已授权抖音店铺的售后/退款订单。
func (s *RefundPullService) Pull(ctx context.Context, opt RefundPullParams) (*RefundPullResult, error) {
	startedAt := time.Now()
	stores, dbErr := g.DB().Model("store").Ctx(ctx).
		Where("platform", "douyin").Where("status", 1).Where("author", 1).All()
	if dbErr != nil {
		return nil, dbErr
	}
	result := &RefundPullResult{StoreCount: len(stores)}
	if len(stores) == 0 {
		return result, nil
	}

	cst := time.FixedZone("CST", 8*3600)
	now := time.Now().In(cst)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cst)
	var requestedStart, requestedEnd time.Time
	var err error
	if opt.StartTime != "" {
		requestedStart, err = time.ParseInLocation("2006-01-02 15:04:05", opt.StartTime, cst)
		if err != nil {
			return nil, fmt.Errorf("开始时间格式错误: %w", err)
		}
	}
	if opt.EndTime != "" {
		requestedEnd, err = time.ParseInLocation("2006-01-02 15:04:05", opt.EndTime, cst)
		if err != nil {
			return nil, fmt.Errorf("结束时间格式错误: %w", err)
		}
	}
	if !requestedStart.IsZero() && !requestedEnd.IsZero() && !requestedStart.Before(requestedEnd) {
		return nil, fmt.Errorf("开始时间必须早于结束时间")
	}

	for _, store := range stores {
		storeID := store["id"].Int()
		storeName := store["store_name"].String()
		shopID := store["store_id"].Int64()
		storeResult := RefundStorePullResult{StoreID: storeID, StoreName: storeName}

		startTimeToUse := requestedStart
		endTimeToUse := requestedEnd
		if startTimeToUse.IsZero() {
			startTimeToUse = todayStart
		}
		if endTimeToUse.IsZero() {
			endTimeToUse = now
		}
		st := startTimeToUse.Format("2006-01-02 15:04:05")
		et := endTimeToUse.Format("2006-01-02 15:04:05")

		g.Log().Infof(ctx, "[DouyinRefundPull] %s 拉取售后/退款: start=%s end=%s useUpdateTime=%v",
			storeName, st, et, opt.UseUpdateTime)

		svc := NewService()
		firstPage := opt.StartPage
		firstResult, apiErr := svc.GetRefundOrderList(ctx, model.PlatformRefundOrderListParams{
			Platform: "douyin", Page: firstPage, PageSize: 100,
			StartTime: st, EndTime: et, StoreId: storeID, UseUpdateTime: opt.UseUpdateTime,
		})
		if apiErr != nil {
			g.Log().Errorf(ctx, "[DouyinRefundPull] %s 第%d页失败: %v", storeName, firstPage, apiErr)
			storeResult.Error = apiErr.Error()
			storeResult.FailedPages = append(storeResult.FailedPages, firstPage)
			result.FailedStores++
			result.FailedPages++
			result.StoreResults = append(result.StoreResults, storeResult)
			continue
		}

		fillRefundStoreFields(firstResult.Orders, shopID, storeName)
		saved, saveErr := saveRefundOrders(ctx, firstResult.Orders)
		if saveErr != nil {
			g.Log().Errorf(ctx, "[DouyinRefundPull] %s 第%d页保存失败: %v", storeName, firstPage, saveErr)
			storeResult.Error = saveErr.Error()
			storeResult.FailedPages = append(storeResult.FailedPages, firstPage)
			result.FailedStores++
			result.FailedPages++
			result.StoreResults = append(result.StoreResults, storeResult)
			continue
		}

		totalPages := 0
		if firstResult.Size > 0 {
			totalPages = (firstResult.Total + firstResult.Size - 1) / firstResult.Size
		}
		storeOrders := len(firstResult.Orders)
		storeSaved := saved
		if firstPage+1 < totalPages && len(firstResult.Orders) > 0 {
			concurrency := opt.Concurrency
			if concurrency <= 0 {
				concurrency = 5
			}
			if concurrency > 45 {
				concurrency = 45
			}
			rateLimiter := time.NewTicker(time.Second / 45)
			var mu sync.Mutex
			var wg sync.WaitGroup
			pageCh := make(chan int, totalPages-firstPage-1)
			for page := firstPage + 1; page < totalPages; page++ {
				pageCh <- page
			}
			close(pageCh)

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					svc := NewService()
					for page := range pageCh {
						select {
						case <-ctx.Done():
							return
						case <-rateLimiter.C:
						}
						var pageResult *model.PlatformRefundOrderListResult
						var apiErr error
						for retry := 0; retry < 3; retry++ {
							pageResult, apiErr = svc.GetRefundOrderList(ctx, model.PlatformRefundOrderListParams{
								Platform: "douyin", Page: page, PageSize: 100,
								StartTime: st, EndTime: et, StoreId: storeID, UseUpdateTime: opt.UseUpdateTime,
							})
							if apiErr == nil {
								break
							}
							if retry < 2 {
								time.Sleep(time.Duration(retry+1) * time.Second)
							}
						}
						if apiErr != nil {
							g.Log().Errorf(ctx, "[DouyinRefundPull] %s worker%d 第%d页失败: %v", storeName, workerID, page, apiErr)
							mu.Lock()
							storeResult.FailedPages = append(storeResult.FailedPages, page)
							mu.Unlock()
							continue
						}
						fillRefundStoreFields(pageResult.Orders, shopID, storeName)
						pageSaved, saveErr := saveRefundOrders(ctx, pageResult.Orders)
						mu.Lock()
						if saveErr != nil {
							g.Log().Errorf(ctx, "[DouyinRefundPull] %s worker%d 第%d页保存失败: %v", storeName, workerID, page, saveErr)
							storeResult.FailedPages = append(storeResult.FailedPages, page)
						} else {
							storeOrders += len(pageResult.Orders)
							storeSaved += pageSaved
						}
						mu.Unlock()
					}
				}(i)
			}
			wg.Wait()
			rateLimiter.Stop()
		}

		storeResult.TotalOrders = storeOrders
		storeResult.TotalSaved = storeSaved
		if len(storeResult.FailedPages) > 0 {
			storeResult.Error = fmt.Sprintf("%d个分页拉取失败", len(storeResult.FailedPages))
			result.FailedStores++
			result.FailedPages += len(storeResult.FailedPages)
		}
		result.TotalOrders += storeOrders
		result.TotalSaved += storeSaved
		result.StoreResults = append(result.StoreResults, storeResult)
	}

	result.DurationMs = int(time.Since(startedAt).Milliseconds())
	if result.FailedStores > 0 {
		return result, fmt.Errorf("抖店售后/退款拉取部分失败: %d个店铺、%d个分页", result.FailedStores, result.FailedPages)
	}
	return result, nil
}

func fillRefundStoreFields(orders []model.PlatformRefundOrderItem, shopID int64, shopName string) {
	for i := range orders {
		orders[i].ShopId = shopID
		orders[i].ShopName = shopName
	}
}

func saveRefundOrders(ctx context.Context, orders []model.PlatformRefundOrderItem) (int, error) {
	if len(orders) == 0 {
		return 0, nil
	}
	sql := `INSERT INTO douyin_refund_order (
		platform, shop_id, shop_name, shop_order_id, sku_order_id, aftersale_id,
		aftersale_order_type, aftersale_type, aftersale_type_text, aftersale_status, aftersale_status_text,
		aftersale_refund_type, aftersale_refund_type_text, refund_type, refund_status, reason_text,
		refund_amount, refund_post_amount, refund_tax_amount, return_promotion_amount, aftersale_num, part_type,
		apply_time, update_time, aftersale_create_time, status_deadline, aftersale_final_time,
		product_id, product_name, product_image, product_spec, shop_sku_code, product_num, product_price,
		pay_amount, post_amount, order_status, order_create_time, return_logistics_code, return_logistics_company_name,
		order_flag, risk_decision_code, risk_decision_reason, risk_decision_description, arbitrate_status, remark,
		raw_json, created_at, updated_at
	) VALUES `
	placeholders := make([]string, 0, len(orders))
	args := make([]interface{}, 0, len(orders)*49)
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, order := range orders {
		if order.AftersaleId == "" {
			continue
		}
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			order.Platform, order.ShopId, order.ShopName, order.ShopOrderId, order.SkuOrderId, order.AftersaleId,
			order.AftersaleOrderType, order.AftersaleType, order.AftersaleTypeText, order.AftersaleStatus, order.AftersaleStatusText,
			order.AftersaleRefundType, order.AftersaleRefundTypeText, order.RefundType, order.RefundStatus, order.ReasonText,
			order.RefundAmount, order.RefundPostAmount, order.RefundTaxAmount, order.ReturnPromotionAmount, order.AftersaleNum, order.PartType,
			nilIfEmpty(order.ApplyTime), nilIfEmpty(order.UpdateTime), nilIfEmpty(order.CreateTime), nilIfEmpty(order.StatusDeadline), nilIfEmpty(order.AftersaleStatusToFinalTime),
			order.ProductId, order.ProductName, order.ProductImage, order.ProductSpec, order.ShopSkuCode, order.ProductNum, order.ProductPrice,
			order.PayAmount, order.PostAmount, order.OrderStatus, nilIfEmpty(order.OrderCreateTime), order.ReturnLogisticsCode, order.ReturnLogisticsCompanyName,
			order.OrderFlag, order.RiskDecisionCode, order.RiskDecisionReason, order.RiskDecisionDescription, order.ArbitrateStatus, order.Remark,
			order.RawJSON, now, now,
		)
	}
	if len(placeholders) == 0 {
		return 0, nil
	}
	sql += strings.Join(placeholders, ",")
	sql += ` ON DUPLICATE KEY UPDATE
		platform = VALUES(platform), shop_id = VALUES(shop_id), shop_name = VALUES(shop_name),
		shop_order_id = VALUES(shop_order_id), aftersale_order_type = VALUES(aftersale_order_type),
		aftersale_type = VALUES(aftersale_type), aftersale_type_text = VALUES(aftersale_type_text),
		aftersale_status = VALUES(aftersale_status), aftersale_status_text = VALUES(aftersale_status_text),
		aftersale_refund_type = VALUES(aftersale_refund_type), aftersale_refund_type_text = VALUES(aftersale_refund_type_text),
		refund_type = VALUES(refund_type), refund_status = VALUES(refund_status), reason_text = VALUES(reason_text),
		refund_amount = VALUES(refund_amount), refund_post_amount = VALUES(refund_post_amount),
		refund_tax_amount = VALUES(refund_tax_amount), return_promotion_amount = VALUES(return_promotion_amount),
		aftersale_num = VALUES(aftersale_num), part_type = VALUES(part_type),
		apply_time = VALUES(apply_time), update_time = VALUES(update_time),
		aftersale_create_time = VALUES(aftersale_create_time), status_deadline = VALUES(status_deadline),
		aftersale_final_time = VALUES(aftersale_final_time), product_id = VALUES(product_id),
		product_name = VALUES(product_name), product_image = VALUES(product_image), product_spec = VALUES(product_spec),
		shop_sku_code = VALUES(shop_sku_code), product_num = VALUES(product_num), product_price = VALUES(product_price),
		pay_amount = VALUES(pay_amount), post_amount = VALUES(post_amount), order_status = VALUES(order_status),
		order_create_time = VALUES(order_create_time), return_logistics_code = VALUES(return_logistics_code),
		return_logistics_company_name = VALUES(return_logistics_company_name), order_flag = VALUES(order_flag),
		risk_decision_code = VALUES(risk_decision_code), risk_decision_reason = VALUES(risk_decision_reason),
		risk_decision_description = VALUES(risk_decision_description), arbitrate_status = VALUES(arbitrate_status),
		remark = VALUES(remark), raw_json = VALUES(raw_json), updated_at = VALUES(updated_at)`

	var err error
	for retry := 0; retry < 3; retry++ {
		_, err = g.DB().Exec(ctx, sql, args...)
		if err == nil {
			return len(placeholders), nil
		}
		if !strings.Contains(err.Error(), "Lock wait timeout") || retry == 2 {
			break
		}
		time.Sleep(time.Duration(retry+1) * 2 * time.Second)
	}
	return 0, err
}
