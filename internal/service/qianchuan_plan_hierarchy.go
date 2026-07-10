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
	QianchuanHierarchyLevelAdvertiser = "advertiser"
	QianchuanHierarchyLevelPlan       = "plan"
)

type QianchuanPlanHierarchyRequest struct {
	Level        string
	StartDate    string
	EndDate      string
	Keyword      string
	AccountID    int64
	AdvertiserID int64
}

type QianchuanPlanHierarchyNode struct {
	NodeKey          string  `json:"nodeKey"`
	NodeType         string  `json:"nodeType"`
	Name             string  `json:"name"`
	SecondaryName    string  `json:"secondaryName"`
	AccountID        int64   `json:"accountId"`
	AdvertiserID     int64   `json:"advertiserId"`
	AdID             int64   `json:"adId"`
	ProductID        string  `json:"productId"`
	ProductName      string  `json:"productName"`
	AwemeName        string  `json:"awemeName"`
	AwemeShowID      string  `json:"awemeShowId"`
	HasChildren      bool    `json:"hasChildren"`
	Matched          bool    `json:"matched"`
	AdvertiserCount  int64   `json:"advertiserCount"`
	PlanCount        int64   `json:"planCount"`
	ProductCount     int64   `json:"productCount"`
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
	AdvertiserCount      int64   `json:"advertiserCount"`
	PlanCount            int64   `json:"planCount"`
	ProductCount         int64   `json:"productCount"`
	TotalCost            float64 `json:"totalCost"`
	TotalCommission      float64 `json:"totalCommission"`
	TotalValidOrderCount float64 `json:"totalValidOrderCount"`
	TotalOrderCost       float64 `json:"totalOrderCost"`
	TotalProfit          float64 `json:"totalProfit"`
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

func (s *QianchuanPlanHierarchyService) Query(ctx context.Context, req QianchuanPlanHierarchyRequest) (*QianchuanPlanHierarchyResult, error) {
	startDate, endDate, endExclusive, err := normalizeQianchuanHierarchyDates(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	req.StartDate = startDate
	req.EndDate = endDate
	req.Keyword = strings.TrimSpace(req.Keyword)

	level := req.Level
	if level == "" {
		level = QianchuanHierarchyLevelStore
	}

	var list []QianchuanPlanHierarchyNode
	switch level {
	case QianchuanHierarchyLevelStore:
		list, err = s.queryStores(ctx, req, endExclusive)
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
	if level == QianchuanHierarchyLevelStore {
		result.Summary = summarizeQianchuanStoreNodes(list)
	}
	return result, nil
}

func normalizeQianchuanHierarchyDates(startDate, endDate string) (string, string, string, error) {
	if startDate == "" || endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate, endDate = today, today
	}
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
		%s.aweme_show_id LIKE ? OR CAST(%s.ad_id AS CHAR) LIKE ? OR
		CAST(%s.advertiser_id AS CHAR) LIKE ? OR EXISTS (
			SELECT 1 FROM account keyword_account
			WHERE keyword_account.id = %s.account_id AND keyword_account.account_name LIKE ?
		)
	)`, alias, alias, alias, alias, alias, alias, alias, alias)
	return clause, []interface{}{like, like, like, like, like, like, like, like}
}

func qianchuanOrderStatsCTE(validCondition string) string {
	return fmt.Sprintf(`
	order_stats AS (
		SELECT
			a.id AS account_id,
			CAST(o.product_id AS CHAR) AS product_id,
			LEFT(o.pay_finish_time, 10) AS stat_date,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS valid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(gd.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN
				(o.product_price * o.product_num)
				- (COALESCE(gd.base_price, 0) * o.product_num)
				- (o.product_price * o.product_num * 0.03)
				- (o.product_price * o.product_num * 0.02)
			ELSE 0 END), 0) AS commission
		FROM order_info o
		INNER JOIN account a
			ON a.platform = 'douyin'
			AND a.status = 1
			AND a.account_name COLLATE utf8mb4_unicode_ci = o.shop_name COLLATE utf8mb4_unicode_ci
		INNER JOIN q_keys qk
			ON qk.account_id = a.id
			AND qk.product_id = CAST(o.product_id AS CHAR)
			AND qk.stat_date = LEFT(o.pay_finish_time, 10)
		LEFT JOIN goods gd
			ON o.seller_code COLLATE utf8mb4_unicode_ci = gd.goods_code COLLATE utf8mb4_unicode_ci
		WHERE o.pay_finish_time >= ? AND o.pay_finish_time < ?
		GROUP BY a.id, CAST(o.product_id AS CHAR), LEFT(o.pay_finish_time, 10)
	)`, validCondition, validCondition, validCondition, validCondition)
}

func (s *QianchuanPlanHierarchyService) queryStores(ctx context.Context, req QianchuanPlanHierarchyRequest, endExclusive string) ([]QianchuanPlanHierarchyNode, error) {
	keywordSQL, keywordArgs := qianchuanHierarchyKeyword("qar", req.Keyword)
	validCondition := FinanceValidOrderCondition("o")
	sql := fmt.Sprintf(`WITH
	q AS (
		SELECT
			qar.account_id, qar.advertiser_id, qar.ad_id,
			CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost, qar.pay_order_count, qar.create_order_count,
			qar.show_cnt, qar.click_cnt, qar.convert_cnt
		FROM qianchuan_ad_report qar
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?%s
	),
	q_agg AS (
		SELECT
			account_id,
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
		GROUP BY account_id
	),
	q_keys AS (
		SELECT DISTINCT account_id, product_id, stat_date
		FROM q
		WHERE account_id > 0 AND product_id != '' AND product_id != '0'
	),%s,
	order_by_account AS (
		SELECT
			account_id,
			COALESCE(SUM(valid_order_count), 0) AS valid_order_count,
			COALESCE(SUM(pay_amount), 0) AS pay_amount,
			COALESCE(SUM(cost_amount), 0) AS cost_amount,
			COALESCE(SUM(commission), 0) AS commission
		FROM order_stats
		GROUP BY account_id
	)
	SELECT
		q.account_id,
		COALESCE(NULLIF(a.account_name, ''), '未匹配店铺') AS account_name,
		q.advertiser_count, q.plan_count, q.product_count,
		q.stat_cost, q.pay_order_count, q.create_order_count,
		q.show_cnt, q.click_cnt, q.convert_cnt,
		COALESCE(o.valid_order_count, 0) AS valid_order_count,
		COALESCE(o.pay_amount, 0) AS pay_amount,
		COALESCE(o.cost_amount, 0) AS cost_amount,
		COALESCE(o.commission, 0) AS commission
	FROM q_agg q
	LEFT JOIN account a ON a.id = q.account_id
	LEFT JOIN order_by_account o ON o.account_id = q.account_id
	ORDER BY (q.account_id = 0) ASC, q.stat_cost DESC, q.account_id ASC`, keywordSQL, qianchuanOrderStatsCTE(validCondition))

	args := []interface{}{req.StartDate, endExclusive}
	args = append(args, keywordArgs...)
	args = append(args, req.StartDate, endExclusive)
	rows, err := g.DB().GetAll(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(rows))
	for _, row := range rows {
		accountID := row["account_id"].Int64()
		statCost := row["stat_cost"].Float64()
		commission := row["commission"].Float64()
		payAmount := row["pay_amount"].Float64()
		showCnt := row["show_cnt"].Int64()
		clickCnt := row["click_cnt"].Int64()
		convertCnt := row["convert_cnt"].Int64()
		list = append(list, QianchuanPlanHierarchyNode{
			NodeKey:          fmt.Sprintf("store:%d", accountID),
			NodeType:         QianchuanHierarchyLevelStore,
			Name:             row["account_name"].String(),
			SecondaryName:    storeHierarchySecondaryName(accountID, row["advertiser_count"].Int64(), row["plan_count"].Int64()),
			AccountID:        accountID,
			HasChildren:      row["advertiser_count"].Int64() > 0,
			Matched:          accountID > 0,
			AdvertiserCount:  row["advertiser_count"].Int64(),
			PlanCount:        row["plan_count"].Int64(),
			ProductCount:     row["product_count"].Int64(),
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
	validCondition := FinanceValidOrderCondition("o")
	sql := fmt.Sprintf(`WITH
	all_q AS (
		SELECT
			qar.account_id, qar.advertiser_id, qar.ad_id,
			CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost
		FROM qianchuan_ad_report qar
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ? AND qar.account_id = ?
	),
	q AS (
		SELECT
			qar.account_id, qar.advertiser_id, qar.advertiser_name, qar.ad_id,
			CAST(qar.product_id AS CHAR) AS product_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost, qar.pay_order_count, qar.create_order_count,
			qar.show_cnt, qar.click_cnt, qar.convert_cnt
		FROM qianchuan_ad_report qar
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ? AND qar.account_id = ?%s
	),
	q_totals AS (
		SELECT account_id, product_id, stat_date,
			COALESCE(SUM(stat_cost), 0) AS total_cost,
			COUNT(DISTINCT ad_id) AS total_plan_count
		FROM all_q
		GROUP BY account_id, product_id, stat_date
	),
	q_advertiser_keys AS (
		SELECT
			account_id, advertiser_id, product_id, stat_date,
			COALESCE(SUM(stat_cost), 0) AS key_cost,
			COUNT(DISTINCT ad_id) AS key_plan_count
		FROM q
		GROUP BY account_id, advertiser_id, product_id, stat_date
	),
	q_keys AS (
		SELECT DISTINCT account_id, product_id, stat_date
		FROM q
		WHERE account_id > 0 AND product_id != '' AND product_id != '0'
	),%s,
	allocated AS (
		SELECT
			k.account_id, k.advertiser_id,
			CASE WHEN t.total_cost > 0 THEN k.key_cost / t.total_cost
				 WHEN t.total_plan_count > 0 THEN k.key_plan_count / t.total_plan_count ELSE 0 END AS allocation_rate,
			COALESCE(o.valid_order_count, 0) AS valid_order_count,
			COALESCE(o.pay_amount, 0) AS pay_amount,
			COALESCE(o.cost_amount, 0) AS cost_amount,
			COALESCE(o.commission, 0) AS commission
		FROM q_advertiser_keys k
		JOIN q_totals t ON t.account_id = k.account_id AND t.product_id = k.product_id AND t.stat_date = k.stat_date
		LEFT JOIN order_stats o ON o.account_id = k.account_id AND o.product_id = k.product_id AND o.stat_date = k.stat_date
	),
	allocated_by_advertiser AS (
		SELECT account_id, advertiser_id,
			SUM(valid_order_count * allocation_rate) AS valid_order_count,
			SUM(pay_amount * allocation_rate) AS pay_amount,
			SUM(cost_amount * allocation_rate) AS cost_amount,
			SUM(commission * allocation_rate) AS commission
		FROM allocated
		GROUP BY account_id, advertiser_id
	),
	q_agg AS (
		SELECT
			account_id, advertiser_id,
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
		GROUP BY account_id, advertiser_id
	)
	SELECT q.*, COALESCE(a.valid_order_count, 0) AS valid_order_count,
		COALESCE(a.pay_amount, 0) AS pay_amount,
		COALESCE(a.cost_amount, 0) AS cost_amount,
		COALESCE(a.commission, 0) AS commission
	FROM q_agg q
	LEFT JOIN allocated_by_advertiser a ON a.account_id = q.account_id AND a.advertiser_id = q.advertiser_id
	ORDER BY q.stat_cost DESC, q.advertiser_id ASC`, keywordSQL, qianchuanOrderStatsCTE(validCondition))

	args := []interface{}{req.StartDate, endExclusive, req.AccountID, req.StartDate, endExclusive, req.AccountID}
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
		list = append(list, QianchuanPlanHierarchyNode{
			NodeKey:          fmt.Sprintf("advertiser:%d", advertiserID),
			NodeType:         QianchuanHierarchyLevelAdvertiser,
			Name:             name,
			SecondaryName:    fmt.Sprintf("ID %d · %d 个计划", advertiserID, row["plan_count"].Int64()),
			AccountID:        row["account_id"].Int64(),
			AdvertiserID:     advertiserID,
			HasChildren:      row["plan_count"].Int64() > 0,
			Matched:          row["account_id"].Int64() > 0,
			PlanCount:        row["plan_count"].Int64(),
			ProductCount:     row["product_count"].Int64(),
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
	sql := fmt.Sprintf(`SELECT daily.*
	FROM (
		SELECT
			qar.account_id, qar.advertiser_id, qar.ad_id, qar.ad_name,
			qar.advertiser_name,
			CAST(qar.product_id AS CHAR) AS product_id, qar.product_name,
			qar.aweme_name, qar.aweme_show_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			qar.stat_cost, qar.pay_order_count, qar.create_order_count,
			qar.show_cnt, qar.click_cnt, qar.convert_cnt,
			SUM(qar.stat_cost) OVER (
				PARTITION BY qar.account_id, qar.product_id, LEFT(qar.stat_datetime, 10)
			) AS total_cost,
			COUNT(*) OVER (
				PARTITION BY qar.account_id, qar.product_id, LEFT(qar.stat_datetime, 10)
			) AS total_plan_count
		FROM qianchuan_ad_report qar
		WHERE qar.stat_datetime >= ? AND qar.stat_datetime < ?
	) daily
	WHERE daily.advertiser_id = ?%s
	ORDER BY daily.stat_date DESC, daily.ad_id ASC`, keywordSQL)

	args := []interface{}{req.StartDate, endExclusive, req.AdvertiserID}
	args = append(args, keywordArgs...)
	rows, err := g.DB().GetAll(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []QianchuanPlanHierarchyNode{}, nil
	}

	productsByAccount := make(map[int64]map[string]struct{})
	for _, row := range rows {
		accountID := row["account_id"].Int64()
		productID := row["product_id"].String()
		if accountID <= 0 || productID == "" || productID == "0" {
			continue
		}
		if productsByAccount[accountID] == nil {
			productsByAccount[accountID] = make(map[string]struct{})
		}
		productsByAccount[accountID][productID] = struct{}{}
	}
	orderStats, err := loadQianchuanPlanOrderStats(ctx, productsByAccount, req.StartDate, endExclusive)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[int64]*QianchuanPlanHierarchyNode)
	payAmountMap := make(map[int64]float64)
	for _, row := range rows {
		adID := row["ad_id"].Int64()
		node := nodeMap[adID]
		if node == nil {
			name := row["ad_name"].String()
			if name == "" {
				name = fmt.Sprintf("计划 %d", adID)
			}
			node = &QianchuanPlanHierarchyNode{
				NodeKey:      fmt.Sprintf("plan:%d", adID),
				NodeType:     QianchuanHierarchyLevelPlan,
				Name:         name,
				AccountID:    row["account_id"].Int64(),
				AdvertiserID: row["advertiser_id"].Int64(),
				AdID:         adID,
				ProductID:    row["product_id"].String(),
				ProductName:  row["product_name"].String(),
				AwemeName:    row["aweme_name"].String(),
				AwemeShowID:  row["aweme_show_id"].String(),
				Matched:      row["account_id"].Int64() > 0,
				PlanCount:    1,
			}
			nodeMap[adID] = node
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
		orderStat := orderStats[qianchuanOrderStatKey(row["account_id"].Int64(), row["product_id"].String(), row["stat_date"].String())]
		node.ValidOrderCount += orderStat.ValidOrderCount * allocationRate
		node.CostAmount += orderStat.CostAmount * allocationRate
		node.Commission += orderStat.Commission * allocationRate
		payAmountMap[adID] += orderStat.PayAmount * allocationRate
	}

	list := make([]QianchuanPlanHierarchyNode, 0, len(nodeMap))
	for adID, node := range nodeMap {
		node.SecondaryName = planHierarchySecondaryName(node.ProductName, node.ProductID, adID)
		node.StatCost = roundQianchuanHierarchy(node.StatCost)
		node.Commission = roundQianchuanHierarchy(node.Commission)
		node.ValidOrderCount = roundQianchuanHierarchy(node.ValidOrderCount)
		node.CostAmount = roundQianchuanHierarchy(node.CostAmount)
		node.Profit = roundQianchuanHierarchy(node.Commission - node.StatCost - payAmountMap[adID]*0.005)
		node.PayOrderCount = roundQianchuanHierarchy(node.PayOrderCount)
		node.CreateOrderCount = roundQianchuanHierarchy(node.CreateOrderCount)
		node.Ctr = ratioQianchuanHierarchy(float64(node.ClickCnt), float64(node.ShowCnt))
		node.ConvertCost = ratioQianchuanHierarchy(node.StatCost, float64(node.ConvertCnt))
		list = append(list, *node)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].StatCost == list[j].StatCost {
			return list[i].AdID < list[j].AdID
		}
		return list[i].StatCost > list[j].StatCost
	})
	return list, nil
}

type qianchuanPlanOrderStat struct {
	ValidOrderCount float64
	PayAmount       float64
	CostAmount      float64
	Commission      float64
}

func loadQianchuanPlanOrderStats(ctx context.Context, productsByAccount map[int64]map[string]struct{}, startDate, endExclusive string) (map[string]qianchuanPlanOrderStat, error) {
	result := make(map[string]qianchuanPlanOrderStat)
	validCondition := FinanceValidOrderCondition("o")
	for accountID, productSet := range productsByAccount {
		account, err := g.DB().Model("account").Ctx(ctx).Where("id", accountID).Fields("account_name").One()
		if err != nil {
			return nil, err
		}
		if account.IsEmpty() || len(productSet) == 0 {
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
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS valid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(gd.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN
				(o.product_price * o.product_num)
				- (COALESCE(gd.base_price, 0) * o.product_num)
				- (o.product_price * o.product_num * 0.03)
				- (o.product_price * o.product_num * 0.02)
			ELSE 0 END), 0) AS commission
		FROM order_info o
		LEFT JOIN goods gd
			ON o.seller_code COLLATE utf8mb4_unicode_ci = gd.goods_code COLLATE utf8mb4_unicode_ci
		WHERE o.shop_name = ?
			AND o.pay_finish_time >= ? AND o.pay_finish_time < ?
			AND o.product_id IN (%s)
		GROUP BY o.product_id, LEFT(o.pay_finish_time, 10)`, validCondition, validCondition, validCondition, validCondition, strings.Join(placeholders, ","))

		args := []interface{}{account["account_name"].String(), startDate, endExclusive}
		for _, productID := range products {
			args = append(args, productID)
		}
		rows, err := g.DB().GetAll(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result[qianchuanOrderStatKey(accountID, row["product_id"].String(), row["stat_date"].String())] = qianchuanPlanOrderStat{
				ValidOrderCount: row["valid_order_count"].Float64(),
				PayAmount:       row["pay_amount"].Float64(),
				CostAmount:      row["cost_amount"].Float64(),
				Commission:      row["commission"].Float64(),
			}
		}
	}
	return result, nil
}

