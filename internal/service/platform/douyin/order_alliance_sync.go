package douyin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

const allianceOrderBatchSize = 20

// OrderAllianceSyncResult 是订单、精选联盟和达人匹配三个阶段的统一结果。
type OrderAllianceSyncResult struct {
	Orders     *PollResult         `json:"orders"`
	Alliance   *AlliancePullResult `json:"alliance"`
	DurationMs int                 `json:"durationMs"`
}

// AlliancePullResult 记录精选联盟拉取及达人匹配结果。
type AlliancePullResult struct {
	StoreCount      int      `json:"storeCount"`
	SubOrderCount   int      `json:"subOrderCount"`
	TotalBatches    int      `json:"totalBatches"`
	SuccessBatches  int      `json:"successBatches"`
	FailedBatches   int      `json:"failedBatches"`
	TotalOrders     int      `json:"totalOrders"`
	TotalSaved      int      `json:"totalSaved"`
	MatchedOrders   int      `json:"matchedOrders"`
	UnmatchedOrders int      `json:"unmatchedOrders"`
	DurationMs      int      `json:"durationMs"`
	Errors          []string `json:"errors,omitempty"`
}

type allianceStoreOrders struct {
	StoreID   int
	StoreName string
	OrderIDs  []string
}

type allianceOrderInfo struct {
	ProductName     string
	ProductNum      int
	ProductPrice    float64
	ShopName        string
	PayFinishTime   string
	OrderSubmitTime string
	OrderFinishTime string
}

type allianceAccount struct {
	UID  string
	Name string
}

type allianceBatchTask struct {
	StoreID   int
	StoreName string
	OrderIDs  []string
}

type allianceBatchResult struct {
	Orders  int
	Saved   int
	Matched map[string]struct{}
	Error   error
}

// OrderAllianceSyncService 编排抖店订单、精选联盟及达人匹配。
type OrderAllianceSyncService struct{}

func NewOrderAllianceSyncService() *OrderAllianceSyncService {
	return &OrderAllianceSyncService{}
}

// Sync 先拉取订单，再仅按本次实际获取的子订单号拉取精选联盟并匹配达人。
// 订单部分失败时仍处理已成功的订单，但最终会返回部分失败错误供任务系统告警。
func (s *OrderAllianceSyncService) Sync(ctx context.Context, opt PollParams) (*OrderAllianceSyncResult, error) {
	startedAt := time.Now()
	orderResult, orderErr := GetPoller().ManualTrigger(ctx, opt)
	result := &OrderAllianceSyncResult{Orders: orderResult}
	if orderResult == nil {
		result.DurationMs = int(time.Since(startedAt).Milliseconds())
		return result, orderErr
	}

	storeOrders := make([]allianceStoreOrders, 0, len(orderResult.StoreResults))
	for _, storeResult := range orderResult.StoreResults {
		if len(storeResult.OrderIDs) == 0 {
			continue
		}
		storeOrders = append(storeOrders, allianceStoreOrders{
			StoreID:   storeResult.StoreID,
			StoreName: storeResult.StoreName,
			OrderIDs:  storeResult.OrderIDs,
		})
	}

	allianceResult, allianceErr := s.pullAlliance(ctx, storeOrders, opt.Concurrency)
	result.Alliance = allianceResult
	result.DurationMs = int(time.Since(startedAt).Milliseconds())
	return result, errors.Join(orderErr, allianceErr)
}

// PullAllianceByTime 保留精选联盟手工补拉能力，按下单时间查出订单号后复用统一匹配流程。
func (s *OrderAllianceSyncService) PullAllianceByTime(ctx context.Context, startTime, endTime string, concurrency int) (*AlliancePullResult, error) {
	stores, err := g.DB().Model("store").Ctx(ctx).
		Where("platform", "douyin").Where("status", 1).Where("author", 1).All()
	if err != nil {
		return nil, fmt.Errorf("查询抖音店铺失败: %w", err)
	}

	storeOrders := make([]allianceStoreOrders, 0, len(stores))
	for _, store := range stores {
		storeID := store["id"].Int()
		storeName := store["store_name"].String()
		douyinStoreID := store["store_id"].String()
		query := g.DB().Model("order_info").Ctx(ctx).
			Fields("sub_order_no").
			Where("order_submit_time >= ?", startTime).
			Where("order_submit_time <= ?", endTime).
			Where("sub_order_no IS NOT NULL AND sub_order_no != ''")

		if douyinStoreID != "" {
			hasStoreID, _ := g.DB().Model("order_info").Ctx(ctx).
				Where("store_id", douyinStoreID).
				Where("order_submit_time >= ?", startTime).
				Where("order_submit_time <= ?", endTime).Count()
			if hasStoreID > 0 {
				query = query.Where("store_id", douyinStoreID)
			} else {
				query = query.Where("shop_name", storeName)
			}
		} else {
			query = query.Where("shop_name", storeName)
		}

		rows, queryErr := query.All()
		if queryErr != nil {
			return nil, fmt.Errorf("查询店铺 %s 的订单失败: %w", storeName, queryErr)
		}
		orderIDs := make([]string, 0, len(rows))
		for _, row := range rows {
			if orderID := row["sub_order_no"].String(); orderID != "" {
				orderIDs = append(orderIDs, orderID)
			}
		}
		storeOrders = append(storeOrders, allianceStoreOrders{StoreID: storeID, StoreName: storeName, OrderIDs: uniqueStrings(orderIDs)})
	}
	return s.pullAlliance(ctx, storeOrders, concurrency)
}

