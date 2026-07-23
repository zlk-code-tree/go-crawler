package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

const (
	QianchuanHierarchyLevelStore      = "store"
	QianchuanHierarchyLevelAweme      = "aweme"
	QianchuanHierarchyLevelTalent     = "talent"
	QianchuanHierarchyLevelAdvertiser = "advertiser"
	QianchuanHierarchyLevelPlan       = "plan"
	qianchuanUnassignedTalentKey      = "unassigned"
)

type QianchuanPlanHierarchyRequest struct {
	Level        string
	StartDate    string
	EndDate      string
	Keyword      string
	ProductID    string
	AwemeKeyword string
	StoreID      int64
	AccountID    int64
	AdvertiserID int64
	AwemeID      int64
	AwemeShowID  string
	TalentKey    string
	GroupByDate  bool
	AllStores    bool
}

type QianchuanPlanHierarchyNode struct {
	NodeKey          string  `json:"nodeKey"`
	NodeType         string  `json:"nodeType"`
	Name             string  `json:"name"`
	SecondaryName    string  `json:"secondaryName"`
	StatDate         string  `json:"statDate"`
	AdvertiserName   string  `json:"advertiserName"`
	StoreID          int64   `json:"storeId"`
	StoreName        string  `json:"storeName"`
	AccountID        int64   `json:"accountId"`
	AdvertiserID     int64   `json:"advertiserId"`
	AwemeID          int64   `json:"awemeId"`
	AdID             int64   `json:"adId"`
	ProductID        string  `json:"productId"`
	ProductName      string  `json:"productName"`
	AwemeName        string  `json:"awemeName"`
	AwemeShowID      string  `json:"awemeShowId"`
	TalentKey        string  `json:"talentKey"`
	HasChildren      bool    `json:"hasChildren"`
	Matched          bool    `json:"matched"`
	AdvertiserCount  int64   `json:"advertiserCount"`
	AccountCount     int64   `json:"accountCount"`
	StoreCount       int64   `json:"storeCount"`
	PlanCount        int64   `json:"planCount"`
	ProductCount     int64   `json:"productCount"`
	ProductQuantity  float64 `json:"productQuantity"`
	TotalOrderCount  int64   `json:"totalOrderCount"`
	PayAmount        float64 `json:"payAmount"`
	StatCost         float64 `json:"statCost"`
	NormalCost       float64 `json:"normalCost"`
	GapCost          float64 `json:"gapCost"`
	UnassignedCost   float64 `json:"unassignedCost"`
	Commission       float64 `json:"commission"`
	ValidOrderCount  float64 `json:"validOrderCount"`
	CostAmount       float64 `json:"costAmount"`
	RefundCost       float64 `json:"refundCost"`
	ReserveAmount    float64 `json:"reserveAmount"`
	ShopServiceFee   float64 `json:"shopServiceFee"`
	CommissionDeduct float64 `json:"commissionDeduction"`
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
	TalentCount          int64   `json:"talentCount"`
	AccountCount         int64   `json:"accountCount"`
	AdvertiserCount      int64   `json:"advertiserCount"`
	PlanCount            int64   `json:"planCount"`
	ProductCount         int64   `json:"productCount"`
	ProductQuantity      int64   `json:"productQuantity"`
	TotalOrderCount      int64   `json:"totalOrderCount"`
	TotalPayAmount       float64 `json:"totalPayAmount"`
	TotalCost            float64 `json:"totalCost"`
	NormalCost           float64 `json:"normalCost"`
	GapCost              float64 `json:"gapCost"`
	UnassignedCost       float64 `json:"unassignedCost"`
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
	req.AwemeKeyword = strings.TrimSpace(req.AwemeKeyword)
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
	case QianchuanHierarchyLevelTalent:
		list, err = s.queryTalents(ctx, req, endExclusive)
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
		breakdown, breakdownErr := loadQianchuanGlobalCostBreakdown(ctx, startDate, endDate)
		if breakdownErr != nil {
			return nil, breakdownErr
		}
		result.Summary.NormalCost = roundQianchuanHierarchy(breakdown.NormalCost)
		result.Summary.GapCost = roundQianchuanHierarchy(breakdown.GapCost)
		result.Summary.UnassignedCost = roundQianchuanHierarchy(
			snapshot.AdCostAmount - result.Summary.NormalCost - result.Summary.GapCost,
		)
		if math.Abs(result.Summary.UnassignedCost) < 0.005 {
			result.Summary.UnassignedCost = 0
		}
	}
	return result, nil
}

