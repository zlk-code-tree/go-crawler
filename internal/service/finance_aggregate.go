package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// FinanceAggregateService 财务管理 - 日聚合任务服务
type FinanceAggregateService struct{}

func NewFinanceAggregateService() *FinanceAggregateService {
	return &FinanceAggregateService{}
}

// DailyAggregateJob 每日聚合任务
// 建议在每天凌晨2点执行，统计前一天的数据
func (s *FinanceAggregateService) DailyAggregateJob(ctx context.Context, date string) error {
	g.Log().Infof(ctx, "[FinanceAggregate] 开始执行日聚合任务, 日期: %s", date)

	// 1. 先删除该日期的旧数据（保证幂等性）
	_, err := g.DB().Model("finance_daily_stats").Where("stat_date", date).Delete()
	if err != nil {
		g.Log().Errorf(ctx, "[FinanceAggregate] 删除旧数据失败: %v", err)
		return err
	}

	// 2. 查询该日期的所有有效订单（跨三张表）
	orders, err := s.getAllValidOrders(ctx, date)
	if err != nil {
		g.Log().Errorf(ctx, "[FinanceAggregate] 查询订单失败: %v", err)
		return err
	}

	g.Log().Infof(ctx, "[FinanceAggregate] 查询到 %d 条订单", len(orders))

	// 3. 按店铺聚合
	shopMap := s.aggregateByShop(orders)

	// 4. 写入各维度统计（去重：同一组合只写一次）
	saved := make(map[string]bool)
	uniqueKey := func(companyId, departmentId, traderId int64, accountId int, platform string) string {
		return fmt.Sprintf("%d-%d-%d-%d-%s", companyId, departmentId, traderId, accountId, platform)
	}

	for accountId, stat := range shopMap {
		// 获取账号信息
		shop, err := g.DB().Model("account").Where("id", accountId).One()
		if err != nil || shop.IsEmpty() {
			continue
		}

		companyId := shop["company_id"].Int64()
		departmentId := shop["department_id"].Int64()

		// 4.2 写入投手维度，同时从投手获取组织信息
		traders, _ := g.DB().Model("trader_account_relation").
			Where("account_id", accountId).
			All()
		for _, trader := range traders {
			traderId := trader["trader_id"].Int64()
			traderInfo, _ := g.DB().Model("trader").Where("id", traderId).One()
			if !traderInfo.IsEmpty() {
				traderCompanyId := traderInfo["company_id"].Int64()
				traderGroupId := traderInfo["group_id"].Int64()

				// 如果 shop 的 company_id/department_id 为0，从投手获取
				if companyId == 0 && traderCompanyId > 0 {
					companyId = traderCompanyId
				}
				if departmentId == 0 && traderGroupId > 0 {
					departmentId = traderGroupId
				}

				key := uniqueKey(traderCompanyId, traderGroupId, traderId, 0, "")
				if !saved[key] {
					saved[key] = true
					s.saveStat(ctx, date, traderCompanyId, traderGroupId, traderId, 0, "", stat)
				}
			}
		}

		// 4.1 写入账号维度
		s.saveStat(ctx, date, companyId, departmentId, 0, accountId, "", stat)

		// 4.3 写入部门维度（汇总）
		if departmentId > 0 {
			key := uniqueKey(companyId, departmentId, 0, 0, "")
			if !saved[key] {
				saved[key] = true
				s.saveStat(ctx, date, companyId, departmentId, 0, 0, "", stat)
			}
		}

		// 4.4 写入公司维度（汇总）
		if companyId > 0 {
			key := uniqueKey(companyId, 0, 0, 0, "")
			if !saved[key] {
				saved[key] = true
				s.saveStat(ctx, date, companyId, 0, 0, 0, "", stat)
			}
		}

		// 4.5 写入平台维度（可选）
		platform := shop["platform"].String()
		if platform != "" {
			s.saveStat(ctx, date, companyId, departmentId, 0, accountId, platform, stat)
		}
	}

	g.Log().Infof(ctx, "[FinanceAggregate] 日聚合任务完成, 日期: %s, 账号数: %d", date, len(shopMap))
	return nil
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
	validCond := "(o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','12')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51'))"
	// 无效订单条件
	invalidCond := "o.after_sale_status IN ('6','7','12')"
	// 退款成本条件：无效 且 已发货
	refundCond := "o.after_sale_status IN ('6','7','12') AND o.order_status = '已发货'"

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

	sql := fmt.Sprintf(`
		SELECT
			o.talent_id,
			COUNT(DISTINCT o.%s) AS total_order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS invalid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_num,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price, 0) * o.product_num ELSE 0 END), 0) AS pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, o.product_price) * o.product_num ELSE 0 END), 0) AS cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_order_count,
			COUNT(DISTINCT CASE WHEN %s AND g.base_price > 0 THEN o.%s END) AS valid_order_cost_match
		FROM %s o
		LEFT JOIN goods g ON o.seller_code%s = g.goods_code
		WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
		  AND o.%s IS NOT NULL AND o.%s != ''
		GROUP BY o.talent_id
	`, orderNoCol, validCond, orderNoCol, invalidCond, orderNoCol, validCond, validCond, validCond, refundCond, refundCond, orderNoCol, validCond, orderNoCol, tableName, collate, orderNoCol, orderNoCol)

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
	reserveAmount := payAmount * 0.03
	orderCount := row["order_count"].Int()
	refundCostAmount := row["refund_cost_amount"].Float64()
	shopServiceFee := 0.0
	if platform == "douyin" {
		shopServiceFee = payAmount * 0.02
	}
	// 佣金 = 定价总额 - 留底 - 货款 - 小店服务费
	commission := payAmount - costAmount - reserveAmount - shopServiceFee
	realtimeProfit := payAmount - costAmount - reserveAmount - refundCostAmount - shopServiceFee - commission

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
		v.RealtimeProfit = roundFloat(v.RealtimeProfit, 2)
		shopMap[k] = v
	}

	return shopMap
}