func (s *OrderAllianceSyncService) pullAlliance(ctx context.Context, stores []allianceStoreOrders, concurrency int) (*AlliancePullResult, error) {
	startedAt := time.Now()
	result := &AlliancePullResult{StoreCount: len(stores)}
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 45 {
		concurrency = 45
	}

	allOrderIDs := make([]string, 0)
	tasks := make([]allianceBatchTask, 0)
	for _, store := range stores {
		orderIDs := uniqueStrings(store.OrderIDs)
		allOrderIDs = append(allOrderIDs, orderIDs...)
		for _, batch := range chunkStrings(orderIDs, allianceOrderBatchSize) {
			tasks = append(tasks, allianceBatchTask{StoreID: store.StoreID, StoreName: store.StoreName, OrderIDs: batch})
		}
	}
	allOrderIDs = uniqueStrings(allOrderIDs)
	result.SubOrderCount = len(allOrderIDs)
	result.TotalBatches = len(tasks)
	if len(tasks) == 0 {
		result.DurationMs = int(time.Since(startedAt).Milliseconds())
		return result, nil
	}

	orderInfoMap, err := loadAllianceOrderInfo(ctx, allOrderIDs)
	if err != nil {
		return result, err
	}
	accountMap, err := loadAllianceAccounts(ctx)
	if err != nil {
		return result, err
	}

	workerCount := concurrency
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}
	taskCh := make(chan allianceBatchTask, len(tasks))
	resultCh := make(chan allianceBatchResult, len(tasks))
	var wg sync.WaitGroup
	var dbMu sync.Mutex

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := NewService()
			for task := range taskCh {
				resultCh <- pullAllianceBatch(ctx, client, task, orderInfoMap, accountMap, &dbMu)
			}
		}()
	}
	for _, task := range tasks {
		taskCh <- task
	}
	close(taskCh)
	wg.Wait()
	close(resultCh)

	matchedOrderIDs := make(map[string]struct{})
	for batchResult := range resultCh {
		result.TotalOrders += batchResult.Orders
		result.TotalSaved += batchResult.Saved
		if batchResult.Error != nil {
			result.FailedBatches++
			if len(result.Errors) < 20 {
				result.Errors = append(result.Errors, batchResult.Error.Error())
			}
			continue
		}
		result.SuccessBatches++
		for orderID := range batchResult.Matched {
			matchedOrderIDs[orderID] = struct{}{}
		}
	}
	result.MatchedOrders = len(matchedOrderIDs)
	result.UnmatchedOrders = result.SubOrderCount - result.MatchedOrders
	if result.UnmatchedOrders < 0 {
		result.UnmatchedOrders = 0
	}
	result.DurationMs = int(time.Since(startedAt).Milliseconds())

	g.Log().Infof(ctx, "[OrderAllianceSync] 联盟完成: 订单号%d, 联盟订单%d, 保存%d, 匹配%d, 未匹配%d, 失败批次%d",
		result.SubOrderCount, result.TotalOrders, result.TotalSaved, result.MatchedOrders, result.UnmatchedOrders, result.FailedBatches)
	if result.FailedBatches > 0 {
		return result, fmt.Errorf("精选联盟拉取部分失败: %d/%d批", result.FailedBatches, result.TotalBatches)
	}
	return result, nil
}

