package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

const kuaishouOrdersTable = "kuaishou_orders"

type KuaishouOrderService struct{}

func NewKuaishouOrderService() *KuaishouOrderService {
	return &KuaishouOrderService{}
}

// List 查询快手订单列表
func (s *KuaishouOrderService) List(ctx context.Context, page, pageSize int, keyword, shopName, orderStatus string, startTime, endTime string) ([]g.Map, int64, error) {
	query := "SELECT * FROM " + kuaishouOrdersTable + " WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (order_no LIKE ? OR product_name LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey)
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
		query += " AND pay_finish_time >= ?"
		params = append(params, startTime)
	}
	if endTime != "" {
		query += " AND pay_finish_time <= ?"
		params = append(params, endTime)
	}

	// 获取总数
	totalQuery := strings.Replace(query, "SELECT *", "SELECT COUNT(*) as total", 1)
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
			"id":              row["id"],
			"shopName":        row["shop_name"],
			"orderNo":         row["order_no"],
			"productName":     row["product_name"],
			"productSpec":     row["product_spec"],
			"productNum":      row["product_num"],
			"productId":       row["product_id"],
			"productPrice":    row["product_price"],
			"orderPayAmount":  row["order_pay_amount"],
			"shippingFee":     row["shipping_fee"],
			"totalDiscount":   row["total_discount"],
			"orderStatus":     row["order_status"],
			"orderSubmitTime": row["order_submit_time"],
			"payFinishTime":   row["pay_finish_time"],
			"orderFinishTime": row["order_finish_time"],
			"receiverName":    row["receiver_name"],
			"receiverPhone":   row["receiver_phone"],
			"province":        row["province"],
			"city":            row["city"],
			"district":        row["district"],
			"talentName":      row["talent_name"],
			"flowSource":      row["flow_source"],
			"afterSaleStatus": row["after_sale_status"],
		}
	}

	return orders, total, nil
}

