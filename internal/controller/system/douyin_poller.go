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

	"github.com/gogf/gf/v2/database/gdb"
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
// 自动遍历所有启用的店铺，前端传: startTime, endTime, concurrency(可选,默认10)
func (c *DouyinPollerController) TriggerAlliance(r *ghttp.Request) {
	sTime := r.Get("startTime").String()
	eTime := r.Get("endTime").String()

	if sTime == "" || eTime == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "startTime和endTime不能为空"})
		return
	}

	concurrency := r.Get("concurrency", g.Cfg().MustGet(r.Context(), "task.concurrency", 10).Int()).Int()
	if concurrency <= 0 {
		concurrency = 10
	}

	// 查询所有启用的店铺
	stores, err := g.DB().Model("store").Where("status", 1).All()
	if err != nil || len(stores) == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "没有启用的店铺"})
		return
	}

	// 记录任务开始
	storeIds := make([]int, 0)
	storeNames := make([]string, 0)
	for _, s := range stores {
		storeIds = append(storeIds, s["id"].Int())
		storeNames = append(storeNames, s["store_name"].String())
	}
	reqParams, _ := json.Marshal(g.Map{"startTime": sTime, "endTime": eTime, "stores": storeNames, "concurrency": concurrency})
	logId, err := service.TaskLogStart(r.Context(), "精选联盟手动拉取", "alliance_manual_pull", "/api/douyin-poller/trigger-alliance", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[AlliancePoller] 写入任务日志失败: %v", err)
	}

	g.Log().Infof(r.Context(), "[AlliancePoller] 收到异步拉取请求: %s~%s, 店铺数=%d, 并发=%d, logId=%d",
		sTime, eTime, len(stores), concurrency, logId)

	// 异步执行，遍历所有店铺拉取
	go c.runAlliancePullAllStores(context.Background(), sTime, eTime, stores, concurrency, logId, r.GetCtxVar("userId").Int64())

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logId, "storeCount": len(stores)}, "msg": "任务已启动，后台拉取中"})
}

// runAlliancePullAllStores 并行拉取所有店铺的精选联盟数据
func (c *DouyinPollerController) runAlliancePullAllStores(ctx context.Context, sTime, eTime string, stores gdb.Result, concurrency int, logId, userId int64) {
	overallStart := time.Now()

	// 多店铺并行拉取
	var (
		mu          sync.Mutex
		wg          sync.WaitGroup
		totalResult struct {
			subOrderCount  int
			totalBatches   int
			successBatches int
			failBatches    int
			totalOrders    int
			totalSaved     int
		}
	)

	g.Log().Infof(ctx, "[AlliancePoller] 开始并行拉取 %d 个店铺", len(stores))

	for _, store := range stores {
		wg.Add(1)
		go func(store gdb.Record) {
			defer wg.Done()

			storeId := store["id"].Int()
			storeName := store["store_name"].String()

			g.Log().Infof(ctx, "[AlliancePoller] ========== 开始拉取店铺: %s (ID=%d) ==========", storeName, storeId)

			result := c.runAlliancePull(ctx, sTime, eTime, storeId, concurrency/len(stores))

			mu.Lock()
			totalResult.subOrderCount += result.subOrderCount
			totalResult.totalBatches += result.totalBatches
			totalResult.successBatches += result.successBatches
			totalResult.failBatches += result.failBatches
			totalResult.totalOrders += result.totalOrders
			totalResult.totalSaved += result.totalSaved
			mu.Unlock()

			g.Log().Infof(ctx, "[AlliancePoller] 店铺 %s 完成: 订单=%d, 保存=%d", storeName, result.totalOrders, result.totalSaved)
		}(store)
	}

	wg.Wait()

	durationMs := int(time.Since(overallStart).Milliseconds())
	g.Log().Infof(ctx, "[AlliancePoller] 全部店铺完成: %d组/%d成功, %d条订单, 保存%d条, %dms",
		totalResult.totalBatches, totalResult.successBatches, totalResult.totalOrders, totalResult.totalSaved, durationMs)

	// 写入任务完成日志
	if logId > 0 {
		respBody, _ := json.Marshal(g.Map{
			"subOrderCount":  totalResult.subOrderCount,
			"totalBatches":   totalResult.totalBatches,
			"successBatches": totalResult.successBatches,
			"failBatches":    totalResult.failBatches,
			"totalOrders":    totalResult.totalOrders,
			"totalSaved":     totalResult.totalSaved,
			"durationMs":     durationMs,
		})
		service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
		level := service.NotifyLevelNormal
		if totalResult.failBatches > 0 {
			level = service.NotifyLevelImportant
		}
		// 增量回填达人信息到 order_info（只处理本次拉取的时间范围）
		backfilled, backfillErr := service.NewAllianceService().BackfillTalentInfoIncremental(ctx, sTime, eTime)
		if backfillErr != nil {
			g.Log().Errorf(ctx, "[AlliancePoller] 回填达人信息失败: %v", backfillErr)
		} else if backfilled > 0 {
			g.Log().Infof(ctx, "[AlliancePoller] 回填达人信息: %d条", backfilled)
		} else {
			g.Log().Infof(ctx, "[AlliancePoller] 没有需要回填的达人信息")
		}
		service.CreateNotification(ctx, "精选联盟拉取完成", string(respBody), "task_result", "alliance_manual_pull", logId, level, userId)
	}
}

