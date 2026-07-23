package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// FinanceAggregateService 财务管理 - 日聚合任务服务
type FinanceAggregateService struct{}

func NewFinanceAggregateService() *FinanceAggregateService {
	return &FinanceAggregateService{}
}

func financeAggregateInferenceContext(ctx context.Context) (context.Context, bool) {
	enabled := IsQianchuanPlanBlankOrderInferenceEnabled(ctx)
	return withQianchuanPlanBlankOrderInferenceSetting(ctx, enabled), enabled
}

// DailyAggregateJob 每日聚合任务
// 建议在每天凌晨2点执行，统计前一天的数据
func (s *FinanceAggregateService) DailyAggregateJob(ctx context.Context, date string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("日期格式错误，期望 2006-01-02: %s", date)
	}
	ctx, qianchuanBlankOrderInferenceEnabled := financeAggregateInferenceContext(ctx)
	g.Log().Infof(
		ctx,
		"[FinanceAggregate] 开始执行日聚合任务, 日期: %s, 空白单反推: %t",
		date,
		qianchuanBlankOrderInferenceEnabled,
	)

	// 历史汇总必须复用实时页面的账号叶子计算器，避免出现两套订单归属、广告费和利润公式。
	shops, err := g.DB().Model("account").Ctx(ctx).
		Where("status", 1).
		Where("platform", "douyin").
		Fields("id, platform, uid, account_name, platform_nickname").
		All()
	if err != nil {
		return fmt.Errorf("查询有效账号失败: %w", err)
	}
	leafStats, err := NewFinanceCommissionService().statsAccountLeaves(ctx, shops, date, date, "")
	if err != nil {
		return fmt.Errorf("计算账号日汇总失败: %w", err)
	}

	rows := make([]g.Map, 0, len(shops))
	for _, shop := range shops {
		accountID := shop["id"].Int()
		platform := shop["platform"].String()
		if accountID <= 0 || platform == "" {
			continue
		}
		stats := leafStats[accountID]
		if stats == nil {
			stats = &model.FinanceStatsSummary{}
		}
		rows = append(rows, financeDailyLeafRow(date, accountID, platform, *stats))
	}

	// 先在事务外完成计算，事务内一次性替换；失败时页面仍可读取旧数据。
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model("finance_daily_stats").Ctx(ctx).Where("stat_date", date).Delete(); err != nil {
			return err
		}
		const batchSize = 200
		for start := 0; start < len(rows); start += batchSize {
			end := start + batchSize
			if end > len(rows) {
				end = len(rows)
			}
			if _, err := tx.Model("finance_daily_stats").Ctx(ctx).Data(rows[start:end]).Insert(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("保存账号日汇总失败: %w", err)
	}

	g.Log().Infof(ctx, "[FinanceAggregate] 日聚合任务完成, 日期: %s, 账号数: %d", date, len(rows))
	return nil
}

func financeDailyLeafRow(date string, accountID int, platform string, stats model.FinanceStatsSummary) g.Map {
	return g.Map{
		"stat_date": date, "company_id": 0, "department_id": 0, "trader_id": 0,
		"account_id": accountID, "platform": platform,
		"total_order_count": stats.TotalOrderCount, "order_count": stats.OrderCount,
		"invalid_order_count": stats.InvalidOrderCount, "product_num": stats.ProductNum,
		"pay_amount": stats.PayAmount, "cost_amount": stats.CostAmount,
		"reserve_amount": stats.ReserveAmount, "commission": stats.Commission,
		"refund_cost_amount": stats.RefundCostAmount, "refund_order_count": stats.RefundOrderCount,
		"shop_service_fee": stats.ShopServiceFee, "ad_cost_amount": stats.AdCostAmount,
		"valid_order_cost_match": stats.ValidOrderCostMatch,
		"commission_order_count": stats.CommissionOrderCount,
		"realtime_profit":        stats.RealtimeProfit,
	}
}

