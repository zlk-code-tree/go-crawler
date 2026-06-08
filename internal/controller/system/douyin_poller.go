package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service"
	douyinSvc "go-crawler/internal/service/platform/douyin"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type DouyinPollerController struct{}

func NewDouyinPollerController() *DouyinPollerController {
	return &DouyinPollerController{}
}

// Start 启动抖店订单轮询 POST /api/douyin-poller/start
func (c *DouyinPollerController) Start(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	if poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已在运行"})
		return
	}
	if err := poller.Start(r.Context()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "启动失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已启动"})
}

// Stop 停止轮询 POST /api/douyin-poller/stop
func (c *DouyinPollerController) Stop(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	if !poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器未在运行"})
		return
	}
	poller.Stop()
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已停止"})
}

// Status 获取轮询器状态 GET /api/douyin-poller/status
func (c *DouyinPollerController) Status(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": poller.Stats(),
		"msg":  "ok",
	})
}

	// TriggerOrder 手动触发抖店订单拉取 POST /api/douyin-poller/trigger-order
	func (c *DouyinPollerController) TriggerOrder(r *ghttp.Request) {
		opt := douyinSvc.PollParams{
			StartTime:   r.Get("startTime").String(),
			EndTime:     r.Get("endTime").String(),
			StartPage:   r.Get("startPage", 0).Int(),
			Concurrency: r.Get("concurrency", 5).Int(),
		}

		// 记录任务开始
		reqParams, _ := json.Marshal(g.Map{
			"startTime":   opt.StartTime,
			"endTime":     opt.EndTime,
			"startPage":   opt.StartPage,
			"concurrency": opt.Concurrency,
		})
		logId, err := service.TaskLogStart(r.Context(), "抖店订单手动拉取", "douyin_order_pull", "/api/douyin-poller/trigger-order", string(reqParams))
		if err != nil {
			g.Log().Errorf(r.Context(), "[DouyinPoller] 写入任务日志失败: %v", err)
		}

		g.Log().Infof(r.Context(), "[DouyinPoller] 收到手动拉取请求: startTime=%s endTime=%s logId=%d",
			opt.StartTime, opt.EndTime, logId)

		go c.runOrderPull(context.Background(), opt, logId, r.GetCtxVar("userId").Int64())

		r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logId}, "msg": "任务已启动，后台拉取中"})
	}

	// runOrderPull 在后台执行抖店订单拉取
	func (c *DouyinPollerController) runOrderPull(ctx context.Context, opt douyinSvc.PollParams, logId, userId int64) {
		overallStart := time.Now()
		result, err := douyinSvc.GetPoller().ManualTrigger(ctx, opt)
		durationMs := int(time.Since(overallStart).Milliseconds())

		if err != nil {
			if logId > 0 {
				service.TaskLogFailed(ctx, logId, err.Error(), durationMs)
				service.CreateNotification(ctx, "抖店订单拉取失败", fmt.Sprintf(`{"error":"%s"}`, err.Error()), "task_result", "douyin_order_pull", logId, service.NotifyLevelImportant, userId)
			}
			return
		}

		respBody, _ := json.Marshal(g.Map{
			"totalOrders": result.TotalOrders,
			"totalSaved":  result.TotalSaved,
			"durationMs":  durationMs,
		})
		if logId > 0 {
			service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
			service.CreateNotification(ctx, "抖店订单拉取完成", string(respBody), "task_result", "douyin_order_pull", logId, service.NotifyLevelNormal, userId)
		}
	}

// QueryShopAllianceOrder 透传调用 /buyin/queryShopAllianceOrder GET /api/douyin/query-shop-alliance-order
func (c *DouyinPollerController) QueryShopAllianceOrder(r *ghttp.Request) {
	svc := douyinSvc.NewService()
	token, _, err := svc.GetTokenForStore(r.Context(), r.Get("storeId", 1).Int())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "获取token失败: " + err.Error()})
		return
	}

	resp, err := svc.QueryShopAllianceOrder(r.Context(), token, douyinSvc.AllianceOrderQuery{
		Size:      r.Get("size", 20).Int64(),
		Cursor:    r.Get("cursor").String(),
		StartTime: r.Get("startTime").String(),
		EndTime:   r.Get("endTime").String(),
		OrderIds:  r.Get("orderIds").String(),
		TimeType:  r.Get("timeType").String(),
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "调用失败: " + err.Error()})
		return
	}
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.WriteExit(resp)
}

