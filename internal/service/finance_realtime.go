package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/xuri/excelize/v2"
)

// ==================== 实时盈亏查询（含千川消耗） ====================

// RealtimeProfitReq 实时盈亏查询请求
type RealtimeProfitReq struct {
	TraderId     int64  `json:"traderId"`
	DepartmentId int64  `json:"departmentId"`
	CompanyId    int64  `json:"companyId"`
	Date         string `json:"date"`      // 单日期，格式 2006-01-02，空=今天（兼容旧版）
	StartDate    string `json:"startDate"` // 开始日期，格式 2006-01-02
	EndDate      string `json:"endDate"`   // 结束日期，格式 2006-01-02
}

// RealtimeProfitResult 实时盈亏结果
type RealtimeProfitResult struct {
	StatDate          string  `json:"statDate"`
	TotalOrderCount   int64   `json:"totalOrderCount"`
	OrderCount        int64   `json:"orderCount"`
	InvalidOrderCount int64   `json:"invalidOrderCount"`
	ProductNum        int64   `json:"productNum"`
	PayAmount         float64 `json:"payAmount"`
	CostAmount        float64 `json:"costAmount"`
	ReserveAmount     float64 `json:"reserveAmount"`
	ShopServiceFee    float64 `json:"shopServiceFee"`
	Commission        float64 `json:"commission"`
	RefundCost        float64 `json:"refundCost"`
	RefundCount       int64   `json:"refundCount"`
	AdCost            float64 `json:"adCost"`
	NetProfit         float64 `json:"netProfit"`
}

// getValidCond 返回有效/无效订单条件SQL
func getValidCond(alias string) (validCond, invalidCond, refundCond string) {
	return financeValidOrderCondition(alias), financeInvalidOrderCondition(alias), financeRefundOrderCondition(alias)
}