func isUnfilteredQianchuanProfitRequest(req QianchuanPlanHierarchyRequest) bool {
	return req.Keyword == "" && req.ProductID == "" && req.AwemeKeyword == "" && req.StoreID == 0 && req.AccountID == 0 &&
		req.AdvertiserID == 0 && req.AwemeID == 0 && req.AwemeShowID == "" && req.TalentKey == ""
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

func loadQianchuanGlobalCostBreakdown(ctx context.Context, startDate, endDate string) (qianchuanTalentCostStats, error) {
	row, err := g.DB().GetOne(ctx, attributedQianchuanCostCTE+`
		SELECT
			COALESCE(SUM(CASE WHEN cost_type = 'normal' THEN stat_cost ELSE 0 END), 0) AS normal_cost,
			COALESCE(SUM(CASE WHEN cost_type = 'gap' THEN stat_cost ELSE 0 END), 0) AS gap_cost,
			COALESCE(SUM(CASE WHEN cost_type = 'unassigned' THEN stat_cost ELSE 0 END), 0) AS unassigned_cost
		FROM attributed_qianchuan_cost`,
		qianchuanAttributionDateArgs(startDate, endDate)...)
	if err != nil {
		return qianchuanTalentCostStats{}, err
	}
	return qianchuanTalentCostStats{
		NormalCost:     row["normal_cost"].Float64(),
		GapCost:        row["gap_cost"].Float64(),
		UnassignedCost: row["unassigned_cost"].Float64(),
	}, nil
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

func qianchuanAwemeKeyword(alias, keyword string) (string, []interface{}) {
	if keyword == "" {
		return "", nil
	}
	like := "%" + keyword + "%"
	clause := fmt.Sprintf(` AND (
		%s.aweme_show_id = ? OR CAST(%s.aweme_id AS CHAR) = ? OR %s.aweme_name LIKE ?
	)`, alias, alias, alias)
	return clause, []interface{}{keyword, keyword, like}
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

type qianchuanTalentOwner struct {
	Key     string
	Name    string
	Matched bool
}

func loadQianchuanTalentOwners(ctx context.Context) (map[int64]qianchuanTalentOwner, error) {
	rows, err := g.DB().Model("qianchuan_account_owner_mapping").Ctx(ctx).
		Fields("advertiser_id, owner_name, trader_id").
		Where("status", 1).
		All()
	if err != nil {
		return nil, err
	}
	owners := make(map[int64]qianchuanTalentOwner, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row["owner_name"].String())
		traderID := row["trader_id"].Int64()
		if name == "" || traderID <= 0 {
			continue
		}
		owners[row["advertiser_id"].Int64()] = qianchuanTalentOwner{
			Key:     fmt.Sprintf("trader:%d", traderID),
			Name:    name,
			Matched: true,
		}
	}
	return owners, nil
}

func qianchuanTalentOwnerForAdvertiser(owners map[int64]qianchuanTalentOwner, advertiserID int64) qianchuanTalentOwner {
	if owner, ok := owners[advertiserID]; ok {
		return owner
	}
	return qianchuanTalentOwner{
		Key:  qianchuanUnassignedTalentKey,
		Name: "未归属达人",
	}
}

type qianchuanTalentCostStats struct {
	NormalCost      float64
	GapCost         float64
	UnassignedCost  float64
	AdvertiserCount int64
	AccountCount    int64
	StoreCount      int64
	PlanCount       int64
}

func (s qianchuanTalentCostStats) TotalCost() float64 {
	return s.NormalCost + s.GapCost + s.UnassignedCost
}

func loadQianchuanTalentCostStats(ctx context.Context, startDate, endDate string) (map[int64]qianchuanTalentCostStats, error) {
	args := qianchuanAttributionDateArgs(startDate, endDate)
	rows, err := g.DB().GetAll(ctx, attributedQianchuanCostCTE+`,
		cost_summary AS (
			SELECT
				resolved_trader_id,
				COALESCE(SUM(CASE WHEN cost_type = 'normal' THEN stat_cost ELSE 0 END), 0) AS normal_cost,
				COALESCE(SUM(CASE WHEN cost_type = 'gap' THEN stat_cost ELSE 0 END), 0) AS gap_cost,
				COALESCE(SUM(CASE WHEN cost_type = 'unassigned' THEN stat_cost ELSE 0 END), 0) AS unassigned_cost
			FROM attributed_qianchuan_cost
			GROUP BY resolved_trader_id
		),
		association_summary AS (
			SELECT
				cost.resolved_trader_id,
				COUNT(DISTINCT cost.advertiser_id) AS advertiser_count,
				COUNT(DISTINCT CASE WHEN cost.account_id > 0 THEN cost.account_id END) AS account_count,
				COUNT(DISTINCT CASE WHEN mapping.store_id > 0 THEN mapping.store_id END) AS store_count
			FROM attributed_qianchuan_cost cost
			LEFT JOIN store_qianchuan_mapping mapping
				ON mapping.qianchuan_advertiser_id = cost.advertiser_id
			GROUP BY cost.resolved_trader_id
		),
		plan_summary AS (
			SELECT
				cost.resolved_trader_id,
				COUNT(DISTINCT CONCAT(report.advertiser_id, ':', report.ad_id)) AS plan_count
			FROM attributed_qianchuan_cost cost
			INNER JOIN qianchuan_ad_report report
				ON cost.cost_type = 'normal'
				AND report.account_id = cost.account_id
				AND report.advertiser_id = cost.advertiser_id
				AND LEFT(report.stat_datetime, 10) = cost.stat_date
			GROUP BY cost.resolved_trader_id
		)
		SELECT
			cost.resolved_trader_id,
			cost.normal_cost,
			cost.gap_cost,
			cost.unassigned_cost,
			COALESCE(association.advertiser_count, 0) AS advertiser_count,
			COALESCE(association.account_count, 0) AS account_count,
			COALESCE(association.store_count, 0) AS store_count,
			COALESCE(plan.plan_count, 0) AS plan_count
		FROM cost_summary cost
		LEFT JOIN association_summary association
			ON association.resolved_trader_id = cost.resolved_trader_id
		LEFT JOIN plan_summary plan
			ON plan.resolved_trader_id = cost.resolved_trader_id`, args...)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]qianchuanTalentCostStats, len(rows))
	for _, row := range rows {
		result[row["resolved_trader_id"].Int64()] = qianchuanTalentCostStats{
			NormalCost:      row["normal_cost"].Float64(),
			GapCost:         row["gap_cost"].Float64(),
			UnassignedCost:  row["unassigned_cost"].Float64(),
			AdvertiserCount: row["advertiser_count"].Int64(),
			AccountCount:    row["account_count"].Int64(),
			StoreCount:      row["store_count"].Int64(),
			PlanCount:       row["plan_count"].Int64(),
		}
	}
	return result, nil
}