// AlliancePullResult 精选联盟拉取结果
type AlliancePullResult struct {
	subOrderCount  int
	totalBatches   int
	successBatches int
	failBatches    int
	totalOrders    int
	totalSaved     int
}

// runAlliancePull 单个店铺的精选联盟拉取（优化版：拉取时直接回填）
func (c *DouyinPollerController) runAlliancePull(ctx context.Context, sTime, eTime string, storeId, concurrency int) AlliancePullResult {
	overallStart := time.Now()
	result := AlliancePullResult{}

	// ========== Step0: 获取店铺信息 ==========
	store, err := g.DB().Model("store").Where("id", storeId).One()
	if err != nil || store.IsEmpty() {
		g.Log().Errorf(ctx, "[AlliancePoller] 查询店铺失败: storeId=%d, err=%v", storeId, err)
		return result
	}
	storeName := store["store_name"].String()
	douyinStoreId := store["store_id"].String()

	// ========== Step0.5: 预加载 account 映射表（short_id → uid, account_name）==========
	accountMap := make(map[string]struct{ uid, name string })
	accountRows, _ := g.DB().Model("account").
		Fields("uid, platform_nickname, account_name").
		Where("platform", "douyin").
		Where("platform_nickname IS NOT NULL AND platform_nickname != ''").
		All()
	for _, row := range accountRows {
		nickname := row["platform_nickname"].String()
		if nickname != "" {
			accountMap[nickname] = struct{ uid, name string }{
				uid:  row["uid"].String(),
				name: row["account_name"].String(),
			}
		}
	}
	g.Log().Infof(ctx, "[AlliancePoller] 预加载 account 映射: %d 条", len(accountMap))

	// ========== Step1: 查询该店铺的 order_info 数据，构建 map ==========
	g.Log().Infof(ctx, "[AlliancePoller] 开始查询 order_info: 店铺=%s(store_id=%s), 时间=%s ~ %s",
		storeName, douyinStoreId, sTime, eTime)

	type OrderInfo struct {
		SubOrderNo      string
		ProductName     string
		ProductNum      int
		ProductPrice    float64
		ShopName        string
		PayFinishTime   string
		OrderSubmitTime string
		OrderFinishTime string
	}

	orderInfoMap := make(map[string]*OrderInfo)

	// 使用下单时间查询 order_info
	query := g.DB().Model("order_info").
		Fields("sub_order_no, product_name, product_num, product_price, shop_name, pay_finish_time, order_submit_time, order_finish_time").
		Where("order_submit_time >= ?", sTime).
		Where("order_submit_time <= ?", eTime).
		Where("sub_order_no IS NOT NULL AND sub_order_no != ''")

	if douyinStoreId != "" {
		hasStoreId, _ := g.DB().Model("order_info").
			Where("store_id", douyinStoreId).
			Where("order_submit_time >= ?", sTime).
			Where("order_submit_time <= ?", eTime).
			Count()
		if hasStoreId > 0 {
			query = query.Where("store_id", douyinStoreId)
		} else {
			query = query.Where("shop_name", storeName)
		}
	} else {
		query = query.Where("shop_name", storeName)
	}

	rows, err := query.All()
	if err != nil {
		g.Log().Errorf(ctx, "[AlliancePoller] 查询order_info失败: %v", err)
		return result
	}

	var allOrderNos []string
	for _, row := range rows {
		subNo := row["sub_order_no"].String()
		if subNo == "" {
			continue
		}
		allOrderNos = append(allOrderNos, subNo)
		orderInfoMap[subNo] = &OrderInfo{
			SubOrderNo:      subNo,
			ProductName:     row["product_name"].String(),
			ProductNum:      row["product_num"].Int(),
			ProductPrice:    row["product_price"].Float64(),
			ShopName:        row["shop_name"].String(),
			PayFinishTime:   row["pay_finish_time"].String(),
			OrderSubmitTime: row["order_submit_time"].String(),
			OrderFinishTime: row["order_finish_time"].String(),
		}
	}

	result.subOrderCount = len(allOrderNos)
	g.Log().Infof(ctx, "[AlliancePoller] 查询到 %d 条 sub_order_no, 耗时 %dms",
		result.subOrderCount, time.Since(overallStart).Milliseconds())

	if result.subOrderCount == 0 {
		return result
	}

	// ========== Step2: 并发调API，每批20个订单（API限制） ==========
	apiBatchSize := 20
	groups := chunkSlice(allOrderNos, apiBatchSize)
	result.totalBatches = len(groups)

	type taskResult struct {
		success    bool
		orderCount int
		savedCount int
		errMsg     string
	}

	var (
		mu      sync.Mutex
		dbMu    sync.Mutex  // 数据库写锁，避免并发写入冲突
		wg      sync.WaitGroup
		taskCh  = make(chan []string, len(groups))
		taskResults []taskResult
	)

	g.Log().Infof(ctx, "[AlliancePoller] %d批, %d并发, 开始拉取API", len(groups), concurrency)

	// 预创建服务实例池
	svcPool := make([]*douyinSvc.Service, concurrency)
	for i := 0; i < concurrency; i++ {
		svcPool[i] = douyinSvc.NewService()
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(svcIdx int) {
			defer wg.Done()
			svc := svcPool[svcIdx]

			for group := range taskCh {
				orderIds := strings.Join(group, ",")

				tr := taskResult{success: true}
				pageStart := time.Now()
				apiResult, err := svc.GetAllianceOrderList(ctx, model.PlatformAllianceOrderListParams{
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
					// 批量收集数据
					var batchData []g.Map
					// 收集需要回填的订单（short_id → order_id）
					backfillOrders := make(map[string][]string) // uid -> []order_id

					for _, o := range apiResult.Orders {
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
						// 从预加载的 map 中获取订单信息
						if oi, ok := orderInfoMap[subNo]; ok {
							data["product_name"] = oi.ProductName
							data["product_num"] = oi.ProductNum
							data["product_price"] = oi.ProductPrice
							data["shop_name"] = oi.ShopName
							data["pay_time"] = oi.PayFinishTime
							data["create_time"] = oi.OrderSubmitTime
							data["finish_time"] = oi.OrderFinishTime
						}
						batchData = append(batchData, data)

						// 收集需要回填的订单
						if o.ShortId != "" {
							if acc, ok := accountMap[o.ShortId]; ok && acc.uid != "" {
								backfillOrders[acc.uid] = append(backfillOrders[acc.uid], subNo)
							}
						}
					}

					// 批量 UPSERT alliance_orders（加锁避免并发冲突）
					if len(batchData) > 0 {
						dbMu.Lock()
						batchUpsertAllianceOrders(ctx, batchData)
						dbMu.Unlock()
						tr.savedCount = len(batchData)
					}

					// 直接回填 order_info（跳过后续的回填步骤）
					if len(backfillOrders) > 0 {
						dbMu.Lock()
						batchBackfillOrderInfo(ctx, backfillOrders, accountMap)
						dbMu.Unlock()
					}

					tr.orderCount = len(apiResult.Orders)
				}

				logThirdParty("douyin", "queryShopAllianceOrder",
					"/alliance/getOrderList", map[string]interface{}{"orderIds": orderIds, "size": 20}, dur, apiResult, err)

				mu.Lock()
				taskResults = append(taskResults, tr)
				mu.Unlock()
			}
		}(i)
	}

	for _, g := range groups {
		taskCh <- g
	}
	close(taskCh)
	wg.Wait()

	// ========== Step3: 汇总 ==========
	for _, r := range taskResults {
		if r.success {
			result.successBatches++
		} else {
			result.failBatches++
		}
		result.totalOrders += r.orderCount
		result.totalSaved += r.savedCount
	}

	durationMs := int(time.Since(overallStart).Milliseconds())
	g.Log().Infof(ctx, "[AlliancePoller] 完成: %d组/%d成功, %d条订单, 保存%d条, %dms",
		len(groups), result.successBatches, result.totalOrders, result.totalSaved, durationMs)

	return result
}

// batchUpsertAllianceOrders 批量插入或更新精选联盟订单
func batchUpsertAllianceOrders(ctx context.Context, batchData []g.Map) {
	if len(batchData) == 0 {
		return
	}

	// 使用 INSERT ... ON DUPLICATE KEY UPDATE 批量写入
	sql := `INSERT INTO alliance_orders (
		platform, order_id, product_id, product_name, product_num, product_price,
		pay_amount, author_account, commission_rate, estimated_comission, real_comission,
		total_pay_amount, order_status, shop_id, shop_name, pay_time, create_time, finish_time,
		alliance_biz_type, phase_id, short_id, created_at, updated_at
	) VALUES `

	var placeholders []string
	var args []interface{}
	now := time.Now().Format("2006-01-02 15:04:05")

	for _, data := range batchData {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		// 处理空值，避免 null 插入到 NOT NULL 字段
		payAmount := data["pay_amount"]
		if payAmount == nil {
			payAmount = 0
		}
		productPrice := data["product_price"]
		if productPrice == nil {
			productPrice = 0
		}
		productNum := data["product_num"]
		if productNum == nil {
			productNum = 0
		}
		commissionRate := data["commission_rate"]
		if commissionRate == nil {
			commissionRate = "0"
		}
		estimatedComission := data["estimated_comission"]
		if estimatedComission == nil {
			estimatedComission = 0
		}
		realComission := data["real_comission"]
		if realComission == nil {
			realComission = 0
		}
		totalPayAmount := data["total_pay_amount"]
		if totalPayAmount == nil {
			totalPayAmount = 0
		}

		// 处理 datetime 字段，空字符串转为 nil
		payTime := data["pay_time"]
		if payTime == nil || payTime == "" {
			payTime = nil
		}
		createTime := data["create_time"]
		if createTime == nil || createTime == "" {
			createTime = nil
		}
		finishTime := data["finish_time"]
		if finishTime == nil || finishTime == "" {
			finishTime = nil
		}

		args = append(args,
			data["platform"], data["order_id"], data["product_id"],
			data["product_name"], productNum, productPrice,
			payAmount, data["author_account"], commissionRate,
			estimatedComission, realComission, totalPayAmount,
			data["order_status"], data["shop_id"], data["shop_name"],
			payTime, createTime, finishTime,
			data["alliance_biz_type"], data["phase_id"], data["short_id"],
			now, now,
		)
	}

	sql += strings.Join(placeholders, ",")
	sql += ` ON DUPLICATE KEY UPDATE
		author_account = VALUES(author_account),
		short_id = VALUES(short_id),
		commission_rate = VALUES(commission_rate),
		order_status = VALUES(order_status),
		updated_at = VALUES(updated_at)`

	// 重试机制：遇到锁超时时重试
	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		_, err := g.DB().Exec(ctx, sql, args...)
		if err == nil {
			return
		}
		// 如果是锁超时错误，等待后重试
		if strings.Contains(err.Error(), "Lock wait timeout") {
			g.Log().Warningf(ctx, "[AlliancePoller] 锁超时，%d秒后重试 (%d/%d)", (retry+1)*2, retry+1, maxRetries)
			time.Sleep(time.Duration(retry+1) * 2 * time.Second)
			continue
		}
		// 其他错误直接报错
		g.Log().Errorf(ctx, "[AlliancePoller] 批量写入失败: %v", err)
		return
	}
	g.Log().Errorf(ctx, "[AlliancePoller] 批量写入失败，重试%d次后放弃", maxRetries)
}

