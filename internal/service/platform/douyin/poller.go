package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
)

// PollParams 手动触发参数
type PollParams struct {
	StartTime     string // 开始时间，为空则用 last_poll_time
	EndTime       string // 结束时间，为空则用当前时间
	StartPage     int    // 起始页码，默认0
	Concurrency   int    // 并发数，默认1（串行），最大45
	RecordLog     bool   // 是否记录任务执行日志（仅定时任务为true）
	UseUpdateTime bool   // 按订单更新时间查询，用于回补退款、发货等状态变化
}

// PollResult 拉取结果
type PollResult struct {
	StoreCount  int `json:"storeCount"`
	TotalOrders int `json:"totalOrders"`
	TotalSaved  int `json:"totalSaved"`
	DurationMs  int `json:"durationMs"`
}

// Poller 抖店订单轮询器（每天最多1000次API调用）
type Poller struct {
	mu         sync.Mutex
	running    bool
	dailyLimit int           // 每日API调用上限
	dailyCount int           // 当日已调用次数
	interval   time.Duration // 两次调用间隔
}

var (
	pollerInstance *Poller
	pollerOnce     sync.Once
)

const douyinValidOrderStatusFilter = "2,3,4,5" // 待发货、已发货、已取消/关闭、已完成

// GetPoller 获取全局轮询器实例
func GetPoller() *Poller {
	pollerOnce.Do(func() {
		pollerInstance = &Poller{
			dailyLimit: 100000,
			interval:   1 * time.Hour, // 每小时调度一次
		}
	})
	return pollerInstance
}

// Start 启动轮询
func (p *Poller) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("抖店订单轮询器已在运行")
	}
	p.running = true
	p.mu.Unlock()

	g.Log().Infof(ctx, "[DouyinPoller] 启动抖店订单轮询，每日上限%d次，间隔%v", p.dailyLimit, p.interval)

	pattern := fmt.Sprintf("@every %ds", int(p.interval.Seconds()))
	_, err := gcron.AddSingleton(ctx, pattern, func(ctx context.Context) {
		p.tick(ctx)
	}, "douyin_order_poller")
	if err != nil {
		return fmt.Errorf("注册抖店轮询定时任务失败: %w", err)
	}

	// 每天凌晨重置计数器
	_, err = gcron.AddSingleton(ctx, "0 1 0 * * *", func(ctx context.Context) {
		p.mu.Lock()
		p.dailyCount = 0
		p.mu.Unlock()
		g.Log().Info(ctx, "[DouyinPoller] 每日API调用计数已重置")
	}, "douyin_poller_reset")
	if err != nil {
		return fmt.Errorf("注册每日重置定时任务失败: %w", err)
	}

	// 每天凌晨3点清理7天前的历史日志详情
	_, err = gcron.AddSingleton(ctx, "0 0 3 * * *", func(ctx context.Context) {
		service.CleanOldLogs(ctx)
	}, "task_log_cleaner")
	if err != nil {
		return fmt.Errorf("注册日志清理任务失败: %w", err)
	}

	return nil
}

// Stop 停止轮询
func (p *Poller) Stop() {
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()

	gcron.Remove("douyin_order_poller")
	gcron.Remove("douyin_poller_reset")
	gcron.Remove("task_log_cleaner")
	g.Log().Info(context.Background(), "[DouyinPoller] 已停止")
}

// IsRunning 是否运行中
func (p *Poller) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// Stats 获取统计信息
func (p *Poller) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]interface{}{
		"running":     p.running,
		"dailyLimit":  p.dailyLimit,
		"dailyCount":  p.dailyCount,
		"interval":    p.interval.String(),
		"storesCount": 0,
	}
}

