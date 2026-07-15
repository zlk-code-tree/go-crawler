package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

const (
	QianchuanHierarchyLevelStore      = "store"
	QianchuanHierarchyLevelAweme      = "aweme"
	QianchuanHierarchyLevelAdvertiser = "advertiser"
	QianchuanHierarchyLevelPlan       = "plan"
)

type QianchuanPlanHierarchyRequest struct {
	Level        string
	StartDate    string
	EndDate      string
	Keyword      string
	ProductID    string
	StoreID      int64
	AccountID    int64
	AdvertiserID int64
	AwemeID      int64
	AwemeShowID  string
	GroupByDate  bool
}

type QianchuanPlanHierarchyNode struct {
	NodeKey          string  `json:"nodeKey"`
	NodeType         string  `json:"nodeType"`
	Name             string  `json:"name"`
	SecondaryName    string  `json:"secondaryName"`
	StatDate         string  `json:"statDate"`
	AdvertiserName   string  `json:"advertiserName"`
	StoreID          int64   `json:"storeId"`
	AccountID        int64   `json:"accountId"`
	AdvertiserID     int64   `json:"advertiserId"`
	AwemeID          int64   `json:"awemeId"`
	AdID             int64   `json:"adId"`
	ProductID        string  `json:"productId"`
	ProductName      string  `json:"productName"`
	AwemeName        string  `json:"awemeName"`
	AwemeShowID      string  `json:"awemeShowId"`
	HasChildren      bool    `json:"hasChildren"`
	Matched          bool    `json:"matched"`
	AdvertiserCount  int64   `json:"advertiserCount"`
	AccountCount     int64   `json:"accountCount"`
	StoreCount       int64   `json:"storeCount"`
	PlanCount        int64   `json:"planCount"`
	ProductCount     int64   `json:"productCount"`
	ProductQuantity  float64 `json:"productQuantity"`
	PayAmount        float64 `json:"payAmount"`
	StatCost         float64 `json:"statCost"`
	Commission       float64 `json:"commission"`
	ValidOrderCount  float64 `json:"validOrderCount"`
	CostAmount       float64 `json:"costAmount"`
	Profit           float64 `json:"profit"`
	PayOrderCount    float64 `json:"payOrderCount"`
	CreateOrderCount float64 `json:"createOrderCount"`
	ShowCnt          int64   `json:"showCnt"`
	ClickCnt         int64   `json:"clickCnt"`
	Ctr              float64 `json:"ctr"`
	ConvertCnt       int64   `json:"convertCnt"`
	ConvertCost      float64 `json:"convertCost"`
}

type QianchuanPlanHierarchySummary struct {
	StoreCount           int64   `json:"storeCount"`
	AwemeCount           int64   `json:"awemeCount"`
	AccountCount         int64   `json:"accountCount"`
	AdvertiserCount      int64   `json:"advertiserCount"`
	PlanCount            int64   `json:"planCount"`
	ProductCount         int64   `json:"productCount"`
	ProductQuantity      int64   `json:"productQuantity"`
	TotalOrderCount      int64   `json:"totalOrderCount"`
	TotalPayAmount       float64 `json:"totalPayAmount"`
	TotalCost            float64 `json:"totalCost"`
	TotalCommission      float64 `json:"totalCommission"`
	TotalValidOrderCount float64 `json:"totalValidOrderCount"`
	TotalOrderCost       float64 `json:"totalOrderCost"`
	TotalRefundCost      float64 `json:"totalRefundCost"`
	ReserveAmount        float64 `json:"reserveAmount"`
	ShopServiceFee       float64 `json:"shopServiceFee"`
	CommissionDeduction  float64 `json:"commissionDeduction"`
	TotalProfit          float64 `json:"totalProfit"`
	FinancialScope       string  `json:"financialScope"`
}

type QianchuanPlanHierarchyResult struct {
	List      []QianchuanPlanHierarchyNode  `json:"list"`
	Level     string                        `json:"level"`
	StartDate string                        `json:"startDate"`
	EndDate   string                        `json:"endDate"`
	Summary   QianchuanPlanHierarchySummary `json:"summary"`
}

type QianchuanPlanHierarchyService struct{}

func NewQianchuanPlanHierarchyService() *QianchuanPlanHierarchyService {
	return &QianchuanPlanHierarchyService{}
}

var qianchuanPlanAwemeExportHeaders = []string{
	"抖音号",
	"关联千川户",
	"关联店铺",
	"计划数",
	"商品数",
	"消耗(¥)",
	"佣金(¥)",
	"有效订单",
	"成本(¥)",
	"利润(¥)",
	"成交订单",
	"展示",
	"点击",
	"点击率",
}

// ExportAwemes 按页面抖音号视角的同一聚合口径导出全部或搜索后的指标。
func (s *QianchuanPlanHierarchyService) ExportAwemes(ctx context.Context, req QianchuanPlanHierarchyRequest) (string, error) {
	req.Keyword = strings.TrimSpace(req.Keyword)
	if err := validateQianchuanPlanAwemeExportRequest(req); err != nil {
		return "", err
	}

	result, err := s.Query(ctx, req)
	if err != nil {
		return "", err
	}
	if len(result.List) == 0 {
		return "", fmt.Errorf("没有可导出的抖音号数据")
	}

	return writeExcelFile(
		"千川计划抖音号指标",
		qianchuanPlanAwemeExportHeaders,
		[]float64{34, 13, 12, 10, 10, 14, 14, 12, 14, 14, 12, 12, 12, 12},
		len(result.List),
		func(rowIdx int) []interface{} {
			return qianchuanPlanAwemeExportRow(result.List[rowIdx])
		},
	)
}

func validateQianchuanPlanAwemeExportRequest(req QianchuanPlanHierarchyRequest) error {
	if req.Level != QianchuanHierarchyLevelAweme {
		return fmt.Errorf("只支持抖音号维度数据导出")
	}
	return nil
}

func qianchuanPlanAwemeExportRow(node QianchuanPlanHierarchyNode) []interface{} {
	return []interface{}{
		qianchuanPlanAwemeExportName(node),
		node.AdvertiserCount,
		node.StoreCount,
		node.PlanCount,
		node.ProductCount,
		node.StatCost,
		node.Commission,
		node.ValidOrderCount,
		node.CostAmount,
		node.Profit,
		node.PayOrderCount,
		node.ShowCnt,
		node.ClickCnt,
		fmt.Sprintf("%.2f%%", node.Ctr*100),
	}
}