// OrderStatItem 订单统计项
type OrderStatItem struct {
	AccountId            int
	TalentId             string
	Platform             string
	TotalOrderCount      int
	OrderCount           int
	InvalidOrderCount    int
	ProductNum           int
	PayAmount            float64
	CostAmount           float64
	ReserveAmount        float64
	Commission           float64
	RefundCostAmount     float64
	RefundOrderCount     int
	ShopServiceFee       float64
	AdCostAmount         float64
	ValidOrderCostMatch  int
	CommissionOrderCount int
	RealtimeProfit       float64
}

// getAllValidOrders 获取某日的所有订单（含全部统计指标）
func (s *FinanceAggregateService) getAllValidOrders(ctx context.Context, date string) ([]OrderStatItem, error) {
	var allOrders []OrderStatItem

	startTime := date + " 00:00:00"
	endTime := date + " 23:59:59"

	// 有效订单条件
	validCond := financeValidOrderCondition("o")
	// 无效订单条件
	invalidCond := financeInvalidOrderCondition("o")
	// 退款成本条件：无效 且 已发货
	refundCond := financeRefundOrderCondition("o")

	// 抖音订单
	allOrders = append(allOrders, s.queryPlatformOrders(ctx, "order_info", "douyin", "main_order_no", "", validCond, invalidCond, refundCond, startTime, endTime)...)

	// 快手订单
	allOrders = append(allOrders, s.queryPlatformOrders(ctx, "kuaishou_orders", "kuaishou", "order_no", "COLLATE utf8mb4_unicode_ci", validCond, invalidCond, refundCond, startTime, endTime)...)

	// 视频号订单
	allOrders = append(allOrders, s.queryPlatformOrders(ctx, "shipinhao_orders", "shipinhao", "order_no", "COLLATE utf8mb4_unicode_ci", validCond, invalidCond, refundCond, startTime, endTime)...)

	return allOrders, nil
}

// queryPlatformOrders 查询单个平台的订单统计
func (s *FinanceAggregateService) queryPlatformOrders(ctx context.Context, tableName, platform, orderNoCol, collateClause, validCond, invalidCond, refundCond, startTime, endTime string) []OrderStatItem {
	collate := ""
	if collateClause != "" {
		collate = " " + collateClause
	}
	timeField := "pay_finish_time"
	if platform == "douyin" {
		timeField = "order_submit_time"
	}

	sql := fmt.Sprintf(`
		SELECT
			o.talent_id,
			COUNT(DISTINCT o.%s) AS total_order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS invalid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_num,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price, 0) * o.product_num ELSE 0 END), 0) AS pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, o.product_price) * o.product_num ELSE 0 END), 0) AS cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_order_count,
			COUNT(DISTINCT CASE WHEN %s AND g.base_price > 0 THEN o.%s END) AS valid_order_cost_match
		FROM %s o
		LEFT JOIN goods g ON o.seller_code%s = g.goods_code
		WHERE o.%s >= ? AND o.%s <= ?
		  AND o.%s IS NOT NULL AND o.%s != ''
		GROUP BY o.talent_id
	`, orderNoCol, validCond, orderNoCol, invalidCond, orderNoCol, validCond, validCond, validCond, refundCond, refundCond, orderNoCol, validCond, orderNoCol, tableName, collate, timeField, timeField, orderNoCol, orderNoCol)

	result, err := g.DB().GetAll(ctx, sql, startTime, endTime)
	if err != nil {
		g.Log().Warningf(ctx, "[FinanceAggregate] 查询 %s 订单失败: %v", platform, err)
		return nil
	}

	var orders []OrderStatItem
	for _, row := range result {
		talentId := row["talent_id"].String()
		if talentId == "" {
			continue
		}
		orders = append(orders, newOrderStatItem(row, talentId, platform))
	}

	return orders
}

