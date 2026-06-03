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

// GetPoller 获取全局轮询器实例
func GetPoller() *Poller {
	pollerOnce.Do(func() {
		pollerInstance = &Poller{
			dailyLimit: 1000,
			interval:   time.Duration(86400/1000) * time.Second,
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
	_, err = gcron.AddSingleton(ctx, "1 0 * * *", func(ctx context.Context) {
		p.mu.Lock()
		p.dailyCount = 0
		p.mu.Unlock()
		g.Log().Info(ctx, "[DouyinPoller] 每日API调用计数已重置")
	}, "douyin_poller_reset")
	if err != nil {
		return fmt.Errorf("注册每日重置定时任务失败: %w", err)
	}

	// 每天凌晨3点清理7天前的历史日志详情
	_, err = gcron.AddSingleton(ctx, "0 3 * * *", func(ctx context.Context) {
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
	defer p.mu.Unlock()
	p.running = false
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
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	const maxPages = 5

	totalSavedAll := 0
	totalOrdersAll := 0

	for _, store := range stores {
		storeId := store["id"].Int()
		storeName := store["store_name"].String()

		// 读取该店铺上次成功拉取时间
		lastPollStr := store["last_poll_time"].String()
		var startTimeToUse time.Time
		if lastPollStr != "" {
			lpt, parseErr := time.Parse("2006-01-02 15:04:05", lastPollStr)
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

		// 翻页拉取
		for page := 0; page < maxPages; page++ {
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
				"shop_id":    store["store_id"].String(),
				"store_id":   storeId,
				"start_time": startTimeToUse.Format("2006-01-02 15:04:05"),
				"end_time":   endTime.Format("2006-01-02 15:04:05"),
				"page":       page,
				"page_size":  100,
			})

			pageStart := time.Now()
			logId, logErr := service.TaskLogStart(ctx, taskName, "douyin_poller",
				"/order/searchList", string(reqParams))
			if logErr != nil {
				g.Log().Errorf(ctx, "[DouyinPoller] 写入开始日志失败: %v", logErr)
			}

			svc := NewService()
			result, err := svc.GetOrderList(ctx, model.PlatformOrderListParams{
				Platform:  "douyin",
				Page:      page,
				PageSize:  100,
				StartTime: startTimeToUse.Format("2006-01-02 15:04:05"),
				EndTime:   endTime.Format("2006-01-02 15:04:05"),
				StoreId:   storeId,
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

// saveOrders 保存订单到 order_info
func (p *Poller) saveOrders(ctx context.Context, orders []model.PlatformOrderItem) int {
	if len(orders) == 0 {
		return 0
	}
	saved := 0
	for _, order := range orders {
		count, err := g.DB().Model("order_info").Ctx(ctx).
			Where("main_order_no", order.OrderNo).
			Where("order_no", order.SubOrderNo).
			Where("platform", "douyin").Count()
		if err != nil {
			continue
		}
		if count > 0 {
			g.DB().Model("order_info").Ctx(ctx).
				Where("main_order_no", order.OrderNo).
				Where("order_no", order.SubOrderNo).
				Data(p.orderToMap(order)).Update()
		} else {
			g.DB().Model("order_info").Ctx(ctx).Data(p.orderToMap(order)).Insert()
		}
		saved++
	}
	return saved
}

func (p *Poller) orderToMap(order model.PlatformOrderItem) map[string]interface{} {
	return map[string]interface{}{
		"platform":          order.Platform,
		"main_order_no":     order.OrderNo,
		"order_no":          order.SubOrderNo,
		"shop_name":         order.ShopName,
		"product_name":      order.ProductName,
		"product_num":       order.ProductNum,
		"product_price":     order.ProductPrice,
		"order_pay_amount":  order.PayAmount,
		"order_status":      order.OrderStatusDesc,
		"pay_finish_time":   order.PayTime,
		"talent_id":         fmt.Sprintf("%d", order.AuthorId),
		"talent_name":       order.AuthorName,
		"seller_code":       order.Code,
		"order_create_time": order.CreateTime,
		"order_finish_time": order.FinishTime,
		"post_amount":       order.PostAmount,
		"post_receiver":     order.PostReceiver,
		"post_tel":          order.PostTel,
		"post_addr":         order.PostAddr,
		"promotion_amount":  order.PromotionAmount,
		"buyer_words":       order.BuyerWords,
		"seller_words":      order.SellerWords,
	}
}
