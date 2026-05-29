package service

import (
	"context"
	"fmt"
	"time"

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

	// 4. 写入各维度统计
	for shopId, stat := range shopMap {
		// 获取店铺信息
		shop, err := g.DB().Model("shop").Where("id", shopId).One()
		if err != nil || shop.IsEmpty() {
			continue
		}

		companyId := shop["company_id"].Int64()
		departmentId := shop["department_id"].Int64()

		// 4.2 写入投手维度，同时从投手获取组织信息
		traders, _ := g.DB().Model("trader_shop_relation").
			Where("shop_id", shopId).
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

				s.saveStat(ctx, date, traderCompanyId, traderGroupId, traderId, 0, "", stat)
			}
		}

		// 4.1 写入店铺维度
		s.saveStat(ctx, date, companyId, departmentId, 0, shopId, "", stat)

		// 4.3 写入部门维度（汇总）
		if departmentId > 0 {
			s.saveStat(ctx, date, companyId, departmentId, 0, 0, "", stat)
		}

		// 4.4 写入公司维度（汇总）
		if companyId > 0 {
			s.saveStat(ctx, date, companyId, 0, 0, 0, "", stat)
		}

		// 4.5 写入平台维度（可选）
		platform := shop["platform"].String()
		if platform != "" {
			s.saveStat(ctx, date, companyId, departmentId, 0, shopId, platform, stat)
		}
	}

	g.Log().Infof(ctx, "[FinanceAggregate] 日聚合任务完成, 日期: %s, 店铺数: %d", date, len(shopMap))
	return nil
}

// OrderStatItem 订单统计项
type OrderStatItem struct {
	ShopId        int
	TalentId      string
	OrderCount    int
	ProductNum    int
	PayAmount     float64
	CostAmount    float64
	ReserveAmount float64
	Commission    float64
}

// getAllValidOrders 获取某日的所有有效订单
func (s *FinanceAggregateService) getAllValidOrders(ctx context.Context, date string) ([]OrderStatItem, error) {
	var allOrders []OrderStatItem

	// 计算时间范围
	startTime := date + " 00:00:00"
	endTime := date + " 23:59:59"

	validFilter := "AND after_sale_status NOT IN ('退款成功', '已全额退款', '待退货') AND order_status != '已关闭'"

	// 1. 抖音订单
	sql := fmt.Sprintf(`
		SELECT
			o.talent_id,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.order_pay_amount), 0) AS pay_amount,
			COALESCE(SUM(COALESCE(g.base_price, o.product_price) * o.product_num), 0) AS cost_amount
		FROM order_info o
		LEFT JOIN goods g ON o.seller_code = g.goods_code
		WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
		  AND o.main_order_no IS NOT NULL AND o.main_order_no != ''
		  %s
		GROUP BY o.talent_id
	`, validFilter)

	result, err := g.DB().GetAll(ctx, sql, startTime, endTime)
	if err == nil {
		for _, row := range result {
			talentId := row["talent_id"].String()
			if talentId == "" {
				continue
			}
			payAmount := row["pay_amount"].Float64()
			costAmount := row["cost_amount"].Float64()
			reserveAmount := payAmount * 0.05
			allOrders = append(allOrders, OrderStatItem{
				TalentId:      talentId,
				OrderCount:    row["order_count"].Int(),
				ProductNum:    row["product_num"].Int(),
				PayAmount:     payAmount,
				CostAmount:    costAmount,
				ReserveAmount: reserveAmount,
				Commission:    payAmount - costAmount - reserveAmount,
			})
		}
	}

	// 2. 快手订单
	sql = fmt.Sprintf(`
		SELECT
			o.talent_id,
			COUNT(DISTINCT o.order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.order_pay_amount), 0) AS pay_amount,
			COALESCE(SUM(COALESCE(g.base_price, o.product_price) * o.product_num), 0) AS cost_amount
		FROM kuaishou_orders o
		LEFT JOIN goods g ON o.seller_code COLLATE utf8mb4_unicode_ci = g.goods_code
		WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
		  AND o.order_no IS NOT NULL AND o.order_no != ''
		  %s
		GROUP BY o.talent_id
	`, validFilter)

	result, err = g.DB().GetAll(ctx, sql, startTime, endTime)
	if err == nil {
		for _, row := range result {
			talentId := row["talent_id"].String()
			if talentId == "" {
				continue
			}
			payAmount := row["pay_amount"].Float64()
			costAmount := row["cost_amount"].Float64()
			reserveAmount := payAmount * 0.05
			allOrders = append(allOrders, OrderStatItem{
				TalentId:      talentId,
				OrderCount:    row["order_count"].Int(),
				ProductNum:    row["product_num"].Int(),
				PayAmount:     payAmount,
				CostAmount:    costAmount,
				ReserveAmount: reserveAmount,
				Commission:    payAmount - costAmount - reserveAmount,
			})
		}
	}

	// 3. 视频号订单
	sql = fmt.Sprintf(`
		SELECT
			o.talent_id,
			COUNT(DISTINCT o.order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.order_pay_amount), 0) AS pay_amount,
			COALESCE(SUM(COALESCE(g.base_price, o.product_price) * o.product_num), 0) AS cost_amount
		FROM shipinhao_orders o
		LEFT JOIN goods g ON o.seller_code COLLATE utf8mb4_unicode_ci = g.goods_code
		WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
		  AND o.order_no IS NOT NULL AND o.order_no != ''
		  %s
		GROUP BY o.talent_id
	`, validFilter)

	result, err = g.DB().GetAll(ctx, sql, startTime, endTime)
	if err == nil {
		for _, row := range result {
			talentId := row["talent_id"].String()
			if talentId == "" {
				continue
			}
			payAmount := row["pay_amount"].Float64()
			costAmount := row["cost_amount"].Float64()
			reserveAmount := payAmount * 0.05
			allOrders = append(allOrders, OrderStatItem{
				TalentId:      talentId,
				OrderCount:    row["order_count"].Int(),
				ProductNum:    row["product_num"].Int(),
				PayAmount:     payAmount,
				CostAmount:    costAmount,
				ReserveAmount: reserveAmount,
				Commission:    payAmount - costAmount - reserveAmount,
			})
		}
	}

	return allOrders, nil
}