// TriggerAlliance 精选联盟全量拉取（异步执行）
// 前端传: startTime, endTime, concurrency(可选,默认10)
func (c *DouyinPollerController) TriggerAlliance(r *ghttp.Request) {
	sTime := r.Get("startTime").String()
	eTime := r.Get("endTime").String()
	storeId := r.Get("store_id", 1).Int()

	if sTime == "" || eTime == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "startTime和endTime不能为空"})
		return
	}

	concurrency := r.Get("concurrency", g.Cfg().MustGet(r.Context(), "task.concurrency", 10).Int()).Int()
	if concurrency <= 0 {
		concurrency = 10
	}

	// 记录任务开始
	reqParams, _ := json.Marshal(g.Map{"startTime": sTime, "endTime": eTime, "storeId": storeId, "concurrency": concurrency})
	logId, err := service.TaskLogStart(r.Context(), "精选联盟手动拉取", "alliance_manual_pull", "/api/douyin-poller/trigger-alliance", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[AlliancePoller] 写入任务日志失败: %v", err)
	}

	g.Log().Infof(r.Context(), "[AlliancePoller] 收到异步拉取请求: %s~%s, storeId=%d, 并发=%d, logId=%d",
		sTime, eTime, storeId, concurrency, logId)

	// 异步执行，避免前端请求超时
	go c.runAlliancePull(context.Background(), sTime, eTime, storeId, concurrency, logId, r.GetCtxVar("userId").Int64())

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logId}, "msg": "任务已启动，后台拉取中"})
}