// ExportToExcel 导出快手订单数据到Excel
func (s *KuaishouOrderService) ExportToExcel(ctx context.Context, keyword, shopName, orderStatus string, startTime, endTime string) (string, error) {
	query := "SELECT * FROM " + kuaishouOrdersTable + " WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (order_no LIKE ? OR product_name LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey)
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
		query += " AND pay_finish_time >= ?"
		params = append(params, startTime)
	}
	if endTime != "" {
		query += " AND pay_finish_time <= ?"
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

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "订单数据"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	headers := []string{
		"店铺", "订单号", "选购商品", "商品规格", "商品数量", "商品ID", "商家编码",
		"商品单价", "订单应付金额", "运费", "优惠总金额", "平台优惠", "商家优惠", "达人优惠",
		"商家改价", "支付优惠", "红包抵扣", "支付方式", "手续费", "收件人", "收件人手机号",
		"省", "市", "区", "街道", "详细地址", "是否修改过地址", "地址修改时段", "买家留言",
		"订单提交时间", "旗帜颜色", "商家备注", "订单完成时间", "支付完成时间", "APP渠道",
		"流量来源", "订单状态", "承诺发货时间", "订单类型", "鲁班落地页ID", "达人ID", "达人昵称",
		"所属门店ID", "售后状态", "取消原因", "预约发货时间", "仓库ID", "仓库名称", "是否安心购",
		"广告渠道", "流量类型", "流量体裁", "流量渠道", "发货主体", "发货主体明细", "发货时间",
		"降价类优惠", "平台实际承担优惠金额", "商家实际承担优惠金额", "达人实际承担优惠金额", "预计送达时间",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	for i, row := range result {
		rowNum := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row["shop_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row["order_no"])
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), row["product_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), row["product_spec"])
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), row["product_num"])
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), row["product_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), row["seller_code"])
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNum), row["product_price"])
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowNum), row["order_pay_amount"])
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowNum), row["shipping_fee"])
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowNum), row["total_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowNum), row["platform_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowNum), row["seller_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowNum), row["talent_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowNum), row["seller_change_price"])
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowNum), row["pay_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", rowNum), row["red_deduction"])
		f.SetCellValue(sheetName, fmt.Sprintf("R%d", rowNum), row["pay_type"])
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", rowNum), row["service_fee"])
		f.SetCellValue(sheetName, fmt.Sprintf("T%d", rowNum), row["receiver_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("U%d", rowNum), row["receiver_phone"])
		f.SetCellValue(sheetName, fmt.Sprintf("V%d", rowNum), row["province"])
		f.SetCellValue(sheetName, fmt.Sprintf("W%d", rowNum), row["city"])
		f.SetCellValue(sheetName, fmt.Sprintf("X%d", rowNum), row["district"])
		f.SetCellValue(sheetName, fmt.Sprintf("Y%d", rowNum), row["street"])
		f.SetCellValue(sheetName, fmt.Sprintf("Z%d", rowNum), row["address_detail"])
		f.SetCellValue(sheetName, fmt.Sprintf("AA%d", rowNum), row["is_address_changed"])
		f.SetCellValue(sheetName, fmt.Sprintf("AB%d", rowNum), row["address_change_time"])
		f.SetCellValue(sheetName, fmt.Sprintf("AC%d", rowNum), row["buyer_note"])
		f.SetCellValue(sheetName, fmt.Sprintf("AD%d", rowNum), formatTime(row["order_submit_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AE%d", rowNum), row["flag_color"])
		f.SetCellValue(sheetName, fmt.Sprintf("AF%d", rowNum), row["seller_note"])
		f.SetCellValue(sheetName, fmt.Sprintf("AG%d", rowNum), formatTime(row["order_finish_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AH%d", rowNum), formatTime(row["pay_finish_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AI%d", rowNum), row["app_channel"])
		f.SetCellValue(sheetName, fmt.Sprintf("AJ%d", rowNum), row["flow_source"])
		f.SetCellValue(sheetName, fmt.Sprintf("AK%d", rowNum), row["order_status"])
		f.SetCellValue(sheetName, fmt.Sprintf("AL%d", rowNum), formatTime(row["promise_send_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AM%d", rowNum), row["order_type"])
		f.SetCellValue(sheetName, fmt.Sprintf("AN%d", rowNum), row["luban_page_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AO%d", rowNum), row["talent_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AP%d", rowNum), row["talent_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("AQ%d", rowNum), row["store_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AR%d", rowNum), row["after_sale_status"])
		f.SetCellValue(sheetName, fmt.Sprintf("AS%d", rowNum), row["cancel_reason"])
		f.SetCellValue(sheetName, fmt.Sprintf("AT%d", rowNum), formatTime(row["appoint_send_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AU%d", rowNum), row["warehouse_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("AV%d", rowNum), row["warehouse_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("AW%d", rowNum), row["is_safe_buy"])
		f.SetCellValue(sheetName, fmt.Sprintf("AX%d", rowNum), row["ad_channel"])
		f.SetCellValue(sheetName, fmt.Sprintf("AY%d", rowNum), row["flow_type"])
		f.SetCellValue(sheetName, fmt.Sprintf("AZ%d", rowNum), row["flow_format"])
		f.SetCellValue(sheetName, fmt.Sprintf("BA%d", rowNum), row["flow_channel"])
		f.SetCellValue(sheetName, fmt.Sprintf("BB%d", rowNum), row["send_user"])
		f.SetCellValue(sheetName, fmt.Sprintf("BC%d", rowNum), row["send_user_detail"])
		f.SetCellValue(sheetName, fmt.Sprintf("BD%d", rowNum), formatTime(row["send_time"]))
		f.SetCellValue(sheetName, fmt.Sprintf("BE%d", rowNum), row["reduce_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BF%d", rowNum), row["platform_real_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BG%d", rowNum), row["seller_real_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BH%d", rowNum), row["talent_real_discount"])
		f.SetCellValue(sheetName, fmt.Sprintf("BI%d", rowNum), formatTime(row["expect_arrive_time"]))
	}

	f.SetColWidth(sheetName, "A", "A", 15)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 30)
	f.SetColWidth(sheetName, "D", "G", 15)
	f.SetColWidth(sheetName, "H", "S", 12)
	f.SetColWidth(sheetName, "T", "AC", 15)
	f.SetColWidth(sheetName, "AD", "BI", 18)

	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, fmt.Sprintf("kuaishou_order_export_%s.xlsx", time.Now().Format("20060102150405")))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}

	return filename, nil
}

// ImportResult 导入结果
type KuaishouOrderImportResult struct {
	TotalRows    int      `json:"totalRows"`
	SuccessCount int      `json:"successCount"`
	SkipCount    int      `json:"skipCount"`
	FailCount    int      `json:"failCount"`
	Errors       []string `json:"errors"`
}

// ImportFromExcel 从Excel导入快手订单数据（流式处理+批量入库）
func (s *KuaishouOrderService) ImportFromExcel(ctx context.Context, filePath string) (*KuaishouOrderImportResult, error) {
	g.Log().Infof(ctx, "开始导入快手订单文件: %s", filePath)

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

	rows, err := f.Rows(sheets[0])
	if err != nil {
		g.Log().Errorf(ctx, "读取Excel数据失败: %v", err)
		return nil, fmt.Errorf("读取Excel数据失败：%w", err)
	}
	defer rows.Close()

	result := &KuaishouOrderImportResult{
		Errors: make([]string, 0),
	}

	if !rows.Next() {
		g.Log().Error(ctx, "Excel文件中没有数据")
		return nil, gerror.New("Excel文件中没有数据")
	}
	headers, err := rows.Columns()
	if err != nil {
		g.Log().Errorf(ctx, "读取表头失败: %v", err)
		return nil, fmt.Errorf("读取表头失败：%w", err)
	}

	g.Log().Infof(ctx, "快手订单Excel表头: %v, 共%d列", headers, len(headers))

	// 快手平台原生表头 → 内部表头映射
	kuaishouHeaderMapping := map[string]string{
		"订单号":                 "订单号",
		"赠品订单号":               "订单号",
		"活动订单编号":              "订单号",
		"商品名称":                "选购商品",
		"SKU编码":               "商家编码",
		"成交数量":                "商品数量",
		"实付款":                 "订单应付金额",
		"快递费":                 "运费",
		"店铺优惠":                "商家优惠",
		"平台补贴":                "平台优惠",
		"主播补贴":                "达人优惠",
		"混资活动优惠":              "优惠总金额",
		"订单创建时间":              "订单提交时间",
		"订单支付时间":              "支付完成时间",
		"收货人姓名":               "收件人",
		"收货人电话":               "收件人手机号",
		"收货地址-省":              "省",
		"收货地址-市":              "市",
		"收货地址-区":              "区",
		"收货地址-街道":             "街道",
		"收货地址":                "详细地址",
		"订单备注":                "商家备注",
		"售后状态":                "售后状态",
		"CPS达人ID":             "达人ID",
		"CPS达人昵称":            "达人昵称",
		"预售/承诺发货时间":           "承诺发货时间",
		"发货时间":                "发货时间",
		"快递公司":                "发货主体",
		"快递单号":                "发货主体明细",
		"订单状态":                "订单状态",
		"商品ID":                "商品ID",
		"商品规格":                "商品规格",
		"商品单价":                "商品单价",
		"买家留言":                "买家留言",
		"旗帜颜色":                "旗帜颜色",
	}

	// 构建 headerMap，同时将快手原生表头映射为内部表头
	// 多个源表头映射到同一个目标时，保留第一个匹配的索引
	headerMap := make(map[string]int)
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if mapped, ok := kuaishouHeaderMapping[h]; ok {
			if _, exists := headerMap[mapped]; !exists {
				headerMap[mapped] = i
			}
		}
		if _, exists := headerMap[h]; !exists {
			headerMap[h] = i
		}
	}

	requiredHeaders := []string{"店铺", "订单号", "选购商品", "商品单价", "订单应付金额", "订单状态"}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			g.Log().Errorf(ctx, "Excel表头缺少必填字段: %s, 实际表头: %v", required, headers)
			return nil, gerror.New(fmt.Sprintf("Excel表头缺少必填字段: %s\n实际读取到的表头: %v", required, headers))
		}
	}

	const batchSize = 1000
	batch := make([]g.Map, 0, batchSize)
	rowNum := 1

	existingOrders := make(map[string]bool)
	existingRows, err := g.DB().Model(kuaishouOrdersTable).Fields("order_no").All()
	if err != nil {
		g.Log().Errorf(ctx, "预加载已存在订单失败: %v", err)
	} else {
		for _, row := range existingRows {
			existingOrders[row["order_no"].String()] = true
		}
		g.Log().Infof(ctx, "已存在订单数量: %d", len(existingOrders))
	}

	var emptyRowCount, validationFailCount, existSkipCount int

	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 读取失败 - %v", rowNum, err))
			result.FailCount++
			continue
		}

		result.TotalRows++

		// 数据列数比表头多时，对齐列数（Excel隐藏空列导致数据多出空列）
		if len(cols) > len(headers) {
			cols = alignCols(cols, len(headers))
		}

		if len(cols) == 0 || (getColFlex(cols, headerMap, "店铺") == "" && getColFlex(cols, headerMap, "订单号") == "" && getColFlex(cols, headerMap, "选购商品") == "") {
			emptyRowCount++
			result.SkipCount++
			continue
		}

		shopName := getColFlex(cols, headerMap, "店铺")
		orderNo := getColFlex(cols, headerMap, "订单号")
		productName := getColFlex(cols, headerMap, "选购商品")
		orderStatus := getColFlex(cols, headerMap, "订单状态")

		// 调试日志：前3行数据
		if rowNum <= 4 {
			g.Log().Infof(ctx, "第%d行调试: headerMap=%v", rowNum, headerMap)
			g.Log().Infof(ctx, "第%d行调试: len(cols)=%d, cols=%v", rowNum, len(cols), cols)
			g.Log().Infof(ctx, "第%d行调试: shopName=[%s] orderNo=[%s] productName=[%s] orderStatus=[%s]", rowNum, shopName, orderNo, productName, orderStatus)
		}

		if shopName == "" {
			validationFailCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 店铺不能为空 (订单号: %s)", rowNum, orderNo))
			g.Log().Warningf(ctx, "订单导入验证失败: 第%d行: 店铺不能为空", rowNum)
			result.FailCount++
			continue
		}
		if orderNo == "" {
			validationFailCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 订单号不能为空 (店铺: %s)", rowNum, shopName))
			g.Log().Warningf(ctx, "订单导入验证失败: 第%d行: 订单号不能为空", rowNum)
			result.FailCount++
			continue
		}
		if productName == "" {
			validationFailCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 选购商品不能为空 (订单号: %s)", rowNum, orderNo))
			g.Log().Warningf(ctx, "订单导入验证失败: 第%d行: 选购商品不能为空", rowNum)
			result.FailCount++
			continue
		}
		if orderStatus == "" {
			validationFailCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 订单状态不能为空 (订单号: %s)", rowNum, orderNo))
			g.Log().Warningf(ctx, "订单导入验证失败: 第%d行: 订单状态不能为空", rowNum)
			result.FailCount++
			continue
		}

		if existingOrders[orderNo] {
			existSkipCount++
			result.SkipCount++
			continue
		}

		data := g.Map{
			"shop_name":              truncateString(shopName, 50),
			"order_no":               truncateString(orderNo, 50),
			"product_name":           truncateString(productName, 255),
			"product_spec":           truncateString(getColFlex(cols, headerMap, "商品规格"), 100),
			"product_num":            parseIntOrDefault(getColFlex(cols, headerMap, "商品数量"), 1),
			"product_id":             truncateString(getColFlex(cols, headerMap, "商品ID"), 50),
			"seller_code":            truncateString(getColFlex(cols, headerMap, "商家编码"), 50),
			"product_price":          parseFloatOrDefault(getColMoney(cols, headerMap, "商品单价"), 0),
			"order_pay_amount":       parseFloatOrDefault(getColMoney(cols, headerMap, "订单应付金额"), 0),
			"shipping_fee":           parseFloatOrDefault(getColMoney(cols, headerMap, "运费"), 0),
			"total_discount":         parseFloatOrDefault(getColMoney(cols, headerMap, "优惠总金额"), 0),
			"platform_discount":      parseFloatOrDefault(getColMoney(cols, headerMap, "平台优惠"), 0),
			"seller_discount":        parseFloatOrDefault(getColMoney(cols, headerMap, "商家优惠"), 0),
			"talent_discount":        parseFloatOrDefault(getColMoney(cols, headerMap, "达人优惠"), 0),
			"seller_change_price":    parseFloatOrDefault(getColMoney(cols, headerMap, "商家改价"), 0),
			"pay_discount":           parseFloatOrDefault(getColMoney(cols, headerMap, "支付优惠"), 0),
			"red_deduction":          parseFloatOrDefault(getColMoney(cols, headerMap, "红包抵扣"), 0),
			"pay_type":               truncateString(getColFlex(cols, headerMap, "支付方式"), 20),
			"service_fee":            parseFloatOrDefault(getColMoney(cols, headerMap, "手续费"), 0),
			"receiver_name":          truncateString(getColFlex(cols, headerMap, "收件人"), 50),
			"receiver_phone":         truncateString(getColFlex(cols, headerMap, "收件人手机号"), 20),
			"province":               truncateString(getColFlex(cols, headerMap, "省"), 20),
			"city":                   truncateString(getColFlex(cols, headerMap, "市"), 20),
			"district":               truncateString(getColFlex(cols, headerMap, "区"), 255),
			"street":                 truncateString(getColFlex(cols, headerMap, "街道"), 100),
			"address_detail":         truncateString(getColFlex(cols, headerMap, "详细地址"), 255),
			"is_address_changed":     truncateString(getColFlex(cols, headerMap, "是否修改过地址"), 10),
			"address_change_time":    truncateString(getColFlex(cols, headerMap, "地址修改时段"), 50),
			"buyer_note":             truncateString(getColFlex(cols, headerMap, "买家留言"), 255),
			"order_submit_time":      parseDateTime(getColFlex(cols, headerMap, "订单提交时间")),
			"flag_color":             truncateString(getColFlex(cols, headerMap, "旗帜颜色"), 10),
			"seller_note":            truncateString(getColFlex(cols, headerMap, "商家备注"), 255),
			"order_finish_time":      parseDateTime(getColFlex(cols, headerMap, "订单完成时间")),
			"pay_finish_time":        parseDateTime(getColFlex(cols, headerMap, "支付完成时间")),
			"app_channel":            truncateString(getColFlex(cols, headerMap, "APP渠道"), 20),
			"flow_source":            truncateString(getColFlex(cols, headerMap, "流量来源"), 20),
			"order_status":           truncateString(orderStatus, 20),
			"promise_send_time":      parseDateTime(getColFlex(cols, headerMap, "承诺发货时间")),
			"order_type":             truncateString(getColFlex(cols, headerMap, "订单类型"), 20),
			"luban_page_id":          truncateString(getColFlex(cols, headerMap, "鲁班落地页ID"), 50),
			"talent_id":              truncateString(getColFlex(cols, headerMap, "达人ID"), 50),
			"talent_name":            truncateString(getColFlex(cols, headerMap, "达人昵称"), 50),
			"store_id":               truncateString(getColFlex(cols, headerMap, "所属门店ID"), 50),
			"after_sale_status":      truncateString(getColFlex(cols, headerMap, "售后状态"), 20),
			"cancel_reason":          truncateString(getColFlex(cols, headerMap, "取消原因"), 100),
			"appoint_send_time":      parseDateTime(getColFlex(cols, headerMap, "预约发货时间")),
			"warehouse_id":           truncateString(getColFlex(cols, headerMap, "仓库ID"), 50),
			"warehouse_name":         truncateString(getColFlex(cols, headerMap, "仓库名称"), 50),
			"is_safe_buy":            truncateString(getColFlex(cols, headerMap, "是否安心购"), 10),
			"ad_channel":             truncateString(getColFlex(cols, headerMap, "广告渠道"), 20),
			"flow_type":              truncateString(getColFlex(cols, headerMap, "流量类型"), 20),
			"flow_format":            truncateString(getColFlex(cols, headerMap, "流量体裁"), 20),
			"flow_channel":           truncateString(getColFlex(cols, headerMap, "流量渠道"), 20),
			"send_user":              truncateString(getColFlex(cols, headerMap, "发货主体"), 20),
			"send_user_detail":       truncateString(getColFlex(cols, headerMap, "发货主体明细"), 50),
			"send_time":              parseDateTime(getColFlex(cols, headerMap, "发货时间")),
			"reduce_discount":        parseFloatOrDefault(getColMoney(cols, headerMap, "降价类优惠"), 0),
			"platform_real_discount": parseFloatOrDefault(getColMoney(cols, headerMap, "平台实际承担优惠金额"), 0),
			"seller_real_discount":   parseFloatOrDefault(getColMoney(cols, headerMap, "商家实际承担优惠金额"), 0),
			"talent_real_discount":   parseFloatOrDefault(getColMoney(cols, headerMap, "达人实际承担优惠金额"), 0),
			"expect_arrive_time":     parseDateTime(getColFlex(cols, headerMap, "预计送达时间")),
		}

		batch = append(batch, data)
		existingOrders[orderNo] = true

		if len(batch) >= batchSize {
			g.Log().Debugf(ctx, "准备批量插入第%d批，共%d条数据", rowNum/batchSize, len(batch))
			if err := s.batchInsert(ctx, batch); err != nil {
				errMsg := fmt.Sprintf("批量插入失败（约第%d-%d行）: %v", rowNum-batchSize+1, rowNum, err)
				result.Errors = append(result.Errors, errMsg)
				g.Log().Errorf(ctx, "快手订单导入%s", errMsg)
				result.FailCount += len(batch)
			} else {
				g.Log().Infof(ctx, "批量插入成功：%d条（约第%d-%d行）", len(batch), rowNum-batchSize+1, rowNum)
				result.SuccessCount += len(batch)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := s.batchInsert(ctx, batch); err != nil {
			errMsg := fmt.Sprintf("批量插入失败（最后%d行）: %v", len(batch), err)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Errorf(ctx, "快手订单导入%s", errMsg)
			result.FailCount += len(batch)
		} else {
			g.Log().Infof(ctx, "批量插入成功：%d条（最后一批）", len(batch))
			result.SuccessCount += len(batch)
		}
	}

	if len(result.Errors) > 0 {
		g.Log().Errorf(ctx, "快手订单导入失败详情（共%d条错误）:", len(result.Errors))
		for i, errMsg := range result.Errors {
			g.Log().Errorf(ctx, "  错误%d: %s", i+1, errMsg)
			if i >= 200 {
				g.Log().Errorf(ctx, "  ... 还有 %d 条错误信息请查看返回结果", len(result.Errors)-200)
				break
			}
		}
	}

	if len(result.Errors) > 50 {
		result.Errors = append(result.Errors[:50], fmt.Sprintf("... 还有 %d 条错误信息未显示，请查看日志获取完整信息", len(result.Errors)-50))
	}

	g.Log().Infof(ctx, "快手订单导入完成: 成功%d条，跳过%d条（含空行%d+已存在%d+验证失败%d），失败%d条",
		result.SuccessCount, result.SkipCount, emptyRowCount, existSkipCount, validationFailCount, result.FailCount)
	return result, nil
}

func (s *KuaishouOrderService) batchInsert(ctx context.Context, batch []g.Map) error {
	if len(batch) == 0 {
		return nil
	}
	_, err := g.DB().Model(kuaishouOrdersTable).Ctx(ctx).Data(batch).Insert()
	return err
}

// getColMoney 获取金额列值，自动去除 ¥ 等货币符号
func getColMoney(cols []string, headerMap map[string]int, header string) string {
	val := getColFlex(cols, headerMap, header)
	val = strings.ReplaceAll(val, "¥", "")
	val = strings.ReplaceAll(val, "￥", "")
	val = strings.ReplaceAll(val, ",", "")
	return strings.TrimSpace(val)
}

// 注意：getCol / truncateString / cleanIllegalChars / parseIntOrDefault / parseFloatOrDefault / parseDateTime
// 已在 order.go 中定义，同一 package 内可直接使用