func qianchuanTalentNodeFromFinance(traderID int64, name string, stats model.FinanceStatsSummary, cost qianchuanTalentCostStats) QianchuanPlanHierarchyNode {
	commissionDeduction := roundQianchuanHierarchy(stats.PayAmount * 0.005)
	return QianchuanPlanHierarchyNode{
		NodeKey:          fmt.Sprintf("talent:trader:%d", traderID),
		NodeType:         QianchuanHierarchyLevelTalent,
		Name:             name,
		SecondaryName:    fmt.Sprintf("%d 个千川账户 · %d 个计划", cost.AdvertiserCount, cost.PlanCount),
		TalentKey:        fmt.Sprintf("trader:%d", traderID),
		HasChildren:      cost.AdvertiserCount > 0,
		Matched:          true,
		AdvertiserCount:  cost.AdvertiserCount,
		AccountCount:     cost.AccountCount,
		StoreCount:       cost.StoreCount,
		PlanCount:        cost.PlanCount,
		ProductQuantity:  float64(stats.ProductNum),
		TotalOrderCount:  int64(stats.TotalOrderCount),
		PayAmount:        stats.PayAmount,
		StatCost:         cost.TotalCost(),
		NormalCost:       cost.NormalCost,
		GapCost:          cost.GapCost,
		UnassignedCost:   cost.UnassignedCost,
		Commission:       stats.Commission,
		ValidOrderCount:  float64(stats.OrderCount),
		CostAmount:       stats.CostAmount,
		RefundCost:       stats.RefundCostAmount,
		ReserveAmount:    stats.ReserveAmount,
		ShopServiceFee:   stats.ShopServiceFee,
		CommissionDeduct: commissionDeduction,
		Profit:           stats.RealtimeProfit,
	}
}

func qianchuanHierarchyNodeHasFinancialData(node QianchuanPlanHierarchyNode) bool {
	return node.TotalOrderCount != 0 || node.ValidOrderCount != 0 || node.ProductQuantity != 0 ||
		math.Abs(node.PayAmount) >= 0.005 || math.Abs(node.CostAmount) >= 0.005 ||
		math.Abs(node.StatCost) >= 0.005 || math.Abs(node.Profit) >= 0.005
}

func qianchuanResidualTalentNode(snapshot *DouyinProfitSnapshot, assigned []QianchuanPlanHierarchyNode, cost qianchuanTalentCostStats) QianchuanPlanHierarchyNode {
	node := QianchuanPlanHierarchyNode{
		NodeKey:         "talent:" + qianchuanUnassignedTalentKey,
		NodeType:        QianchuanHierarchyLevelTalent,
		Name:            "未归属达人 / 口径差额",
		SecondaryName:   "未匹配订单与未归属消耗",
		TalentKey:       qianchuanUnassignedTalentKey,
		Matched:         false,
		HasChildren:     cost.AdvertiserCount > 0,
		AdvertiserCount: cost.AdvertiserCount,
		AccountCount:    cost.AccountCount,
		StoreCount:      cost.StoreCount,
		PlanCount:       cost.PlanCount,
	}
	for _, item := range assigned {
		node.TotalOrderCount -= item.TotalOrderCount
		node.ProductQuantity -= item.ProductQuantity
		node.PayAmount -= item.PayAmount
		node.StatCost -= item.StatCost
		node.NormalCost -= item.NormalCost
		node.GapCost -= item.GapCost
		node.Commission -= item.Commission
		node.ValidOrderCount -= item.ValidOrderCount
		node.CostAmount -= item.CostAmount
		node.RefundCost -= item.RefundCost
		node.ReserveAmount -= item.ReserveAmount
		node.ShopServiceFee -= item.ShopServiceFee
		node.CommissionDeduct -= item.CommissionDeduct
		node.Profit -= item.Profit
	}
	node.TotalOrderCount += snapshot.TotalOrderCount
	node.ProductQuantity += float64(snapshot.ProductQuantity)
	node.PayAmount += snapshot.SalesAmount
	node.StatCost += snapshot.AdCostAmount
	node.Commission += snapshot.Commission
	node.ValidOrderCount += float64(snapshot.ValidOrderCount)
	node.CostAmount += snapshot.CostAmount
	node.RefundCost += snapshot.RefundCostAmount
	node.ReserveAmount += snapshot.ReserveAmount
	node.ShopServiceFee += snapshot.ShopServiceFee
	node.CommissionDeduct += snapshot.CommissionDeduction
	node.Profit += snapshot.Profit
	node.NormalCost = 0
	node.GapCost = 0
	node.UnassignedCost = node.StatCost

	node.ProductQuantity = roundQianchuanHierarchy(node.ProductQuantity)
	node.PayAmount = roundQianchuanHierarchy(node.PayAmount)
	node.StatCost = roundQianchuanHierarchy(node.StatCost)
	node.NormalCost = roundQianchuanHierarchy(node.NormalCost)
	node.GapCost = roundQianchuanHierarchy(node.GapCost)
	node.UnassignedCost = roundQianchuanHierarchy(node.UnassignedCost)
	node.Commission = roundQianchuanHierarchy(node.Commission)
	node.ValidOrderCount = roundQianchuanHierarchy(node.ValidOrderCount)
	node.CostAmount = roundQianchuanHierarchy(node.CostAmount)
	node.RefundCost = roundQianchuanHierarchy(node.RefundCost)
	node.ReserveAmount = roundQianchuanHierarchy(node.ReserveAmount)
	node.ShopServiceFee = roundQianchuanHierarchy(node.ShopServiceFee)
	node.CommissionDeduct = roundQianchuanHierarchy(node.CommissionDeduct)
	node.Profit = roundQianchuanHierarchy(node.Profit)
	return node
}