// GetRealtimeProfit 查询实时盈亏（含千川消耗），支持按投手/部门/公司维度
func (s *FinanceCommissionService) GetRealtimeProfit(ctx context.Context, req *RealtimeProfitReq) (*RealtimeProfitResult, error) {
	// 解析日期范围：优先使用 StartDate/EndDate，兼容旧版 Date
	startDate, endDate := req.StartDate, req.EndDate
	if startDate == "" && endDate == "" {
		if req.Date != "" {
			startDate, endDate = req.Date, req.Date
		} else {
			today := time.Now().Format("2006-01-02")
			startDate, endDate = today, today
		}
	} else {
		if startDate == "" {
			startDate = endDate
		}
		if endDate == "" {
			endDate = startDate
		}
	}
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	startTime := startDate + " 00:00:00"
	endTime := endDate + " 23:59:59"

	result := &RealtimeProfitResult{StatDate: startDate + " ~ " + endDate}
	if req.CompanyId > 0 {
		isRoot, rootErr := isRootFinanceCompany(ctx, req.CompanyId)
		if rootErr != nil {
			return nil, rootErr
		}
		if isRoot {
			snapshot, snapshotErr := GetDouyinProfitSnapshot(ctx, startDate, endDate)
			if snapshotErr != nil {
				return nil, snapshotErr
			}
			return &RealtimeProfitResult{
				StatDate:          startDate + " ~ " + endDate,
				TotalOrderCount:   snapshot.TotalOrderCount,
				OrderCount:        snapshot.ValidOrderCount,
				InvalidOrderCount: snapshot.InvalidOrderCount,
				ProductNum:        snapshot.ProductQuantity,
				PayAmount:         snapshot.SalesAmount,
				CostAmount:        snapshot.CostAmount,
				ReserveAmount:     snapshot.ReserveAmount,
				ShopServiceFee:    snapshot.ShopServiceFee,
				Commission:        snapshot.Commission,
				RefundCost:        snapshot.RefundCostAmount,
				RefundCount:       snapshot.RefundOrderCount,
				AdCost:            snapshot.AdCostAmount,
				NetProfit:         snapshot.Profit,
			}, nil
		}
	}

	// 1. 确定要查询的账号ID列表
	accountIds, err := s.getAccountIdsByDimension(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(accountIds) == 0 {
		return result, nil
	}

	// 2. 查询三平台订单统计
	// 投手维度：使用 statsOrdersByAccountIds（含归属规则），与佣金统计口径一致
	if req.TraderId > 0 {
		traderName, _ := s.getTraderName(ctx, req.TraderId)
		if traderName != "" {
			stats, err := s.statsOrdersByAccountIds(ctx, accountIds, startDate, endDate, traderName)
			if err != nil {
				return nil, fmt.Errorf("查询投手订单归属失败: %w", err)
			}
			if stats != nil {
				result.OrderCount = int64(stats.OrderCount)
				result.PayAmount = stats.PayAmount
				result.CostAmount = stats.CostAmount
				result.ReserveAmount = stats.ReserveAmount
				result.ShopServiceFee = stats.ShopServiceFee
				result.Commission = stats.Commission
				result.RefundCost = stats.RefundCostAmount
				result.RefundCount = int64(stats.RefundOrderCount)
			}
		}
	} else {
		validCond, invalidCond, refundCond := getValidCond("o")
		s.orderStatsByAccounts(ctx, accountIds, startTime, endTime, validCond, invalidCond, refundCond, result)
		if req.CompanyId > 0 {
			if selfSaleStats, selfSaleErr := s.statsSelfSaleByAccountIds(ctx, accountIds, startDate, endDate); selfSaleErr != nil {
				g.Log().Warningf(ctx, "[RealtimeProfit] 查询公司自营销售额失败: companyId=%d, err=%v", req.CompanyId, selfSaleErr)
			} else {
				result.OrderCount += int64(selfSaleStats.OrderCount)
				result.PayAmount = roundFloatFinance(result.PayAmount+selfSaleStats.PayAmount, 2)
				result.CostAmount = roundFloatFinance(result.CostAmount+selfSaleStats.CostAmount, 2)
				result.ReserveAmount = roundFloatFinance(result.ReserveAmount+selfSaleStats.ReserveAmount, 2)
				result.ShopServiceFee = roundFloatFinance(result.ShopServiceFee+selfSaleStats.ShopServiceFee, 2)
				result.Commission = roundFloatFinance(result.Commission+selfSaleStats.Commission, 2)
				result.RefundCost = roundFloatFinance(result.RefundCost+selfSaleStats.RefundCostAmount, 2)
				result.RefundCount += int64(selfSaleStats.RefundOrderCount)
			}
		}
	}

	// 3. 查询千川广告消耗（按日期范围）
	result.AdCost = s.getAdCostByAccountsRange(ctx, accountIds, startDate, endDate)

	// 4. 计算净利润
	commissionDeduction := result.PayAmount * 0.005
	result.NetProfit = CalculateOrderProfit(result.PayAmount, result.CostAmount, result.ReserveAmount, result.RefundCost, result.ShopServiceFee, commissionDeduction, 0, result.AdCost)
	result.NetProfit = roundFloatFinance(result.NetProfit, 2)
	result.AdCost = roundFloatFinance(result.AdCost, 2)

	return result, nil
}

// getAccountIdsByDimension 根据维度获取账号ID列表
func (s *FinanceCommissionService) getAccountIdsByDimension(ctx context.Context, req *RealtimeProfitReq) ([]int, error) {
	var (
		accountIds []int
		err        error
	)

	switch {
	case req.TraderId > 0:
		traderId := req.TraderId
		if !model.IsTraderID(traderId) {
			traderId = model.EncodeTraderID(traderId)
		}
		accountIds, err = s.getAccountIdsByNode(ctx, traderId, "trader", "")
	case req.DepartmentId > 0:
		departmentId := req.DepartmentId
		if !model.IsDeptID(departmentId) {
			departmentId = model.EncodeDeptID(departmentId)
		}
		accountIds, err = s.getAccountIdsByNode(ctx, departmentId, "department", "")
	case req.CompanyId > 0:
		// 公司维度需要汇总当前公司及所有子公司，口径与佣金统计保持一致。
		accountIds, err = s.getAccountIdsByNode(ctx, req.CompanyId, "company", "")
	}
	if err != nil {
		return nil, err
	}

	return uniqueIntSlice(accountIds), nil
}

func (s *FinanceCommissionService) getDepartmentTreeIds(ctx context.Context, departmentId int64) ([]int64, error) {
	rows, err := g.DB().GetAll(ctx, `
		WITH RECURSIVE dept_tree AS (
			SELECT id FROM sys_department WHERE id = ?
			UNION ALL
			SELECT d.id FROM sys_department d
			INNER JOIN dept_tree dt ON d.parent_id = dt.id
		)
		SELECT id FROM dept_tree
	`, departmentId)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if id := row["id"].Int64(); id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func uniqueIntSlice(values []int) []int {
	if len(values) <= 1 {
		return values
	}

	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// orderStatsByAccounts 查询指定账号集合的订单统计
func (s *FinanceCommissionService) orderStatsByAccounts(ctx context.Context, accountIds []int, startTime, endTime, validCond, invalidCond, refundCond string, result *RealtimeProfitResult) {
	rows, err := g.DB().Model("account").WhereIn("id", accountIds).Fields("uid").All()
	if err != nil {
		g.Log().Warningf(ctx, "[RealtimeProfit] 查询账号UID失败: %v", err)
		return
	}

	uids := make([]string, 0, len(rows))
	for _, row := range rows {
		if uid := row["uid"].String(); uid != "" {
			uids = append(uids, uid)
		}
	}
	if len(uids) == 0 {
		return
	}

	platforms := []struct {
		table      string
		orderNoCol string
		isDouyin   bool
	}{
		{"order_info", "main_order_no", true},
		{"kuaishou_orders", "order_no", false},
		{"shipinhao_orders", "order_no", false},
	}

	for _, p := range platforms {
		var row gdb.Record
		placeholders := strings.TrimRight(strings.Repeat("?,", len(uids)), ",")
		sql := fmt.Sprintf(`
			SELECT
				COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
				COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
				COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
					COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost,
				COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_count
			FROM %s o
			LEFT JOIN goods g ON o.seller_code COLLATE utf8mb4_unicode_ci = g.goods_code COLLATE utf8mb4_unicode_ci
			WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
			  AND o.talent_id IN (%s)
		`, validCond, p.orderNoCol, validCond, validCond, refundCond, refundCond, p.orderNoCol, p.table, placeholders)

		params := []interface{}{startTime, endTime}
		for _, uid := range uids {
			params = append(params, uid)
		}

		row, err = g.DB().GetOne(ctx, sql, params...)
		if err != nil {
			g.Log().Warningf(ctx, "[RealtimeProfit] 查询%s订单失败: %v", p.table, err)
			continue
		}

		count := row["order_count"].Int64()
		if count == 0 {
			continue
		}

		result.OrderCount += count
		payAmount := row["pay_amount"].Float64()
		result.PayAmount += payAmount
		result.CostAmount += row["cost_amount"].Float64()
		result.RefundCost += row["refund_cost"].Float64()
		result.RefundCount += row["refund_count"].Int64()

		if p.isDouyin {
			result.ShopServiceFee += CalculateDouyinShopServiceFee(payAmount)
		}
	}
	result.ReserveAmount = CalculateReserveAmount(result.PayAmount)

	result.Commission = result.PayAmount - result.ReserveAmount - result.CostAmount - result.ShopServiceFee
	if result.Commission < 0 {
		result.Commission = 0
	}

	result.PayAmount = roundFloatFinance(result.PayAmount, 2)
	result.CostAmount = roundFloatFinance(result.CostAmount, 2)
	result.ReserveAmount = roundFloatFinance(result.ReserveAmount, 2)
	result.ShopServiceFee = roundFloatFinance(result.ShopServiceFee, 2)
	result.Commission = roundFloatFinance(result.Commission, 2)
	result.RefundCost = roundFloatFinance(result.RefundCost, 2)
}

// qianchuanAdReportJoinCond 返回千川广告日报表的 JOIN ON 条件
// 注意：GoFrame 的 Join 方法签名为 ...string，不支持 ? 占位符传参，
// 第三个参数会被识别为表别名，因此日期需要直接拼接到条件中。
func qianchuanAdReportJoinCond(date string) string {
	return fmt.Sprintf("aqm.qianchuan_advertiser_id = qar.advertiser_id AND LEFT(qar.stat_datetime, 10) = '%s'", date)
}

// qianchuanAdReportJoinCondRange 返回千川广告日报表按日期范围筛选的 JOIN ON 条件
func qianchuanAdReportJoinCondRange(startDate, endDate string) string {
	return fmt.Sprintf("aqm.qianchuan_advertiser_id = qar.advertiser_id AND LEFT(qar.stat_datetime, 10) BETWEEN '%s' AND '%s'", startDate, endDate)
}

// getAdCostByAccounts 查询指定账号集合的单日千川广告消耗（兼容旧版）
func (s *FinanceCommissionService) getAdCostByAccounts(ctx context.Context, accountIds []int, date string) float64 {
	return s.getAdCostByAccountsRange(ctx, accountIds, date, date)
}

// getAdCostByAccountsRange 查询指定账号集合在日期范围内的千川广告消耗
func (s *FinanceCommissionService) getAdCostByAccountsRange(ctx context.Context, accountIds []int, startDate, endDate string) float64 {
	if len(accountIds) == 0 {
		return 0
	}

	// 优先使用达人维度日报表，避免同一个千川广告主下多个达人时按广告主总消耗重复分摊。
	row, err := g.DB().Model("qianchuan_author_daily_report").
		WhereIn("account_id", accountIds).
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COALESCE(SUM(stat_cost), 0) AS ad_cost").
		One()
	if err == nil && row["ad_cost"].Float64() > 0 {
		return row["ad_cost"].Float64()
	}

	// 兼容历史数据：如果达人维度还未回填，再回退到广告主总消耗。
	row, err = g.DB().Model("account_qianchuan_mapping aqm").
		LeftJoin("qianchuan_advertiser_daily_report qar", qianchuanAdReportJoinCondRange(startDate, endDate)).
		WhereIn("aqm.account_id", accountIds).
		Fields("COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		One()
	if err != nil {
		return 0
	}
	return row["ad_cost"].Float64()
}

// AuthorCostItem 达人消耗明细
type AuthorCostItem struct {
	AccountID      int64   `json:"accountId"`
	AwemeID        int64   `json:"awemeId"`
	AwemeName      string  `json:"awemeName"`      // 兼容旧字段：抖音昵称
	DouyinNickname string  `json:"douyinNickname"` // 抖音昵称
	AuthorName     string  `json:"authorName"`     // 内部账号表维护的达人名称
	OperatorName   string  `json:"operatorName"`   // 投手名称
	StatCost       float64 `json:"statCost"`
	PayOrderGmv    float64 `json:"payOrderGmv"`
	OrderCount     int64   `json:"orderCount"`
}

// QianchuanCostDetailItem 组织/运营消耗明细
type QianchuanCostDetailItem struct {
	AccountID      int64   `json:"accountId"`
	AwemeID        int64   `json:"awemeId"`
	AwemeName      string  `json:"awemeName"`      // 兼容旧字段：抖音昵称
	DouyinNickname string  `json:"douyinNickname"` // 抖音昵称
	AuthorName     string  `json:"authorName"`     // 内部账号表维护的达人名称
	AdCost         float64 `json:"adCost"`
	PayOrderGmv    float64 `json:"payOrderGmv"`
	OrderCount     int64   `json:"orderCount"`
}

// QianchuanStoreCostItem 店铺消耗统计。
type QianchuanStoreCostItem struct {
	StoreID      int64  `json:"storeId"`
	StoreName    string `json:"storeName"`
	StoreStatus  int    `json:"storeStatus"`
	MatchedStore bool   `json:"matchedStore"`
	// 以下主体字段保留一段时间，兼容旧接口调用方。
	SubjectID       int64   `json:"subjectId"`
	AdvertiserID    int64   `json:"advertiserId"`
	AdvertiserName  string  `json:"advertiserName"`
	SubjectName     string  `json:"subjectName"`
	AccountName     string  `json:"accountName"`
	Status          string  `json:"status"`
	StatusType      string  `json:"statusType"`
	StatCost        float64 `json:"statCost"`
	PayOrderAmount  float64 `json:"payOrderAmount"`
	AllOrderPayGmv  float64 `json:"allOrderPayGmv"`
	ROI             float64 `json:"roi"`
	Percent         float64 `json:"percent"`
	MappingCount    int64   `json:"mappingCount"`
	AdvertiserCount int64   `json:"advertiserCount"`
	MatchedSubject  bool    `json:"matchedSubject"`
}

// GetStoreCostStats 按店铺汇总千川广告主账户消耗。
func (s *FinanceCommissionService) GetStoreCostStats(ctx context.Context, startDate, endDate string, limit int) ([]QianchuanStoreCostItem, error) {
	if startDate == "" && endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate, endDate = today, today
	} else {
		if startDate == "" {
			startDate = endDate
		}
		if endDate == "" {
			endDate = startDate
		}
	}
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := g.DB().GetAll(ctx, `
		SELECT
			COALESCE(st.id, 0) AS store_id,
			COALESCE(NULLIF(st.store_name, ''), '未绑定店铺') AS store_name,
			COALESCE(st.status, 0) AS store_status,
			COUNT(DISTINCT adv.advertiser_id) AS advertiser_count,
			COALESCE(SUM(adv.stat_cost), 0) AS stat_cost,
			COALESCE(SUM(adv.pay_order_amount), 0) AS pay_order_amount,
			COALESCE(SUM(adv.all_order_pay_gmv), 0) AS all_order_pay_gmv,
			COUNT(DISTINCT CASE WHEN st.id IS NOT NULL THEN adv.advertiser_id END) AS mapping_count,
			MAX(CASE WHEN st.id IS NOT NULL THEN 1 ELSE 0 END) AS matched_store
		FROM (
			SELECT
				advertiser_id,
				COALESCE(SUM(stat_cost), 0) AS stat_cost,
				COALESCE(SUM(pay_order_amount), 0) AS pay_order_amount,
				COALESCE(SUM(all_order_pay_gmv_7days), 0) AS all_order_pay_gmv
			FROM qianchuan_advertiser_daily_report
			WHERE LEFT(stat_datetime, 10) >= ?
			  AND LEFT(stat_datetime, 10) <= ?
			GROUP BY advertiser_id
		) adv
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN store st ON st.id = sqm.store_id
		GROUP BY COALESCE(st.id, 0), COALESCE(NULLIF(st.store_name, ''), '未绑定店铺'), COALESCE(st.status, 0)
		ORDER BY stat_cost DESC
		LIMIT ?
	`, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}

	result := make([]QianchuanStoreCostItem, len(rows))
	maxCost := 0.0
	for _, row := range rows {
		if cost := row["stat_cost"].Float64(); cost > maxCost {
			maxCost = cost
		}
	}

	for i, row := range rows {
		statCost := row["stat_cost"].Float64()
		payOrderAmount := row["pay_order_amount"].Float64()
		allOrderPayGmv := row["all_order_pay_gmv"].Float64()
		roiAmount := payOrderAmount
		if roiAmount <= 0 {
			roiAmount = allOrderPayGmv
		}
		roi := 0.0
		if statCost > 0 {
			roi = roiAmount / statCost
		}

		matchedStore := row["matched_store"].Int() > 0
		status := "待绑定店铺"
		statusType := "warning"
		if matchedStore && row["store_status"].Int() == 1 {
			status = "已归属"
			statusType = "known"
		} else if matchedStore {
			status = "店铺已停用"
		}

		percent := 0.0
		if maxCost > 0 {
			percent = statCost / maxCost * 100
		}

		storeID := row["store_id"].Int64()
		storeName := row["store_name"].String()
		result[i] = QianchuanStoreCostItem{
			StoreID:         storeID,
			StoreName:       storeName,
			StoreStatus:     row["store_status"].Int(),
			MatchedStore:    matchedStore,
			SubjectID:       storeID,
			AdvertiserID:    storeID,
			AdvertiserName:  storeName,
			SubjectName:     storeName,
			AccountName:     fmt.Sprintf("%d 个千川账户", row["advertiser_count"].Int64()),
			Status:          status,
			StatusType:      statusType,
			StatCost:        roundFloatFinance(statCost, 2),
			PayOrderAmount:  roundFloatFinance(payOrderAmount, 2),
			AllOrderPayGmv:  roundFloatFinance(allOrderPayGmv, 2),
			ROI:             roundFloatFinance(roi, 2),
			Percent:         roundFloatFinance(percent, 2),
			MappingCount:    row["mapping_count"].Int64(),
			AdvertiserCount: row["advertiser_count"].Int64(),
			MatchedSubject:  matchedStore,
		}
	}
	return result, nil
}

// GetSubjectCostStats 兼容旧主体消耗接口，数据已统一为店铺口径。
func (s *FinanceCommissionService) GetSubjectCostStats(ctx context.Context, startDate, endDate string, limit int) ([]QianchuanStoreCostItem, error) {
	return s.GetStoreCostStats(ctx, startDate, endDate, limit)
}

// QianchuanStoreCostDetailItem 店铺消耗明细（按千川广告主账户维度）。
type QianchuanStoreCostDetailItem struct {
	AdvertiserID   int64   `json:"advertiserId"`
	AdvertiserName string  `json:"advertiserName"`
	AccountID      int64   `json:"accountId"`
	AccountName    string  `json:"accountName"`
	UID            string  `json:"uid"`
	TraderName     string  `json:"traderName"`
	StatCost       float64 `json:"statCost"`
	PayOrderAmount float64 `json:"payOrderAmount"`
	AllOrderPayGmv float64 `json:"allOrderPayGmv"`
	ShowCnt        int64   `json:"showCnt"`
	ClickCnt       int64   `json:"clickCnt"`
	PayOrderCount  int64   `json:"payOrderCount"`
	ROI            float64 `json:"roi"`
}

// GetStoreCostDetail 查询指定店铺下所有千川广告主账户的消耗明细。
// storeId 为 store.id，0 表示未绑定到有效店铺的账户。
func (s *FinanceCommissionService) GetStoreCostDetail(ctx context.Context, startDate, endDate string, storeId int64, page, pageSize int) ([]QianchuanStoreCostDetailItem, int64, error) {
	if startDate == "" && endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate, endDate = today, today
	} else {
		if startDate == "" {
			startDate = endDate
		}
		if endDate == "" {
			endDate = startDate
		}
	}
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, 0, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, 0, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize

	// 统计总数
	countRow, err := g.DB().GetOne(ctx, `
		SELECT COUNT(*) AS total FROM (
			SELECT adv.advertiser_id
			FROM (
				SELECT advertiser_id
				FROM qianchuan_advertiser_daily_report
				WHERE LEFT(stat_datetime, 10) >= ? AND LEFT(stat_datetime, 10) <= ?
				GROUP BY advertiser_id
			) adv
			LEFT JOIN store_qianchuan_mapping sqm ON sqm.qianchuan_advertiser_id = adv.advertiser_id
			LEFT JOIN store st ON st.id = sqm.store_id
			WHERE COALESCE(st.id, 0) = ?
			GROUP BY adv.advertiser_id
		) t
	`, startDate, endDate, storeId)
	if err != nil {
		return nil, 0, err
	}
	total := countRow["total"].Int64()

	rows, err := g.DB().GetAll(ctx, `
		SELECT
			adv.advertiser_id,
			COALESCE(
				NULLIF(MAX(adv.advertiser_name), ''),
				NULLIF(MAX(sqm.qianchuan_advertiser_name), ''),
				NULLIF(MAX(r.advertiser_name), ''),
				CONCAT('千川账户 ', adv.advertiser_id)
			) AS advertiser_name,
			COALESCE(NULLIF(MAX(a.platform_nickname), ''), NULLIF(MAX(a.account_name), ''), '') AS account_name,
			COALESCE(MAX(a.id), 0) AS account_id,
			COALESCE(NULLIF(MAX(a.uid), ''), '') AS uid,
			COALESCE(NULLIF(MAX(t.name), ''), '') AS trader_name,
			MAX(adv.stat_cost) AS stat_cost,
			MAX(adv.pay_order_amount) AS pay_order_amount,
			MAX(adv.all_order_pay_gmv) AS all_order_pay_gmv,
			MAX(adv.show_cnt) AS show_cnt,
			MAX(adv.click_cnt) AS click_cnt,
			MAX(adv.pay_order_count) AS pay_order_count
		FROM (
			SELECT
				advertiser_id,
				MAX(NULLIF(advertiser_name, '')) AS advertiser_name,
				COALESCE(SUM(stat_cost), 0) AS stat_cost,
				COALESCE(SUM(pay_order_amount), 0) AS pay_order_amount,
				COALESCE(SUM(all_order_pay_gmv_7days), 0) AS all_order_pay_gmv,
				COALESCE(SUM(show_cnt), 0) AS show_cnt,
				COALESCE(SUM(click_cnt), 0) AS click_cnt,
				COALESCE(SUM(pay_order_count), 0) AS pay_order_count
			FROM qianchuan_advertiser_daily_report
			WHERE LEFT(stat_datetime, 10) >= ? AND LEFT(stat_datetime, 10) <= ?
			GROUP BY advertiser_id
		) adv
		LEFT JOIN store_qianchuan_mapping sqm ON sqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN store st ON st.id = sqm.store_id
		LEFT JOIN account_qianchuan_mapping aqm ON aqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN account a ON a.id = aqm.account_id
		LEFT JOIN trader_account_relation tar ON tar.account_id = a.id
		LEFT JOIN trader t ON t.id = tar.trader_id
		LEFT JOIN (
			SELECT advertiser_id, MAX(advertiser_name) AS advertiser_name
			FROM qianchuan_ad_report
			GROUP BY advertiser_id
		) r ON r.advertiser_id = adv.advertiser_id
		WHERE COALESCE(st.id, 0) = ?
		GROUP BY adv.advertiser_id
		ORDER BY stat_cost DESC
		LIMIT ?, ?
	`, startDate, endDate, storeId, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	result := make([]QianchuanStoreCostDetailItem, len(rows))
	for i, row := range rows {
		statCost := row["stat_cost"].Float64()
		payOrderAmount := row["pay_order_amount"].Float64()
		allOrderPayGmv := row["all_order_pay_gmv"].Float64()
		roiAmount := payOrderAmount
		if roiAmount <= 0 {
			roiAmount = allOrderPayGmv
		}
		roi := 0.0
		if statCost > 0 {
			roi = roiAmount / statCost
		}

		result[i] = QianchuanStoreCostDetailItem{
			AdvertiserID:   row["advertiser_id"].Int64(),
			AdvertiserName: row["advertiser_name"].String(),
			AccountID:      row["account_id"].Int64(),
			AccountName:    row["account_name"].String(),
			UID:            row["uid"].String(),
			TraderName:     row["trader_name"].String(),
			StatCost:       roundFloatFinance(statCost, 2),
			PayOrderAmount: roundFloatFinance(payOrderAmount, 2),
			AllOrderPayGmv: roundFloatFinance(allOrderPayGmv, 2),
			ShowCnt:        row["show_cnt"].Int64(),
			ClickCnt:       row["click_cnt"].Int64(),
			PayOrderCount:  row["pay_order_count"].Int64(),
			ROI:            roundFloatFinance(roi, 2),
		}
	}
	return result, total, nil
}

// GetSubjectCostDetail 兼容旧主体明细接口，subjectId 参数按 storeId 解释。
func (s *FinanceCommissionService) GetSubjectCostDetail(ctx context.Context, startDate, endDate string, subjectId int64, page, pageSize int) ([]QianchuanStoreCostDetailItem, int64, error) {
	return s.GetStoreCostDetail(ctx, startDate, endDate, subjectId, page, pageSize)
}

// GetAuthorCostRanking 查询达人维度消耗统计
func (s *FinanceCommissionService) GetAuthorCostRanking(ctx context.Context, startDate, endDate string, page, pageSize int) ([]AuthorCostItem, int64, error) {
	if startDate == "" && endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate, endDate = today, today
	} else {
		if startDate == "" {
			startDate = endDate
		}
		if endDate == "" {
			endDate = startDate
		}
	}
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, 0, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, 0, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100000 {
		pageSize = 100000
	}
	offset := (page - 1) * pageSize

	totalRow, err := g.DB().Model("qianchuan_author_daily_report").
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COUNT(DISTINCT aweme_id) AS total").
		One()
	if err != nil {
		return nil, 0, err
	}
	total := totalRow["total"].Int64()

	rows, err := g.DB().GetAll(ctx, `
		SELECT
			MAX(qar.account_id) AS account_id,
			qar.aweme_id,
			COALESCE(MAX(NULLIF(qar.aweme_name, '')), MAX(NULLIF(a_uid.platform_nickname, '')), MAX(NULLIF(a_id.platform_nickname, '')), '') AS douyin_nickname,
			COALESCE(MAX(NULLIF(a_uid.account_name, '')), MAX(NULLIF(a_id.account_name, '')), MAX(NULLIF(qar.aweme_name, '')), '') AS author_name,
			COALESCE(MAX(NULLIF(a_uid.operator, '')), MAX(NULLIF(a_id.operator, '')), '') AS operator_name,
			COALESCE(SUM(qar.stat_cost), 0) AS stat_cost,
			COALESCE(SUM(qar.pay_order_gmv), 0) AS pay_order_gmv,
			COALESCE(SUM(qar.pay_order_count), 0) AS pay_order_count
		FROM qianchuan_author_daily_report qar
		LEFT JOIN account a_id ON a_id.id = qar.account_id
		LEFT JOIN account a_uid ON a_uid.uid REGEXP '^[0-9]+$' AND CAST(a_uid.uid AS UNSIGNED) = qar.aweme_id
		WHERE LEFT(qar.stat_datetime, 10) >= ?
		  AND LEFT(qar.stat_datetime, 10) <= ?
		GROUP BY qar.aweme_id
		ORDER BY stat_cost DESC
		LIMIT ? OFFSET ?
	`, startDate, endDate, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]AuthorCostItem, len(rows))
	for i, row := range rows {
		douyinNickname := row["douyin_nickname"].String()
		result[i] = AuthorCostItem{
			AccountID:      row["account_id"].Int64(),
			AwemeID:        row["aweme_id"].Int64(),
			AwemeName:      douyinNickname,
			DouyinNickname: douyinNickname,
			AuthorName:     row["author_name"].String(),
			OperatorName:   row["operator_name"].String(),
			StatCost:       roundFloatFinance(row["stat_cost"].Float64(), 2),
			PayOrderGmv:    roundFloatFinance(row["pay_order_gmv"].Float64(), 2),
			OrderCount:     row["pay_order_count"].Int64(),
		}
	}
	return result, total, nil
}

// GetCostRanking 查询消耗排行
func (s *FinanceCommissionService) GetCostRanking(ctx context.Context, startDate, endDate, dimension string, limit int) ([]g.Map, error) {
	if startDate == "" && endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate, endDate = today, today
	} else {
		if startDate == "" {
			startDate = endDate
		}
		if endDate == "" {
			endDate = startDate
		}
	}
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if limit <= 0 {
		limit = 20
	}

	switch dimension {
	case "trader":
		return s.costRankingByTrader(ctx, startDate, endDate, limit)
	case "department":
		return s.costRankingByDept(ctx, startDate, endDate, limit)
	case "company":
		return s.costRankingByCompany(ctx, startDate, endDate, limit)
	default:
		return nil, fmt.Errorf("不支持的维度: %s，支持: trader/department/company", dimension)
	}
}