// newOrderStatItem 从查询行构造 OrderStatItem
func newOrderStatItem(row gdb.Record, talentId, platform string) OrderStatItem {
	payAmount := row["pay_amount"].Float64()
	costAmount := row["cost_amount"].Float64()
	reserveAmount := CalculateReserveAmount(payAmount)
	orderCount := row["order_count"].Int()
	refundCostAmount := row["refund_cost_amount"].Float64()
	shopServiceFee := 0.0
	if platform == "douyin" {
		shopServiceFee = CalculateDouyinShopServiceFee(payAmount)
	}
	// 佣金 = 定价总额 - 留底 - 货款 - 小店服务费
	commission := payAmount - costAmount - reserveAmount
	commissionDeduction := payAmount * 0.005
	realtimeProfit := CalculateOrderProfit(payAmount, costAmount, reserveAmount, refundCostAmount, shopServiceFee, commissionDeduction, 0, 0)

	return OrderStatItem{
		TalentId:             talentId,
		Platform:             platform,
		TotalOrderCount:      row["total_order_count"].Int(),
		OrderCount:           orderCount,
		InvalidOrderCount:    row["invalid_order_count"].Int(),
		ProductNum:           row["product_num"].Int(),
		PayAmount:            payAmount,
		CostAmount:           costAmount,
		ReserveAmount:        reserveAmount,
		Commission:           commission,
		RefundCostAmount:     refundCostAmount,
		RefundOrderCount:     row["refund_order_count"].Int(),
		ShopServiceFee:       shopServiceFee,
		AdCostAmount:         0,
		ValidOrderCostMatch:  row["valid_order_cost_match"].Int(),
		CommissionOrderCount: orderCount,
		RealtimeProfit:       realtimeProfit,
	}
}

// aggregateByShop 按店铺聚合订单
func (s *FinanceAggregateService) aggregateByShop(orders []OrderStatItem) map[int]OrderStatItem {
	shopMap := make(map[int]OrderStatItem)

	for _, order := range orders {
		// 通过 talent_id 查找店铺
		shop, err := g.DB().Model("account").Where("uid", order.TalentId).One()
		if err != nil || shop.IsEmpty() {
			continue
		}

		accountId := shop["id"].Int()
		if existing, ok := shopMap[accountId]; ok {
			existing.TotalOrderCount += order.TotalOrderCount
			existing.OrderCount += order.OrderCount
			existing.InvalidOrderCount += order.InvalidOrderCount
			existing.ProductNum += order.ProductNum
			existing.PayAmount += order.PayAmount
			existing.CostAmount += order.CostAmount
			existing.ReserveAmount += order.ReserveAmount
			existing.Commission += order.Commission
			existing.RefundCostAmount += order.RefundCostAmount
			existing.RefundOrderCount += order.RefundOrderCount
			existing.ShopServiceFee += order.ShopServiceFee
			existing.AdCostAmount += order.AdCostAmount
			existing.ValidOrderCostMatch += order.ValidOrderCostMatch
			existing.CommissionOrderCount += order.CommissionOrderCount
			existing.RealtimeProfit += order.RealtimeProfit
			shopMap[accountId] = existing
		} else {
			order.AccountId = accountId
			shopMap[accountId] = order
		}
	}

	// 四舍五入
	for k, v := range shopMap {
		v.PayAmount = roundFloat(v.PayAmount, 2)
		v.CostAmount = roundFloat(v.CostAmount, 2)
		v.ReserveAmount = roundFloat(v.ReserveAmount, 2)
		v.Commission = roundFloat(v.Commission, 2)
		v.RefundCostAmount = roundFloat(v.RefundCostAmount, 2)
		v.ShopServiceFee = roundFloat(v.ShopServiceFee, 2)
		v.AdCostAmount = roundFloat(v.AdCostAmount, 2)
		// 净利润 = 定价总额 - 货款 - 留底 - 退款成本 - 小店服务费 - 佣金扣除 - 广告消耗
		v.RealtimeProfit = roundFloat(CalculateOrderProfit(v.PayAmount, v.CostAmount, v.ReserveAmount, v.RefundCostAmount, v.ShopServiceFee, v.PayAmount*0.005, 0, v.AdCostAmount), 2)
		shopMap[k] = v
	}

	return shopMap
}