func (s *QianchuanPlanHierarchyService) queryTalents(ctx context.Context, req QianchuanPlanHierarchyRequest, _ string) ([]QianchuanPlanHierarchyNode, error) {
	costStats, err := loadQianchuanTalentCostStats(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	traders, err := g.DB().Model("trader").Ctx(ctx).
		Fields("id, name").
		Where("status", 1).
		Order("id ASC").
		All()
	if err != nil {
		return nil, err
	}

	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	financeService := NewFinanceCommissionService()
	list := make([]QianchuanPlanHierarchyNode, 0, len(traders)+1)
	for _, trader := range traders {
		traderID := trader["id"].Int64()
		name := strings.TrimSpace(trader["name"].String())
		if traderID <= 0 || name == "" {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(name), keyword) {
			continue
		}
		stats, statsErr := financeService.GetStats(ctx, &model.FinanceStatsReq{
			NodeId:    model.EncodeTraderID(traderID),
			NodeType:  "trader",
			TimeType:  "custom",
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
			Platform:  "douyin",
		})
		if statsErr != nil {
			return nil, fmt.Errorf("查询达人 %s 千川盈亏失败: %w", name, statsErr)
		}
		node := qianchuanTalentNodeFromFinance(traderID, name, stats.Summary, costStats[traderID])
		if qianchuanHierarchyNodeHasFinancialData(node) {
			list = append(list, node)
		}
	}

	if keyword == "" {
		snapshot, snapshotErr := GetDouyinProfitSnapshot(ctx, req.StartDate, req.EndDate)
		if snapshotErr != nil {
			return nil, snapshotErr
		}
		residual := qianchuanResidualTalentNode(snapshot, list, costStats[0])
		if qianchuanHierarchyNodeHasFinancialData(residual) {
			list = append(list, residual)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].StatCost == list[j].StatCost {
			return list[i].Name < list[j].Name
		}
		return list[i].StatCost > list[j].StatCost
	})
	return list, nil
}

type qianchuanTalentNodeAccumulator struct {
	node        QianchuanPlanHierarchyNode
	advertisers map[int64]struct{}
	accounts    map[int64]struct{}
	stores      map[int64]struct{}
}

