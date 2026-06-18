package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

type OrderService struct{}

func NewOrderService() *OrderService {
	return &OrderService{}
}

// OrderStats 订单统计数据
type OrderStats struct {
	TotalOrderCount      int64   `json:"totalOrderCount"`      // 总订单 = 有效 + 无效 + 未命中
	ValidOrderCount      int64   `json:"validOrderCount"`      // 有效订单
	InvalidOrderCount    int64   `json:"invalidOrderCount"`    // 无效订单
	UnmatchedOrderCount  int64   `json:"unmatchedOrderCount"`  // 未命中有效/无效规则的订单
	ValidPayAmount       float64 `json:"validPayAmount"`       // 有效定价总额 = product_price × num
	ProductCostAmount    float64 `json:"productCostAmount"`    // 货款 = base_price × num
	ReserveAmount        float64 `json:"reserveAmount"`        // 留底 = product_price × num × 3%
	RefundCostAmount     float64 `json:"refundCostAmount"`     // 发货后退款成本
	ShopServiceFee       float64 `json:"shopServiceFee"`       // 小店服务费 = 总商品金额 × 2%
	AdCostAmount         float64 `json:"adCostAmount"`         // 广告费消耗（千川）
	OnlineCommission     float64 `json:"onlineCommission"`     // 线上佣金支出
	RealtimeProfit       float64 `json:"realtimeProfit"`       // 实时利润
	ValidOrderCostMatch  int64   `json:"validOrderCostMatch"`  // 有效订单已匹配数量
	RefundOrderCount     int64   `json:"refundOrderCount"`     // 退款订单数量
	CommissionOrderCount int64   `json:"commissionOrderCount"` // 有佣金的订单数量
	SyncDate             string  `json:"syncDate"`             // 最近同步日期
}