// batchBackfillOrderInfo 批量回填 order_info 的达人信息
func batchBackfillOrderInfo(ctx context.Context, backfillOrders map[string][]string, accountMap map[string]struct{ uid, name string }) {
	for uid, orderIds := range backfillOrders {
		// 获取 account_name
		accountName := ""
		for _, acc := range accountMap {
			if acc.uid == uid {
				accountName = acc.name
				break
			}
		}

		// 分批更新（每批 500 个订单）
		batchSize := 500
		for i := 0; i < len(orderIds); i += batchSize {
			end := i + batchSize
			if end > len(orderIds) {
				end = len(orderIds)
			}
			batch := orderIds[i:end]

			placeholders := ""
			args := []interface{}{uid, accountName}
			for j, orderId := range batch {
				if j > 0 {
					placeholders += ","
				}
				placeholders += "?"
				args = append(args, orderId)
			}

			sql := fmt.Sprintf(`
				UPDATE order_info
				SET talent_id = ?, talent_name = ?
				WHERE sub_order_no IN (%s)
				  AND (talent_id IS NULL OR talent_id = '' OR talent_id = '0')
			`, placeholders)

			result, err := g.DB().Exec(ctx, sql, args...)
			if err != nil {
				g.Log().Warningf(ctx, "[Backfill] 直接回填失败 uid=%s: %v", uid, err)
				continue
			}
			n, _ := result.RowsAffected()
			if n > 0 {
				g.Log().Debugf(ctx, "[Backfill] 直接回填 uid=%s: %d 条", uid, n)
			}
		}
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