func qianchuanPlanAwemeExportName(node QianchuanPlanHierarchyNode) string {
	name := strings.TrimSpace(node.Name)
	if name == "" {
		name = "未识别抖音号"
	}
	if showID := strings.TrimSpace(node.AwemeShowID); showID != "" {
		return fmt.Sprintf("%s（抖音号：%s）", name, showID)
	}
	if node.AwemeID > 0 {
		return fmt.Sprintf("%s（UID：%d）", name, node.AwemeID)
	}
	return name
}

func (s *QianchuanPlanHierarchyService) Query(ctx context.Context, req QianchuanPlanHierarchyRequest) (*QianchuanPlanHierarchyResult, error) {
	if err := NewQianchuanStoreMappingService().EnsureSchema(ctx); err != nil {
		return nil, err
	}
	startDate, endDate, endExclusive, err := normalizeQianchuanHierarchyDates(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	req.StartDate = startDate
	req.EndDate = endDate
	req.Keyword = strings.TrimSpace(req.Keyword)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.AwemeShowID = strings.TrimSpace(req.AwemeShowID)

	level := req.Level
	if level == "" {
		level = QianchuanHierarchyLevelStore
	}

	var list []QianchuanPlanHierarchyNode
	switch level {
	case QianchuanHierarchyLevelStore:
		list, err = s.queryStores(ctx, req, endExclusive)
	case QianchuanHierarchyLevelAweme:
		list, err = s.queryAwemes(ctx, req, endExclusive)
	case QianchuanHierarchyLevelAdvertiser:
		list, err = s.queryAdvertisers(ctx, req, endExclusive)
	case QianchuanHierarchyLevelPlan:
		list, err = s.queryPlans(ctx, req, endExclusive)
	default:
		return nil, fmt.Errorf("不支持的层级: %s", level)
	}
	if err != nil {
		return nil, err
	}

	result := &QianchuanPlanHierarchyResult{
		List:      list,
		Level:     level,
		StartDate: startDate,
		EndDate:   endDate,
	}
	result.Summary = summarizeQianchuanHierarchyNodes(list, level)
	if isUnfilteredQianchuanProfitRequest(req) {
		snapshot, snapshotErr := GetDouyinProfitSnapshot(ctx, startDate, endDate)
		if snapshotErr != nil {
			return nil, snapshotErr
		}
		applyDouyinProfitSnapshot(&result.Summary, snapshot)
	}
	return result, nil
}

func isUnfilteredQianchuanProfitRequest(req QianchuanPlanHierarchyRequest) bool {
	return req.Keyword == "" && req.ProductID == "" && req.StoreID == 0 && req.AccountID == 0 &&
		req.AdvertiserID == 0 && req.AwemeID == 0 && req.AwemeShowID == ""
}

func applyDouyinProfitSnapshot(summary *QianchuanPlanHierarchySummary, snapshot *DouyinProfitSnapshot) {
	if summary == nil || snapshot == nil {
		return
	}
	summary.TotalOrderCount = snapshot.TotalOrderCount
	summary.TotalValidOrderCount = float64(snapshot.ValidOrderCount)
	summary.ProductQuantity = snapshot.ProductQuantity
	summary.TotalPayAmount = snapshot.SalesAmount
	summary.TotalOrderCost = snapshot.CostAmount
	summary.TotalRefundCost = snapshot.RefundCostAmount
	summary.ReserveAmount = snapshot.ReserveAmount
	summary.ShopServiceFee = snapshot.ShopServiceFee
	summary.TotalCommission = snapshot.Commission
	summary.CommissionDeduction = snapshot.CommissionDeduction
	summary.TotalCost = snapshot.AdCostAmount
	summary.TotalProfit = snapshot.Profit
	summary.FinancialScope = "douyin_order_snapshot"
}

func normalizeQianchuanHierarchyDates(startDate, endDate string) (string, string, string, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	start, startErr := time.Parse("2006-01-02", startDate)
	end, endErr := time.Parse("2006-01-02", endDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		return "", "", "", fmt.Errorf("日期格式错误或结束日期早于开始日期")
	}
	return startDate, endDate, end.AddDate(0, 0, 1).Format("2006-01-02"), nil
}

func qianchuanHierarchyKeyword(alias, keyword string) (string, []interface{}) {
	if keyword == "" {
		return "", nil
	}
	like := "%" + keyword + "%"
	clause := fmt.Sprintf(` AND (
		%s.advertiser_name LIKE ? OR %s.ad_name LIKE ? OR %s.product_name LIKE ? OR %s.aweme_name LIKE ? OR
		%s.aweme_show_id LIKE ? OR CAST(%s.aweme_id AS CHAR) LIKE ? OR CAST(%s.ad_id AS CHAR) LIKE ? OR
		CAST(%s.advertiser_id AS CHAR) LIKE ? OR EXISTS (
			SELECT 1 FROM account keyword_account
			WHERE keyword_account.id = %s.account_id AND keyword_account.account_name LIKE ?
		) OR EXISTS (
			SELECT 1 FROM store_qianchuan_mapping keyword_mapping
			JOIN store keyword_store ON keyword_store.id = keyword_mapping.store_id
			WHERE keyword_mapping.qianchuan_advertiser_id = %s.advertiser_id
				AND keyword_store.store_name LIKE ?
		)
	)`, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias)
	return clause, []interface{}{like, like, like, like, like, like, like, like, like, like}
}

func qianchuanOrderStatsCTE(validCondition string) string {
	return fmt.Sprintf(`
	order_stats AS (
		SELECT
			s.id AS store_id,
			CAST(o.product_id AS CHAR) AS product_id,
			LEFT(o.pay_finish_time, 10) AS stat_date,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS valid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_quantity,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(gd.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN
				(o.product_price * o.product_num)
				- (COALESCE(gd.base_price, 0) * o.product_num)
				- (o.product_price * o.product_num * 0.03)
				- (o.product_price * o.product_num * 0.02)
			ELSE 0 END), 0) AS commission
		FROM order_info o
		INNER JOIN store s
			ON s.platform = 'douyin'
			AND s.status = 1
			AND s.store_name COLLATE utf8mb4_unicode_ci = o.shop_name COLLATE utf8mb4_unicode_ci
		INNER JOIN q_keys qk
			ON qk.store_id = s.id
			AND qk.product_id = CAST(o.product_id AS CHAR)
			AND qk.stat_date = LEFT(o.pay_finish_time, 10)
		LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) gd
			ON o.seller_code COLLATE utf8mb4_unicode_ci = gd.goods_code COLLATE utf8mb4_unicode_ci
		WHERE o.pay_finish_time >= ? AND o.pay_finish_time < ?
		GROUP BY s.id, CAST(o.product_id AS CHAR), LEFT(o.pay_finish_time, 10)
	)`, validCondition, validCondition, validCondition, validCondition, validCondition)
}

func (s *QianchuanPlanHierarchyService) queryStores(ctx context.Context, req QianchuanPlanHierarchyRequest, endExclusive string) ([]QianchuanPlanHierarchyNode, error) {
	keywordSQL, keywordArgs := qianchuanHierarchyKeyword("qar", req.Keyword)
	validCondition := FinanceValidOrderCondition("o")
	sql := fmt.Sprintf(`WITH
	q AS (
		SELECT
			COALESCE(sqm.store_id, 0) AS store_id,
			qar.account_id, qar.advertiser_id, qar.ad_id,
			CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost, qar.pay_order_count, qar.create_order_count,
			qar.show_cnt, qar.click_cnt, qar.convert_cnt
		FROM qianchuan_ad_report qar
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = qar.advertiser_id
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?%s
	),
	q_agg AS (
		SELECT
			store_id,
			MAX(account_id) AS account_id,
			COUNT(DISTINCT advertiser_id) AS advertiser_count,
			COUNT(DISTINCT ad_id) AS plan_count,
			COUNT(DISTINCT product_id) AS product_count,
			COALESCE(SUM(stat_cost), 0) AS stat_cost,
			COALESCE(SUM(pay_order_count), 0) AS pay_order_count,
			COALESCE(SUM(create_order_count), 0) AS create_order_count,
			COALESCE(SUM(show_cnt), 0) AS show_cnt,
			COALESCE(SUM(click_cnt), 0) AS click_cnt,
			COALESCE(SUM(convert_cnt), 0) AS convert_cnt
		FROM q
		GROUP BY store_id
	),
	q_keys AS (
		SELECT DISTINCT store_id, product_id, stat_date
		FROM q
		WHERE store_id > 0 AND product_id != '' AND product_id != '0'
	),%s,
	order_by_store AS (
		SELECT
			store_id,
			COALESCE(SUM(valid_order_count), 0) AS valid_order_count,
			COALESCE(SUM(product_quantity), 0) AS product_quantity,
			COALESCE(SUM(pay_amount), 0) AS pay_amount,
			COALESCE(SUM(cost_amount), 0) AS cost_amount,
			COALESCE(SUM(commission), 0) AS commission
		FROM order_stats
		GROUP BY store_id
	)
	SELECT
		q.store_id, q.account_id,
		COALESCE(NULLIF(st.store_name, ''), '未匹配店铺') AS store_name,
		q.advertiser_count, q.plan_count, q.product_count,
		q.stat_cost, q.pay_order_count, q.create_order_count,
		q.show_cnt, q.click_cnt, q.convert_cnt,
		COALESCE(o.valid_order_count, 0) AS valid_order_count,
		COALESCE(o.product_quantity, 0) AS product_quantity,
		COALESCE(o.pay_amount, 0) AS pay_amount,
		COALESCE(o.cost_amount, 0) AS cost_amount,
		COALESCE(o.commission, 0) AS commission
	FROM q_agg q
	LEFT JOIN store st ON st.id = q.store_id
	LEFT JOIN order_by_store o ON o.store_id = q.store_id
	ORDER BY (q.store_id = 0) ASC, q.stat_cost DESC, q.store_id ASC`, keywordSQL, qianchuanOrderStatsCTE(validCondition))

	args := []interface{}{req.StartDate, endExclusive}
	args = append(args, keywordArgs...)
	args = append(args, req.StartDate, endExclusive)
	rows, err := g.DB().GetAll(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(rows))
	for _, row := range rows {
		storeID := row["store_id"].Int64()
		accountID := row["account_id"].Int64()
		statCost := row["stat_cost"].Float64()
		commission := row["commission"].Float64()
		payAmount := row["pay_amount"].Float64()
		showCnt := row["show_cnt"].Int64()
		clickCnt := row["click_cnt"].Int64()
		convertCnt := row["convert_cnt"].Int64()
		list = append(list, QianchuanPlanHierarchyNode{
			NodeKey:          fmt.Sprintf("store:%d", storeID),
			NodeType:         QianchuanHierarchyLevelStore,
			Name:             row["store_name"].String(),
			SecondaryName:    storeHierarchySecondaryName(storeID, row["advertiser_count"].Int64(), row["plan_count"].Int64()),
			StoreID:          storeID,
			AccountID:        accountID,
			HasChildren:      row["advertiser_count"].Int64() > 0,
			Matched:          storeID > 0,
			AdvertiserCount:  row["advertiser_count"].Int64(),
			PlanCount:        row["plan_count"].Int64(),
			ProductCount:     row["product_count"].Int64(),
			ProductQuantity:  roundQianchuanHierarchy(row["product_quantity"].Float64()),
			PayAmount:        roundQianchuanHierarchy(payAmount),
			StatCost:         roundQianchuanHierarchy(statCost),
			Commission:       roundQianchuanHierarchy(commission),
			ValidOrderCount:  roundQianchuanHierarchy(row["valid_order_count"].Float64()),
			CostAmount:       roundQianchuanHierarchy(row["cost_amount"].Float64()),
			Profit:           roundQianchuanHierarchy(commission - statCost - payAmount*0.005),
			PayOrderCount:    roundQianchuanHierarchy(row["pay_order_count"].Float64()),
			CreateOrderCount: roundQianchuanHierarchy(row["create_order_count"].Float64()),
			ShowCnt:          showCnt,
			ClickCnt:         clickCnt,
			Ctr:              ratioQianchuanHierarchy(float64(clickCnt), float64(showCnt)),
			ConvertCnt:       convertCnt,
			ConvertCost:      ratioQianchuanHierarchy(statCost, float64(convertCnt)),
		})
	}
	return list, nil
}

func (s *QianchuanPlanHierarchyService) queryAwemes(ctx context.Context, req QianchuanPlanHierarchyRequest, endExclusive string) ([]QianchuanPlanHierarchyNode, error) {
	keywordSQL, keywordArgs := qianchuanHierarchyKeyword("qar", req.Keyword)
	storeSQL := ""
	if req.StoreID > 0 {
		storeSQL = " AND COALESCE(sqm.store_id, 0) = ?"
	}
	validCondition := FinanceValidOrderCondition("o")
	awemeKeySQL := `COALESCE(
		NULLIF(CAST(qar.aweme_id AS CHAR), '0'),
		NULLIF(qar.aweme_show_id, ''),
		CONCAT('unknown:', qar.advertiser_id)
	)`
	sql := fmt.Sprintf(`WITH
	all_q AS (
		SELECT
			COALESCE(sqm.store_id, 0) AS store_id,
			%s AS aweme_key,
			qar.ad_id, CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date, qar.stat_cost
		FROM qianchuan_ad_report qar
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = qar.advertiser_id
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?%s
	),
	q AS (
		SELECT
			COALESCE(sqm.store_id, 0) AS store_id,
			%s AS aweme_key,
			qar.aweme_id, qar.aweme_name, qar.aweme_show_id,
			qar.account_id, qar.advertiser_id, qar.ad_id,
			CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost, qar.pay_order_count, qar.create_order_count,
			qar.show_cnt, qar.click_cnt, qar.convert_cnt
		FROM qianchuan_ad_report qar
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = qar.advertiser_id
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?%s%s
	),
	q_totals AS (
		SELECT store_id, product_id, stat_date,
			COALESCE(SUM(stat_cost), 0) AS total_cost,
			COUNT(DISTINCT ad_id) AS total_plan_count
		FROM all_q
		GROUP BY store_id, product_id, stat_date
	),
	q_aweme_keys AS (
		SELECT store_id, aweme_key, product_id, stat_date,
			COALESCE(SUM(stat_cost), 0) AS key_cost,
			COUNT(DISTINCT ad_id) AS key_plan_count
		FROM q
		GROUP BY store_id, aweme_key, product_id, stat_date
	),
	q_keys AS (
		SELECT DISTINCT store_id, product_id, stat_date
		FROM q
		WHERE store_id > 0 AND product_id != '' AND product_id != '0'
	),%s,
	allocated AS (
		SELECT k.aweme_key,
			CASE WHEN t.total_cost > 0 THEN k.key_cost / t.total_cost
				 WHEN t.total_plan_count > 0 THEN k.key_plan_count / t.total_plan_count ELSE 0 END AS allocation_rate,
			COALESCE(o.valid_order_count, 0) AS valid_order_count,
			COALESCE(o.product_quantity, 0) AS product_quantity,
			COALESCE(o.pay_amount, 0) AS pay_amount,
			COALESCE(o.cost_amount, 0) AS cost_amount,
			COALESCE(o.commission, 0) AS commission
		FROM q_aweme_keys k
		JOIN q_totals t ON t.store_id = k.store_id AND t.product_id = k.product_id AND t.stat_date = k.stat_date
		LEFT JOIN order_stats o ON o.store_id = k.store_id AND o.product_id = k.product_id AND o.stat_date = k.stat_date
	),
	allocated_by_aweme AS (
		SELECT aweme_key,
			SUM(valid_order_count * allocation_rate) AS valid_order_count,
			SUM(product_quantity * allocation_rate) AS product_quantity,
			SUM(pay_amount * allocation_rate) AS pay_amount,
			SUM(cost_amount * allocation_rate) AS cost_amount,
			SUM(commission * allocation_rate) AS commission
		FROM allocated
		GROUP BY aweme_key
	),
	q_agg AS (
		SELECT aweme_key,
			MAX(aweme_id) AS aweme_id,
			MAX(NULLIF(aweme_name, '')) AS aweme_name,
			MAX(NULLIF(aweme_show_id, '')) AS aweme_show_id,
			CASE WHEN COUNT(DISTINCT NULLIF(store_id, 0)) = 1 THEN MAX(store_id) ELSE 0 END AS store_id,
			CASE WHEN COUNT(DISTINCT NULLIF(account_id, 0)) = 1 THEN MAX(account_id) ELSE 0 END AS account_id,
			COUNT(DISTINCT NULLIF(store_id, 0)) AS store_count,
			COUNT(DISTINCT NULLIF(account_id, 0)) AS account_count,
			COUNT(DISTINCT advertiser_id) AS advertiser_count,
			COUNT(DISTINCT ad_id) AS plan_count,
			COUNT(DISTINCT product_id) AS product_count,
			COALESCE(SUM(stat_cost), 0) AS stat_cost,
			COALESCE(SUM(pay_order_count), 0) AS pay_order_count,
			COALESCE(SUM(create_order_count), 0) AS create_order_count,
			COALESCE(SUM(show_cnt), 0) AS show_cnt,
			COALESCE(SUM(click_cnt), 0) AS click_cnt,
			COALESCE(SUM(convert_cnt), 0) AS convert_cnt
		FROM q
		GROUP BY aweme_key
	)
	SELECT q.*, COALESCE(a.valid_order_count, 0) AS valid_order_count,
		COALESCE(a.product_quantity, 0) AS product_quantity,
		COALESCE(a.pay_amount, 0) AS pay_amount,
		COALESCE(a.cost_amount, 0) AS cost_amount,
		COALESCE(a.commission, 0) AS commission
	FROM q_agg q
	LEFT JOIN allocated_by_aweme a ON a.aweme_key = q.aweme_key
	ORDER BY q.stat_cost DESC, q.aweme_key ASC`, awemeKeySQL, storeSQL, awemeKeySQL, storeSQL, keywordSQL, qianchuanOrderStatsCTE(validCondition))

	args := []interface{}{req.StartDate, endExclusive}
	if req.StoreID > 0 {
		args = append(args, req.StoreID)
	}
	args = append(args, req.StartDate, endExclusive)
	if req.StoreID > 0 {
		args = append(args, req.StoreID)
	}
	args = append(args, keywordArgs...)
	args = append(args, req.StartDate, endExclusive)
	rows, err := g.DB().GetAll(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(rows))
	for _, row := range rows {
		awemeID := row["aweme_id"].Int64()
		awemeShowID := row["aweme_show_id"].String()
		name := row["aweme_name"].String()
		if name == "" {
			name = awemeHierarchyFallbackName(awemeID, awemeShowID)
		}
		statCost := row["stat_cost"].Float64()
		commission := row["commission"].Float64()
		payAmount := row["pay_amount"].Float64()
		showCnt := row["show_cnt"].Int64()
		clickCnt := row["click_cnt"].Int64()
		convertCnt := row["convert_cnt"].Int64()
		list = append(list, QianchuanPlanHierarchyNode{
			NodeKey:          "aweme:" + row["aweme_key"].String(),
			NodeType:         QianchuanHierarchyLevelAweme,
			Name:             name,
			SecondaryName:    awemeHierarchySecondaryName(awemeShowID, row["advertiser_count"].Int64(), row["plan_count"].Int64()),
			StoreID:          row["store_id"].Int64(),
			AccountID:        row["account_id"].Int64(),
			AwemeID:          awemeID,
			AwemeName:        row["aweme_name"].String(),
			AwemeShowID:      awemeShowID,
			HasChildren:      row["plan_count"].Int64() > 0 && (awemeID > 0 || awemeShowID != ""),
			Matched:          awemeID > 0 || awemeShowID != "",
			StoreCount:       row["store_count"].Int64(),
			AccountCount:     row["account_count"].Int64(),
			AdvertiserCount:  row["advertiser_count"].Int64(),
			PlanCount:        row["plan_count"].Int64(),
			ProductCount:     row["product_count"].Int64(),
			ProductQuantity:  roundQianchuanHierarchy(row["product_quantity"].Float64()),
			PayAmount:        roundQianchuanHierarchy(payAmount),
			StatCost:         roundQianchuanHierarchy(statCost),
			Commission:       roundQianchuanHierarchy(commission),
			ValidOrderCount:  roundQianchuanHierarchy(row["valid_order_count"].Float64()),
			CostAmount:       roundQianchuanHierarchy(row["cost_amount"].Float64()),
			Profit:           roundQianchuanHierarchy(commission - statCost - payAmount*0.005),
			PayOrderCount:    roundQianchuanHierarchy(row["pay_order_count"].Float64()),
			CreateOrderCount: roundQianchuanHierarchy(row["create_order_count"].Float64()),
			ShowCnt:          showCnt,
			ClickCnt:         clickCnt,
			Ctr:              ratioQianchuanHierarchy(float64(clickCnt), float64(showCnt)),
			ConvertCnt:       convertCnt,
			ConvertCost:      ratioQianchuanHierarchy(statCost, float64(convertCnt)),
		})
	}
	return list, nil
}

func (s *QianchuanPlanHierarchyService) queryAdvertisers(ctx context.Context, req QianchuanPlanHierarchyRequest, endExclusive string) ([]QianchuanPlanHierarchyNode, error) {
	keywordSQL, keywordArgs := qianchuanHierarchyKeyword("qar", req.Keyword)
	allScopeSQL := ""
	qScopeSQL := ""
	allScopeArgs := make([]interface{}, 0, 1)
	qScopeArgs := make([]interface{}, 0, 2)
	hasAwemeScope := req.AwemeID > 0 || req.AwemeShowID != ""
	if req.StoreID > 0 {
		allScopeSQL += " AND COALESCE(sqm.store_id, 0) = ?"
		qScopeSQL += " AND COALESCE(sqm.store_id, 0) = ?"
		allScopeArgs = append(allScopeArgs, req.StoreID)
		qScopeArgs = append(qScopeArgs, req.StoreID)
	} else if !hasAwemeScope {
		allScopeSQL += " AND COALESCE(sqm.store_id, 0) = 0"
		qScopeSQL += " AND COALESCE(sqm.store_id, 0) = 0"
	}
	if req.AwemeID > 0 {
		qScopeSQL += " AND qar.aweme_id = ?"
		qScopeArgs = append(qScopeArgs, req.AwemeID)
	} else if req.AwemeShowID != "" {
		qScopeSQL += " AND qar.aweme_show_id = ?"
		qScopeArgs = append(qScopeArgs, req.AwemeShowID)
	}
	validCondition := FinanceValidOrderCondition("o")
	sql := fmt.Sprintf(`WITH
	all_q AS (
		SELECT
			COALESCE(sqm.store_id, 0) AS store_id,
			qar.account_id, qar.advertiser_id, qar.ad_id,
			CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost
		FROM qianchuan_ad_report qar
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = qar.advertiser_id
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?%s
	),
	q AS (
		SELECT
			COALESCE(sqm.store_id, 0) AS store_id,
			qar.account_id, qar.advertiser_id,
			qar.aweme_id, qar.aweme_name, qar.aweme_show_id,
			COALESCE(NULLIF(qar.advertiser_name, ''), NULLIF(sqm.qianchuan_advertiser_name, ''), '') AS advertiser_name,
			qar.ad_id,
			CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost, qar.pay_order_count, qar.create_order_count,
			qar.show_cnt, qar.click_cnt, qar.convert_cnt
		FROM qianchuan_ad_report qar
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = qar.advertiser_id
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?%s%s
	),
	q_totals AS (
		SELECT store_id, product_id, stat_date,
			COALESCE(SUM(stat_cost), 0) AS total_cost,
			COUNT(DISTINCT ad_id) AS total_plan_count
		FROM all_q
		GROUP BY store_id, product_id, stat_date
	),
	q_advertiser_keys AS (
		SELECT
			store_id, advertiser_id, product_id, stat_date,
			COALESCE(SUM(stat_cost), 0) AS key_cost,
			COUNT(DISTINCT ad_id) AS key_plan_count
		FROM q
		GROUP BY store_id, advertiser_id, product_id, stat_date
	),
	q_keys AS (
		SELECT DISTINCT store_id, product_id, stat_date
		FROM q
		WHERE store_id > 0 AND product_id != '' AND product_id != '0'
	),%s,
	allocated AS (
		SELECT
			k.store_id, k.advertiser_id,
			CASE WHEN t.total_cost > 0 THEN k.key_cost / t.total_cost
				 WHEN t.total_plan_count > 0 THEN k.key_plan_count / t.total_plan_count ELSE 0 END AS allocation_rate,
			COALESCE(o.valid_order_count, 0) AS valid_order_count,
			COALESCE(o.product_quantity, 0) AS product_quantity,
			COALESCE(o.pay_amount, 0) AS pay_amount,
			COALESCE(o.cost_amount, 0) AS cost_amount,
			COALESCE(o.commission, 0) AS commission
		FROM q_advertiser_keys k
		JOIN q_totals t ON t.store_id = k.store_id AND t.product_id = k.product_id AND t.stat_date = k.stat_date
		LEFT JOIN order_stats o ON o.store_id = k.store_id AND o.product_id = k.product_id AND o.stat_date = k.stat_date
	),
	allocated_by_advertiser AS (
		SELECT store_id, advertiser_id,
			SUM(valid_order_count * allocation_rate) AS valid_order_count,
			SUM(product_quantity * allocation_rate) AS product_quantity,
			SUM(pay_amount * allocation_rate) AS pay_amount,
			SUM(cost_amount * allocation_rate) AS cost_amount,
			SUM(commission * allocation_rate) AS commission
		FROM allocated
		GROUP BY store_id, advertiser_id
	),
	q_agg AS (
		SELECT
			store_id, advertiser_id,
			MAX(account_id) AS account_id,
			MAX(aweme_id) AS aweme_id,
			MAX(NULLIF(aweme_name, '')) AS aweme_name,
			MAX(NULLIF(aweme_show_id, '')) AS aweme_show_id,
			MAX(NULLIF(advertiser_name, '')) AS advertiser_name,
			COUNT(DISTINCT ad_id) AS plan_count,
			COUNT(DISTINCT product_id) AS product_count,
			SUM(stat_cost) AS stat_cost,
			SUM(pay_order_count) AS pay_order_count,
			SUM(create_order_count) AS create_order_count,
			SUM(show_cnt) AS show_cnt,
			SUM(click_cnt) AS click_cnt,
			SUM(convert_cnt) AS convert_cnt
		FROM q
		GROUP BY store_id, advertiser_id
	)
	SELECT q.*, COALESCE(a.valid_order_count, 0) AS valid_order_count,
		COALESCE(a.product_quantity, 0) AS product_quantity,
		COALESCE(a.pay_amount, 0) AS pay_amount,
		COALESCE(a.cost_amount, 0) AS cost_amount,
		COALESCE(a.commission, 0) AS commission
	FROM q_agg q
	LEFT JOIN allocated_by_advertiser a ON a.store_id = q.store_id AND a.advertiser_id = q.advertiser_id
	ORDER BY q.stat_cost DESC, q.advertiser_id ASC`, allScopeSQL, qScopeSQL, keywordSQL, qianchuanOrderStatsCTE(validCondition))

	args := []interface{}{req.StartDate, endExclusive}
	args = append(args, allScopeArgs...)
	args = append(args, req.StartDate, endExclusive)
	args = append(args, qScopeArgs...)
	args = append(args, keywordArgs...)
	args = append(args, req.StartDate, endExclusive)
	rows, err := g.DB().GetAll(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(rows))
	for _, row := range rows {
		advertiserID := row["advertiser_id"].Int64()
		name := row["advertiser_name"].String()
		if name == "" {
			name = fmt.Sprintf("千川账户 %d", advertiserID)
		}
		statCost := row["stat_cost"].Float64()
		commission := row["commission"].Float64()
		payAmount := row["pay_amount"].Float64()
		showCnt := row["show_cnt"].Int64()
		clickCnt := row["click_cnt"].Int64()
		convertCnt := row["convert_cnt"].Int64()
		nodeKey := fmt.Sprintf("advertiser:%d", advertiserID)
		if req.AwemeID > 0 {
			nodeKey += fmt.Sprintf(":aweme:%d", req.AwemeID)
		} else if req.AwemeShowID != "" {
			nodeKey += ":aweme-show:" + req.AwemeShowID
		}
		list = append(list, QianchuanPlanHierarchyNode{
			NodeKey:          nodeKey,
			NodeType:         QianchuanHierarchyLevelAdvertiser,
			Name:             name,
			SecondaryName:    fmt.Sprintf("ID %d · %d 个计划", advertiserID, row["plan_count"].Int64()),
			StoreID:          row["store_id"].Int64(),
			AccountID:        row["account_id"].Int64(),
			AdvertiserID:     advertiserID,
			AwemeID:          row["aweme_id"].Int64(),
			AwemeName:        row["aweme_name"].String(),
			AwemeShowID:      row["aweme_show_id"].String(),
			HasChildren:      row["plan_count"].Int64() > 0,
			Matched:          row["store_id"].Int64() > 0,
			PlanCount:        row["plan_count"].Int64(),
			ProductCount:     row["product_count"].Int64(),
			ProductQuantity:  roundQianchuanHierarchy(row["product_quantity"].Float64()),
			PayAmount:        roundQianchuanHierarchy(payAmount),
			StatCost:         roundQianchuanHierarchy(statCost),
			Commission:       roundQianchuanHierarchy(commission),
			ValidOrderCount:  roundQianchuanHierarchy(row["valid_order_count"].Float64()),
			CostAmount:       roundQianchuanHierarchy(row["cost_amount"].Float64()),
			Profit:           roundQianchuanHierarchy(commission - statCost - payAmount*0.005),
			PayOrderCount:    roundQianchuanHierarchy(row["pay_order_count"].Float64()),
			CreateOrderCount: roundQianchuanHierarchy(row["create_order_count"].Float64()),
			ShowCnt:          showCnt,
			ClickCnt:         clickCnt,
			Ctr:              ratioQianchuanHierarchy(float64(clickCnt), float64(showCnt)),
			ConvertCnt:       convertCnt,
			ConvertCost:      ratioQianchuanHierarchy(statCost, float64(convertCnt)),
		})
	}
	return list, nil
}

func (s *QianchuanPlanHierarchyService) queryPlans(ctx context.Context, req QianchuanPlanHierarchyRequest, endExclusive string) ([]QianchuanPlanHierarchyNode, error) {
	keywordSQL, keywordArgs := qianchuanHierarchyKeyword("daily", req.Keyword)
	filterSQL := ""
	filterArgs := make([]interface{}, 0, 6)
	if req.AdvertiserID > 0 {
		filterSQL += " AND daily.advertiser_id = ?"
		filterArgs = append(filterArgs, req.AdvertiserID)
	}
	if req.AwemeID > 0 {
		filterSQL += " AND daily.aweme_id = ?"
		filterArgs = append(filterArgs, req.AwemeID)
	} else if req.AwemeShowID != "" {
		filterSQL += " AND daily.aweme_show_id = ?"
		filterArgs = append(filterArgs, req.AwemeShowID)
	}
	if req.StoreID > 0 {
		filterSQL += " AND daily.store_id = ?"
		filterArgs = append(filterArgs, req.StoreID)
	}
	if req.AccountID > 0 {
		filterSQL += " AND daily.account_id = ?"
		filterArgs = append(filterArgs, req.AccountID)
	}
	if req.ProductID != "" {
		filterSQL += " AND daily.product_id = ?"
		filterArgs = append(filterArgs, req.ProductID)
	}
	sql := fmt.Sprintf(`SELECT daily.*
	FROM (
		SELECT
			COALESCE(sqm.store_id, 0) AS store_id,
			qar.account_id, qar.advertiser_id, qar.ad_id, qar.ad_name,
			COALESCE(NULLIF(qar.advertiser_name, ''), NULLIF(sqm.qianchuan_advertiser_name, ''), '') AS advertiser_name,
			CAST(qar.product_id AS CHAR) AS product_id, qar.product_name,
			qar.aweme_id, qar.aweme_name, qar.aweme_show_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost, qar.pay_order_count, qar.create_order_count,
			qar.show_cnt, qar.click_cnt, qar.convert_cnt,
			SUM(qar.stat_cost) OVER (
				PARTITION BY COALESCE(sqm.store_id, 0), qar.product_id, LEFT(qar.stat_datetime, 10)
			) AS total_cost,
			COUNT(*) OVER (
				PARTITION BY COALESCE(sqm.store_id, 0), qar.product_id, LEFT(qar.stat_datetime, 10)
			) AS total_plan_count
		FROM qianchuan_ad_report qar
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = qar.advertiser_id
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?
	) daily
	WHERE 1 = 1%s%s
	ORDER BY daily.stat_date DESC, daily.ad_id ASC`, filterSQL, keywordSQL)

	args := []interface{}{req.StartDate, endExclusive}
	args = append(args, filterArgs...)
	args = append(args, keywordArgs...)
	rows, err := g.DB().GetAll(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []QianchuanPlanHierarchyNode{}, nil
	}

	productsByStore := make(map[int64]map[string]struct{})
	for _, row := range rows {
		storeID := row["store_id"].Int64()
		productID := row["product_id"].String()
		if storeID <= 0 || productID == "" || productID == "0" {
			continue
		}
		if productsByStore[storeID] == nil {
			productsByStore[storeID] = make(map[string]struct{})
		}
		productsByStore[storeID][productID] = struct{}{}
	}
	orderStats, err := loadQianchuanPlanOrderStats(ctx, productsByStore, req.StartDate, endExclusive)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[string]*QianchuanPlanHierarchyNode)
	payAmountMap := make(map[string]float64)
	for _, row := range rows {
		adID := row["ad_id"].Int64()
		statDate := row["stat_date"].String()
		mapKey := fmt.Sprintf("%d", adID)
		if req.GroupByDate {
			mapKey += "|" + statDate
		}
		node := nodeMap[mapKey]
		if node == nil {
			name := row["ad_name"].String()
			if name == "" {
				name = fmt.Sprintf("计划 %d", adID)
			}
			nodeKey := fmt.Sprintf("plan:%d", adID)
			if req.GroupByDate {
				nodeKey += ":" + statDate
			}
			node = &QianchuanPlanHierarchyNode{
				NodeKey:        nodeKey,
				NodeType:       QianchuanHierarchyLevelPlan,
				Name:           name,
				StatDate:       statDate,
				AdvertiserName: row["advertiser_name"].String(),
				StoreID:        row["store_id"].Int64(),
				AccountID:      row["account_id"].Int64(),
				AdvertiserID:   row["advertiser_id"].Int64(),
				AwemeID:        row["aweme_id"].Int64(),
				AdID:           adID,
				ProductID:      row["product_id"].String(),
				ProductName:    row["product_name"].String(),
				AwemeName:      row["aweme_name"].String(),
				AwemeShowID:    row["aweme_show_id"].String(),
				Matched:        row["store_id"].Int64() > 0,
				PlanCount:      1,
			}
			if node.ProductID != "" && node.ProductID != "0" {
				node.ProductCount = 1
			}
			nodeMap[mapKey] = node
		}

		statCost := row["stat_cost"].Float64()
		node.StatCost += statCost
		node.PayOrderCount += row["pay_order_count"].Float64()
		node.CreateOrderCount += row["create_order_count"].Float64()
		node.ShowCnt += row["show_cnt"].Int64()
		node.ClickCnt += row["click_cnt"].Int64()
		node.ConvertCnt += row["convert_cnt"].Int64()

		totalCost := row["total_cost"].Float64()
		totalPlanCount := row["total_plan_count"].Float64()
		allocationRate := 0.0
		if totalCost > 0 {
			allocationRate = statCost / totalCost
		} else if totalPlanCount > 0 {
			allocationRate = 1 / totalPlanCount
		}
		orderStat := orderStats[qianchuanOrderStatKey(row["store_id"].Int64(), row["product_id"].String(), row["stat_date"].String())]
		node.ValidOrderCount += orderStat.ValidOrderCount * allocationRate
		node.ProductQuantity += orderStat.ProductQuantity * allocationRate
		node.PayAmount += orderStat.PayAmount * allocationRate
		node.CostAmount += orderStat.CostAmount * allocationRate
		node.Commission += orderStat.Commission * allocationRate
		payAmountMap[mapKey] += orderStat.PayAmount * allocationRate
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(nodeMap))
	for mapKey, node := range nodeMap {
		adID := node.AdID
		node.SecondaryName = planHierarchySecondaryName(node.ProductName, node.ProductID, adID)
		node.StatCost = roundQianchuanHierarchy(node.StatCost)
		node.Commission = roundQianchuanHierarchy(node.Commission)
		node.ValidOrderCount = roundQianchuanHierarchy(node.ValidOrderCount)
		node.ProductQuantity = roundQianchuanHierarchy(node.ProductQuantity)
		node.PayAmount = roundQianchuanHierarchy(node.PayAmount)
		node.CostAmount = roundQianchuanHierarchy(node.CostAmount)
		node.Profit = roundQianchuanHierarchy(node.Commission - node.StatCost - payAmountMap[mapKey]*0.005)
		node.PayOrderCount = roundQianchuanHierarchy(node.PayOrderCount)
		node.CreateOrderCount = roundQianchuanHierarchy(node.CreateOrderCount)
		node.Ctr = ratioQianchuanHierarchy(float64(node.ClickCnt), float64(node.ShowCnt))
		node.ConvertCost = ratioQianchuanHierarchy(node.StatCost, float64(node.ConvertCnt))
		list = append(list, *node)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].StatDate != list[j].StatDate {
			return list[i].StatDate > list[j].StatDate
		}
		if list[i].StatCost == list[j].StatCost {
			return list[i].AdID < list[j].AdID
		}
		return list[i].StatCost > list[j].StatCost
	})
	return list, nil
}