// Stats 查询订单统计
func (s *OrderService) Stats(ctx context.Context, shopName string, startTime, endTime string) (*OrderStats, error) {
	stats := &OrderStats{
		SyncDate: time.Now().Format("2006-01-02"),
	}

	// 纯历史数据 → 优先从预聚合表读取
	today := time.Now().Format("2006-01-02")
	if endTime != "" && endTime < today {
		if result := s.getOrderStatsFromTable(ctx, shopName, startTime, endTime); result != nil {
			return result, nil
		}
	}

	// 构建日期筛选条件（使用下单时间）
	dateWhere := "DATE(order_submit_time) = CURDATE()"
	dateParams := []interface{}{}
	if startTime != "" {
		dateWhere = "order_submit_time >= ?"
		dateParams = append(dateParams, startTime)
	}
	if endTime != "" {
		if startTime != "" {
			dateWhere += " AND order_submit_time <= ?"
		} else {
			dateWhere = "order_submit_time <= ?"
		}
		dateParams = append(dateParams, endTime)
	}

	// 联盟订单的日期筛选（按 pay_time）
	allianceDateWhere := "DATE(pay_time) = CURDATE()"
	allianceDateParams := []interface{}{}
	if startTime != "" {
		allianceDateWhere = "pay_time >= ?"
		allianceDateParams = append(allianceDateParams, startTime)
	}
	if endTime != "" {
		if startTime != "" {
			allianceDateWhere += " AND pay_time <= ?"
		} else {
			allianceDateWhere = "pay_time <= ?"
		}
		allianceDateParams = append(allianceDateParams, endTime)
	}

	// 店铺筛选
	shopWhere := ""
	shopParams := []interface{}{}
	if shopName != "" {
		shopWhere = " AND shop_name = ?"
		shopParams = append(shopParams, shopName)
	}

	// 有效订单状态
	validStatuses := "('已发货', '待发货', '已完成')"
	// 无效订单：after_sale_status 售后申请(6)、售后退货中(7)、售后成功(12)
	invalidAfterSaleStatuses := "('6', '7', '12')"
	// 有效订单条件：
	//   1. order_status在(已发货,待发货,已完成) 且 after_sale已关闭或未发生售后
	//   2. order_status=已关闭 且 after_sale为关闭状态(拒绝27/失败28/取消51)
	validWhereClause := "((order_status IN " + validStatuses + " AND (after_sale_status NOT IN " + invalidAfterSaleStatuses + " OR after_sale_status IS NULL OR after_sale_status = '')) OR (order_status = '已关闭' AND after_sale_status IN ('27', '28', '51')))"
	validJoinClause := "((o.order_status IN " + validStatuses + " AND (o.after_sale_status NOT IN " + invalidAfterSaleStatuses + " OR o.after_sale_status IS NULL OR o.after_sale_status = '')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27', '28', '51')))"

	// 0. 全部订单总数（不限状态）
	totalQuery := "SELECT COUNT(*) as total FROM order_info WHERE " + dateWhere + shopWhere
	row, err := g.DB().GetOne(ctx, totalQuery, append(dateParams, shopParams...)...)
	if err != nil {
		return nil, err
	}
	stats.TotalOrderCount = row["total"].Int64()

	// 1. 无效订单（无效优先，先统计）
	invalidQuery := "SELECT COUNT(*) as total FROM order_info WHERE " + dateWhere + shopWhere + " AND after_sale_status IN " + invalidAfterSaleStatuses
	row, err = g.DB().GetOne(ctx, invalidQuery, append(dateParams, shopParams...)...)
	if err != nil {
		return nil, err
	}
	stats.InvalidOrderCount = row["total"].Int64()

	// 2. 有效订单 + 有效定价总额（product_price × product_num），排除已计入无效的
	validQuery := "SELECT COUNT(*) as total, COALESCE(SUM(product_price * product_num), 0) as amount FROM order_info WHERE " + dateWhere + shopWhere + " AND " + validWhereClause
	row, err = g.DB().GetOne(ctx, validQuery, append(dateParams, shopParams...)...)
	if err != nil {
		return nil, err
	}
	stats.ValidOrderCount = row["total"].Int64()
	stats.ValidPayAmount = row["amount"].Float64()

	// 3. 未命中 = 总数 - 有效 - 无效
	stats.UnmatchedOrderCount = stats.TotalOrderCount - stats.ValidOrderCount - stats.InvalidOrderCount

	// 4. 货款（商品成本支出）= base_price × product_num（有效订单，排除无效售后）
	costQuery := "SELECT COALESCE(SUM(g.base_price * o.product_num), 0) as cost FROM order_info o LEFT JOIN goods g ON o.seller_code = g.goods_code WHERE " + dateWhere + shopWhere + " AND " + validJoinClause
	row, err = g.DB().GetOne(ctx, costQuery, append(dateParams, shopParams...)...)
	if err != nil {
		return nil, err
	}
	stats.ProductCostAmount = row["cost"].Float64()

	// 5. 总商品金额 = SUM(product_price × product_num)（有效订单，排除无效售后）
	amountQuery := "SELECT COALESCE(SUM(product_price * product_num), 0) as total FROM order_info WHERE " + dateWhere + shopWhere + " AND " + validWhereClause
	row, err = g.DB().GetOne(ctx, amountQuery, append(dateParams, shopParams...)...)
	if err != nil {
		return nil, err
	}
	totalProductAmount := row["total"].Float64()

	// 6. 留底 = product_price × product_num × 3%（有效订单，排除无效售后）
	reserveQuery := "SELECT COALESCE(SUM(product_price * product_num * 0.03), 0) as reserve FROM order_info WHERE " + dateWhere + shopWhere + " AND " + validWhereClause
	row, err = g.DB().GetOne(ctx, reserveQuery, append(dateParams, shopParams...)...)
	if err != nil {
		return nil, err
	}
	stats.ReserveAmount = row["reserve"].Float64()

	// 7. 线上佣金 = 总商品金额 - 留底 - 货款（显示用）
	stats.OnlineCommission = totalProductAmount - stats.ReserveAmount - stats.ProductCostAmount
	if stats.OnlineCommission < 0 { stats.OnlineCommission = 0 }

	// 8. 小店服务费 = 总商品金额 × 2%
	stats.ShopServiceFee = totalProductAmount * 0.02

	// 9. 发货后退款成本 = 无效订单的 base_price × product_num
	refundCostQuery := "SELECT COALESCE(SUM(g.base_price * o.product_num), 0) as cost, COUNT(*) as cnt FROM order_info o LEFT JOIN goods g ON o.seller_code = g.goods_code WHERE " + dateWhere + shopWhere + " AND o.after_sale_status IN " + invalidAfterSaleStatuses
	row, err = g.DB().GetOne(ctx, refundCostQuery, append(dateParams, shopParams...)...)
	if err != nil {
		return nil, err
	}
	stats.RefundCostAmount = row["cost"].Float64()
	stats.RefundOrderCount = row["cnt"].Int64()

	// 10. 广告费：从千川报表获取
	stats.AdCostAmount = s.getAdCost(ctx, startTime, endTime)

	// 11. 有效订单成本已匹配数
	matchQuery := "SELECT COUNT(*) as total FROM order_info o INNER JOIN goods g ON o.seller_code = g.goods_code WHERE " + dateWhere + shopWhere + " AND " + validJoinClause + " AND g.base_price > 0"
	row, err = g.DB().GetOne(ctx, matchQuery, append(dateParams, shopParams...)...)
	if err == nil {
		stats.ValidOrderCostMatch = row["total"].Int64()
	}

	// 12. 有佣金的订单数（从 alliance_orders 表）
	commissionQuery := "SELECT COUNT(*) as total FROM alliance_orders WHERE " + allianceDateWhere
	if shopName != "" {
		commissionQuery += " AND shop_name = ?"
		row, err = g.DB().GetOne(ctx, commissionQuery, append(allianceDateParams, shopName)...)
	} else {
		row, err = g.DB().GetOne(ctx, commissionQuery, allianceDateParams...)
	}
	if err == nil {
		stats.CommissionOrderCount = row["total"].Int64()
	}

	// 13. 实时利润 = 总商品金额 - 货款 - 留底 - 退款成本 - 小店服务费 - 线上佣金 - 广告费
	// 利润 = 定价总额 - 货款 - 留底 - 退款成本 - 小店服务费 - 佣金(5%×0.1) - 广告费
	// 佣金扣除 = (定价总额 - 留底 - 货款) × 5% × 0.1
	commissionDeduction := (totalProductAmount - stats.ReserveAmount - stats.ProductCostAmount) * 0.005
	stats.RealtimeProfit = totalProductAmount - stats.ProductCostAmount - stats.ReserveAmount - stats.RefundCostAmount - stats.ShopServiceFee - commissionDeduction - stats.AdCostAmount

	return stats, nil
}