func qianchuanOrderStatKey(accountID int64, productID, statDate string) string {
	return fmt.Sprintf("%d|%s|%s", accountID, productID, statDate)
}

func summarizeQianchuanStoreNodes(list []QianchuanPlanHierarchyNode) QianchuanPlanHierarchySummary {
	var summary QianchuanPlanHierarchySummary
	for _, item := range list {
		if item.AccountID > 0 {
			summary.StoreCount++
		}
		summary.AdvertiserCount += item.AdvertiserCount
		summary.PlanCount += item.PlanCount
		summary.ProductCount += item.ProductCount
		summary.TotalCost += item.StatCost
		summary.TotalCommission += item.Commission
		summary.TotalValidOrderCount += item.ValidOrderCount
		summary.TotalOrderCost += item.CostAmount
		summary.TotalProfit += item.Profit
	}
	summary.TotalCost = roundQianchuanHierarchy(summary.TotalCost)
	summary.TotalCommission = roundQianchuanHierarchy(summary.TotalCommission)
	summary.TotalValidOrderCount = roundQianchuanHierarchy(summary.TotalValidOrderCount)
	summary.TotalOrderCost = roundQianchuanHierarchy(summary.TotalOrderCost)
	summary.TotalProfit = roundQianchuanHierarchy(summary.TotalProfit)
	return summary
}

func storeHierarchySecondaryName(accountID, advertiserCount, planCount int64) string {
	if accountID == 0 {
		return fmt.Sprintf("%d 个千川账户待绑定 · %d 个计划", advertiserCount, planCount)
	}
	return fmt.Sprintf("%d 个千川账户 · %d 个计划", advertiserCount, planCount)
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