func groupQianchuanTalentNodes(advertisers []QianchuanPlanHierarchyNode, owners map[int64]qianchuanTalentOwner) []QianchuanPlanHierarchyNode {
	groups := make(map[string]*qianchuanTalentNodeAccumulator)
	for _, item := range advertisers {
		owner := qianchuanTalentOwnerForAdvertiser(owners, item.AdvertiserID)
		group := groups[owner.Key]
		if group == nil {
			group = &qianchuanTalentNodeAccumulator{
				node: QianchuanPlanHierarchyNode{
					NodeKey:   "talent:" + owner.Key,
					NodeType:  QianchuanHierarchyLevelTalent,
					Name:      owner.Name,
					TalentKey: owner.Key,
					Matched:   owner.Matched,
				},
				advertisers: make(map[int64]struct{}),
				accounts:    make(map[int64]struct{}),
				stores:      make(map[int64]struct{}),
			}
			groups[owner.Key] = group
		}
		group.advertisers[item.AdvertiserID] = struct{}{}
		if item.AccountID > 0 {
			group.accounts[item.AccountID] = struct{}{}
		}
		if item.StoreID > 0 {
			group.stores[item.StoreID] = struct{}{}
		}
		group.node.PlanCount += item.PlanCount
		group.node.ProductCount += item.ProductCount
		group.node.ProductQuantity += item.ProductQuantity
		group.node.TotalOrderCount += item.TotalOrderCount
		group.node.PayAmount += item.PayAmount
		group.node.StatCost += item.StatCost
		group.node.NormalCost += item.NormalCost
		group.node.GapCost += item.GapCost
		group.node.UnassignedCost += item.UnassignedCost
		group.node.Commission += item.Commission
		group.node.ValidOrderCount += item.ValidOrderCount
		group.node.CostAmount += item.CostAmount
		group.node.RefundCost += item.RefundCost
		group.node.ReserveAmount += item.ReserveAmount
		group.node.ShopServiceFee += item.ShopServiceFee
		group.node.CommissionDeduct += item.CommissionDeduct
		group.node.Profit += item.Profit
		group.node.PayOrderCount += item.PayOrderCount
		group.node.CreateOrderCount += item.CreateOrderCount
		group.node.ShowCnt += item.ShowCnt
		group.node.ClickCnt += item.ClickCnt
		group.node.ConvertCnt += item.ConvertCnt
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(groups))
	for _, group := range groups {
		group.node.AdvertiserCount = int64(len(group.advertisers))
		group.node.AccountCount = int64(len(group.accounts))
		group.node.StoreCount = int64(len(group.stores))
		group.node.HasChildren = group.node.AdvertiserCount > 0
		group.node.SecondaryName = fmt.Sprintf("%d 个千川账户 · %d 个计划", group.node.AdvertiserCount, group.node.PlanCount)
		group.node.ProductQuantity = roundQianchuanHierarchy(group.node.ProductQuantity)
		group.node.PayAmount = roundQianchuanHierarchy(group.node.PayAmount)
		group.node.StatCost = roundQianchuanHierarchy(group.node.StatCost)
		group.node.NormalCost = roundQianchuanHierarchy(group.node.NormalCost)
		group.node.GapCost = roundQianchuanHierarchy(group.node.GapCost)
		group.node.UnassignedCost = roundQianchuanHierarchy(group.node.UnassignedCost)
		group.node.Commission = roundQianchuanHierarchy(group.node.Commission)
		group.node.ValidOrderCount = roundQianchuanHierarchy(group.node.ValidOrderCount)
		group.node.CostAmount = roundQianchuanHierarchy(group.node.CostAmount)
		group.node.RefundCost = roundQianchuanHierarchy(group.node.RefundCost)
		group.node.ReserveAmount = roundQianchuanHierarchy(group.node.ReserveAmount)
		group.node.ShopServiceFee = roundQianchuanHierarchy(group.node.ShopServiceFee)
		group.node.CommissionDeduct = roundQianchuanHierarchy(group.node.CommissionDeduct)
		group.node.Profit = roundQianchuanHierarchy(group.node.Profit)
		group.node.PayOrderCount = roundQianchuanHierarchy(group.node.PayOrderCount)
		group.node.CreateOrderCount = roundQianchuanHierarchy(group.node.CreateOrderCount)
		group.node.Ctr = ratioQianchuanHierarchy(float64(group.node.ClickCnt), float64(group.node.ShowCnt))
		group.node.ConvertCost = ratioQianchuanHierarchy(group.node.StatCost, float64(group.node.ConvertCnt))
		list = append(list, group.node)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].StatCost == list[j].StatCost {
			return list[i].Name < list[j].Name
		}
		return list[i].StatCost > list[j].StatCost
	})
	return list
}

type qianchuanAdvertiserTalentCost struct {
	ResolvedTraderID int64
	AdvertiserID     int64
	AccountID        int64
	NormalCost       float64
	GapCost          float64
	UnassignedCost   float64
}

func (c qianchuanAdvertiserTalentCost) TotalCost() float64 {
	return c.NormalCost + c.GapCost + c.UnassignedCost
}

func loadQianchuanAdvertiserTalentCosts(ctx context.Context, startDate, endDate string) (map[int64][]qianchuanAdvertiserTalentCost, error) {
	rows, err := g.DB().GetAll(ctx, attributedQianchuanCostCTE+`
		SELECT
			resolved_trader_id,
			advertiser_id,
			MAX(CASE WHEN account_id > 0 THEN account_id ELSE 0 END) AS account_id,
			COALESCE(SUM(CASE WHEN cost_type = 'normal' THEN stat_cost ELSE 0 END), 0) AS normal_cost,
			COALESCE(SUM(CASE WHEN cost_type = 'gap' THEN stat_cost ELSE 0 END), 0) AS gap_cost,
			COALESCE(SUM(CASE WHEN cost_type = 'unassigned' THEN stat_cost ELSE 0 END), 0) AS unassigned_cost
		FROM attributed_qianchuan_cost
		GROUP BY resolved_trader_id, advertiser_id`,
		qianchuanAttributionDateArgs(startDate, endDate)...)
	if err != nil {
		return nil, err
	}
	result := make(map[int64][]qianchuanAdvertiserTalentCost)
	for _, row := range rows {
		item := qianchuanAdvertiserTalentCost{
			ResolvedTraderID: row["resolved_trader_id"].Int64(),
			AdvertiserID:     row["advertiser_id"].Int64(),
			AccountID:        row["account_id"].Int64(),
			NormalCost:       row["normal_cost"].Float64(),
			GapCost:          row["gap_cost"].Float64(),
			UnassignedCost:   row["unassigned_cost"].Float64(),
		}
		result[item.AdvertiserID] = append(result[item.AdvertiserID], item)
	}
	return result, nil
}

