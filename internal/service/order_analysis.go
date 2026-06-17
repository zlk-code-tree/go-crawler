package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
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
func (s *OrderAnalysisService) GroupStats(ctx context.Context, startDate, endDate string) ([]model.GroupStatItem, error) {
	// 构建统计SQL - 先按 main_order_no 聚合，避免 JOIN 导致重复计算
	sql := `
		SELECT
			COALESCE(a.operator, '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				SUM(o1.product_num) AS product_num,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			WHERE o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN account a ON o.talent_id = a.uid
		GROUP BY a.operator
		ORDER BY a.operator
	`

	result, err := g.DB().GetAll(ctx, sql, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	// 转换结果为结构体
	stats := make([]model.GroupStatItem, 0, len(result))
	for _, row := range result {
		productTotal := row["pay_amount"].Float64()
		costAmount := row["cost_amount"].Float64()
		reserveAmount := row["reserve_amount"].Float64()

		stats = append(stats, model.GroupStatItem{
			GroupName:     row["group_name"].String(),
			OrderCount:    row["order_count"].Int(),
			ProductNum:    row["product_num"].Int(),
			PayAmount:     roundFloat(productTotal, 2),
			CostAmount:    roundFloat(costAmount, 2),
			ReserveAmount: roundFloat(reserveAmount, 2),
			Commission:    roundFloat(productTotal-reserveAmount-costAmount, 2),
		})
	}

	return stats, nil
}

// ShopStats 账号维度统计
func (s *OrderAnalysisService) ShopStats(ctx context.Context, startDate, endDate string, groupId int) ([]model.ShopStatItem, error) {
	// 构建账号筛选条件
	shopFilter := ""
	params := []interface{}{startDate, endDate}
	if groupId > 0 {
		shopFilter = " AND a.id = ?"
		params = append(params, groupId)
	}

	// 构建统计SQL
	sql := `
		SELECT
			a.id AS account_id,
			o.shop_name AS shop_name,
			COALESCE(a.operator, '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				SUM(o1.product_num) AS product_num,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			WHERE o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN account a ON o.talent_id = a.uid
		WHERE 1=1 ` + shopFilter + `
		GROUP BY a.id, o.shop_name, a.operator
		ORDER BY o.shop_name
	`

	result, err := g.DB().GetAll(ctx, sql, params...)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	// 转换结果为结构体
	stats := make([]model.ShopStatItem, 0, len(result))
	for _, row := range result {
		productTotal := row["pay_amount"].Float64()
		costAmount := row["cost_amount"].Float64()
		reserveAmount := row["reserve_amount"].Float64()

		stats = append(stats, model.ShopStatItem{
			AccountId:        row["account_id"].Int(),
			ShopName:      row["shop_name"].String(),
			GroupName:     row["group_name"].String(),
			OrderCount:    row["order_count"].Int(),
			ProductNum:    row["product_num"].Int(),
			PayAmount:     roundFloat(productTotal, 2),
			CostAmount:    roundFloat(costAmount, 2),
			ReserveAmount: roundFloat(reserveAmount, 2),
			Commission:    roundFloat(productTotal-reserveAmount-costAmount, 2),
		})
	}

	return stats, nil
}

// GetAllGroups 获取所有小组列表（用于下拉筛选）
func (s *OrderAnalysisService) GetAllGroups(ctx context.Context) ([]map[string]interface{}, error) {
	sql := `
		SELECT DISTINCT 
			COALESCE(operator, '未分配小组') AS group_name
		FROM account 
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

// GetAllShops 获取所有账号列表（用于下拉筛选）
func (s *OrderAnalysisService) GetAllShops(ctx context.Context) ([]map[string]interface{}, error) {
	sql := `
		SELECT id, account_name, COALESCE(operator, '未分配小组') AS group_name
		FROM account 
		ORDER BY account_name
	`

	result, err := g.DB().GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询账号列表失败: %v", err)
	}

	shops := make([]map[string]interface{}, 0, len(result))
	for _, row := range result {
		shops = append(shops, map[string]interface{}{
			"accountId":    row["id"].Int(),
			"shopName":  row["account_name"],
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
func (s *OrderAnalysisService) buildBaseQuery(startDate, endDate, extraWhere string) string {
	validFilter := s.GetValidOrderFilter()
	return `
		SELECT
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			WHERE o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + validFilter + `
			GROUP BY o1.main_order_no
		) o
		WHERE 1=1 ` + extraWhere + `
	`
}

// Overview 统计概览：期间统计 + 历史累计
func (s *OrderAnalysisService) Overview(ctx context.Context, startDate, endDate string) (period model.OverviewPeriod, total model.OverviewTotal, err error) {
	// 1. 期间统计
	periodSQL := s.buildBaseQuery(startDate, endDate, "")
	periodResult, e := g.DB().GetAll(ctx, periodSQL, startDate, endDate)
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

	// 2. 退款统计
	refundSQL := `
		SELECT
			COUNT(DISTINCT o.main_order_no) AS refund_count,
			COALESCE(SUM(o.refund_amount), 0) AS refund_amount
		FROM (
			SELECT
				o1.main_order_no,
				MAX(o1.order_pay_amount + o1.platform_discount + o1.talent_discount) AS refund_amount
			FROM order_info o1
			WHERE o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  AND o1.after_sale_status IN ('退款成功', '已全额退款')
			GROUP BY o1.main_order_no
		) o
	`
	refundResult, e := g.DB().GetAll(ctx, refundSQL, startDate, endDate)
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

	period = model.OverviewPeriod{
		OrderCount:   orderCount,
		PayAmount:    roundFloat(payAmount, 2),
		CostAmount:   roundFloat(costAmount, 2),
		Commission:   roundFloat(payAmount-reserveAmount-costAmount, 2),
		RefundCount:  refundCount,
		RefundAmount: roundFloat(refundAmount, 2),
	}

	// 3. 历史累计（不限制时间）
	totalSQL := s.buildBaseQuery("2020-01-01", time.Now().Add(24*time.Hour).Format("2006-01-02"), "")
	totalResult, e := g.DB().GetAll(ctx, totalSQL, "2020-01-01", time.Now().Add(24*time.Hour).Format("2006-01-02"))
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

	total = model.OverviewTotal{
		OrderCount: totalOrders,
		PayAmount:  roundFloat(totalPay, 2),
		Commission: roundFloat(totalPay-totalReserve-totalCost, 2),
	}

	return
}

// SalesTrend 销售额趋势：当前期 + 上月同期对比
func (s *OrderAnalysisService) SalesTrend(ctx context.Context, startDate, endDate string) (current, compare []model.DailyTrend, err error) {
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
func (s *OrderAnalysisService) queryDailyStats(ctx context.Context, startDate, endDate string, days int) ([]model.DailyTrend, error) {
	sql := `
		SELECT
			DATE(o.pay_finish_time) AS stat_date,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				o1.pay_finish_time,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			WHERE o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.pay_finish_time
		) o
		GROUP BY DATE(o.pay_finish_time)
		ORDER BY stat_date
	`

	result, err := g.DB().GetAll(ctx, sql, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("日统计查询失败: %v", err)
	}

	// 构建日期 map
	dataMap := make(map[string]model.DailyTrend, len(result))
	for _, row := range result {
		dateStr := row["stat_date"].String()
		pay := row["pay_amount"].Float64()
		cost := row["cost_amount"].Float64()
		reserve := row["reserve_amount"].Float64()
		dataMap[dateStr] = model.DailyTrend{
			Date:       dateStr,
			OrderCount: row["order_count"].Int(),
			PayAmount:  roundFloat(pay, 2),
			Commission: roundFloat(pay-reserve-cost, 2),
		}
	}

	// 填充空白天为0
	start, _ := time.Parse("2006-01-02", startDate)
	rows := make([]model.DailyTrend, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		if item, ok := dataMap[d]; ok {
			rows = append(rows, item)
		} else {
			rows = append(rows, model.DailyTrend{
				Date:       d,
				OrderCount: 0,
				PayAmount:  0.00,
				Commission: 0.00,
			})
		}
	}

	return rows, nil
}

// Rank 排名榜单：小组排名 + 账号排名
func (s *OrderAnalysisService) Rank(ctx context.Context, startDate, endDate string) (groupRank, shopRank []model.RankItem, err error) {
	// 小组排名
	groupSQL := `
		SELECT
			COALESCE(a.operator, '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS goods_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				SUM(o1.product_num) AS product_num,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			WHERE o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN account a ON o.talent_id = a.uid
		WHERE a.operator IS NOT NULL AND a.operator != ''
		GROUP BY a.operator
		ORDER BY order_count DESC
	`

	groupResult, e := g.DB().GetAll(ctx, groupSQL, startDate, endDate)
	if e != nil {
		err = fmt.Errorf("小组排名查询失败: %v", e)
		return
	}

	groupRank = make([]model.RankItem, 0, len(groupResult))
	for i, row := range groupResult {
		pay := row["pay_amount"].Float64()
		cost := row["cost_amount"].Float64()
		reserve := row["reserve_amount"].Float64()
		groupRank = append(groupRank, model.RankItem{
			Rank:       i + 1,
			Name:       row["group_name"].String(),
			OrderCount: row["order_count"].Int(),
			GoodsCount: row["goods_count"].Int(),
			PayAmount:  roundFloat(pay, 2),
			Commission: roundFloat(pay-reserve-cost, 2),
		})
	}

	// 账号排名 + 达人明细（一次查询，在内存中聚合）
	// 按 shop_name + talent_id 分组，得到每个账号内每位达人的统计数据
	shopTalentSQL := `
		SELECT
			o.shop_name,
			o.talent_id,
			COALESCE(MAX(a.account_name), '未知达人') AS talent_name,
			COALESCE(GROUP_CONCAT(DISTINCT a.operator ORDER BY a.operator SEPARATOR ', '), '未分配小组') AS group_name,
			COALESCE(MAX(a.uid), 0) AS account_id,
			COUNT(DISTINCT o.main_order_no) AS talent_order_count,
			COALESCE(SUM(o.product_num), 0) AS talent_goods_count,
			COALESCE(SUM(o.pay_amount), 0) AS talent_pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS talent_cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS talent_reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				o1.shop_name,
				o1.talent_id,
				SUM(o1.product_num) AS product_num,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0)) AS pay_amount,
				SUM(COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0)) AS cost_amount,
				SUM(COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05) AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			WHERE o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  ` + s.GetValidOrderFilter() + `
			GROUP BY o1.main_order_no, o1.shop_name, o1.talent_id
		) o
		LEFT JOIN account a ON o.talent_id = a.uid
		WHERE a.operator IS NOT NULL AND a.operator != ''
		GROUP BY o.shop_name, o.talent_id
		ORDER BY o.shop_name, talent_order_count DESC
	`

	shopTalentResult, e := g.DB().GetAll(ctx, shopTalentSQL, startDate, endDate)
	if e != nil {
		err = fmt.Errorf("账号排名查询失败: %v", e)
		return
	}

	// 在内存中聚合：shopName -> { totals, talents[] }
	type shopAggregate struct {
		accountId     int
		groupName  string
		orderCount int
		goodsCount int
		payAmount  float64
		costAmount float64
		reserveAmt float64
		talents    []model.ShopTalentStatItem
	}
	shopMap := make(map[string]*shopAggregate)
	shopOrder := make([]string, 0) // 保持首次出现顺序

	for _, row := range shopTalentResult {
		shopName := row["shop_name"].String()
		groupName := row["group_name"].String()
		accountId := row["account_id"].Int()

		talentPay := row["talent_pay_amount"].Float64()
		talentCost := row["talent_cost_amount"].Float64()
		talentReserve := row["talent_reserve_amount"].Float64()
		talentCommission := roundFloat(talentPay-talentReserve-talentCost, 2)

		talentItem := model.ShopTalentStatItem{
			TalentId:   row["talent_id"].Int(),
			TalentName: row["talent_name"].String(),
			GroupName:  row["group_name"].String(),
			OrderCount: row["talent_order_count"].Int(),
			GoodsCount: row["talent_goods_count"].Int(),
			PayAmount:  roundFloat(talentPay, 2),
			Commission: talentCommission,
		}

		if agg, ok := shopMap[shopName]; ok {
			// 累加店铺总计
			agg.orderCount += talentItem.OrderCount
			agg.goodsCount += talentItem.GoodsCount
			agg.payAmount += talentPay
			agg.costAmount += talentCost
			agg.reserveAmt += talentReserve
			agg.talents = append(agg.talents, talentItem)
		} else {
			shopMap[shopName] = &shopAggregate{
				accountId:     accountId,
				groupName:  groupName,
				orderCount: talentItem.OrderCount,
				goodsCount: talentItem.GoodsCount,
				payAmount:  talentPay,
				costAmount: talentCost,
				reserveAmt: talentReserve,
				talents:    []model.ShopTalentStatItem{talentItem},
			}
			shopOrder = append(shopOrder, shopName)
		}
	}

	// 按订单数降序排序
	sort.SliceStable(shopOrder, func(i, j int) bool {
		return shopMap[shopOrder[i]].orderCount > shopMap[shopOrder[j]].orderCount
	})

	shopRank = make([]model.RankItem, 0, len(shopOrder))
	for i, name := range shopOrder {
		agg := shopMap[name]
		shopRank = append(shopRank, model.RankItem{
			Rank:       i + 1,
			Name:       name,
			GroupName:  agg.groupName,
			AccountId:     agg.accountId,
			OrderCount: agg.orderCount,
			GoodsCount: agg.goodsCount,
			PayAmount:  roundFloat(agg.payAmount, 2),
			Commission: roundFloat(agg.payAmount-agg.reserveAmt-agg.costAmount, 2),
			Details:    agg.talents,
		})
	}

	return
}

// GetGroupList 获取所有小组列表
func (s *OrderAnalysisService) GetGroupList(ctx context.Context) ([]string, error) {
	sql := `
		SELECT DISTINCT COALESCE(operator, '未分配小组') AS group_name
		FROM account 
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

// GetShopList 获取所有账号列表
func (s *OrderAnalysisService) GetShopList(ctx context.Context, groupName string) ([]model.ShopInfo, error) {
	query := "SELECT id, account_name AS shop_name, COALESCE(operator, '未分配小组') AS group_name FROM shop"
	params := []interface{}{}

	if groupName != "" && groupName != "全部" {
		query += " WHERE operator = ? OR (operator IS NULL AND '未分配小组' = ?)"
		params = append(params, groupName, groupName)
	}

	query += " ORDER BY account_name"

	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("查询账号列表失败: %v", err)
	}

	shops := make([]model.ShopInfo, 0, len(result))
	for _, row := range result {
		shops = append(shops, model.ShopInfo{
			AccountId:    row["id"].Int(),
			ShopName:  row["shop_name"].String(),
			GroupName: row["group_name"].String(),
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

// queryOrderDetailsByFilter 根据过滤条件查询订单明细
func (s *OrderAnalysisService) queryOrderDetailsByFilter(ctx context.Context, filterType, filterValue, startDate, endDate string) (gdb.Result, error) {
	var sql string
	var params []interface{}

	// 处理 endDate：如果是纯日期，转换为当天的最后一刻
	endTime := endDate
	if len(endDate) == 10 {
		// 纯日期格式 (YYYY-MM-DD)，追加时间到当天最后一刻
		endTime = endDate + " 23:59:59"
	}

	if filterType == "group" {
		sql = `
			SELECT o.*
			FROM order_info o
			LEFT JOIN account a ON o.talent_id = a.uid
			WHERE a.operator = ?
			  AND o.pay_finish_time >= ?
			  AND o.pay_finish_time <= ?
			  AND o.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货')
			  AND o.order_status != '已关闭'
			  AND o.main_order_no IS NOT NULL
			  AND o.main_order_no != ''
			ORDER BY o.pay_finish_time DESC
		`
		params = []interface{}{filterValue, startDate, endTime}
	} else if filterType == "shop" {
		sql = `
			SELECT o.*
			FROM order_info o
			WHERE o.shop_name = ?
			  AND o.pay_finish_time >= ?
			  AND o.pay_finish_time <= ?
			  AND o.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货')
			  AND o.order_status != '已关闭'
			  AND o.main_order_no IS NOT NULL
			  AND o.main_order_no != ''
			ORDER BY o.pay_finish_time DESC
		`
		params = []interface{}{filterValue, startDate, endTime}
	} else {
		return nil, gerror.New("不支持的过滤类型")
	}

	result, err := g.DB().GetAll(ctx, sql, params...)
	if err != nil {
		return nil, fmt.Errorf("查询订单明细失败: %v", err)
	}

	return result, nil
}

// ExportGroupDetails 导出小组订单明细
func (s *OrderAnalysisService) ExportGroupDetails(ctx context.Context, groupName, timeType, startDate, endDate string) (string, error) {
	return s.ExportOrderDetails(ctx, "group", groupName, timeType, startDate, endDate)
}

// GetShopTalentStats 获取账号下的达人订单统计明细
func (s *OrderAnalysisService) GetShopTalentStats(ctx context.Context, shopName, startDate, endDate string) ([]model.ShopTalentStatItem, error) {
		sql := `
		SELECT
			o.talent_id,
			COALESCE(MAX(a.account_name), '未知达人') AS talent_name,
			COALESCE(MAX(a.operator), '未分配小组') AS group_name,
			COUNT(DISTINCT o.main_order_no) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS goods_count,
			COALESCE(SUM(o.pay_amount), 0) AS pay_amount,
			COALESCE(SUM(o.cost_amount), 0) AS cost_amount,
			COALESCE(SUM(o.reserve_amount), 0) AS reserve_amount
		FROM (
			SELECT
				o1.main_order_no,
				o1.talent_id,
				o1.product_num,
				COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) AS pay_amount,
				COALESCE(g.base_price, 0) * COALESCE(o1.product_num, 0) AS cost_amount,
				COALESCE(o1.product_price, 0) * COALESCE(o1.product_num, 0) * 0.05 AS reserve_amount
			FROM order_info o1
			LEFT JOIN goods g ON o1.seller_code = g.goods_code AND g.status = 1
			WHERE o1.shop_name = ?
			  AND o1.pay_finish_time >= ?
			  AND o1.pay_finish_time < ?
			  AND o1.main_order_no IS NOT NULL
			  AND o1.main_order_no != ''
			  AND o1.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货')
			  AND o1.order_status != '已关闭'
		) o
		LEFT JOIN account a ON o.talent_id = a.uid
		GROUP BY o.talent_id
		ORDER BY order_count DESC
	`
	result, err := g.DB().GetAll(ctx, sql, shopName, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("查询账号达人明细失败: %v", err)
	}

	stats := make([]model.ShopTalentStatItem, 0, len(result))
	for _, row := range result {
		pay := row["pay_amount"].Float64()
		cost := row["cost_amount"].Float64()
		reserve := row["reserve_amount"].Float64()
		stats = append(stats, model.ShopTalentStatItem{
			TalentId:   row["talent_id"].Int(),
			TalentName: row["talent_name"].String(),
			GroupName:  row["group_name"].String(),
			OrderCount: row["order_count"].Int(),
			GoodsCount: row["goods_count"].Int(),
			PayAmount:  roundFloat(pay, 2),
			Commission: roundFloat(pay-reserve-cost, 2),
		})
	}
	return stats, nil
}

// ExportShopDetails 导出账号订单明细（accountId为达人UID，直接用于匹配订单表的talent_id）
func (s *OrderAnalysisService) ExportShopDetails(ctx context.Context, accountId int, timeType, startDate, endDate string) (string, error) {
	return s.ExportOrderDetails(ctx, "shop", strconv.Itoa(accountId), timeType, startDate, endDate)
}

// getTimeLabel 获取时间维度标签
func getTimeLabel(timeType string) string {
	switch timeType {
	case "today":
		return "今日"
	case "yesterday":
		return "昨日"
	case "last7":
		return "近7日"
	case "last30":
		return "近30日"
	case "thisMonth":
		return "本月"
	case "all":
		return "全部"
	default:
		return "未知"
	}
}

// normalizeTimeType 标准化时间类型参数
// 兼容新旧两种时间类型命名，确保前端传入的 last7/last30/thisMonth/all 等值能被正确识别
func normalizeTimeType(timeType string) string {
	switch timeType {
	case "today":
		return "today"
	case "yesterday":
		return "yesterday"
	case "last7", "7days", "week":
		return "last7"
	case "last30", "30days":
		return "last30"
	case "thisMonth", "month":
		return "thisMonth"
	case "all":
		return "all"
	default:
		// 如果无法识别的类型，则原样返回（让 ParseTimeRange 处理）
		return timeType
	}
}

// ExportOrderDetails 导出订单详情（支持小组和账号维度）
func (s *OrderAnalysisService) ExportOrderDetails(ctx context.Context, exportType, key, timeType, startDate, endDate string) (string, error) {
	// 标准化时间类型
	normalizedTimeType := normalizeTimeType(timeType)
	startDate, endDate = s.ParseTimeRange(normalizedTimeType, startDate, endDate)
	
	var result gdb.Result
	var err error
	var targetName string
	
	if exportType == "group" {
		// 使用统一的查询方法
		result, err = s.queryOrderDetailsByFilter(ctx, "group", key, startDate, endDate)
		targetName = key
	} else if exportType == "shop" {
		// key 可能是 accountId（达人UID），先查出对应账号名，再按账号名导出所有订单
		shopInfo, _ := g.DB().Model("account").Fields("account_name").Where("uid = ?", key).One()
		if !shopInfo.IsEmpty() {
			shopName := shopInfo["account_name"].String()
			result, err = s.queryOrderDetailsByFilter(ctx, "shop", shopName, startDate, endDate)
			targetName = shopName
		} else {
			// 直接作为账号名查询
			result, err = s.queryOrderDetailsByFilter(ctx, "shop", key, startDate, endDate)
			targetName = key
		}
	} else {
		return "", gerror.New("不支持的导出类型")
	}
	
	if err != nil {
		return "", fmt.Errorf("查询订单详情失败: %v", err)
	}
	
	if len(result) == 0 {
		return "", gerror.New("没有数据可导出")
	}
	
	// 使用中文表头（与数据库字段顺序对应）
	headers := []string{
		"店铺", "主订单编号", "子订单编号", "选购商品", "商品规格", "商品数量", "商品ID", "商家编码",
		"商品单价", "订单应付金额", "运费", "优惠总金额", "平台优惠", "商家优惠", "达人优惠",
		"商家改价", "支付优惠", "红包抵扣", "支付方式", "手续费", "收件人", "收件人手机号",
		"省", "市", "区", "街道", "详细地址", "是否修改过地址", "地址修改时段", "买家留言",
		"订单提交时间", "旗帜颜色", "商家备注", "订单完成时间", "支付完成时间", "APP渠道",
		"流量来源", "订单状态", "承诺发货时间", "订单类型", "鲁班落地页ID", "达人ID", "达人昵称",
		"所属门店ID", "售后状态", "取消原因", "预约发货时间", "仓库ID", "仓库名称", "是否安心购",
		"广告渠道", "流量类型", "流量体裁", "流量渠道", "发货主体", "发货主体明细", "发货时间",
		"降价类优惠", "平台实际承担优惠金额", "商家实际承担优惠金额", "达人实际承担优惠金额", "预计送达时间",
	}
	
	// 字段映射关系（数据库字段名 -> 表头索引）
	fieldMapping := map[string]int{
		"shop_name":                0,  // 店铺
		"main_order_no":            1,  // 主订单编号
		"sub_order_no":             2,  // 子订单编号
		"product_name":             3,  // 选购商品
		"product_spec":             4,  // 商品规格
		"product_num":              5,  // 商品数量
		"product_id":               6,  // 商品ID
		"seller_code":              7,  // 商家编码
		"product_price":            8,  // 商品单价
		"order_pay_amount":         9,  // 订单应付金额
		"shipping_fee":             10, // 运费
		"total_discount":           11, // 优惠总金额
		"platform_discount":        12, // 平台优惠
		"seller_discount":          13, // 商家优惠
		"talent_discount":          14, // 达人优惠
		"seller_change_price":      15, // 商家改价
		"pay_discount":             16, // 支付优惠
		"red_deduction":            17, // 红包抵扣
		"pay_type":                 18, // 支付方式
		"service_fee":              19, // 手续费
		"receiver_name":            20, // 收件人
		"receiver_phone":           21, // 收件人手机号
		"province":                 22, // 省
		"city":                     23, // 市
		"district":                 24, // 区
		"street":                   25, // 街道
		"address_detail":           26, // 详细地址
		"is_address_changed":       27, // 是否修改过地址
		"address_change_time":      28, // 地址修改时段
		"buyer_note":               29, // 买家留言
		"order_submit_time":        30, // 订单提交时间
		"flag_color":               31, // 旗帜颜色
		"seller_note":              32, // 商家备注
		"order_finish_time":        33, // 订单完成时间
		"pay_finish_time":          34, // 支付完成时间
		"app_channel":              35, // APP渠道
		"flow_source":              36, // 流量来源
		"order_status":             37, // 订单状态
		"promise_send_time":        38, // 承诺发货时间
		"order_type":               39, // 订单类型
		"luban_page_id":            40, // 鲁班落地页ID
		"talent_id":                41, // 达人ID
		"talent_name":              42, // 达人昵称
		"store_id":                 43, // 所属门店ID
		"after_sale_status":        44, // 售后状态
		"cancel_reason":            45, // 取消原因
		"appoint_send_time":        46, // 预约发货时间
		"warehouse_id":             47, // 仓库ID
		"warehouse_name":           48, // 仓库名称
		"is_safe_buy":              49, // 是否安心购
		"ad_channel":               50, // 广告渠道
		"flow_type":                51, // 流量类型
		"flow_format":              52, // 流量体裁
		"flow_channel":             53, // 流量渠道
		"send_user":                54, // 发货主体
		"send_user_detail":         55, // 发货主体明细
		"send_time":                56, // 发货时间
		"reduce_discount":          57, // 降价类优惠
		"platform_real_discount":   58, // 平台实际承担优惠金额
		"seller_real_discount":     59, // 商家实际承担优惠金额
		"talent_real_discount":     60, // 达人实际承担优惠金额
		"expect_arrive_time":       61, // 预计送达时间
	}
	
	// 列宽配置
	colWidths := map[string]float64{
		"A": 15, "B": 20, "C": 20, "D": 30, "E": 15, "F": 12, "G": 15, "H": 15,
		"I": 12, "J": 15, "K": 12, "L": 15, "M": 12, "N": 12, "O": 12, "P": 12,
		"Q": 12, "R": 12, "S": 12, "T": 12, "U": 15, "V": 15, "W": 10, "X": 10,
		"Y": 10, "Z": 12, "AA": 15, "AB": 15, "AC": 15, "AD": 20, "AE": 18,
		"AF": 10, "AG": 20, "AH": 18, "AI": 18, "AJ": 12, "AK": 12, "AL": 12,
		"AM": 18, "AN": 12, "AO": 15, "AP": 15, "AQ": 15, "AR": 15, "AS": 12,
		"AT": 15, "AU": 18, "AV": 15, "AW": 15, "AX": 10, "AY": 12, "AZ": 12,
		"BA": 12, "BB": 12, "BC": 12, "BD": 15, "BE": 18, "BF": 12, "BG": 12,
		"BH": 12, "BI": 12, "BJ": 18,
	}
	
	// 生成文件名
	timeStr := time.Now().Format("200601021504")
	timeLabel := getTimeLabel(normalizedTimeType)
	var fileName string
	// 清理目标名称中的非法字符（如 / 等路径分隔符），避免被解析为目录
	safeTargetName := sanitizeFileName(targetName)
	if exportType == "group" {
		fileName = fmt.Sprintf("小组订单详情_%s_%s_%s.xlsx", safeTargetName, timeLabel, timeStr)
	} else {
		fileName = fmt.Sprintf("账号订单详情_%s_%s_%s.xlsx", safeTargetName, timeLabel, timeStr)
	}
	
	// 生成Excel文件
	return generateExcelFile(result, headers, fieldMapping, "订单详情", fileName, colWidths)
}





// generateExcelFile 通用Excel文件生成函数
func generateExcelFile(result gdb.Result, headers []string, fieldMapping map[string]int, sheetName, fileName string, colWidths map[string]float64) (string, error) {
	if len(result) == 0 {
		return "", gerror.New("没有数据可导出")
	}
	
	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()
	
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)
	
	// 写入表头
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}
	
	// 设置表头样式
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, style)
	}
	
	// 写入数据
	if fieldMapping != nil {
		// 使用字段映射
		for rowIdx, row := range result {
			rowNum := rowIdx + 2
			// 初始化所有单元格为空
			for colIdx := 0; colIdx < len(headers); colIdx++ {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
				f.SetCellValue(sheetName, cell, "")
			}
			
			// 根据字段映射填充数据
			for dbField, colIdx := range fieldMapping {
				if value, ok := row[dbField]; ok {
					cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
					// 特殊处理时间字段
					if strings.HasSuffix(dbField, "_time") || dbField == "order_submit_time" || dbField == "pay_finish_time" || 
					   dbField == "order_finish_time" || dbField == "promise_send_time" || dbField == "appoint_send_time" ||
					   dbField == "send_time" || dbField == "expect_arrive_time" {
						f.SetCellValue(sheetName, cell, formatTime(value))
					} else {
						f.SetCellValue(sheetName, cell, value)
					}
				}
			}
		}
	} else {
		// 直接使用数据库字段名（按表头顺序）
		for rowIdx, data := range result {
			rowNum := rowIdx + 2
			for colIdx, col := range headers {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
				value := data[col]
				f.SetCellValue(sheetName, cell, value)
			}
		}
	}
	
	// 设置列宽
	if colWidths != nil {
		for col, width := range colWidths {
			f.SetColWidth(sheetName, col, col, width)
		}
	} else {
		// 默认列宽
		for i := 1; i <= len(headers); i++ {
			colName, _ := excelize.ColumnNumberToName(i)
			f.SetColWidth(sheetName, colName, colName, 15)
		}
	}
	
	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filePath := filepath.Join(exportDir, fileName)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}
	
	return filePath, nil
}

// sanitizeFileName 清理文件名，移除非法字符
func sanitizeFileName(name string) string {
	// 移除文件名中的非法字符
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "[", "]", "【", "】"}
	result := name
	for _, ch := range invalidChars {
		result = strings.ReplaceAll(result, ch, "")
	}
	// 限制长度
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

// exportRankToExcel 导出排名数据到Excel
func exportRankToExcel(rankItems []model.RankItem, rankType, timeLabel string) (string, error) {
	if len(rankItems) == 0 {
		return "", gerror.New("没有排名数据可导出")
	}
	
	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()
	
	sheetName := rankType + "排名"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)
	
	// 定义表头
	headers := []string{"排名", "名称", "所属小组", "订单数", "商品数", "销售额", "佣金"}
	
	// 写入表头
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}
	
	// 设置表头样式
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, style)
	}
	
	// 写入数据
	for rowIdx, item := range rankItems {
		rowNum := rowIdx + 2
		// 排名
		cell, _ := excelize.CoordinatesToCellName(1, rowNum)
		f.SetCellValue(sheetName, cell, item.Rank)
		// 名称
		cell, _ = excelize.CoordinatesToCellName(2, rowNum)
		f.SetCellValue(sheetName, cell, item.Name)
		// 所属小组
		cell, _ = excelize.CoordinatesToCellName(3, rowNum)
		f.SetCellValue(sheetName, cell, item.GroupName)
		// 订单数
		cell, _ = excelize.CoordinatesToCellName(4, rowNum)
		f.SetCellValue(sheetName, cell, item.OrderCount)
		// 商品数
		cell, _ = excelize.CoordinatesToCellName(5, rowNum)
		f.SetCellValue(sheetName, cell, item.GoodsCount)
		// 销售额
		cell, _ = excelize.CoordinatesToCellName(6, rowNum)
		f.SetCellValue(sheetName, cell, item.PayAmount)
		// 佣金
		cell, _ = excelize.CoordinatesToCellName(7, rowNum)
		f.SetCellValue(sheetName, cell, item.Commission)
	}
	
	// 设置列宽
	colWidths := map[string]float64{
		"A": 8,  // 排名
		"B": 20, // 名称
		"C": 15, // 所属小组
		"D": 10, // 订单数
		"E": 10, // 商品数
		"F": 15, // 销售额
		"G": 15, // 佣金
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetName, col, col, width)
	}
	
	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	timeStr := time.Now().Format("200601021504")
	fileName := fmt.Sprintf("%s排名_%s_%s.xlsx", rankType, timeLabel, timeStr)
	filePath := filepath.Join(exportDir, fileName)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}
	
	return filePath, nil
}

// ExportGroupRank 导出小组排名
func (s *OrderAnalysisService) ExportGroupRank(ctx context.Context, timeType string) (string, error) {
	startDate, endDate := s.ParseTimeRange(timeType, "", "")
	groupRank, _, err := s.Rank(ctx, startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("获取小组排名失败: %v", err)
	}
	
	timeLabel := getTimeLabel(timeType)
	return exportRankToExcel(groupRank, "小组", timeLabel)
}

// ExportShopRank 导出账号排名
func (s *OrderAnalysisService) ExportShopRank(ctx context.Context, timeType string) (string, error) {
	startDate, endDate := s.ParseTimeRange(timeType, "", "")
	_, shopRank, err := s.Rank(ctx, startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("获取账号排名失败: %v", err)
	}

	timeLabel := getTimeLabel(timeType)
	return exportRankToExcel(shopRank, "店铺", timeLabel)
}