// tick 每次触发时执行一次API调用
func (p *Poller) tick(ctx context.Context) {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	startTime := time.Now()

	p.mu.Lock()
	if p.dailyCount >= p.dailyLimit {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	// 获取所有已授权的抖音店铺
	stores, err := g.DB().Model("store").Ctx(ctx).
		Where("platform", "douyin").Where("status", 1).Where("author", 1).All()
	if err != nil {
		g.Log().Errorf(ctx, "[DouyinPoller] 获取店铺列表失败: %v", err)
		return
	}
	if len(stores) == 0 {
		return
	}

	// 遍历所有店铺，每个店铺翻页拉取其增量订单
	cst := time.FixedZone("CST", 8*3600)
	now := time.Now().In(cst)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cst)
	totalSavedAll := 0
	totalOrdersAll := 0

	for _, store := range stores {
		storeId := store["id"].Int()
		storeName := store["store_name"].String()

		// 读取该店铺上次成功拉取时间
		lastPollStr := store["last_poll_time"].String()
		var startTimeToUse time.Time
		if lastPollStr != "" {
			lpt, parseErr := time.ParseInLocation("2006-01-02 15:04:05", lastPollStr, cst)
			if parseErr == nil {
				startTimeToUse = lpt
			} else {
				startTimeToUse = todayStart
			}
		} else {
			startTimeToUse = todayStart
		}
		endTime := now

		taskName := fmt.Sprintf("抖店订单轮询-%s", storeName)
		storeSaved := 0
		storeOrders := 0
		storeSuccess := true

		// 翻页拉取，直到拉完为止
		for page := 0; ; page++ {
			// 检查每日配额
			p.mu.Lock()
			if p.dailyCount >= p.dailyLimit {
				p.mu.Unlock()
				g.Log().Debug(ctx, "[DouyinPoller] 已达每日上限，停止翻页")
				goto nextStore
			}
			p.dailyCount++
			p.mu.Unlock()

			reqParams, _ := json.Marshal(map[string]interface{}{
				"shop_id":      store["store_id"].String(),
				"store_id":     storeId,
				"start_time":   startTimeToUse.Format("2006-01-02 15:04:05"),
				"end_time":     endTime.Format("2006-01-02 15:04:05"),
				"page":         page,
				"page_size":    100,
				"order_status": douyinValidOrderStatusFilter,
			})

			pageStart := time.Now()
			logId, logErr := service.TaskLogStart(ctx, taskName, "douyin_poller",
				"/order/searchList", string(reqParams))
			if logErr != nil {
				g.Log().Errorf(ctx, "[DouyinPoller] 写入开始日志失败: %v", logErr)
			}

			svc := NewService()
			result, err := svc.GetOrderList(ctx, model.PlatformOrderListParams{
				Platform:      "douyin",
				Page:          page,
				PageSize:      100,
				StartTime:     startTimeToUse.Format("2006-01-02 15:04:05"),
				EndTime:       endTime.Format("2006-01-02 15:04:05"),
				StoreId:       storeId,
				OrderStatus:   douyinValidOrderStatusFilter,
				UseUpdateTime: true,
			})
			dur := int(time.Since(pageStart).Milliseconds())

			if err != nil {
				storeSuccess = false
				errMsg := err.Error()
				if strings.Contains(errMsg, "未找到授权记录") || strings.Contains(errMsg, "access_token") {
					g.Log().Warningf(ctx, "[DouyinPoller] %s 尚未授权，跳过", storeName)
					if logId > 0 {
						service.TaskLogSkip(ctx, taskName, "douyin_poller", "/order/searchList", errMsg)
					}
				} else {
					g.Log().Errorf(ctx, "[DouyinPoller] %s 第%d页失败: %v", storeName, page, err)
					if logId > 0 {
						service.TaskLogFailed(ctx, logId, errMsg, dur)
					}
				}
				break
			}

			respSummary, _ := json.Marshal(map[string]interface{}{
				"total": result.Total, "page": result.Page,
				"size": result.Size, "order_count": len(result.Orders),
			})
			if logId > 0 {
				service.TaskLogSuccess(ctx, logId, "10000", string(respSummary), dur)
			}

			saved := p.saveOrders(ctx, result.Orders)
			storeSaved += saved
			storeOrders += len(result.Orders)

			// 还有下一页？
			totalPages := 0
			if result.Size > 0 {
				totalPages = (result.Total + result.Size - 1) / result.Size
			}
			if page+1 >= totalPages || len(result.Orders) == 0 {
				break
			}
		}

		// 全部页成功 → 持久化 last_poll_time
		if storeSuccess && storeOrders > 0 {
			g.DB().Model("store").Ctx(ctx).Where("id", storeId).
				Data(g.Map{"last_poll_time": endTime.Format("2006-01-02 15:04:05")}).Update()
		}

	nextStore:
		totalSavedAll += storeSaved
		totalOrdersAll += storeOrders
		g.Log().Debugf(ctx, "[DouyinPoller] %s: %d条/%d条保存", storeName, storeOrders, storeSaved)
	}

	durationMs := int(time.Since(startTime).Milliseconds())
	p.mu.Lock()
	used := p.dailyCount
	p.mu.Unlock()
	g.Log().Infof(ctx, "[DouyinPoller] 本轮完成: %d个店铺, 共%d条/%d条保存, %dms, 当日%d/%d",
		len(stores), totalOrdersAll, totalSavedAll, durationMs, used, p.dailyLimit)
}