// saveStat 保存统计数据
func (s *FinanceAggregateService) saveStat(ctx context.Context, date string, companyId, departmentId, traderId int64, accountId int, platform string, stat OrderStatItem) {
	data := g.Map{
		"stat_date":              date,
		"company_id":             companyId,
		"department_id":          departmentId,
		"trader_id":              traderId,
		"account_id":                accountId,
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

	_, err := g.DB().Model("finance_daily_stats").Data(data).InsertIgnore()
	if err != nil {
		g.Log().Warningf(ctx, "[FinanceAggregate] 保存统计失败: %v, data: %+v", err, data)
	}
}

// AggregateRange 回填历史数据（3并发）
func (s *FinanceAggregateService) AggregateRange(ctx context.Context, startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("开始日期格式错误: %v", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("结束日期格式错误: %v", err)
	}

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
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for date := range dateCh {
				if err := s.DailyAggregateJob(ctx, date); err != nil {
					g.Log().Errorf(ctx, "[FinanceAggregate] 回填数据失败, 日期: %s, 错误: %v", date)
				}
			}
		}()
	}
	wg.Wait()

	return nil
}

// OrderDailyAggregateJob 抖音订单每日聚合任务
func (s *FinanceAggregateService) OrderDailyAggregateJob(ctx context.Context, date string) error {
	g.Log().Infof(ctx, "[OrderAggregate] 开始订单聚合, 日期: %s", date)

	startTime := date + " 00:00:00"
	endTime := date + " 23:59:59"

	validCond := "(o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','12')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51'))"
	invalidCond := "o.after_sale_status IN ('6','7','12')"
	refundCond := "o.after_sale_status IN ('6','7','12') AND o.order_status = '已发货'"

	sql := fmt.Sprintf(`
		SELECT
			o.shop_name,
			COUNT(DISTINCT o.main_order_no) AS total_order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS valid_order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS invalid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price,0)*o.product_num ELSE 0 END),0) AS valid_pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price,0)*o.product_num ELSE 0 END),0) AS product_cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price,0)*o.product_num*0.03 ELSE 0 END),0) AS reserve_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(o.product_price,0)*o.product_num ELSE 0 END),0) AS refund_cost_amount,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS refund_order_count,
			COUNT(DISTINCT CASE WHEN %s AND g.base_price>0 THEN o.main_order_no END) AS valid_order_cost_match
		FROM order_info o
		LEFT JOIN goods g ON o.seller_code = g.goods_code
		WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
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
		shopServiceFee := roundFloat(payAmount*0.02, 2)
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