// getAdCost 从千川广告报表获取指定时间段内的广告费消耗
func (s *OrderService) getAdCost(ctx context.Context, startTime, endTime string) float64 {
	// 提取日期部分（stat_datetime 存的是日期字符串 2026-06-01）
	qStart := ""
	qEnd := ""
	if startTime != "" && len(startTime) >= 10 {
		qStart = startTime[:10]
	}
	if endTime != "" && len(endTime) >= 10 {
		qEnd = endTime[:10]
	}

	query := "SELECT COALESCE(SUM(stat_cost), 0) as cost FROM qianchuan_ad_report WHERE 1=1"
	params := []interface{}{}
	if qStart != "" {
		query += " AND stat_datetime >= ?"
		params = append(params, qStart)
	}
	if qEnd != "" {
		query += " AND stat_datetime <= ?"
		params = append(params, qEnd)
	}

	row, err := g.DB().GetOne(ctx, query, params...)
	if err != nil {
		g.Log().Warningf(ctx, "查询千川广告费失败: %v", err)
		return 0
	}
	return row["cost"].Float64()
}

// List 查询订单列表
func (s *OrderService) List(ctx context.Context, page, pageSize int, keyword, shopName, orderStatus, talentId string, startTime, endTime string) ([]g.Map, int64, error) {
	query := "SELECT o.*, COALESCE(g.base_price, 0) as base_price FROM order_info o LEFT JOIN goods g ON o.seller_code = g.goods_code WHERE 1=1"
	params := []interface{}{}

	// 关键词搜索（订单号、商品名称、达人名称）
	if keyword != "" {
		query += " AND (sub_order_no LIKE ? OR main_order_no LIKE ? OR product_name LIKE ? OR talent_name LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey, searchKey, searchKey)
	}

	// 账号筛选
	if shopName != "" {
		query += " AND shop_name = ?"
		params = append(params, shopName)
	}

	// 订单状态筛选
	if orderStatus != "" {
		query += " AND order_status = ?"
		params = append(params, orderStatus)
	}

	// 达人UID筛选
	if talentId != "" {
		query += " AND talent_id = ?"
		params = append(params, talentId)
	}

	// 时间范围筛选（支付完成时间）
	if startTime != "" {
		query += " AND order_submit_time >= ?"
		params = append(params, startTime)
	}
	if endTime != "" {
		query += " AND order_submit_time <= ?"
		params = append(params, endTime)
	}

	// 获取总数
	totalQuery := strings.Replace(query, "SELECT o.*, COALESCE(g.base_price, 0) as base_price", "SELECT COUNT(*) as total", 1)
	resultTotal, err := g.DB().GetOne(ctx, totalQuery, params...)
	if err != nil {
		return nil, 0, err
	}
	total := resultTotal["total"].Int64()

	// 分页查询
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	params = append(params, pageSize, offset)

	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}

	orders := make([]g.Map, len(result))
	for i, row := range result {
		orders[i] = g.Map{
			"id":                   row["id"],
			"shopName":             row["shop_name"],
			"mainOrderNo":          row["main_order_no"],
			"subOrderNo":           row["sub_order_no"],
			"productName":          row["product_name"],
			"productSpec":          row["product_spec"],
			"productNum":           row["product_num"],
			"productId":            row["product_id"],
			"productPrice":         row["product_price"],
			"orderPayAmount":       row["order_pay_amount"],
			"shippingFee":          row["shipping_fee"],
			"totalDiscount":        row["total_discount"],
			"orderStatus":          row["order_status"],
			"orderSubmitTime":      row["order_submit_time"],
			"payFinishTime":        row["pay_finish_time"],
			"orderFinishTime":      row["order_finish_time"],
			"receiverName":         row["receiver_name"],
			"receiverPhone":        row["receiver_phone"],
			"province":             row["province"],
			"city":                 row["city"],
			"district":             row["district"],
			"talentName":           row["talent_name"],
			"flowSource":           row["flow_source"],
			"afterSaleStatus":      row["after_sale_status"],
			"talentId":             row["talent_id"],
			"productCost":          row["product_cost"],
			"shopServiceFee":       row["shop_service_fee"],
			"commissionRate":       row["commission_rate"],
			"douyinAccount":        row["douyin_account"],
			"uid":                  row["uid"],
			"teamOwner":            row["team_owner"],
			"isShopSelfSale":       row["is_shop_self_sale"],
		}
	}

	return orders, total, nil
}