type qianchuanPlanOrderStat struct {
	ValidOrderCount float64
	ProductQuantity float64
	PayAmount       float64
	CostAmount      float64
	Commission      float64
}

func loadQianchuanPlanOrderStats(ctx context.Context, productsByStore map[int64]map[string]struct{}, startDate, endExclusive string) (map[string]qianchuanPlanOrderStat, error) {
	result := make(map[string]qianchuanPlanOrderStat)
	validCondition := FinanceValidOrderCondition("o")
	for storeID, productSet := range productsByStore {
		store, err := g.DB().Model("store").Ctx(ctx).Where("id", storeID).Fields("store_name").One()
		if err != nil {
			return nil, err
		}
		if store.IsEmpty() || len(productSet) == 0 {
			continue
		}

		products := make([]string, 0, len(productSet))
		placeholders := make([]string, 0, len(productSet))
		for productID := range productSet {
			products = append(products, productID)
			placeholders = append(placeholders, "?")
		}
		sort.Strings(products)

		sql := fmt.Sprintf(`SELECT
			o.product_id,
			LEFT(o.pay_finish_time, 10) AS stat_date,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS valid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_quantity,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(gd.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN
				(o.product_price * o.product_num)
				- (COALESCE(gd.base_price, 0) * o.product_num)
				- (o.product_price * o.product_num * 0.03)
				- (o.product_price * o.product_num * 0.02)
			ELSE 0 END), 0) AS commission
		FROM order_info o
		LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) gd
			ON o.seller_code COLLATE utf8mb4_unicode_ci = gd.goods_code COLLATE utf8mb4_unicode_ci
		WHERE o.shop_name = ?
			AND o.pay_finish_time >= ? AND o.pay_finish_time < ?
			AND o.product_id IN (%s)
		GROUP BY o.product_id, LEFT(o.pay_finish_time, 10)`, validCondition, validCondition, validCondition, validCondition, validCondition, strings.Join(placeholders, ","))

		args := []interface{}{store["store_name"].String(), startDate, endExclusive}
		for _, productID := range products {
			args = append(args, productID)
		}
		rows, err := g.DB().GetAll(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result[qianchuanOrderStatKey(storeID, row["product_id"].String(), row["stat_date"].String())] = qianchuanPlanOrderStat{
				ValidOrderCount: row["valid_order_count"].Float64(),
				ProductQuantity: row["product_quantity"].Float64(),
				PayAmount:       row["pay_amount"].Float64(),
				CostAmount:      row["cost_amount"].Float64(),
				Commission:      row["commission"].Float64(),
			}
		}
	}
	return result, nil
}