func pullAllianceBatch(
	ctx context.Context,
	client *Service,
	task allianceBatchTask,
	orderInfoMap map[string]allianceOrderInfo,
	accountMap map[string]allianceAccount,
	dbMu *sync.Mutex,
) allianceBatchResult {
	orderIDs := strings.Join(task.OrderIDs, ",")
	startedAt := time.Now()
	var apiResult *model.PlatformAllianceOrderListResult
	var err error
	for retry := 0; retry < 3; retry++ {
		apiResult, err = client.GetAllianceOrderList(ctx, model.PlatformAllianceOrderListParams{
			Platform: "douyin",
			Cursor:   "0",
			Size:     allianceOrderBatchSize,
			StoreId:  task.StoreID,
			OrderIds: orderIDs,
		})
		if err == nil {
			break
		}
		if retry < 2 {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				retry = 3
			case <-time.After(time.Duration(retry+1) * time.Second):
			}
		}
	}
	logAllianceThirdParty(task, int(time.Since(startedAt).Milliseconds()), apiResult, err)
	if err != nil {
		return allianceBatchResult{Error: fmt.Errorf("店铺%s订单[%s]拉取失败: %w", task.StoreName, orderIDs, err)}
	}

	batchData := make([]g.Map, 0, len(apiResult.Orders))
	backfill := make(map[allianceAccount][]string)
	matched := make(map[string]struct{})
	for _, order := range apiResult.Orders {
		data := g.Map{
			"platform":            "douyin",
			"order_id":            order.OrderId,
			"product_id":          order.ProductId,
			"author_account":      order.AuthorAccount,
			"commission_rate":     order.CommissionRate,
			"estimated_comission": order.EstimatedComission,
			"real_comission":      order.RealComission,
			"total_pay_amount":    order.TotalPayAmount,
			"order_status":        order.OrderStatus,
			"shop_id":             order.ShopId,
			"alliance_biz_type":   order.AllianceBizType,
			"phase_id":            order.PhaseId,
			"short_id":            order.ShortId,
		}
		if info, ok := orderInfoMap[order.OrderId]; ok {
			data["product_name"] = info.ProductName
			data["product_num"] = info.ProductNum
			data["product_price"] = info.ProductPrice
			data["shop_name"] = info.ShopName
			data["pay_time"] = info.PayFinishTime
			data["create_time"] = info.OrderSubmitTime
			data["finish_time"] = info.OrderFinishTime
		}
		batchData = append(batchData, data)
		if account, ok := accountMap[order.ShortId]; ok && account.UID != "" {
			backfill[account] = append(backfill[account], order.OrderId)
			matched[order.OrderId] = struct{}{}
		}
	}

	dbMu.Lock()
	err = upsertAllianceOrders(ctx, batchData)
	if err == nil {
		_, err = backfillAllianceTalent(ctx, backfill)
	}
	dbMu.Unlock()
	if err != nil {
		return allianceBatchResult{Orders: len(apiResult.Orders), Error: fmt.Errorf("店铺%s联盟数据保存或回填失败: %w", task.StoreName, err)}
	}
	return allianceBatchResult{Orders: len(apiResult.Orders), Saved: len(batchData), Matched: matched}
}

func loadAllianceOrderInfo(ctx context.Context, orderIDs []string) (map[string]allianceOrderInfo, error) {
	result := make(map[string]allianceOrderInfo, len(orderIDs))
	for _, batch := range chunkStrings(orderIDs, 500) {
		rows, err := g.DB().Model("order_info").Ctx(ctx).
			Fields("sub_order_no, product_name, product_num, product_price, shop_name, pay_finish_time, order_submit_time, order_finish_time").
			WhereIn("sub_order_no", batch).All()
		if err != nil {
			return nil, fmt.Errorf("查询本次订单明细失败: %w", err)
		}
		for _, row := range rows {
			result[row["sub_order_no"].String()] = allianceOrderInfo{
				ProductName:     row["product_name"].String(),
				ProductNum:      row["product_num"].Int(),
				ProductPrice:    row["product_price"].Float64(),
				ShopName:        row["shop_name"].String(),
				PayFinishTime:   row["pay_finish_time"].String(),
				OrderSubmitTime: row["order_submit_time"].String(),
				OrderFinishTime: row["order_finish_time"].String(),
			}
		}
	}
	return result, nil
}

func loadAllianceAccounts(ctx context.Context) (map[string]allianceAccount, error) {
	rows, err := g.DB().Model("account").Ctx(ctx).
		Fields("uid, platform_nickname, account_name").
		Where("platform", "douyin").
		Where("platform_nickname IS NOT NULL AND platform_nickname != ''").All()
	if err != nil {
		return nil, fmt.Errorf("查询抖音达人账号映射失败: %w", err)
	}
	result := make(map[string]allianceAccount, len(rows))
	for _, row := range rows {
		shortID := row["platform_nickname"].String()
		if shortID != "" {
			result[shortID] = allianceAccount{UID: row["uid"].String(), Name: row["account_name"].String()}
		}
	}
	return result, nil
}

