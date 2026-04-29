package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// OrderAnalysisService 订单分析服务
type OrderAnalysisService struct{}

func NewOrderAnalysisService() *OrderAnalysisService {
	return &OrderAnalysisService{}
}

// GetTimeRange 根据时间类型获取日期范围
func (s *OrderAnalysisService) GetTimeRange(timeType string) (startDate, endDate string) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch timeType {
	case "today":
		// 今日
		startDate = today.Format("2006-01-02")
		endDate = today.Add(24 * time.Hour).Format("2006-01-02")
	case "yesterday":
		// 昨日
		yesterday := today.AddDate(0, 0, -1)
		startDate = yesterday.Format("2006-01-02")
		endDate = today.Format("2006-01-02")
	case "week":
		// 近7日（含今天）
		weekStart := today.AddDate(0, 0, -6)
		startDate = weekStart.Format("2006-01-02")
		endDate = today.Add(24 * time.Hour).Format("2006-01-02")
	case "month":
		// 本月
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		startDate = monthStart.Format("2006-01-02")
		endDate = today.Add(24 * time.Hour).Format("2006-01-02")
	default:
		// 默认今日
		startDate = today.Format("2006-01-02")
		endDate = today.Add(24 * time.Hour).Format("2006-01-02")
	}
	return
}

// GetValidOrderFilter 获取有效订单的筛选条件SQL片段
// 注意：此方法用于子查询内部，订单表别名为 o1
func (s *OrderAnalysisService) GetValidOrderFilter() string {
	return " " +
		"AND o1.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货') " +
		"AND o1.order_status != '已关闭' "
}

// GroupStats 小组维度统计
func (s *OrderAnalysisService) GroupStats(ctx context.Context, startDate, endDate string) ([]map[string]interface{}, error) {
	// 构建统计SQL - 先按 main_order_no 聚合，避免 JOIN 导致重复计算
	// 问题：一个订单可能包含多个商品（多行），order_pay_amount 是订单级金额，不能直接 SUM
	// 解决：使用子查询先按主订单号聚合，再去 JOIN
	// 注意：使用 main_order_no 统计订单数量，一个主订单只算一次
	sql := `
		SELECT 
			COALESCE(s.operation_leader, '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			-- 子查询：按主订单号聚合，确保每个主订单只计算一次订单级金额
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				-- 商品数量：求和（一个订单多个商品）
				SUM(o1.product_num) AS product_num,
				-- 商品总价 = Σ(商品单价 × 数量)，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				-- 成本：按商品分别计算后求和，处理NULL值
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				-- 留底 = 商品总价 × 5%，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON TRIM(o1.seller_code) = TRIM(g.goods_code) AND g.status = 1
			WHERE o1.pay_finish_time >= '` + startDate + `'
			  AND o1.pay_finish_time < '` + endDate + `'
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN shop s ON o.talent_id = s.uid
		GROUP BY s.operation_leader
		ORDER BY s.operation_leader
	`

	result, err := g.DB().GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	// 转换结果
	stats := make([]map[string]interface{}, 0, len(result))
	for _, row := range result {
		productTotal := row["pay_amount"].Float64()
		costAmount := row["cost_amount"].Float64()
		reserveAmount := row["reserve_amount"].Float64()
		// 佣金 = 商品总价 - 留底 - 货款
		commission := productTotal - reserveAmount - costAmount

		stats = append(stats, map[string]interface{}{
			"groupName":     row["group_name"],
			"orderCount":    row["order_count"].Int(),
			"productNum":    row["product_num"].Int(),
			"payAmount":     roundFloat(productTotal, 2),
			"costAmount":    roundFloat(costAmount, 2),
			"reserveAmount": roundFloat(reserveAmount, 2),
			"commission":    roundFloat(commission, 2),
		})
	}

	return stats, nil
}