// saveOrders 保存订单到 order_info，按 sub_order_no 去重
func (p *Poller) saveOrders(ctx context.Context, orders []model.PlatformOrderItem) int {
	if len(orders) == 0 {
		return 0
	}

	// 批量查询 seller_code → base_price 映射
	priceMap := make(map[string]float64)
	rows, _ := g.DB().Model("goods").Ctx(ctx).Fields("goods_code, base_price").All()
	for _, r := range rows {
		priceMap[r["goods_code"].String()] = r["base_price"].Float64()
	}

	saved := 0
	for _, order := range orders {
		data := p.orderToMap(order)
		// 商品成本 = base_price × product_num
		if bp, ok := priceMap[order.Code]; ok && bp > 0 {
			data["product_cost"] = bp * float64(order.ProductNum)
		}
		// 小店服务费按商品原价总额计算，优惠后的实付金额不参与该口径。
		data["shop_service_fee"] = order.ProductPrice * float64(order.ProductNum) * service.DouyinShopServiceFeeRate

		count, _ := g.DB().Model("order_info").Ctx(ctx).
			Where("sub_order_no", order.SubOrderNo).Count()
		if count > 0 {
			_, err := g.DB().Model("order_info").Ctx(ctx).
				Where("sub_order_no", order.SubOrderNo).
				Data(data).Update()
			if err != nil {
				g.Log().Errorf(ctx, "[DouyinPoller] 更新订单失败 sub_order_no=%s: %v", order.SubOrderNo, err)
				continue
			}
		} else {
			data["create_time"] = time.Now().Format("2006-01-02 15:04:05")
			_, err := g.DB().Model("order_info").Ctx(ctx).Data(data).Insert()
			if err != nil {
				g.Log().Errorf(ctx, "[DouyinPoller] 插入订单失败 sub_order_no=%s product_name=%s: %v",
					order.SubOrderNo, truncateStr(order.ProductName, 100), err)
				continue
			}
		}
		saved++
	}
	return saved
}

func douyinOrderStatusDesc(order model.PlatformOrderItem) string {
	if order.OrderStatus == 4 {
		return "已关闭"
	}
	return order.OrderStatusDesc
}