// enrichAdCost 通过千川映射表补全各账号的广告消耗
func (s *FinanceAggregateService) enrichAdCost(ctx context.Context, shopMap map[int]OrderStatItem, date string) {
	if len(shopMap) == 0 {
		return
	}

	// 收集所有 accountId
	accountIds := make([]int, 0, len(shopMap))
	for accountId := range shopMap {
		accountIds = append(accountIds, accountId)
	}

	// 使用 GoFrame 查询构建器批量查询千川消耗
	rows, err := g.DB().Model("account_qianchuan_mapping aqm").
		LeftJoin("qianchuan_advertiser_daily_report qar", qianchuanAdReportJoinCond(date)).
		WhereIn("aqm.account_id", accountIds).
		Fields("aqm.account_id, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		Group("aqm.account_id").
		All()
	if err != nil {
		g.Log().Warningf(ctx, "[FinanceAggregate] 查询千川消耗失败: %v", err)
		return
	}

	for _, row := range rows {
		accountId := row["account_id"].Int()
		adCost := row["ad_cost"].Float64()
		if stat, ok := shopMap[accountId]; ok {
			stat.AdCostAmount = adCost
			shopMap[accountId] = stat
		}
	}
}

// saveStat 保存统计数据
func (s *FinanceAggregateService) saveStat(ctx context.Context, date string, companyId, departmentId, traderId int64, accountId int, platform string, stat OrderStatItem) {
	data := g.Map{
		"stat_date":              date,
		"company_id":             companyId,
		"department_id":          departmentId,
		"trader_id":              traderId,
		"account_id":             accountId,
		"platform":               platform,
		"total_order_count":      stat.TotalOrderCount,
		"order_count":            stat.OrderCount,
		"invalid_order_count":    stat.InvalidOrderCount,
		"product_num":            stat.ProductNum,
		"pay_amount":             stat.PayAmount,
		"cost_amount":            stat.CostAmount,
		"reserve_amount":         stat.ReserveAmount,
		"commission":             stat.Commission,
		"refund_cost_amount":     stat.RefundCostAmount,
		"refund_order_count":     stat.RefundOrderCount,
		"shop_service_fee":       stat.ShopServiceFee,
		"ad_cost_amount":         stat.AdCostAmount,
		"valid_order_cost_match": stat.ValidOrderCostMatch,
		"commission_order_count": stat.CommissionOrderCount,
		"realtime_profit":        stat.RealtimeProfit,
	}

	_, err := g.DB().Model("finance_daily_stats").Ctx(ctx).Data(data).OnDuplicate(
		"total_order_count", "order_count", "invalid_order_count", "product_num",
		"pay_amount", "cost_amount", "reserve_amount", "commission",
		"refund_cost_amount", "refund_order_count", "shop_service_fee",
		"ad_cost_amount", "valid_order_cost_match", "commission_order_count", "realtime_profit",
	).Insert()
	if err != nil {
		g.Log().Warningf(ctx, "[FinanceAggregate] 保存统计失败: %v, data: %+v", err, data)
	}
}

// AggregateRange 回填历史账号日汇总（2并发）。
func (s *FinanceAggregateService) AggregateRange(ctx context.Context, startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("开始日期格式错误: %v", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("结束日期格式错误: %v", err)
	}

	ctx, qianchuanBlankOrderInferenceEnabled := financeAggregateInferenceContext(ctx)
	g.Log().Infof(
		ctx,
		"[FinanceAggregate] 开始区间回填, 日期: %s~%s, 空白单反推: %t",
		startDate,
		endDate,
		qianchuanBlankOrderInferenceEnabled,
	)

	// 收集所有日期
	var dates []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	// 2 并发 worker（日订单量 4-5万，避免 RDS 压力过大）
	const workers = 2
	dateCh := make(chan string, len(dates))
	for _, date := range dates {
		dateCh <- date
	}
	close(dateCh)

	var wg sync.WaitGroup
	errCh := make(chan error, len(dates))
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for date := range dateCh {
				if err := s.DailyAggregateJob(ctx, date); err != nil {
					g.Log().Errorf(ctx, "[FinanceAggregate] 回填数据失败, 日期: %s, 错误: %v", date, err)
					errCh <- fmt.Errorf("日期%s: %w", date, err)
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for aggregateErr := range errCh {
		return aggregateErr
	}

	return nil
}

// OrderDailyAggregateJob 抖音订单每日聚合任务
func (s *FinanceAggregateService) OrderDailyAggregateJob(ctx context.Context, date string) error {
	g.Log().Infof(ctx, "[OrderAggregate] 开始订单聚合, 日期: %s", date)

	startTime := date + " 00:00:00"
	endTime := date + " 23:59:59"

	validCond := financeValidOrderCondition("o")
	invalidCond := financeInvalidOrderCondition("o")
	refundCond := financeRefundOrderCondition("o")

	sql := fmt.Sprintf(`
		SELECT
			o.shop_name,
			COUNT(DISTINCT o.main_order_no) AS total_order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS valid_order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS invalid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price,0)*o.product_num ELSE 0 END),0) AS valid_pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price,0)*o.product_num ELSE 0 END),0) AS product_cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price,0)*o.product_num*0.03 ELSE 0 END),0) AS reserve_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price,0)*o.product_num ELSE 0 END),0) AS refund_cost_amount,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS refund_order_count,
			COUNT(DISTINCT CASE WHEN %s AND g.base_price>0 THEN o.main_order_no END) AS valid_order_cost_match
		FROM order_info o
		LEFT JOIN goods g ON o.seller_code = g.goods_code
		WHERE o.order_submit_time >= ? AND o.order_submit_time <= ?
		  AND o.main_order_no IS NOT NULL AND o.main_order_no != ''
		GROUP BY o.shop_name
	`, validCond, invalidCond, validCond, validCond, validCond, refundCond, refundCond, validCond)

	result, err := g.DB().GetAll(ctx, sql, startTime, endTime)
	if err != nil {
		return fmt.Errorf("查询订单失败: %w", err)
	}

	// 先删后插
	g.DB().Model("daily_order_stats").Where("stat_date", date).Delete()

	for _, row := range result {
		shopName := row["shop_name"].String()
		if shopName == "" {
			continue
		}
		payAmount := row["valid_pay_amount"].Float64()
		costAmount := row["product_cost_amount"].Float64()
		reserveAmount := row["reserve_amount"].Float64()
		reserveAmount = CalculateReserveAmount(payAmount)
		shopServiceFee := CalculateDouyinShopServiceFee(payAmount)
		commission := roundFloat(payAmount-costAmount-reserveAmount-shopServiceFee, 2)

		g.DB().Model("daily_order_stats").Data(g.Map{
			"stat_date":              date,
			"shop_name":              shopName,
			"total_order_count":      row["total_order_count"].Int(),
			"valid_order_count":      row["valid_order_count"].Int(),
			"invalid_order_count":    row["invalid_order_count"].Int(),
			"valid_pay_amount":       roundFloat(payAmount, 2),
			"product_cost_amount":    roundFloat(costAmount, 2),
			"reserve_amount":         roundFloat(reserveAmount, 2),
			"refund_cost_amount":     roundFloat(row["refund_cost_amount"].Float64(), 2),
			"refund_order_count":     row["refund_order_count"].Int(),
			"shop_service_fee":       shopServiceFee,
			"commission":             commission,
			"valid_order_cost_match": row["valid_order_cost_match"].Int(),
		}).InsertIgnore()
	}

	g.Log().Infof(ctx, "[OrderAggregate] 订单聚合完成, 日期: %s, 店铺数: %d", date, len(result))
	return nil
}