func (s *FinanceCommissionService) costRankingByTrader(ctx context.Context, startDate, endDate string, limit int) ([]g.Map, error) {
	if s.hasAuthorCostData(ctx, startDate, endDate) {
		rows, err := g.DB().Model("trader t").
			LeftJoin("qianchuan_author_daily_report qar", fmt.Sprintf("t.id = qar.trader_id AND LEFT(qar.stat_datetime, 10) BETWEEN '%s' AND '%s'", startDate, endDate)).
			Fields("t.id, t.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
			Group("t.id, t.name").
			OrderDesc("ad_cost").
			Limit(limit).
			All()
		if err != nil {
			return nil, err
		}
		result := make([]g.Map, len(rows))
		for i, row := range rows {
			traderID := row["id"].Int64()
			result[i] = g.Map{
				"id":          row["id"],
				"name":        row["name"],
				"adCost":      roundFloatFinance(row["ad_cost"].Float64(), 2),
				"costDetails": s.costDetailsByDimension(ctx, startDate, endDate, "trader", traderID),
			}
		}
		return result, nil
	}

	rows, err := g.DB().Model("trader t").
		InnerJoin("trader_account_relation tar", "t.id = tar.trader_id").
		InnerJoin("account_qianchuan_mapping aqm", "tar.account_id = aqm.account_id").
		InnerJoin("qianchuan_advertiser_daily_report qar", qianchuanAdReportJoinCondRange(startDate, endDate)).
		Fields("t.id, t.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		Group("t.id, t.name").
		OrderDesc("ad_cost").
		Limit(limit).
		All()
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		result[i] = g.Map{
			"id":     row["id"],
			"name":   row["name"],
			"adCost": roundFloatFinance(row["ad_cost"].Float64(), 2),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costRankingByDept(ctx context.Context, startDate, endDate string, limit int) ([]g.Map, error) {
	if s.hasAuthorCostData(ctx, startDate, endDate) {
		rows, err := g.DB().Model("sys_department d").
			LeftJoin("qianchuan_author_daily_report qar", fmt.Sprintf("d.id = qar.department_id AND LEFT(qar.stat_datetime, 10) BETWEEN '%s' AND '%s'", startDate, endDate)).
			Fields("d.id, d.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
			Group("d.id, d.name").
			OrderDesc("ad_cost").
			Limit(limit).
			All()
		if err != nil {
			return nil, err
		}
		result := make([]g.Map, len(rows))
		for i, row := range rows {
			departmentID := row["id"].Int64()
			result[i] = g.Map{
				"id":          row["id"],
				"name":        row["name"],
				"adCost":      roundFloatFinance(row["ad_cost"].Float64(), 2),
				"costDetails": s.costDetailsByDimension(ctx, startDate, endDate, "department", departmentID),
			}
		}
		return result, nil
	}

	rows, err := g.DB().Model("sys_department d").
		InnerJoin("account a", "a.department_id = d.id").
		InnerJoin("account_qianchuan_mapping aqm", "a.id = aqm.account_id").
		InnerJoin("qianchuan_advertiser_daily_report qar", qianchuanAdReportJoinCondRange(startDate, endDate)).
		Fields("d.id, d.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		Group("d.id, d.name").
		OrderDesc("ad_cost").
		Limit(limit).
		All()
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		result[i] = g.Map{
			"id":     row["id"],
			"name":   row["name"],
			"adCost": roundFloatFinance(row["ad_cost"].Float64(), 2),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costRankingByCompany(ctx context.Context, startDate, endDate string, limit int) ([]g.Map, error) {
	if s.hasAuthorCostData(ctx, startDate, endDate) {
		rows, err := g.DB().Model("sys_company c").
			LeftJoin("qianchuan_author_daily_report qar", fmt.Sprintf("c.id = qar.company_id AND LEFT(qar.stat_datetime, 10) BETWEEN '%s' AND '%s'", startDate, endDate)).
			Fields("c.id, c.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
			Group("c.id, c.name").
			OrderDesc("ad_cost").
			Limit(limit).
			All()
		if err != nil {
			return nil, err
		}
		result := make([]g.Map, len(rows))
		for i, row := range rows {
			companyID := row["id"].Int64()
			result[i] = g.Map{
				"id":          row["id"],
				"name":        row["name"],
				"adCost":      roundFloatFinance(row["ad_cost"].Float64(), 2),
				"costDetails": s.costDetailsByDimension(ctx, startDate, endDate, "company", companyID),
			}
		}
		return result, nil
	}

	rows, err := g.DB().Model("sys_company c").
		InnerJoin("account a", "a.company_id = c.id").
		InnerJoin("account_qianchuan_mapping aqm", "a.id = aqm.account_id").
		InnerJoin("qianchuan_advertiser_daily_report qar", qianchuanAdReportJoinCondRange(startDate, endDate)).
		Fields("c.id, c.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		Group("c.id, c.name").
		OrderDesc("ad_cost").
		Limit(limit).
		All()
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		result[i] = g.Map{
			"id":     row["id"],
			"name":   row["name"],
			"adCost": roundFloatFinance(row["ad_cost"].Float64(), 2),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costDetailsByDimension(ctx context.Context, startDate, endDate, dimension string, id int64) []QianchuanCostDetailItem {
	whereColumn := map[string]string{
		"trader":     "qar.trader_id",
		"department": "qar.department_id",
		"company":    "qar.company_id",
	}[dimension]
	if whereColumn == "" || id <= 0 {
		return []QianchuanCostDetailItem{}
	}

	rows, err := g.DB().GetAll(ctx, fmt.Sprintf(`
		SELECT
			MAX(qar.account_id) AS account_id,
			qar.aweme_id,
			COALESCE(MAX(NULLIF(qar.aweme_name, '')), MAX(NULLIF(a_uid.platform_nickname, '')), MAX(NULLIF(a_id.platform_nickname, '')), '') AS douyin_nickname,
			COALESCE(MAX(NULLIF(a_uid.account_name, '')), MAX(NULLIF(a_id.account_name, '')), MAX(NULLIF(qar.aweme_name, '')), '') AS author_name,
			COALESCE(SUM(qar.stat_cost), 0) AS ad_cost,
			COALESCE(SUM(qar.pay_order_gmv), 0) AS pay_order_gmv,
			COALESCE(SUM(qar.pay_order_count), 0) AS pay_order_count
		FROM qianchuan_author_daily_report qar
		LEFT JOIN account a_id ON a_id.id = qar.account_id
		LEFT JOIN account a_uid ON a_uid.uid REGEXP '^[0-9]+$' AND CAST(a_uid.uid AS UNSIGNED) = qar.aweme_id
		WHERE LEFT(qar.stat_datetime, 10) >= ?
		  AND LEFT(qar.stat_datetime, 10) <= ?
		  AND %s = ?
		GROUP BY qar.aweme_id
		ORDER BY ad_cost DESC
	`, whereColumn), startDate, endDate, id)
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanCostDetail] 查询%s=%d消耗明细失败: %v", dimension, id, err)
		return []QianchuanCostDetailItem{}
	}

	result := make([]QianchuanCostDetailItem, len(rows))
	for i, row := range rows {
		douyinNickname := row["douyin_nickname"].String()
		result[i] = QianchuanCostDetailItem{
			AccountID:      row["account_id"].Int64(),
			AwemeID:        row["aweme_id"].Int64(),
			AwemeName:      douyinNickname,
			DouyinNickname: douyinNickname,
			AuthorName:     row["author_name"].String(),
			AdCost:         roundFloatFinance(row["ad_cost"].Float64(), 2),
			PayOrderGmv:    roundFloatFinance(row["pay_order_gmv"].Float64(), 2),
			OrderCount:     row["pay_order_count"].Int64(),
		}
	}
	return result
}

func (s *FinanceCommissionService) hasAuthorCostData(ctx context.Context, startDate, endDate string) bool {
	row, err := g.DB().Model("qianchuan_author_daily_report").
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COUNT(1) AS cnt").
		One()
	if err != nil {
		return false
	}
	return row["cnt"].Int64() > 0
}

// ExportAuthorCostRanking 导出达人维度消耗统计全量数据
func (s *FinanceCommissionService) ExportAuthorCostRanking(ctx context.Context, startDate, endDate string) (string, error) {
	list, _, err := s.GetAuthorCostRanking(ctx, startDate, endDate, 1, 1000000)
	if err != nil {
		return "", err
	}
	return writeExcelFile("达人维度消耗统计", []string{
		"排名", "抖音账号昵称", "投手名称", "达人ID", "千川消耗(元)", "成交金额(元)", "成交订单数",
	}, []float64{10, 28, 20, 24, 18, 18, 16}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			item.DouyinNickname,
			item.OperatorName,
			fmt.Sprintf("%d", item.AwemeID),
			item.StatCost,
			item.PayOrderGmv,
			item.OrderCount,
		}
	})
}

// ExportCostRanking 导出千川消耗统计全量数据
func (s *FinanceCommissionService) ExportCostRanking(ctx context.Context, startDate, endDate, dimension string) (string, error) {
	list, err := s.GetCostRanking(ctx, startDate, endDate, dimension, 1000000)
	if err != nil {
		return "", err
	}
	dimensionName := map[string]string{
		"trader":     "达人统计",
		"department": "部门统计",
		"company":    "公司统计",
	}[dimension]
	if dimensionName == "" {
		dimensionName = dimension
	}
	return writeExcelFile("千川消耗统计", []string{
		"排名", "统计维度", "名称", "千川消耗(元)",
	}, []float64{10, 18, 28, 18}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			dimensionName,
			gconv.String(item["name"]),
			roundFloatFinance(gconv.Float64(item["adCost"]), 2),
		}
	})
}

// ExportCostDetails 导出某个统计行下的账号消耗明细
func (s *FinanceCommissionService) ExportCostDetails(ctx context.Context, startDate, endDate, dimension string, id int64) (string, error) {
	list := s.costDetailsByDimension(ctx, startDate, endDate, dimension, id)
	return writeExcelFile("账号消耗明细", []string{
		"排名", "抖音账号昵称", "达人名称", "达人ID", "实际消耗(元)", "成交金额(元)", "成交订单数",
	}, []float64{10, 28, 28, 24, 18, 18, 16}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			item.DouyinNickname,
			item.AuthorName,
			fmt.Sprintf("%d", item.AwemeID),
			item.AdCost,
			item.PayOrderGmv,
			item.OrderCount,
		}
	})
}

// ExportStoreCostDetail 导出店铺消耗明细（按千川广告主账户维度，全量数据）。
func (s *FinanceCommissionService) ExportStoreCostDetail(ctx context.Context, startDate, endDate string, storeId int64) (string, error) {
	list, _, err := s.GetStoreCostDetail(ctx, startDate, endDate, storeId, 1, 1000000)
	if err != nil {
		return "", err
	}
	return writeExcelFile("店铺消耗明细", []string{
		"排名", "千川广告主ID", "千川账户", "千川消耗(元)", "成交金额(元)", "成交订单数", "展示数", "点击数", "ROI",
	}, []float64{8, 18, 24, 16, 16, 12, 12, 12, 10}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			fmt.Sprintf("%d", item.AdvertiserID),
			item.AdvertiserName,
			item.StatCost,
			item.PayOrderAmount,
			item.PayOrderCount,
			item.ShowCnt,
			item.ClickCnt,
			item.ROI,
		}
	})
}

// ExportSubjectCostDetail 兼容旧主体明细导出接口，subjectId 参数按 storeId 解释。
func (s *FinanceCommissionService) ExportSubjectCostDetail(ctx context.Context, startDate, endDate string, subjectId int64) (string, error) {
	return s.ExportStoreCostDetail(ctx, startDate, endDate, subjectId)
}

// OverallDetailExportItem 乘方明细导出项（供 controller 层填充）
type OverallDetailExportItem struct {
	AnchorName                                 string  `json:"anchorName"`
	AnchorShowID                               string  `json:"anchorShowId"`
	StatCostForOverallRoi2                     float64 `json:"statCostForOverallRoi2"`
	TotalPrepayAndPaySettleOverallRoi21h       float64 `json:"totalPrepayAndPaySettleOverallRoi21h"`
	TotalCostPerPayOrderSettleForOverallRoi21h float64 `json:"totalCostPerPayOrderSettleForOverallRoi21h"`
	ShopEstimatedComissionCost                 float64 `json:"shopEstimatedComissionCost"`
	TotalOrderSettleAmountForRoi21h            float64 `json:"totalOrderSettleAmountForRoi21h"`
	StatCostForRoi2                            float64 `json:"statCostForRoi2"`
	TotalOrderSettleCountForRoi21h             int64   `json:"totalOrderSettleCountForRoi21h"`
	TotalOrderRealSettleAmountForRoi21h        float64 `json:"totalOrderRealSettleAmountForRoi21h"`
}

// ExportOverallDetail 导出乘方抖音号明细（接收已拉取的明细数据列表）
func (s *FinanceCommissionService) ExportOverallDetail(list []OverallDetailExportItem) (string, error) {
	return writeExcelFile("乘方抖音号明细", []string{
		"排名", "抖音号", "抖音号ID", "综合成本(元)", "综合ROI", "综合订单成本", "预估达人佣金(元)",
		"净成交金额(元)", "整体消耗(元)", "净成交订单数", "用户实际支付净成交金额(元)",
	}, []float64{8, 22, 18, 16, 12, 16, 18, 18, 16, 14, 22}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			item.AnchorName,
			item.AnchorShowID,
			item.StatCostForOverallRoi2,
			item.TotalPrepayAndPaySettleOverallRoi21h,
			item.TotalCostPerPayOrderSettleForOverallRoi21h,
			item.ShopEstimatedComissionCost,
			item.TotalOrderSettleAmountForRoi21h,
			item.StatCostForRoi2,
			item.TotalOrderSettleCountForRoi21h,
			item.TotalOrderRealSettleAmountForRoi21h,
		}
	})
}

func writeExcelFile(sheetName string, headers []string, colWidths []float64, rowCount int, rowData func(rowIdx int) []interface{}) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := f.GetSheetName(0)
	if defaultSheet != sheetName {
		f.SetSheetName(defaultSheet, sheetName)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8F3FF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
		if i < len(colWidths) {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheetName, col, col, colWidths[i])
		}
	}

	for i := 0; i < rowCount; i++ {
		values := rowData(i)
		for colIdx, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, i+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	if err := os.MkdirAll("./exports", 0755); err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%s_%s.xlsx", sheetName, time.Now().Format("20060102150405"))
	filePath := filepath.Join("./exports", fileName)
	if err := f.SaveAs(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}