func (p *Poller) orderToMap(order model.PlatformOrderItem) map[string]interface{} {
	m := map[string]interface{}{
		"sub_order_no":        order.SubOrderNo,
		"main_order_no":       order.OrderNo,
		"shop_name":           order.ShopName,
		"product_name":        order.ProductName,
		"product_num":         order.ProductNum,
		"product_price":       order.ProductPrice,
		"order_pay_amount":    order.PayAmount,
		"order_status":        douyinOrderStatusDesc(order),
		"talent_name":         order.AuthorName,
		"total_discount":      order.PromotionAmount,
		"platform_discount":   order.PromotionPlatformAmount,
		"talent_discount":     order.PromotionTalentAmount,
		"seller_change_price": order.ModifyAmount,
		"pay_discount":        order.PromotionPayAmount,
		"shipping_fee":        order.PostAmount,
	}
	setIf := func(key, val string) {
		if val != "" {
			m[key] = val
		}
	}
	setIf("product_id", fmt.Sprintf("%d", order.ProductId))
	setIf("seller_code", order.Code)
	setIf("pay_finish_time", order.PayTime)
	setIf("talent_id", fmt.Sprintf("%d", order.AuthorId))
	setIf("order_submit_time", order.CreateTime)
	setIf("order_finish_time", order.FinishTime)
	setIf("send_time", order.ShipTime)
	setIf("receiver_name", order.PostReceiver)
	setIf("receiver_phone", order.PostTel)
	setIf("province", order.PostProvince)
	setIf("city", order.PostCity)
	setIf("district", order.PostTown)
	setIf("address_detail", order.PostDetail)
	setIf("buyer_note", order.BuyerWords)
	setIf("seller_note", order.SellerWords)
	setIf("pay_type", fmt.Sprintf("%d", order.PayType))
	setIf("flow_source", order.BTypeDesc)
	setIf("after_sale_status", fmt.Sprintf("%d", order.AfterSaleStatus))
	setIf("cancel_reason", order.CancelReason)
	return m
}