// aggregateByShop 按店铺聚合订单
func (s *FinanceAggregateService) aggregateByShop(orders []OrderStatItem) map[int]OrderStatItem {
	shopMap := make(map[int]OrderStatItem)

	for _, order := range orders {
		// 通过 talent_id 查找店铺
		shop, err := g.DB().Model("shop").Where("uid", order.TalentId).One()
		if err != nil || shop.IsEmpty() {
			continue
		}

		shopId := shop["id"].Int()
		if existing, ok := shopMap[shopId]; ok {
			// 合并统计
			existing.OrderCount += order.OrderCount
			existing.ProductNum += order.ProductNum
			existing.PayAmount += order.PayAmount
			existing.CostAmount += order.CostAmount
			existing.ReserveAmount += order.ReserveAmount
			existing.Commission += order.Commission
			shopMap[shopId] = existing
		} else {
			order.ShopId = shopId
			shopMap[shopId] = order
		}
	}

	// 四舍五入
	for k, v := range shopMap {
		v.PayAmount = roundFloat(v.PayAmount, 2)
		v.CostAmount = roundFloat(v.CostAmount, 2)
		v.ReserveAmount = roundFloat(v.ReserveAmount, 2)
		v.Commission = roundFloat(v.Commission, 2)
		shopMap[k] = v
	}

	return shopMap
}

// saveStat 保存统计数据
func (s *FinanceAggregateService) saveStat(ctx context.Context, date string, companyId, departmentId, traderId int64, shopId int, platform string, stat OrderStatItem) {
	data := g.Map{
		"stat_date":      date,
		"company_id":     companyId,
		"department_id":  departmentId,
		"trader_id":      traderId,
		"shop_id":        shopId,
		"platform":       platform,
		"order_count":    stat.OrderCount,
		"product_num":    stat.ProductNum,
		"pay_amount":     stat.PayAmount,
		"cost_amount":    stat.CostAmount,
		"reserve_amount": stat.ReserveAmount,
		"commission":     stat.Commission,
	}

	_, err := g.DB().Model("finance_daily_stats").Data(data).Insert()
	if err != nil {
		g.Log().Warningf(ctx, "[FinanceAggregate] 保存统计失败: %v, data: %+v", err, data)
	}
}

// AggregateRange 回填历史数据
func (s *FinanceAggregateService) AggregateRange(ctx context.Context, startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("开始日期格式错误: %v", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("结束日期格式错误: %v", err)
	}

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		if err := s.DailyAggregateJob(ctx, date); err != nil {
			g.Log().Errorf(ctx, "[FinanceAggregate] 回填数据失败, 日期: %s, 错误: %v", date, err)
			// 继续执行，不中断
		}
	}

	return nil
}