// ExportToExcel 导出订单数据到Excel
func (s *OrderService) ExportToExcel(ctx context.Context, keyword, shopName, orderStatus string, startTime, endTime string) (string, error) {
	// 查询所有符合条件的订单
	query := "SELECT * FROM order_info WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (sub_order_no LIKE ? OR main_order_no LIKE ? OR product_name LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey, searchKey)
	}

	if shopName != "" {
		query += " AND shop_name = ?"
		params = append(params, shopName)
	}

	if orderStatus != "" {
		query += " AND order_status = ?"
		params = append(params, orderStatus)
	}

	if startTime != "" {
		query += " AND order_submit_time >= ?"
		params = append(params, startTime)
	}
	if endTime != "" {
		query += " AND order_submit_time <= ?"
		params = append(params, endTime)
	}

	query += " ORDER BY id DESC"
	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "", gerror.New("没有数据可导出")
	}

	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "订单数据"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头（与数据库字段对应）
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
	for i, row := range result {
		rowNum := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row["shop_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row["main_order_no"])
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), row["sub_order_no"])
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), row["product_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), row["product_spec"])
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), row["product_num"])
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), row["product_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNum), row["seller_code"])
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowNum), row["product_price"])
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowNum), row["order_pay_amount"])
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowNum), row["shipping_fee"])
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowNum), row["total_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowNum), row["platform_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowNum), row["seller_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowNum), row["talent_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowNum), row["seller_change_price"])
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", rowNum), row["pay_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("R%d", rowNum), row["red_deduction"])
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", rowNum), row["pay_type"])
		f.SetCellValue(sheetName, fmt.Sprintf("T%d", rowNum), row["service_fee"])
		f.SetCellValue(sheetName, fmt.Sprintf("U%d", rowNum), row["receiver_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("V%d", rowNum), row["receiver_phone"])
		f.SetCellValue(sheetName, fmt.Sprintf("W%d", rowNum), row["province"])
		f.SetCellValue(sheetName, fmt.Sprintf("X%d", rowNum), row["city"])
		f.SetCellValue(sheetName, fmt.Sprintf("Y%d", rowNum), row["district"])
		f.SetCellValue(sheetName, fmt.Sprintf("Z%d", rowNum), row["street"])
		f.SetCellValue(sheetName, fmt.Sprintf("AA%d", rowNum), row["address_detail"])
		f.SetCellValue(sheetName, fmt.Sprintf("AB%d", rowNum), row["is_address_changed"])
		f.SetCellValue(sheetName, fmt.Sprintf("AC%d", rowNum), row["address_change_time"])
		f.SetCellValue(sheetName, fmt.Sprintf("AD%d", rowNum), row["buyer_note"])
		f.SetCellValue(sheetName, fmt.Sprintf("AE%d", rowNum), formatTime(row["order_submit_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AF%d", rowNum), row["flag_color"])
		f.SetCellValue(sheetName, fmt.Sprintf("AG%d", rowNum), row["seller_note"])
		f.SetCellValue(sheetName, fmt.Sprintf("AH%d", rowNum), formatTime(row["order_finish_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AI%d", rowNum), formatTime(row["pay_finish_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AJ%d", rowNum), row["app_channel"])
		f.SetCellValue(sheetName, fmt.Sprintf("AK%d", rowNum), row["flow_source"])
		f.SetCellValue(sheetName, fmt.Sprintf("AL%d", rowNum), row["order_status"])
		f.SetCellValue(sheetName, fmt.Sprintf("AM%d", rowNum), formatTime(row["promise_send_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AN%d", rowNum), row["order_type"])
		f.SetCellValue(sheetName, fmt.Sprintf("AO%d", rowNum), row["luban_page_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AP%d", rowNum), row["talent_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AQ%d", rowNum), row["talent_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("AR%d", rowNum), row["store_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AS%d", rowNum), row["after_sale_status"])
		f.SetCellValue(sheetName, fmt.Sprintf("AT%d", rowNum), row["cancel_reason"])
		f.SetCellValue(sheetName, fmt.Sprintf("AU%d", rowNum), formatTime(row["appoint_send_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AV%d", rowNum), row["warehouse_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AW%d", rowNum), row["warehouse_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("AX%d", rowNum), row["is_safe_buy"])
		f.SetCellValue(sheetName, fmt.Sprintf("AY%d", rowNum), row["ad_channel"])
		f.SetCellValue(sheetName, fmt.Sprintf("AZ%d", rowNum), row["flow_type"])
		f.SetCellValue(sheetName, fmt.Sprintf("BA%d", rowNum), row["flow_format"])
		f.SetCellValue(sheetName, fmt.Sprintf("BB%d", rowNum), row["flow_channel"])
		f.SetCellValue(sheetName, fmt.Sprintf("BC%d", rowNum), row["send_user"])
		f.SetCellValue(sheetName, fmt.Sprintf("BD%d", rowNum), row["send_user_detail"])
		f.SetCellValue(sheetName, fmt.Sprintf("BE%d", rowNum), formatTime(row["send_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("BF%d", rowNum), row["reduce_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BG%d", rowNum), row["platform_real_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BH%d", rowNum), row["seller_real_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BI%d", rowNum), row["talent_real_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BJ%d", rowNum), formatTime(row["expect_arrive_time"]))
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 15)  // 店铺
	f.SetColWidth(sheetName, "B", "C", 20)  // 订单编号
	f.SetColWidth(sheetName, "D", "D", 30)  // 商品名称
	f.SetColWidth(sheetName, "E", "H", 15)  // 规格等
	f.SetColWidth(sheetName, "I", "T", 12)  // 金额相关
	f.SetColWidth(sheetName, "U", "AD", 15) // 地址相关
	f.SetColWidth(sheetName, "AE", "BJ", 18) // 时间等

	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, fmt.Sprintf("order_export_%s.xlsx", time.Now().Format("20060102150405")))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}

	return filename, nil
}

// formatTime 格式化时间
func formatTime(val interface{}) string {
	if val == nil {
		return ""
	}
	if t, ok := val.(time.Time); ok {
		return t.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%v", val)
}

// ImportResult 导入结果
type OrderImportResult struct {
	TotalRows    int      `json:"totalRows"`
	SuccessCount int      `json:"successCount"`
	SkipCount    int      `json:"skipCount"`
	FailCount    int      `json:"failCount"`
	Errors       []string `json:"errors"`
}

// ImportFromExcel 从Excel导入订单数据（流式处理+批量入库）
func (s *OrderService) ImportFromExcel(ctx context.Context, filePath string) (*OrderImportResult, error) {
	g.Log().Infof(ctx, "开始导入订单文件: %s", filePath)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		g.Log().Errorf(ctx, "打开Excel文件失败: %s, 错误: %v", filePath, err)
		return nil, fmt.Errorf("打开Excel文件失败(filePath=%s): %w", filePath, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		g.Log().Error(ctx, "Excel文件中没有工作表")
		return nil, gerror.New("Excel文件中没有工作表")
	}

	// 使用流式读取
	rows, err := f.Rows(sheets[0])
	if err != nil {
		g.Log().Errorf(ctx, "读取Excel数据失败: %v", err)
		return nil, fmt.Errorf("读取Excel数据失败：%w", err)
	}
	defer rows.Close()

	result := &OrderImportResult{
		Errors: make([]string, 0),
	}

	// 读取表头
	if !rows.Next() {
		g.Log().Error(ctx, "Excel文件中没有数据")
		return nil, gerror.New("Excel文件中没有数据")
	}
	headers, err := rows.Columns()
	if err != nil {
		g.Log().Errorf(ctx, "读取表头失败: %v", err)
		return nil, fmt.Errorf("读取表头失败：%w", err)
	}

	g.Log().Infof(ctx, "订单Excel表头: %v", headers)

	// 验证表头（至少包含必填字段）
	requiredHeaders := []string{"店铺", "子订单编号", "选购商品", "商品单价", "订单应付金额", "订单状态"}
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			g.Log().Errorf(ctx, "Excel表头缺少必填字段: %s, 实际表头: %v", required, headers)
			return nil, gerror.New(fmt.Sprintf("Excel表头缺少必填字段: %s\n实际读取到的表头: %v", required, headers))
		}
	}

	// 批量处理配置
	const batchSize = 1000
	batch := make([]g.Map, 0, batchSize)
	rowNum := 1

	// 预加载已存在的子订单号
	existingOrders := make(map[string]bool)
	existingRows, err := g.DB().Model("order_info").Fields("sub_order_no").All()
	if err != nil {
		g.Log().Errorf(ctx, "预加载已存在订单失败: %v", err)
	} else {
		for _, row := range existingRows {
			existingOrders[row["sub_order_no"].String()] = true
		}
		g.Log().Infof(ctx, "已存在订单数量: %d", len(existingOrders))
	}

	// 用于统计的空行、验证失败、已存在跳过数量
	var emptyRowCount, validationFailCount, existSkipCount int

	// 流式读取并批量处理
	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 读取失败 - %v", rowNum, err))
			result.FailCount++
			continue
		}

		result.TotalRows++

		// 跳过空行
		if len(cols) == 0 || (getCol(cols, headerMap, "店铺") == "" && getCol(cols, headerMap, "子订单编号") == "" && getCol(cols, headerMap, "选购商品") == "") {
			emptyRowCount++
			result.SkipCount++
			continue
		}

		// 验证必填字段
		shopName := getCol(cols, headerMap, "店铺")
		subOrderNo := getCol(cols, headerMap, "子订单编号")
		productName := getCol(cols, headerMap, "选购商品")
		orderStatus := getCol(cols, headerMap, "订单状态")

		if shopName == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 店铺不能为空 (子订单号: %s)", rowNum, subOrderNo)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}
		if subOrderNo == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 子订单编号不能为空 (店铺: %s)", rowNum, shopName)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}
		if productName == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 选购商品不能为空 (子订单号: %s)", rowNum, subOrderNo)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}
		if orderStatus == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 订单状态不能为空 (子订单号: %s)", rowNum, subOrderNo)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}

		// 检查子订单号是否已存在
		if existingOrders[subOrderNo] {
			existSkipCount++
			result.SkipCount++
			continue
		}

		// 构建数据（截断可能超长的字段，避免数据库插入失败）
		data := g.Map{
			"shop_name":                truncateString(shopName, 50),
			"main_order_no":            truncateString(getCol(cols, headerMap, "主订单编号"), 50),
			"sub_order_no":             truncateString(subOrderNo, 50),
			"product_name":             truncateString(productName, 255),
			"product_spec":             truncateString(getCol(cols, headerMap, "商品规格"), 100),
			"product_num":              parseIntOrDefault(getCol(cols, headerMap, "商品数量"), 1),
			"product_id":               truncateString(getCol(cols, headerMap, "商品ID"), 50),
			"seller_code":              truncateString(getCol(cols, headerMap, "商家编码"), 50),
			"product_price":            parseFloatOrDefault(getCol(cols, headerMap, "商品单价"), 0),
			"order_pay_amount":         parseFloatOrDefault(getCol(cols, headerMap, "订单应付金额"), 0),
			"shipping_fee":             parseFloatOrDefault(getCol(cols, headerMap, "运费"), 0),
			"total_discount":           parseFloatOrDefault(getCol(cols, headerMap, "优惠总金额"), 0),
			"platform_discount":        parseFloatOrDefault(getCol(cols, headerMap, "平台优惠"), 0),
			"seller_discount":          parseFloatOrDefault(getCol(cols, headerMap, "商家优惠"), 0),
			"talent_discount":          parseFloatOrDefault(getCol(cols, headerMap, "达人优惠"), 0),
			"seller_change_price":      parseFloatOrDefault(getCol(cols, headerMap, "商家改价"), 0),
			"pay_discount":             parseFloatOrDefault(getCol(cols, headerMap, "支付优惠"), 0),
			"red_deduction":            parseFloatOrDefault(getCol(cols, headerMap, "红包抵扣"), 0),
			"pay_type":                 truncateString(getCol(cols, headerMap, "支付方式"), 20),
			"service_fee":              parseFloatOrDefault(getCol(cols, headerMap, "手续费"), 0),
			"receiver_name":            truncateString(getCol(cols, headerMap, "收件人"), 50),
			"receiver_phone":           truncateString(getCol(cols, headerMap, "收件人手机号"), 20),
			"province":                 truncateString(getCol(cols, headerMap, "省"), 20),
			"city":                     truncateString(getCol(cols, headerMap, "市"), 20),
			"district":                 truncateString(getCol(cols, headerMap, "区"), 255),
			"street":                   truncateString(getCol(cols, headerMap, "街道"), 100),
			"address_detail":           truncateString(getCol(cols, headerMap, "详细地址"), 255),
			"is_address_changed":       truncateString(getCol(cols, headerMap, "是否修改过地址"), 10),
			"address_change_time":      truncateString(getCol(cols, headerMap, "地址修改时段"), 50),
			"buyer_note":               truncateString(getCol(cols, headerMap, "买家留言"), 255),
			"order_submit_time":        parseDateTime(getCol(cols, headerMap, "订单提交时间")),
			"flag_color":               truncateString(getCol(cols, headerMap, "旗帜颜色"), 10),
			"seller_note":              truncateString(getCol(cols, headerMap, "商家备注"), 255),
			"order_finish_time":        parseDateTime(getCol(cols, headerMap, "订单完成时间")),
			"pay_finish_time":          parseDateTime(getCol(cols, headerMap, "支付完成时间")),
			"app_channel":              truncateString(getCol(cols, headerMap, "APP渠道"), 20),
			"flow_source":              truncateString(getCol(cols, headerMap, "流量来源"), 20),
			"order_status":             truncateString(orderStatus, 20),
			"promise_send_time":        parseDateTime(getCol(cols, headerMap, "承诺发货时间")),
			"order_type":               truncateString(getCol(cols, headerMap, "订单类型"), 20),
			"luban_page_id":            truncateString(getCol(cols, headerMap, "鲁班落地页ID"), 50),
			"talent_id":                truncateString(getCol(cols, headerMap, "达人ID"), 50),
			"talent_name":              truncateString(getCol(cols, headerMap, "达人昵称"), 50),
			"store_id":                 truncateString(getCol(cols, headerMap, "所属门店ID"), 50),
			"after_sale_status":        truncateString(getCol(cols, headerMap, "售后状态"), 20),
			"cancel_reason":            truncateString(getCol(cols, headerMap, "取消原因"), 100),
			"appoint_send_time":        parseDateTime(getCol(cols, headerMap, "预约发货时间")),
			"warehouse_id":             truncateString(getCol(cols, headerMap, "仓库ID"), 50),
			"warehouse_name":           truncateString(getCol(cols, headerMap, "仓库名称"), 50),
			"is_safe_buy":              truncateString(getCol(cols, headerMap, "是否安心购"), 10),
			"ad_channel":               truncateString(getCol(cols, headerMap, "广告渠道"), 20),
			"flow_type":                truncateString(getCol(cols, headerMap, "流量类型"), 20),
			"flow_format":              truncateString(getCol(cols, headerMap, "流量体裁"), 20),
			"flow_channel":             truncateString(getCol(cols, headerMap, "流量渠道"), 20),
			"send_user":                truncateString(getCol(cols, headerMap, "发货主体"), 20),
			"send_user_detail":         truncateString(getCol(cols, headerMap, "发货主体明细"), 50),
			"send_time":                parseDateTime(getCol(cols, headerMap, "发货时间")),
			"reduce_discount":          parseFloatOrDefault(getCol(cols, headerMap, "降价类优惠"), 0),
			"platform_real_discount":   parseFloatOrDefault(getCol(cols, headerMap, "平台实际承担优惠金额"), 0),
			"seller_real_discount":     parseFloatOrDefault(getCol(cols, headerMap, "商家实际承担优惠金额"), 0),
			"talent_real_discount":     parseFloatOrDefault(getCol(cols, headerMap, "达人实际承担优惠金额"), 0),
			"expect_arrive_time":       parseDateTime(getCol(cols, headerMap, "预计送达时间")),
		}

		batch = append(batch, data)
		existingOrders[subOrderNo] = true

		// 达到批次大小，执行批量插入
		if len(batch) >= batchSize {
			g.Log().Debugf(ctx, "准备批量插入第%d批，共%d条数据", rowNum/batchSize, len(batch))
			if err := s.batchInsert(ctx, batch); err != nil {
				errMsg := fmt.Sprintf("批量插入失败（约第%d-%d行）: %v", rowNum-batchSize+1, rowNum, err)
				result.Errors = append(result.Errors, errMsg)
				g.Log().Errorf(ctx, "订单导入%s", errMsg)
				result.FailCount += len(batch)
			} else {
				g.Log().Infof(ctx, "批量插入成功：%d条（约第%d-%d行）", len(batch), rowNum-batchSize+1, rowNum)
				result.SuccessCount += len(batch)
			}
			batch = batch[:0]
		}
	}

	// 处理剩余数据
	if len(batch) > 0 {
		if err := s.batchInsert(ctx, batch); err != nil {
			errMsg := fmt.Sprintf("批量插入失败（最后%d行）: %v", len(batch), err)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Errorf(ctx, "订单导入%s", errMsg)
			result.FailCount += len(batch)
		} else {
			g.Log().Infof(ctx, "批量插入成功：%d条（最后一批）", len(batch))
			result.SuccessCount += len(batch)
		}
	}

	// 记录所有错误信息到日志
	if len(result.Errors) > 0 {
		g.Log().Errorf(ctx, "订单导入失败详情（共%d条错误）:", len(result.Errors))
		for i, errMsg := range result.Errors {
			g.Log().Errorf(ctx, "  错误%d: %s", i+1, errMsg)
			// 只打印前200条到日志，避免日志过大
			if i >= 200 {
				g.Log().Errorf(ctx, "  ... 还有 %d 条错误信息请查看返回结果", len(result.Errors)-200)
				break
			}
		}
	}

	// 限制返回给前端的错误信息数量（避免响应过大）
	if len(result.Errors) > 50 {
		result.Errors = append(result.Errors[:50], fmt.Sprintf("... 还有 %d 条错误信息未显示，请查看日志获取完整信息", len(result.Errors)-50))
	}

	g.Log().Infof(ctx, "订单导入完成: 成功%d条，跳过%d条（含空行%d+已存在%d+验证失败%d），失败%d条",
		result.SuccessCount, result.SkipCount, emptyRowCount, existSkipCount, validationFailCount, result.FailCount)
	return result, nil
}

// batchInsert 批量插入订单数据
func (s *OrderService) batchInsert(ctx context.Context, batch []g.Map) error {
	if len(batch) == 0 {
		return nil
	}
	_, err := g.DB().Model("order_info").Ctx(ctx).Data(batch).Insert()
	return err
}

	// 辅助函数：获取列值
func getCol(cols []string, headerMap map[string]int, header string) string {
	if idx, exists := headerMap[header]; exists && idx < len(cols) {
		return strings.TrimSpace(cols[idx])
	}
	return ""
}

// alignCols 移除数据列尾部的空列，使列数不超过目标列数
func alignCols(cols []string, targetLen int) []string {
	if len(cols) <= targetLen {
		return cols
	}
	result := make([]string, len(cols))
	copy(result, cols)
	for len(result) > targetLen && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}
	return result
}

// getColFlex 获取列值，支持模糊匹配：当精确索引为空时，搜索附近索引
func getColFlex(cols []string, headerMap map[string]int, header string) string {
	idx, exists := headerMap[header]
	if !exists {
		return ""
	}
	if idx < len(cols) {
		if v := strings.TrimSpace(cols[idx]); v != "" {
			return v
		}
	}
	// 精确索引为空或越界，搜索附近（Excel列错位通常偏移1-2列）
	for _, offset := range []int{1, -1, 2, -2} {
		adj := idx + offset
		if adj >= 0 && adj < len(cols) {
			if v := strings.TrimSpace(cols[adj]); v != "" {
				return v
			}
		}
	}
	return ""
}

// 辅助函数：截断字符串到指定字符数（不是字节数），并清理非法字符
func truncateString(s string, maxLen int) string {
	s = cleanIllegalChars(s)
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return s
}

// cleanIllegalChars 清理非法UTF-8字符和不完整序列
func cleanIllegalChars(s string) string {
	result := strings.Builder{}
	for i := 0; i < len(s); {
		// 解码一个 rune 和它的字节宽度
		r, w := utf8.DecodeRuneInString(s[i:])
		
		// 情况1: 空字节
		if r == 0 {
			i += w
			continue
		}
		
		// 情况2: C0 控制字符（保留换行和回车）
		if r < 32 && r != '\n' && r != '\r' {
			i += w
			continue
		}
		
		// 情况3: 替换字符（表示输入中有不完整的 UTF-8 序列）
		if r == unicode.ReplacementChar {
			// 检查是否是不完整的多字节序列
			// 通过尝试解码下一个位置来判断
			remaining := len(s) - i - w
			if remaining > 0 {
				// 跳过不完整序列的剩余字节
				// 寻找下一个有效字符的起始位置
				i += w
				// 跳过后续的 continuation byte (10xxxxxx) 或不完整序列
				for i < len(s) && (s[i]&0xC0) == 0x80 {
					i++
				}
				continue
			}
			i += w
			continue
		}
		
		// 情况4: 代理对（surrogate pairs）无效
		if r >= 0xD800 && r <= 0xDFFF {
			i += w
			continue
		}
		
		// 情况5: 非 Unicode 平面字符（大于 0x10FFFF）
		if r > 0x10FFFF {
			i += w
			continue
		}
		
		// 有效字符，添加到结果
		result.WriteRune(r)
		i += w
	}
	return result.String()
}

// 辅助函数：解析整数
func parseIntOrDefault(s string, defaultVal int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// 辅助函数：解析浮点数
func parseFloatOrDefault(s string, defaultVal float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// 辅助函数：解析日期时间
func parseDateTime(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// 尝试多种日期格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02",
		"2006/01/02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return nil
}

// getOrderStatsFromTable 从预聚合表读取订单统计
func (s *OrderService) getOrderStatsFromTable(ctx context.Context, shopName, startTime, endTime string) *OrderStats {
	query := g.DB().Model("daily_order_stats").
		Where("stat_date >= ?", startTime[:10]).
		Where("stat_date <= ?", endTime[:10])
	if shopName != "" {
		query = query.Where("shop_name = ?", shopName)
	}

	result, err := query.Fields(`
		COALESCE(SUM(total_order_count),0) as total_order_count,
		COALESCE(SUM(valid_order_count),0) as valid_order_count,
		COALESCE(SUM(invalid_order_count),0) as invalid_order_count,
		COALESCE(SUM(valid_pay_amount),0) as valid_pay_amount,
		COALESCE(SUM(product_cost_amount),0) as product_cost_amount,
		COALESCE(SUM(reserve_amount),0) as reserve_amount,
		COALESCE(SUM(refund_cost_amount),0) as refund_cost_amount,
		COALESCE(SUM(refund_order_count),0) as refund_order_count,
		COALESCE(SUM(shop_service_fee),0) as shop_service_fee,
		COALESCE(SUM(commission),0) as commission,
		COALESCE(SUM(valid_order_cost_match),0) as valid_order_cost_match
	`).One()

	if err != nil || result == nil {
		return nil
	}

	totalOrderCount := result["total_order_count"].Int64()
	validOrderCount := result["valid_order_count"].Int64()
	invalidOrderCount := result["invalid_order_count"].Int64()
	if totalOrderCount == 0 && validOrderCount == 0 {
		return nil
	}

	validPayAmount := result["valid_pay_amount"].Float64()
	productCostAmount := result["product_cost_amount"].Float64()
	reserveAmount := result["reserve_amount"].Float64()
	shopServiceFee := result["shop_service_fee"].Float64()
	commission := result["commission"].Float64()

	return &OrderStats{
		TotalOrderCount:      totalOrderCount,
		ValidOrderCount:      validOrderCount,
		InvalidOrderCount:    invalidOrderCount,
		UnmatchedOrderCount:  totalOrderCount - validOrderCount - invalidOrderCount,
		ValidPayAmount:       validPayAmount,
		ProductCostAmount:    productCostAmount,
		ReserveAmount:        reserveAmount,
		RefundCostAmount:     result["refund_cost_amount"].Float64(),
		RefundOrderCount:     result["refund_order_count"].Int64(),
		ShopServiceFee:       shopServiceFee,
		AdCostAmount:         0,
		OnlineCommission:     commission,
		RealtimeProfit:       validPayAmount - productCostAmount - reserveAmount - result["refund_cost_amount"].Float64() - shopServiceFee - commission,
		ValidOrderCostMatch:  result["valid_order_cost_match"].Int64(),
		CommissionOrderCount: validOrderCount,
		SyncDate:             time.Now().Format("2006-01-02"),
	}
}