// ShopStats 店铺维度统计
func (s *OrderAnalysisService) ShopStats(ctx context.Context, startDate, endDate string, groupId int) ([]map[string]interface{}, error) {
	// 构建店铺筛选条件
	shopFilter := ""
	if groupId > 0 {
		shopFilter = fmt.Sprintf(" AND s.id = %d", groupId)
	}

	// 构建统计SQL - 先按 main_order_no 聚合，避免 JOIN 导致重复计算
	// 注意：使用 main_order_no 统计订单数量，一个主订单只算一次
	sql := `
		SELECT 
			s.id AS shop_id,
			o.shop_name AS shop_name,
			COALESCE(s.operation_leader, '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			-- 子查询：按主订单号聚合，确保每个主订单只计算一次订单级金额
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				-- 商品数量：求和（一个订单多个商品）
				SUM(o1.product_num) AS product_num,
				-- 商品总价 = Σ(商品单价 × 数量)，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				-- 成本：按商品分别计算后求和，处理NULL值
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				-- 留底 = 商品总价 × 5%，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON TRIM(o1.seller_code) = TRIM(g.goods_code) AND g.status = 1
			WHERE o1.pay_finish_time >= '` + startDate + `'
			  AND o1.pay_finish_time < '` + endDate + `'
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN shop s ON o.talent_id = s.uid
		WHERE 1=1 ` + shopFilter + `
		GROUP BY s.id, o.shop_name, s.operation_leader
		ORDER BY o.shop_name
	`

	result, err := g.DB().GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	// 转换结果
	stats := make([]map[string]interface{}, 0, len(result))
	for _, row := range result {
		productTotal := row["pay_amount"].Float64()
		costAmount := row["cost_amount"].Float64()
		reserveAmount := row["reserve_amount"].Float64()
		// 佣金 = 商品总价 - 留底 - 货款
		commission := productTotal - reserveAmount - costAmount

		stats = append(stats, map[string]interface{}{
			"shopId":        row["shop_id"].Int(),
			"shopName":      row["shop_name"],
			"groupName":     row["group_name"],
			"orderCount":    row["order_count"].Int(),
			"productNum":    row["product_num"].Int(),
			"payAmount":     roundFloat(productTotal, 2),
			"costAmount":    roundFloat(costAmount, 2),
			"reserveAmount": roundFloat(reserveAmount, 2),
			"commission":    roundFloat(commission, 2),
		})
	}

	return stats, nil
}

// GetAllGroups 获取所有小组列表（用于下拉筛选）
func (s *OrderAnalysisService) GetAllGroups(ctx context.Context) ([]map[string]interface{}, error) {
	sql := `
		SELECT DISTINCT 
			COALESCE(operation_leader, '未分配小组') AS group_name
		FROM shop 
		ORDER BY group_name
	`

	result, err := g.DB().GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询小组列表失败: %v", err)
	}

	groups := make([]map[string]interface{}, 0, len(result))
	for _, row := range result {
		groups = append(groups, map[string]interface{}{
			"groupName": row["group_name"],
		})
	}

	return groups, nil
}

// GetAllShops 获取所有店铺列表（用于下拉筛选）
func (s *OrderAnalysisService) GetAllShops(ctx context.Context) ([]map[string]interface{}, error) {
	sql := `
		SELECT id, douyin_name, COALESCE(operation_leader, '未分配小组') AS group_name
		FROM shop 
		ORDER BY douyin_name
	`

	result, err := g.DB().GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询店铺列表失败: %v", err)
	}

	shops := make([]map[string]interface{}, 0, len(result))
	for _, row := range result {
		shops = append(shops, map[string]interface{}{
			"shopId":    row["id"].Int(),
			"shopName":  row["douyin_name"],
			"groupName": row["group_name"],
		})
	}

	return shops, nil
}

// roundFloat 四舍五入保留指定小数位
func roundFloat(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	return float64(int(val*ratio+0.5)) / ratio
}

// ParseTimeRange 统一时间解析：preset 与自定义日期二选一
// 返回 startDate 和 endDate（格式 2006-01-02）
func (s *OrderAnalysisService) ParseTimeRange(preset, customStart, customEnd string) (startDate, endDate string) {
	// 如果传了自定义日期，优先使用
	if customStart != "" && customEnd != "" {
		return customStart, customEnd
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)

	switch preset {
	case "today":
		return today.Format("2006-01-02"), tomorrow.Format("2006-01-02")
	case "yesterday":
		y := today.AddDate(0, 0, -1)
		return y.Format("2006-01-02"), today.Format("2006-01-02")
	case "last7":
		return today.AddDate(0, 0, -6).Format("2006-01-02"), tomorrow.Format("2006-01-02")
	case "last30":
		return today.AddDate(0, 0, -29).Format("2006-01-02"), tomorrow.Format("2006-01-02")
	case "thisMonth":
		ms := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return ms.Format("2006-01-02"), tomorrow.Format("2006-01-02")
	case "all":
		// 从最早的数据开始
		return "2020-01-01", tomorrow.Format("2006-01-02")
	default:
		return today.Format("2006-01-02"), tomorrow.Format("2006-01-02")
	}
}