// runAlliancePull 在后台执行精选联盟全量拉取
func (c *DouyinPollerController) runAlliancePull(ctx context.Context, sTime, eTime string, storeId, concurrency int, logId, userId int64) {
	overallStart := time.Now()

	// ========== Step1: 分批查 order_info 的 sub_order_no ==========
	dbBatchSize := 2000
	offset := 0
	var allOrderNos []string

	for {
		rows, err := g.DB().Model("order_info").
			Fields("DISTINCT sub_order_no").
			Where("pay_finish_time >= ?", sTime).
			Where("pay_finish_time <= ?", eTime).
			Where("sub_order_no IS NOT NULL AND sub_order_no != ''").
			Order("sub_order_no ASC").
			Limit(dbBatchSize).Offset(offset).
			All()
		if err != nil {
			g.Log().Errorf(ctx, "[AlliancePoller] 查询订单号失败: %v", err)
			if logId > 0 {
				service.TaskLogFailed(ctx, logId, "查询订单号失败: "+err.Error(), int(time.Since(overallStart).Milliseconds()))
			service.CreateNotification(ctx, "精选联盟拉取失败", `{"error":"查询订单号失败"}`, "task_result", "alliance_manual_pull", logId, service.NotifyLevelImportant, userId)
			}
			return
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			allOrderNos = append(allOrderNos, row["sub_order_no"].String())
		}
		offset += dbBatchSize
		if len(rows) < dbBatchSize {
			break
		}
	}

	totalSubOrders := len(allOrderNos)
	if totalSubOrders == 0 {
		g.Log().Infof(ctx, "[AlliancePoller] 该时间范围内没有订单")
		if logId > 0 {
			respBody, _ := json.Marshal(g.Map{"subOrderCount": 0, "msg": "该时间范围内没有订单"})
			service.TaskLogSuccess(ctx, logId, "", string(respBody), int(time.Since(overallStart).Milliseconds()))
			service.CreateNotification(ctx, "精选联盟拉取完成", `{"subOrderCount":0,"msg":"该时间范围内没有订单"}`, "task_result", "alliance_manual_pull", logId, service.NotifyLevelNormal, userId)
		}
		return
	}

	// ========== Step2: 每20个一组，并发调API ==========
	groups := chunkSlice(allOrderNos, 20)

	type taskResult struct {
		success    bool
		orderCount int
		savedCount int
		errMsg     string
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		taskCh  = make(chan []string, len(groups))
		results []taskResult
	)

	g.Log().Infof(ctx, "[AlliancePoller] 查询到%d条sub_order_no, %d批, %d并发, 开始拉取", totalSubOrders, len(groups), concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := douyinSvc.NewService()
			for group := range taskCh {
				orderIds := strings.Join(group, ",")

				tr := taskResult{success: true}
				pageStart := time.Now()
				result, err := svc.GetAllianceOrderList(ctx, model.PlatformAllianceOrderListParams{
					Platform: "douyin",
					Cursor:   "0",
					Size:     20,
					StoreId:  storeId,
					OrderIds: orderIds,
				})
				dur := int(time.Since(pageStart).Milliseconds())

				if err != nil {
					tr.success = false
					tr.errMsg = err.Error()
					g.Log().Errorf(ctx, "[AlliancePoller] 拉取失败 orderIds=%s: %v", orderIds, err)
				} else {
					for _, o := range result.Orders {
						subNo := o.OrderId
						data := g.Map{
							"platform":            "douyin",
							"order_id":            subNo,
							"product_id":          o.ProductId,
							"author_account":      o.AuthorAccount,
							"commission_rate":     o.CommissionRate,
							"estimated_comission": o.EstimatedComission,
							"real_comission":      o.RealComission,
							"total_pay_amount":    o.TotalPayAmount,
							"order_status":        o.OrderStatus,
							"shop_id":             o.ShopId,
							"alliance_biz_type":   o.AllianceBizType,
							"phase_id":            o.PhaseId,
							"short_id":            o.ShortId,
						}
						oi, _ := g.DB().Model("order_info").
							Where("sub_order_no", subNo).One()
						if !oi.IsEmpty() {
							data["product_name"] = oi["product_name"]
							data["product_num"] = oi["product_num"]
							data["product_price"] = oi["product_price"]
							data["shop_name"] = oi["shop_name"]
							data["pay_time"] = oi["pay_finish_time"]
							data["create_time"] = oi["order_submit_time"]
							data["finish_time"] = oi["order_finish_time"]
						}
						count, _ := g.DB().Model("alliance_orders").
							Where("platform", "douyin").Where("order_id", subNo).Count()
						if count > 0 {
							g.DB().Model("alliance_orders").
								Where("platform", "douyin").Where("order_id", subNo).
								Data(data).Update()
						} else {
							g.DB().Model("alliance_orders").Data(data).Insert()
						}
						tr.savedCount++
					}
					tr.orderCount = len(result.Orders)
				}

				logThirdParty("douyin", "queryShopAllianceOrder",
					"/alliance/getOrderList", map[string]interface{}{"orderIds": orderIds, "size": 20}, dur, result, err)

				mu.Lock()
				results = append(results, tr)
				mu.Unlock()
			}
		}()
	}

	for _, g := range groups {
		taskCh <- g
	}
	close(taskCh)
	wg.Wait()

	// ========== Step3: 汇总 ==========
	successBatches := 0
	failBatches := 0
	totalOrders := 0
	totalSaved := 0
	for _, r := range results {
		if r.success {
			successBatches++
		} else {
			failBatches++
		}
		totalOrders += r.orderCount
		totalSaved += r.savedCount
	}

	durationMs := int(time.Since(overallStart).Milliseconds())
	g.Log().Infof(ctx, "[AlliancePoller] 完成: %d组/%d成功, %d条订单, 保存%d条, %dms",
		len(groups), successBatches, totalOrders, totalSaved, durationMs)

	// 写入任务完成日志
	if logId > 0 {
		respBody, _ := json.Marshal(g.Map{
			"subOrderCount":  totalSubOrders,
			"totalBatches":   len(groups),
			"successBatches": successBatches,
			"failBatches":    failBatches,
			"totalOrders":    totalOrders,
			"totalSaved":     totalSaved,
			"durationMs":     durationMs,
		})
		service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
			level := service.NotifyLevelNormal
			if failBatches > 0 {
				level = service.NotifyLevelImportant
			}
			service.CreateNotification(ctx, "精选联盟拉取完成", string(respBody), "task_result", "alliance_manual_pull", logId, level, userId)
	}
}

func chunkSlice(s []string, size int) [][]string {
	var chunks [][]string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

func logThirdParty(platform, apiName, apiPath string,
	params map[string]interface{}, durationMs int, resp interface{}, err error) {
	reqJSON, _ := json.Marshal(params)
	status := 1
	respCode := ""
	respBody := ""
	if err != nil {
		status = 0
	} else {
		if r, _ := json.Marshal(resp); r != nil {
			respBody = string(r)
		}
	}
	if len(respBody) > 50000 {
		respBody = respBody[:50000] + "...[truncated]"
	}
	g.DB().Model("third_party_api_log").Data(g.Map{
		"platform":       platform,
		"api_name":       apiName,
		"api_path":       apiPath,
		"request_params": string(reqJSON),
		"response_code":  respCode,
		"response_body":  respBody,
		"duration_ms":    durationMs,
		"status":         status,
	}).Insert()
}