// ManualTrigger 手动触发一次订单拉取（不改变定时调度）
func (p *Poller) ManualTrigger(ctx context.Context, opt PollParams) (*PollResult, error) {
	g.Log().Info(ctx, "[DouyinPoller] 手动触发订单拉取 start=%s end=%s", opt.StartTime, opt.EndTime)
	totalOrders, totalSaved, err := p.tickWithParams(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &PollResult{
		StoreCount:  1,
		TotalOrders: totalOrders,
		TotalSaved:  totalSaved,
	}, nil
}

// tickWithParams 使用指定时间参数拉取订单
func (p *Poller) tickWithParams(ctx context.Context, opt PollParams) (totalOrders, totalSaved int, err error) {
	startTime := time.Now()

	stores, dbErr := g.DB().Model("store").Ctx(ctx).
		Where("platform", "douyin").Where("status", 1).Where("author", 1).All()
	if dbErr != nil {
		return 0, 0, dbErr
	}
	if len(stores) == 0 {
		return 0, 0, nil
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, store := range stores {
		storeId := store["id"].Int()
		storeName := store["store_name"].String()

		cst := time.FixedZone("CST", 8*3600)
		var startTimeToUse, endTimeToUse time.Time
		if opt.StartTime != "" {
			startTimeToUse, _ = time.ParseInLocation("2006-01-02 15:04:05", opt.StartTime, cst)
		}
		if opt.EndTime != "" {
			endTimeToUse, _ = time.ParseInLocation("2006-01-02 15:04:05", opt.EndTime, cst)
		}
		if startTimeToUse.IsZero() {
			lastPollStr := store["last_poll_time"].String()
			if lastPollStr != "" {
				if lpt, pe := time.ParseInLocation("2006-01-02 15:04:05", lastPollStr, cst); pe == nil {
					startTimeToUse = lpt
				}
			}
			if startTimeToUse.IsZero() {
				startTimeToUse = todayStart
			}
		}
		if endTimeToUse.IsZero() {
			endTimeToUse = now
		}

		timeRange := startTimeToUse.Format("2006-01-02 15:04:05") + " ~ " + endTimeToUse.Format("2006-01-02 15:04:05")
		g.Log().Infof(ctx, "[DouyinPoller] %s 手动触发: opt.Start=%s opt.End=%s 实际拉取: %s",
			storeName, opt.StartTime, opt.EndTime, timeRange)

		st := startTimeToUse.Format("2006-01-02 15:04:05")
		et := endTimeToUse.Format("2006-01-02 15:04:05")

		// 第一步：调第0页（或指定起始页）获取 total
		svc := NewService()
		firstPage := opt.StartPage
		firstResult, apiErr := svc.GetOrderList(ctx, model.PlatformOrderListParams{
			Platform: "douyin", Page: firstPage, PageSize: 100,
			StartTime: st, EndTime: et, StoreId: storeId, OrderStatus: douyinValidOrderStatusFilter,
			UseUpdateTime: opt.UseUpdateTime,
		})
		if apiErr != nil {
			g.Log().Errorf(ctx, "[DouyinPoller] %s 第%d页失败: %v", storeName, firstPage, apiErr)
			continue
		}
		saved := p.saveOrders(ctx, firstResult.Orders)
		g.Log().Infof(ctx, "[DouyinPoller] %s 第%d页: %d条保存, total=%d", storeName, firstPage, len(firstResult.Orders), firstResult.Total)

		totalPages := 0
		if firstResult.Size > 0 {
			totalPages = (firstResult.Total + firstResult.Size - 1) / firstResult.Size
		}
		storeOrders := len(firstResult.Orders)
		storeSaved := saved

		// 如果没有更多页，直接返回
		if firstPage+1 >= totalPages || len(firstResult.Orders) == 0 {
			g.Log().Infof(ctx, "[DouyinPoller] %s: %d条/%d条保存 (单页完成)", storeName, storeOrders, storeSaved)
			totalOrders += storeOrders
			totalSaved += storeSaved
			continue
		}

		// 第二步：并发拉取剩余页
		concurrency := opt.Concurrency
		if concurrency <= 0 {
			concurrency = 5 // 默认5并发
		}
		if concurrency > 45 {
			concurrency = 45
		}

		// 速率限制器：45次/秒
		rateLimiter := make(chan struct{}, concurrency)
		go func() {
			ticker := time.NewTicker(time.Second / 45)
			defer ticker.Stop()
			for range ticker.C {
				select {
				case rateLimiter <- struct{}{}:
				default:
				}
			}
		}()

		var mu sync.Mutex
		var wg sync.WaitGroup
		var failPages []int
		pageCh := make(chan int, totalPages-firstPage-1)
		for page := firstPage + 1; page < totalPages; page++ {
			pageCh <- page
		}
		close(pageCh)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerId int) {
				defer wg.Done()
				svc := NewService()
				for page := range pageCh {
					<-rateLimiter
					var result *model.PlatformOrderListResult
					var apiErr error
					// 失败重试3次
					for retry := 0; retry < 3; retry++ {
						result, apiErr = svc.GetOrderList(ctx, model.PlatformOrderListParams{
							Platform: "douyin", Page: page, PageSize: 100,
							StartTime: st, EndTime: et, StoreId: storeId, OrderStatus: douyinValidOrderStatusFilter,
							UseUpdateTime: opt.UseUpdateTime,
						})
						if apiErr == nil {
							break
						}
						if retry < 2 {
							time.Sleep(time.Duration(retry+1) * time.Second)
						}
					}
					if apiErr != nil {
						g.Log().Errorf(ctx, "[DouyinPoller] %s worker%d 第%d页失败(重试3次): %v", storeName, workerId, page, apiErr)
						mu.Lock()
						failPages = append(failPages, page)
						mu.Unlock()
						continue
					}
					saved := p.saveOrders(ctx, result.Orders)
					mu.Lock()
					storeOrders += len(result.Orders)
					storeSaved += saved
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()

		g.Log().Infof(ctx, "[DouyinPoller] %s: %d条/%d条保存 (并发%d, 共%d页, 失败%d页)",
			storeName, storeOrders, storeSaved, concurrency, totalPages, len(failPages))
		totalOrders += storeOrders
		totalSaved += storeSaved
	}

	_ = time.Since(startTime)
	return totalOrders, totalSaved, nil
}

// truncateStr 截断字符串到指定长度
func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