func qianchuanOrderStatKey(storeID int64, productID, statDate string) string {
	return fmt.Sprintf("%d|%s|%s", storeID, productID, statDate)
}

func summarizeQianchuanHierarchyNodes(list []QianchuanPlanHierarchyNode, level string) QianchuanPlanHierarchySummary {
	var summary QianchuanPlanHierarchySummary
	for _, item := range list {
		if level == QianchuanHierarchyLevelStore && item.StoreID > 0 {
			summary.StoreCount++
		}
		if level == QianchuanHierarchyLevelAweme {
			summary.AwemeCount++
			summary.StoreCount += item.StoreCount
			summary.AccountCount += item.AccountCount
		}
		summary.AdvertiserCount += item.AdvertiserCount
		summary.PlanCount += item.PlanCount
		summary.ProductCount += item.ProductCount
		summary.ProductQuantity += int64(math.Round(item.ProductQuantity))
		summary.TotalPayAmount += item.PayAmount
		summary.TotalCost += item.StatCost
		summary.TotalCommission += item.Commission
		summary.TotalValidOrderCount += item.ValidOrderCount
		summary.TotalOrderCost += item.CostAmount
		summary.TotalProfit += item.Profit
	}
	summary.TotalCost = roundQianchuanHierarchy(summary.TotalCost)
	summary.TotalPayAmount = roundQianchuanHierarchy(summary.TotalPayAmount)
	summary.TotalCommission = roundQianchuanHierarchy(summary.TotalCommission)
	summary.TotalValidOrderCount = roundQianchuanHierarchy(summary.TotalValidOrderCount)
	summary.TotalOrderCost = roundQianchuanHierarchy(summary.TotalOrderCost)
	summary.ReserveAmount = CalculateReserveAmount(summary.TotalPayAmount)
	summary.ShopServiceFee = CalculateDouyinShopServiceFee(summary.TotalPayAmount)
	summary.CommissionDeduction = roundQianchuanHierarchy(summary.TotalPayAmount * 0.005)
	summary.TotalProfit = roundQianchuanHierarchy(summary.TotalProfit)
	summary.FinancialScope = "allocated_plan_orders"
	return summary
}