func qianchuanTalentIDFromKey(key string) (int64, bool) {
	key = strings.TrimSpace(key)
	if key == qianchuanUnassignedTalentKey {
		return 0, true
	}
	if !strings.HasPrefix(key, "trader:") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(key, "trader:"), 10, 64)
	return id, err == nil && id > 0
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
	} else if !hasAwemeScope && req.TalentKey == "" && !req.AllStores {
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

	var advertiserTalentCosts map[int64][]qianchuanAdvertiserTalentCost
	targetTraderID := int64(0)
	if req.TalentKey != "" {
		var validTalentKey bool
		targetTraderID, validTalentKey = qianchuanTalentIDFromKey(req.TalentKey)
		if !validTalentKey {
			return []QianchuanPlanHierarchyNode{}, nil
		}
		advertiserTalentCosts, err = loadQianchuanAdvertiserTalentCosts(ctx, req.StartDate, req.EndDate)
		if err != nil {
			return nil, err
		}
	}
	list := make([]QianchuanPlanHierarchyNode, 0, len(rows))
	for _, row := range rows {
		advertiserID := row["advertiser_id"].Int64()
		talentCost := qianchuanAdvertiserTalentCost{}
		allNormalCost := 0.0
		if req.TalentKey != "" {
			found := false
			for _, item := range advertiserTalentCosts[advertiserID] {
				allNormalCost += item.NormalCost
				if item.ResolvedTraderID == targetTraderID {
					talentCost = item
					found = true
				}
			}
			if !found || talentCost.TotalCost() < 0.005 {
				continue
			}
		}
		name := row["advertiser_name"].String()
		if name == "" {
			name = fmt.Sprintf("千川账户 %d", advertiserID)
		}
		statCost := row["stat_cost"].Float64()
		commission := row["commission"].Float64()
		payAmount := row["pay_amount"].Float64()
		normalCost := statCost
		gapCost := 0.0
		unassignedCost := 0.0
		allocationRate := 1.0
		if req.TalentKey != "" {
			if allNormalCost > 0 {
				allocationRate = talentCost.NormalCost / allNormalCost
			} else {
				allocationRate = 0
			}
			statCost = talentCost.TotalCost()
			normalCost = talentCost.NormalCost
			gapCost = talentCost.GapCost
			unassignedCost = talentCost.UnassignedCost
			commission *= allocationRate
			payAmount *= allocationRate
		}
		showCnt := row["show_cnt"].Int64()
		clickCnt := row["click_cnt"].Int64()
		convertCnt := row["convert_cnt"].Int64()
		nodeKey := fmt.Sprintf("advertiser:%d", advertiserID)
		if req.TalentKey != "" {
			nodeKey += ":" + req.TalentKey
			showCnt = int64(math.Round(float64(showCnt) * allocationRate))
			clickCnt = int64(math.Round(float64(clickCnt) * allocationRate))
			convertCnt = int64(math.Round(float64(convertCnt) * allocationRate))
		}
		if req.AwemeID > 0 {
			nodeKey += fmt.Sprintf(":aweme:%d", req.AwemeID)
		} else if req.AwemeShowID != "" {
			nodeKey += ":aweme-show:" + req.AwemeShowID
		}
		list = append(list, QianchuanPlanHierarchyNode{
			NodeKey:       nodeKey,
			NodeType:      QianchuanHierarchyLevelAdvertiser,
			Name:          name,
			SecondaryName: fmt.Sprintf("ID %d · %d 个计划", advertiserID, row["plan_count"].Int64()),
			StoreID:       row["store_id"].Int64(),
			AccountID: func() int64 {
				if req.TalentKey != "" && talentCost.AccountID > 0 {
					return talentCost.AccountID
				}
				return row["account_id"].Int64()
			}(),
			AdvertiserID:     advertiserID,
			AwemeID:          row["aweme_id"].Int64(),
			AwemeName:        row["aweme_name"].String(),
			AwemeShowID:      row["aweme_show_id"].String(),
			TalentKey:        req.TalentKey,
			HasChildren:      row["plan_count"].Int64() > 0,
			Matched:          row["store_id"].Int64() > 0,
			PlanCount:        row["plan_count"].Int64(),
			ProductCount:     row["product_count"].Int64(),
			ProductQuantity:  roundQianchuanHierarchy(row["product_quantity"].Float64() * allocationRate),
			PayAmount:        roundQianchuanHierarchy(payAmount),
			StatCost:         roundQianchuanHierarchy(statCost),
			NormalCost:       roundQianchuanHierarchy(normalCost),
			GapCost:          roundQianchuanHierarchy(gapCost),
			UnassignedCost:   roundQianchuanHierarchy(unassignedCost),
			Commission:       roundQianchuanHierarchy(commission),
			ValidOrderCount:  roundQianchuanHierarchy(row["valid_order_count"].Float64() * allocationRate),
			CostAmount:       roundQianchuanHierarchy(row["cost_amount"].Float64() * allocationRate),
			Profit:           roundQianchuanHierarchy(commission - statCost - payAmount*0.005),
			PayOrderCount:    roundQianchuanHierarchy(row["pay_order_count"].Float64() * allocationRate),
			CreateOrderCount: roundQianchuanHierarchy(row["create_order_count"].Float64() * allocationRate),
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
	awemeKeywordSQL, awemeKeywordArgs := qianchuanAwemeKeyword("daily", req.AwemeKeyword)
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
	if req.TalentKey != "" {
		traderID, validTalentKey := qianchuanTalentIDFromKey(req.TalentKey)
		if !validTalentKey {
			return []QianchuanPlanHierarchyNode{}, nil
		}
		filterSQL += " AND daily.resolved_trader_id = ?"
		filterArgs = append(filterArgs, traderID)
	}
	if req.ProductID != "" {
		filterSQL += " AND daily.product_id = ?"
		filterArgs = append(filterArgs, req.ProductID)
	}
	sql := fmt.Sprintf(`SELECT daily.*
	FROM (
		SELECT
			COALESCE(sqm.store_id, 0) AS store_id,
			COALESCE(NULLIF(st.store_name, ''), '未匹配店铺') AS store_name,
			qar.account_id, qar.advertiser_id, qar.ad_id, qar.ad_name,
			COALESCE(NULLIF(qar.advertiser_name, ''), NULLIF(sqm.qianchuan_advertiser_name, ''), '') AS advertiser_name,
			CAST(qar.product_id AS CHAR) AS product_id, qar.product_name,
			qar.aweme_id, qar.aweme_name, qar.aweme_show_id,
			COALESCE(
				(
					SELECT rule.target_trader_id
					FROM qianchuan_advertiser_attribution_rule rule
					WHERE rule.advertiser_id = qar.advertiser_id
					  AND rule.status = 1
					  AND LEFT(qar.stat_datetime, 10) >= rule.effective_start_date
					  AND (rule.effective_end_date IS NULL OR LEFT(qar.stat_datetime, 10) <= rule.effective_end_date)
					  AND FIND_IN_SET(
						LOWER(TRIM(COALESCE(NULLIF(matched_account.account_name, ''), NULLIF(qar.aweme_name, ''), ''))) COLLATE utf8mb4_unicode_ci,
						LOWER(REPLACE(rule.source_account_names, '，', ',')) COLLATE utf8mb4_unicode_ci
					  ) > 0
					ORDER BY rule.effective_start_date DESC, rule.id DESC
					LIMIT 1
				),
				(
					SELECT MIN(account_owner.trader_id)
					FROM trader_account_relation account_owner
					INNER JOIN trader account_trader
						ON account_trader.id = account_owner.trader_id
						AND account_trader.status = 1
					WHERE account_owner.account_id = qar.account_id
				),
				0
			) AS resolved_trader_id,
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
		LEFT JOIN store st
			ON st.id = COALESCE(sqm.store_id, 0)
		LEFT JOIN account matched_account
			ON matched_account.id = qar.account_id
			AND matched_account.platform = 'douyin'
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?
	) daily
	WHERE 1 = 1%s%s%s
	ORDER BY daily.stat_date DESC, daily.ad_id ASC`, filterSQL, keywordSQL, awemeKeywordSQL)

	args := []interface{}{req.StartDate, endExclusive}
	args = append(args, filterArgs...)
	args = append(args, keywordArgs...)
	args = append(args, awemeKeywordArgs...)
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
				StoreName:      row["store_name"].String(),
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

	var auditedOrderStats map[int64]qianchuanPlanOrderStat
	if req.TalentKey != "" {
		traderID, validTalentKey := qianchuanTalentIDFromKey(req.TalentKey)
		if validTalentKey && traderID > 0 {
			auditedOrderStats, err = loadQianchuanAuditedPlanOrderStats(ctx, traderID, req.StartDate, req.EndDate)
			if err != nil {
				return nil, err
			}
		}
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(nodeMap))
	for mapKey, node := range nodeMap {
		if auditedOrderStats != nil {
			audited := auditedOrderStats[node.AdID]
			node.ValidOrderCount = audited.ValidOrderCount
			node.ProductQuantity = audited.ProductQuantity
			node.PayAmount = audited.PayAmount
			node.CostAmount = audited.CostAmount
			node.Commission = audited.Commission
			payAmountMap[mapKey] = audited.PayAmount
		}
		adID := node.AdID
		node.SecondaryName = planHierarchySecondaryNameWithStore(node.ProductName, node.ProductID, node.StoreName, adID)
		node.StatCost = roundQianchuanHierarchy(node.StatCost)
		node.NormalCost = node.StatCost
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
	if req.TalentKey != "" && req.AdvertiserID > 0 {
		traderID, validTalentKey := qianchuanTalentIDFromKey(req.TalentKey)
		if validTalentKey {
			costsByAdvertiser, costErr := loadQianchuanAdvertiserTalentCosts(ctx, req.StartDate, req.EndDate)
			if costErr != nil {
				return nil, costErr
			}
			targetCost := qianchuanAdvertiserTalentCost{}
			for _, item := range costsByAdvertiser[req.AdvertiserID] {
				if item.ResolvedTraderID == traderID {
					targetCost = item
					break
				}
			}
			rawNormalCost := 0.0
			for _, item := range list {
				rawNormalCost += item.StatCost
			}
			if rawNormalCost > 0 && targetCost.NormalCost >= 0 {
				for i := range list {
					list[i].StatCost = roundQianchuanHierarchy(list[i].StatCost * targetCost.NormalCost / rawNormalCost)
					list[i].NormalCost = list[i].StatCost
					list[i].Profit = roundQianchuanHierarchy(list[i].Commission - list[i].StatCost - list[i].PayAmount*0.005)
					list[i].ConvertCost = ratioQianchuanHierarchy(list[i].StatCost, float64(list[i].ConvertCnt))
				}
			}
			if targetCost.GapCost >= 0.005 {
				list = append(list, QianchuanPlanHierarchyNode{
					NodeKey:       fmt.Sprintf("plan:gap:%d:%d", req.AdvertiserID, traderID),
					NodeType:      QianchuanHierarchyLevelPlan,
					Name:          "千川户差额消耗",
					SecondaryName: "账户日实扣减已归属计划消耗",
					AdvertiserID:  req.AdvertiserID,
					StatCost:      roundQianchuanHierarchy(targetCost.GapCost),
					GapCost:       roundQianchuanHierarchy(targetCost.GapCost),
					Profit:        roundQianchuanHierarchy(-targetCost.GapCost),
					Matched:       true,
				})
			}
			if targetCost.UnassignedCost >= 0.005 {
				list = append(list, QianchuanPlanHierarchyNode{
					NodeKey:        fmt.Sprintf("plan:unassigned:%d:%d", req.AdvertiserID, traderID),
					NodeType:       QianchuanHierarchyLevelPlan,
					Name:           "未归属消耗",
					SecondaryName:  "计划账号或千川户负责人未识别",
					AdvertiserID:   req.AdvertiserID,
					StatCost:       roundQianchuanHierarchy(targetCost.UnassignedCost),
					UnassignedCost: roundQianchuanHierarchy(targetCost.UnassignedCost),
					Profit:         roundQianchuanHierarchy(-targetCost.UnassignedCost),
					Matched:        false,
				})
			}
		}
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

func loadQianchuanAuditedPlanOrderStats(ctx context.Context, traderID int64, startDate, endDate string) (map[int64]qianchuanPlanOrderStat, error) {
	rows, _, _, err := NewFinanceCommissionService().financeValidOrderAuditRows(ctx, &model.FinanceValidOrderAttributionReq{
		NodeId:    model.EncodeTraderID(traderID),
		NodeType:  "trader",
		TimeType:  "custom",
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		return nil, fmt.Errorf("查询达人有效订单计划归属失败: %w", err)
	}
	result := make(map[int64]qianchuanPlanOrderStat)
	for _, row := range rows {
		if row.QianchuanPlanId <= 0 {
			continue
		}
		item := result[row.QianchuanPlanId]
		item.ValidOrderCount++
		item.ProductQuantity += float64(row.ProductNum)
		item.PayAmount += row.OrderPayAmount
		item.CostAmount += row.ProductCost
		item.Commission += row.Commission
		result[row.QianchuanPlanId] = item
	}
	return result, nil
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
		if level == QianchuanHierarchyLevelTalent {
			if item.Matched {
				summary.TalentCount++
			}
			summary.StoreCount += item.StoreCount
			summary.AccountCount += item.AccountCount
		}
		summary.AdvertiserCount += item.AdvertiserCount
		summary.PlanCount += item.PlanCount
		summary.ProductCount += item.ProductCount
		summary.ProductQuantity += int64(math.Round(item.ProductQuantity))
		summary.TotalOrderCount += item.TotalOrderCount
		summary.TotalPayAmount += item.PayAmount
		summary.TotalCost += item.StatCost
		summary.NormalCost += item.NormalCost
		summary.GapCost += item.GapCost
		summary.UnassignedCost += item.UnassignedCost
		summary.TotalCommission += item.Commission
		summary.TotalValidOrderCount += item.ValidOrderCount
		summary.TotalOrderCost += item.CostAmount
		summary.TotalRefundCost += item.RefundCost
		summary.ReserveAmount += item.ReserveAmount
		summary.ShopServiceFee += item.ShopServiceFee
		summary.CommissionDeduction += item.CommissionDeduct
		summary.TotalProfit += item.Profit
	}
	summary.TotalCost = roundQianchuanHierarchy(summary.TotalCost)
	summary.NormalCost = roundQianchuanHierarchy(summary.NormalCost)
	summary.GapCost = roundQianchuanHierarchy(summary.GapCost)
	summary.UnassignedCost = roundQianchuanHierarchy(summary.UnassignedCost)
	summary.TotalPayAmount = roundQianchuanHierarchy(summary.TotalPayAmount)
	summary.TotalCommission = roundQianchuanHierarchy(summary.TotalCommission)
	summary.TotalValidOrderCount = roundQianchuanHierarchy(summary.TotalValidOrderCount)
	summary.TotalOrderCost = roundQianchuanHierarchy(summary.TotalOrderCost)
	summary.TotalRefundCost = roundQianchuanHierarchy(summary.TotalRefundCost)
	if math.Abs(summary.ReserveAmount) < 0.005 {
		summary.ReserveAmount = CalculateReserveAmount(summary.TotalPayAmount)
	} else {
		summary.ReserveAmount = roundQianchuanHierarchy(summary.ReserveAmount)
	}
	if math.Abs(summary.ShopServiceFee) < 0.005 {
		summary.ShopServiceFee = CalculateDouyinShopServiceFee(summary.TotalPayAmount)
	} else {
		summary.ShopServiceFee = roundQianchuanHierarchy(summary.ShopServiceFee)
	}
	if math.Abs(summary.CommissionDeduction) < 0.005 {
		summary.CommissionDeduction = roundQianchuanHierarchy(summary.TotalPayAmount * 0.005)
	} else {
		summary.CommissionDeduction = roundQianchuanHierarchy(summary.CommissionDeduction)
	}
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

func planHierarchySecondaryNameWithStore(productName, productID, storeName string, adID int64) string {
	planName := planHierarchySecondaryName(productName, productID, adID)
	storeName = strings.TrimSpace(storeName)
	if storeName == "" {
		return planName
	}
	return fmt.Sprintf("%s · %s", storeName, planName)
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