// buildBaseQuery 构建基础统计查询（复用，避免重复SQL）
// 先按 main_order_no 聚合，避免 JOIN 导致重复计算
// 注意：使用 main_order_no 统计订单数量，一个主订单只算一次
func (s *OrderAnalysisService) buildBaseQuery(startDate, endDate, extraWhere string) string {
	validFilter := s.GetValidOrderFilter()
	return `
		SELECT 
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			-- 子查询：按主订单号聚合，确保每个主订单只计算一次订单级金额
			SELECT 
				o1.main_order_no,
			-- 商品总价 = Σ(商品单价 × 数量)，处理NULL值
			SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				-- 成本：按商品分别计算后求和，处理NULL值
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				-- 留底 = 商品总价 × 5%，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON TRIM(o1.seller_code) = TRIM(g.goods_code) AND g.status = 1
			WHERE o1.pay_finish_time >= '` + startDate + `' 
			  AND o1.pay_finish_time < '` + endDate + `'
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + validFilter + `
			GROUP BY o1.main_order_no
		) o
		WHERE 1=1 ` + extraWhere + `
	`
}

// Overview 统计概览：期间统计 + 历史累计
func (s *OrderAnalysisService) Overview(ctx context.Context, startDate, endDate string) (period, total map[string]interface{}, err error) {
	// 1. 期间统计
	periodSQL := s.buildBaseQuery(startDate, endDate, "")
	periodResult, e := g.DB().GetAll(ctx, periodSQL)
	if e != nil {
		err = fmt.Errorf("期间统计查询失败: %v", e)
		return
	}

	payAmount := 0.0
	costAmount := 0.0
	reserveAmount := 0.0
	orderCount := 0
	if len(periodResult) > 0 {
		row := periodResult[0]
		payAmount = row["pay_amount"].Float64()
		costAmount = row["cost_amount"].Float64()
		reserveAmount = row["reserve_amount"].Float64()
		orderCount = row["order_count"].Int()
	}

	// 2. 退款统计 - 先按主订单号聚合，避免重复计算
	// 注意：使用 main_order_no 统计退款订单数量，一个主订单只算一次
	// 注意：退款金额仍使用订单应付金额（保持原逻辑）
	refundSQL := `
		SELECT 
			COUNT(DISTINCT o.main_order_no) AS refund_count,
			COALESCE(SUM(o.refund_amount), 0) AS refund_amount
		FROM (
			-- 子查询：按主订单号聚合，确保每个主订单只计算一次退款金额
			SELECT 
				o1.main_order_no,
				-- 退款金额：取最大值（所有行相同，都是订单级金额）
				MAX(o1.order_pay_amount + o1.platform_discount + o1.talent_discount) AS refund_amount
			FROM order_info o1
			WHERE o1.pay_finish_time >= '` + startDate + `' 
			  AND o1.pay_finish_time < '` + endDate + `'
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  AND o1.after_sale_status IN ('退款成功', '已全额退款')
			GROUP BY o1.main_order_no
		) o
	`
	refundResult, e := g.DB().GetAll(ctx, refundSQL)
	if e != nil {
		err = fmt.Errorf("退款统计查询失败: %v", e)
		return
	}
	refundCount := 0
	refundAmount := 0.0
	if len(refundResult) > 0 {
		refundCount = refundResult[0]["refund_count"].Int()
		refundAmount = refundResult[0]["refund_amount"].Float64()
	}

	period = map[string]interface{}{
		"orderCount":   orderCount,
		"payAmount":    roundFloat(payAmount, 2),
		"costAmount":   roundFloat(costAmount, 2),
		"commission":   roundFloat(payAmount-reserveAmount-costAmount, 2),
		"refundCount":  refundCount,
		"refundAmount": roundFloat(refundAmount, 2),
	}

	// 3. 历史累计（不限制时间）
	totalSQL := s.buildBaseQuery("2020-01-01", time.Now().Add(24*time.Hour).Format("2006-01-02"), "")
	totalResult, e := g.DB().GetAll(ctx, totalSQL)
	if e != nil {
		err = fmt.Errorf("累计统计查询失败: %v", e)
		return
	}
	totalPay := 0.0
	totalCost := 0.0
	totalReserve := 0.0
	totalOrders := 0
	if len(totalResult) > 0 {
		r := totalResult[0]
		totalPay = r["pay_amount"].Float64()
		totalCost = r["cost_amount"].Float64()
		totalReserve = r["reserve_amount"].Float64()
		totalOrders = r["order_count"].Int()
	}

	total = map[string]interface{}{
		"orderCount": totalOrders,
		"payAmount":  roundFloat(totalPay, 2),
		"commission": roundFloat(totalPay-totalReserve-totalCost, 2),
	}

	return
}

// SalesTrend 销售额趋势：当前期 + 上月同期对比
func (s *OrderAnalysisService) SalesTrend(ctx context.Context, startDate, endDate string) (current, compare []map[string]interface{}, err error) {
	// 解析日期
	start, e1 := time.Parse("2006-01-02", startDate)
	end, e2 := time.Parse("2006-01-02", endDate)
	if e1 != nil || e2 != nil {
		err = fmt.Errorf("日期格式错误")
		return
	}

	// 计算天数
	days := int(end.Sub(start).Hours()/24) + 1
	// 上月同期
	compareStart := start.AddDate(0, -1, 0)
	compareEnd := end.AddDate(0, -1, 0)

	// 查询当前期
	current, err = s.queryDailyStats(ctx, startDate, endDate, days)
	if err != nil {
		return
	}

	// 查询对比期
	compare, err = s.queryDailyStats(ctx, compareStart.Format("2006-01-02"), compareEnd.Format("2006-01-02"), days)
	if err != nil {
		return
	}

	return
}

// queryDailyStats 查询每日统计
func (s *OrderAnalysisService) queryDailyStats(ctx context.Context, startDate, endDate string, days int) ([]map[string]interface{}, error) {
	sql := `
		SELECT 
			DATE(o.pay_finish_time) AS stat_date,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			-- 子查询：按主订单号聚合，确保每个主订单只计算一次订单级金额
			SELECT 
				o1.main_order_no,
				o1.pay_finish_time,
			-- 商品总价 = Σ(商品单价 × 数量)，处理NULL值
			SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				-- 成本：按商品分别计算后求和，处理NULL值
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				-- 留底 = 商品总价 × 5%，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON TRIM(o1.seller_code) = TRIM(g.goods_code) AND g.status = 1
			WHERE o1.pay_finish_time >= '` + startDate + `' 
			  AND o1.pay_finish_time < '` + endDate + `'
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.pay_finish_time
		) o
		GROUP BY DATE(o.pay_finish_time)
		ORDER BY stat_date
	`

	result, err := g.DB().GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("日统计查询失败: %v", err)
	}

	// 构建日期 map
	dataMap := make(map[string]map[string]interface{}, len(result))
	for _, row := range result {
		dateStr := row["stat_date"].String()
		pay := row["pay_amount"].Float64()
		cost := row["cost_amount"].Float64()
		reserve := row["reserve_amount"].Float64()
		dataMap[dateStr] = map[string]interface{}{
			"date":       dateStr,
			"orderCount": row["order_count"].Int(),
			"payAmount":  roundFloat(pay, 2),
			"commission": roundFloat(pay-reserve-cost, 2),
		}
	}

	// 填充空白天为0
	start, _ := time.Parse("2006-01-02", startDate)
	rows := make([]map[string]interface{}, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		if item, ok := dataMap[d]; ok {
			rows = append(rows, item)
		} else {
			rows = append(rows, map[string]interface{}{
				"date":       d,
				"orderCount": 0,
				"payAmount":  0.00,
				"commission": 0.00,
			})
		}
	}

	return rows, nil
}

// Rank 排名榜单：小组排名 + 店铺排名
func (s *OrderAnalysisService) Rank(ctx context.Context, startDate, endDate string) (groupRank, shopRank []map[string]interface{}, err error) {
	// 小组排名 - 先按主订单号聚合，避免重复计算
	// 注意：使用 main_order_no 统计订单数量，一个主订单只算一次
	groupSQL := `
		SELECT 
			COALESCE(s.operation_leader, '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS goods_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			-- 子查询：按主订单号聚合，确保每个主订单只计算一次订单级金额
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				-- 商品数量：求和（一个订单多个商品）
				SUM(o1.product_num) AS product_num,
			-- 商品总价 = Σ(商品单价 × 数量)，处理NULL值
			SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				-- 成本：按商品分别计算后求和，处理NULL值
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				-- 留底 = 商品总价 × 5%，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON TRIM(o1.seller_code) = TRIM(g.goods_code) AND g.status = 1
			WHERE o1.pay_finish_time >= '` + startDate + `'
			  AND o1.pay_finish_time < '` + endDate + `'
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN shop s ON o.talent_id = s.uid
		GROUP BY s.operation_leader
		ORDER BY order_count DESC
	`

	groupResult, e := g.DB().GetAll(ctx, groupSQL)
	if e != nil {
		err = fmt.Errorf("小组排名查询失败: %v", e)
		return
	}

	groupRank = make([]map[string]interface{}, 0, len(groupResult))
	for i, row := range groupResult {
		pay := row["pay_amount"].Float64()
		cost := row["cost_amount"].Float64()
		reserve := row["reserve_amount"].Float64()
		groupRank = append(groupRank, map[string]interface{}{
			"rank":       i + 1,
			"groupName":  row["group_name"],
			"orderCount": row["order_count"].Int(),
			"goodsCount": row["goods_count"].Int(),
			"payAmount":  roundFloat(pay, 2),
			"commission": roundFloat(pay-reserve-cost, 2),
		})
	}

	// 店铺排名 - 先按主订单号聚合，避免重复计算
	// 注意：使用 main_order_no 统计订单数量，一个主订单只算一次
	shopSQL := `
		SELECT 
			o.shop_name AS shop_name,
			COALESCE(s.operation_leader, '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS goods_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			-- 子查询：按主订单号聚合，确保每个主订单只计算一次订单级金额
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				-- 商品数量：求和（一个订单多个商品）
				SUM(o1.product_num) AS product_num,
			-- 商品总价 = Σ(商品单价 × 数量)，处理NULL值
			SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				-- 成本：按商品分别计算后求和，处理NULL值
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				-- 留底 = 商品总价 × 5%，处理NULL值
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON TRIM(o1.seller_code) = TRIM(g.goods_code) AND g.status = 1
			WHERE o1.pay_finish_time >= '` + startDate + `'
			  AND o1.pay_finish_time < '` + endDate + `'
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN shop s ON o.talent_id = s.uid
		GROUP BY o.shop_name, s.operation_leader
		ORDER BY order_count DESC
	`

	shopResult, e := g.DB().GetAll(ctx, shopSQL)
	if e != nil {
		err = fmt.Errorf("店铺排名查询失败: %v", e)
		return
	}

	shopRank = make([]map[string]interface{}, 0, len(shopResult))
	for i, row := range shopResult {
		pay := row["pay_amount"].Float64()
		cost := row["cost_amount"].Float64()
		reserve := row["reserve_amount"].Float64()
		shopRank = append(shopRank, map[string]interface{}{
			"rank":       i + 1,
			"shopName":   row["shop_name"],
			"groupName":  row["group_name"],
			"orderCount": row["order_count"].Int(),
			"goodsCount": row["goods_count"].Int(),
			"payAmount":  roundFloat(pay, 2),
			"commission": roundFloat(pay-reserve-cost, 2),
		})
	}

	return
}

// GetGroupList 获取所有小组列表
func (s *OrderAnalysisService) GetGroupList(ctx context.Context) ([]string, error) {
	sql := `
		SELECT DISTINCT COALESCE(operation_leader, '未分配小组') AS group_name
		FROM shop 
		ORDER BY group_name
	`

	result, err := g.DB().GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询小组列表失败: %v", err)
	}

	groups := make([]string, 0, len(result))
	for _, row := range result {
		if name := row["group_name"].String(); name != "" {
			groups = append(groups, name)
		}
	}

	return groups, nil
}

// GetShopList 获取所有店铺列表
func (s *OrderAnalysisService) GetShopList(ctx context.Context, groupName string) ([]map[string]interface{}, error) {
	query := "SELECT id, douyin_name AS shop_name, COALESCE(operation_leader, '未分配小组') AS group_name FROM shop"
	params := []interface{}{}

	if groupName != "" && groupName != "全部" {
		query += " WHERE operation_leader = ? OR (operation_leader IS NULL AND '未分配小组' = ?)"
		params = append(params, groupName, groupName)
	}

	query += " ORDER BY douyin_name"

	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("查询店铺列表失败: %v", err)
	}

	shops := make([]map[string]interface{}, 0, len(result))
	for _, row := range result {
		shops = append(shops, map[string]interface{}{
			"shopId":    row["id"].Int(),
			"shopName":  row["shop_name"],
			"groupName": row["group_name"],
		})
	}

	return shops, nil
}

// FormatGroupName 格式化小组名称
func FormatGroupName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "未分配小组"
	}
	return name
}