func summarizeQianchuanStoreNodes(list []QianchuanPlanHierarchyNode) QianchuanPlanHierarchySummary {
	return summarizeQianchuanHierarchyNodes(list, QianchuanHierarchyLevelStore)
}

func storeHierarchySecondaryName(storeID, advertiserCount, planCount int64) string {
	if storeID == 0 {
		return fmt.Sprintf("%d 个千川账户待绑定 · %d 个计划", advertiserCount, planCount)
	}
	return fmt.Sprintf("%d 个千川账户 · %d 个计划", advertiserCount, planCount)
}

func awemeHierarchyFallbackName(awemeID int64, awemeShowID string) string {
	if awemeShowID != "" {
		return "抖音号 " + awemeShowID
	}
	if awemeID > 0 {
		return fmt.Sprintf("抖音号 UID %d", awemeID)
	}
	return "未识别抖音号"
}

func awemeHierarchySecondaryName(awemeShowID string, advertiserCount, planCount int64) string {
	prefix := "未获取抖音号"
	if awemeShowID != "" {
		prefix = "抖音号 " + awemeShowID
	}
	return fmt.Sprintf("%s · %d 个千川账户 · %d 个计划", prefix, advertiserCount, planCount)
}

func planHierarchySecondaryName(productName, productID string, adID int64) string {
	if productName != "" {
		return fmt.Sprintf("%s · 计划ID %d", productName, adID)
	}
	if productID != "" && productID != "0" {
		return fmt.Sprintf("商品ID %s · 计划ID %d", productID, adID)
	}
	return fmt.Sprintf("计划ID %d", adID)
}

func roundQianchuanHierarchy(value float64) float64 {
	return math.Round(value*100) / 100
}

func ratioQianchuanHierarchy(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return math.Round((numerator/denominator)*10000) / 10000
}