func upsertAllianceOrders(ctx context.Context, batchData []g.Map) error {
	if len(batchData) == 0 {
		return nil
	}
	sql := `INSERT INTO alliance_orders (
		platform, order_id, product_id, product_name, product_num, product_price,
		pay_amount, author_account, commission_rate, estimated_comission, real_comission,
		total_pay_amount, order_status, shop_id, shop_name, pay_time, create_time, finish_time,
		alliance_biz_type, phase_id, short_id, created_at, updated_at
	) VALUES `
	placeholders := make([]string, 0, len(batchData))
	args := make([]interface{}, 0, len(batchData)*23)
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, data := range batchData {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			data["platform"], data["order_id"], data["product_id"], valueOr(data["product_name"], ""),
			valueOr(data["product_num"], 0), valueOr(data["product_price"], 0), valueOr(data["pay_amount"], 0),
			data["author_account"], valueOr(data["commission_rate"], 0), valueOr(data["estimated_comission"], 0),
			valueOr(data["real_comission"], 0), valueOr(data["total_pay_amount"], 0), data["order_status"],
			data["shop_id"], valueOr(data["shop_name"], ""), nilIfEmpty(data["pay_time"]),
			nilIfEmpty(data["create_time"]), nilIfEmpty(data["finish_time"]), data["alliance_biz_type"],
			data["phase_id"], data["short_id"], now, now,
		)
	}
	sql += strings.Join(placeholders, ",")
	sql += ` ON DUPLICATE KEY UPDATE
		product_id = VALUES(product_id), product_name = VALUES(product_name),
		product_num = VALUES(product_num), product_price = VALUES(product_price),
		author_account = VALUES(author_account), commission_rate = VALUES(commission_rate),
		estimated_comission = VALUES(estimated_comission), real_comission = VALUES(real_comission),
		total_pay_amount = VALUES(total_pay_amount), order_status = VALUES(order_status),
		shop_id = VALUES(shop_id), shop_name = VALUES(shop_name), short_id = VALUES(short_id),
		updated_at = VALUES(updated_at)`

	var err error
	for retry := 0; retry < 3; retry++ {
		_, err = g.DB().Exec(ctx, sql, args...)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "Lock wait timeout") || retry == 2 {
			break
		}
		time.Sleep(time.Duration(retry+1) * 2 * time.Second)
	}
	return err
}

func backfillAllianceTalent(ctx context.Context, groups map[allianceAccount][]string) (int, error) {
	total := 0
	for account, orderIDs := range groups {
		for _, batch := range chunkStrings(orderIDs, 500) {
			_, err := g.DB().Model("order_info").Ctx(ctx).
				WhereIn("sub_order_no", batch).
				Where("(talent_id IS NULL OR talent_id = '' OR talent_id = '0')").
				Data(g.Map{"talent_id": account.UID, "talent_name": account.Name}).Update()
			if err != nil {
				return total, err
			}
			total += len(batch)
		}
	}
	return total, nil
}

func logAllianceThirdParty(task allianceBatchTask, durationMs int, response interface{}, requestErr error) {
	requestJSON, _ := json.Marshal(g.Map{"storeId": task.StoreID, "orderIds": task.OrderIDs, "size": allianceOrderBatchSize})
	responseJSON, _ := json.Marshal(response)
	status := 1
	if requestErr != nil {
		status = 0
		responseJSON, _ = json.Marshal(g.Map{"error": requestErr.Error()})
	}
	if len(responseJSON) > 50000 {
		responseJSON = append(responseJSON[:50000], []byte("...[truncated]")...)
	}
	_, _ = g.DB().Model("third_party_api_log").Data(g.Map{
		"platform": "douyin", "api_name": "queryShopAllianceOrder", "api_path": "/alliance/getOrderList",
		"request_params": string(requestJSON), "response_body": string(responseJSON),
		"duration_ms": durationMs, "status": status,
	}).Insert()
}

func valueOr(value interface{}, fallback interface{}) interface{} {
	if value == nil || value == "" {
		return fallback
	}
	return value
}

func nilIfEmpty(value interface{}) interface{} {
	if value == nil || value == "" {
		return nil
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func chunkStrings(values []string, size int) [][]string {
	if size <= 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}
