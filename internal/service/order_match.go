package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"go-crawler/internal/model"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

type OrderMatchService struct{}

func NewOrderMatchService() *OrderMatchService {
	return &OrderMatchService{}
}

// 平台→生产表映射
var matchPlatformTableMap = map[string]string{
	"douyin":    "order_info",
	"kuaishou":  "kuaishou_orders",
	"shipinhao": "shipinhao_orders",
}

// 平台→回流时匹配的订单号字段（生产表中的字段）
var matchPlatformUpdateField = map[string]string{
	"douyin":    "main_order_no",
	"kuaishou":  "order_no",
	"shipinhao": "order_no",
}

// normalizePlatform 统一平台值
func normalizeMatchPlatform(p string) string {
	switch strings.TrimSpace(p) {
	case "抖音", "douyin":
		return "douyin"
	case "快手", "kuaishou":
		return "kuaishou"
	case "视频号", "shipinhao":
		return "shipinhao"
	default:
		return strings.ToLower(strings.TrimSpace(p))
	}
}

// ============================================
// 平台表头映射（按平台隔离）
// ============================================

// 普通订单 - 通用映射
var ordersHeaderCommon = map[string]string{
	"订单号":  "order_no",
	"店铺名称": "shop_name",
	"商品名称": "product_name",
	"商品数量": "product_num",
	"付款金额": "order_pay_amount",
	"订单状态": "order_status",
	"达人ID": "talent_id",
	"达人昵称": "talent_name",
	"付款时间": "pay_finish_time",
}

// 普通订单 - 平台专属映射
var ordersHeaderPlatform = map[string]map[string]string{
	"douyin": {
		"主订单编号":  "order_no",
		"子订单编号":  "sub_order_no",
		"店铺":     "shop_name",
		"选购商品":   "product_name",
		"订单应付金额": "order_pay_amount",
		"支付完成时间": "pay_finish_time",
	},
	"kuaishou": {
		"订单编号": "order_no",
		"店铺名":  "shop_name",
		"商品名":  "product_name",
		"下单时间": "pay_finish_time",
		"售后状态": "after_sale_status",
	},
	"shipinhao": {
		"订单编号":    "order_no",
		"商品数量（件）": "product_num",
		"实付金额":    "order_pay_amount",
		"下单时间":    "pay_finish_time",
		"售后状态":    "after_sale_status",
		"带货账号类型":  "account_type",
	},
}

// 精选联盟 - 通用映射
var jingxuanHeaderCommon = map[string]string{
	"订单号":  "order_no",
	"达人ID": "talent_id",
	"达人昵称": "talent_name",
	"商品名称": "product_name",
	"佣金比例": "commission_rate",
}

// 精选联盟 - 平台专属映射
var jingxuanHeaderPlatform = map[string]map[string]string{
	"douyin": {
		"订单id":   "order_no",
		"抖音/火山号": "talent_id",
		"作者账号":   "talent_name",
		"佣金率":    "commission_rate",
	},
	"kuaishou": {
		"订单编号": "order_no",
		"达人账号": "talent_id",
		"达人名称": "talent_name",
		"佣金率":  "commission_rate",
		"预估佣金": "commission_rate",
	},
	"shipinhao": {
		"订单编号":    "order_no",
		"推广带货者ID": "talent_id",
		"推广带货者昵称": "talent_name",
		"带货者佣金率":  "commission_rate",
	},
}

// mergeHeaderMapping 合并通用映射和平台专属映射
func mergeHeaderMapping(common map[string]string, platformMap map[string]map[string]string, platform string) map[string]string {
	merged := make(map[string]string)
	for k, v := range common {
		merged[k] = v
	}
	if pm, ok := platformMap[platform]; ok {
		for k, v := range pm {
			merged[k] = v
		}
	}
	return merged
}

// readRows 读取文件行（支持 Excel 和 CSV）
func readRows(filePath string) (headers []string, rows [][]string, err error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".csv" {
		return readCSV(filePath)
	}
	return readExcel(filePath)
}

func readCSV(filePath string) (headers []string, rows [][]string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("打开CSV文件失败: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("读取CSV数据失败: %w", err)
	}
	if len(allRows) < 2 {
		return nil, nil, fmt.Errorf("CSV文件中没有数据")
	}

	for i, h := range allRows[0] {
		allRows[0][i] = strings.TrimSpace(h)
	}
	return allRows[0], allRows[1:], nil
}

func readExcel(filePath string) (headers []string, rows [][]string, err error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("打开Excel文件失败: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("Excel文件中没有工作表")
	}

	rowsIter, err := f.Rows(sheets[0])
	if err != nil {
		return nil, nil, fmt.Errorf("读取Excel数据失败: %w", err)
	}
	defer rowsIter.Close()

	if !rowsIter.Next() {
		return nil, nil, fmt.Errorf("Excel文件中没有数据")
	}
	headers, err = rowsIter.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("读取表头失败: %w", err)
	}

	var dataRows [][]string
	for rowsIter.Next() {
		cols, err := rowsIter.Columns()
		if err != nil {
			continue
		}
		dataRows = append(dataRows, cols)
	}

	return headers, dataRows, nil
}

// ============================================
// 导入普通订单
// ============================================

func (s *OrderMatchService) ImportOrders(ctx context.Context, filePath, platform string) (*ImportResult, error) {
	platform = normalizeMatchPlatform(platform)
	if platform == "" || matchPlatformTableMap[platform] == "" {
		return nil, fmt.Errorf("无效的平台: %s，支持 douyin/kuaishou/shipinhao", platform)
	}

	g.Log().Infof(ctx, "[OrderMatch] 导入普通订单: platform=%s, file=%s", platform, filePath)

	headers, dataRows, err := readRows(filePath)
	if err != nil {
		return nil, err
	}

	headerMapping := mergeHeaderMapping(ordersHeaderCommon, ordersHeaderPlatform, platform)

	requiredHeaders := []string{"order_no"}
	headerIndex := make(map[string]int)
	for _, required := range requiredHeaders {
		found := false
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if mapped, ok := headerMapping[h]; ok && mapped == required {
				headerIndex[required] = i
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("缺少必要的列: %s, 实际表头: %v", required, headers)
		}
	}

	optionalHeaders := []string{"shop_name", "product_name", "product_num", "order_pay_amount", "order_status", "talent_id", "talent_name", "pay_finish_time", "after_sale_status", "account_type"}
	for _, opt := range optionalHeaders {
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if mapped, ok := headerMapping[h]; ok && mapped == opt {
				headerIndex[opt] = i
				break
			}
		}
	}

	// 预加载已存在的 (platform, order_no)
	existingKeys := make(map[string]bool)
	existingRows, _ := g.DB().Model("match_orders").Where("platform", platform).Fields("order_no").All()
	for _, row := range existingRows {
		existingKeys[platform+"|"+row["order_no"].String()] = true
	}

	result := &ImportResult{Errors: make([]string, 0)}
	const batchSize = 5000
	batch := make([]g.Map, 0, batchSize)

	for rowNum, cols := range dataRows {
		result.TotalRows++

		getCol := func(key string) string {
			if idx, ok := headerIndex[key]; ok && idx < len(cols) {
				return strings.TrimSpace(cols[idx])
			}
			return ""
		}

		orderNo := getCol("order_no")
		if orderNo == "" {
			result.SkipCount++
			continue
		}

		key := platform + "|" + orderNo
		if existingKeys[key] {
			result.SkipCount++
			continue
		}
		existingKeys[key] = true

		data := g.Map{
			"platform":          platform,
			"order_no":          orderNo,
			"shop_name":         getCol("shop_name"),
			"product_name":      getCol("product_name"),
			"product_num":       getCol("product_num"),
			"order_pay_amount":  getCol("order_pay_amount"),
			"order_status":      getCol("order_status"),
			"talent_id":         getCol("talent_id"),
			"talent_name":       getCol("talent_name"),
			"pay_finish_time":   getCol("pay_finish_time"),
			"after_sale_status": getCol("after_sale_status"),
			"account_type":      getCol("account_type"),
			"created_at":        time.Now(),
		}
		batch = append(batch, data)

		if len(batch) >= batchSize {
			success, skip, err := s.batchInsertTable(ctx, "match_orders", batch)
			result.SuccessCount += success
			result.SkipCount += skip
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("第%d行起批量插入失败: %v", rowNum+1, err))
				result.FailCount += len(batch) - success - skip
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		success, skip, err := s.batchInsertTable(ctx, "match_orders", batch)
		result.SuccessCount += success
		result.SkipCount += skip
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("批量插入失败: %v", err))
			result.FailCount += len(batch) - success - skip
		}
	}

	if len(result.Errors) > 100 {
		result.Errors = append(result.Errors[:100], fmt.Sprintf("... 还有 %d 条错误", len(result.Errors)-100))
	}

	return result, nil
}

// ============================================
// 导入精选联盟订单
// ============================================

func (s *OrderMatchService) ImportJingxuan(ctx context.Context, filePath, platform string) (*ImportResult, error) {
	platform = normalizeMatchPlatform(platform)
	if platform == "" || matchPlatformTableMap[platform] == "" {
		return nil, fmt.Errorf("无效的平台: %s，支持 douyin/kuaishou/shipinhao", platform)
	}

	g.Log().Infof(ctx, "[OrderMatch] 导入精选联盟: platform=%s, file=%s", platform, filePath)

	headers, dataRows, err := readRows(filePath)
	if err != nil {
		return nil, err
	}

	headerMapping := mergeHeaderMapping(jingxuanHeaderCommon, jingxuanHeaderPlatform, platform)

	requiredHeaders := []string{"order_no", "talent_id"}
	headerIndex := make(map[string]int)
	for _, required := range requiredHeaders {
		found := false
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if mapped, ok := headerMapping[h]; ok && mapped == required {
				headerIndex[required] = i
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("缺少必要的列: %s, 实际表头: %v", required, headers)
		}
	}

	optionalHeaders := []string{"talent_name", "product_name", "commission_rate"}
	for _, opt := range optionalHeaders {
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if mapped, ok := headerMapping[h]; ok && mapped == opt {
				headerIndex[opt] = i
				break
			}
		}
	}

	existingKeys := make(map[string]bool)
	existingRows, _ := g.DB().Model("match_jingxuan_orders").Where("platform", platform).Fields("order_no").All()
	for _, row := range existingRows {
		existingKeys[platform+"|"+row["order_no"].String()] = true
	}

	result := &ImportResult{Errors: make([]string, 0)}
	const batchSize = 5000
	batch := make([]g.Map, 0, batchSize)

	for rowNum, cols := range dataRows {
		result.TotalRows++

		getCol := func(key string) string {
			if idx, ok := headerIndex[key]; ok && idx < len(cols) {
				return strings.TrimSpace(cols[idx])
			}
			return ""
		}

		orderNo := getCol("order_no")
		talentId := getCol("talent_id")
		if orderNo == "" || talentId == "" {
			result.SkipCount++
			continue
		}

		key := platform + "|" + orderNo
		if existingKeys[key] {
			result.SkipCount++
			continue
		}
		existingKeys[key] = true

		data := g.Map{
			"platform":        platform,
			"order_no":        orderNo,
			"talent_id":       talentId,
			"talent_name":     getCol("talent_name"),
			"product_name":    getCol("product_name"),
			"commission_rate": getCol("commission_rate"),
			"created_at":      time.Now(),
		}
		batch = append(batch, data)

		if len(batch) >= batchSize {
			success, skip, err := s.batchInsertTable(ctx, "match_jingxuan_orders", batch)
			result.SuccessCount += success
			result.SkipCount += skip
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("第%d行起批量插入失败: %v", rowNum+1, err))
				result.FailCount += len(batch) - success - skip
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		success, skip, err := s.batchInsertTable(ctx, "match_jingxuan_orders", batch)
		result.SuccessCount += success
		result.SkipCount += skip
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("批量插入失败: %v", err))
			result.FailCount += len(batch) - success - skip
		}
	}

	if len(result.Errors) > 100 {
		result.Errors = append(result.Errors[:100], fmt.Sprintf("... 还有 %d 条错误", len(result.Errors)-100))
	}

	return result, nil
}

// ============================================
// 核心匹配逻辑（纯 SQL 批量操作，按平台隔离）
// ============================================

func (s *OrderMatchService) MatchOrders(ctx context.Context) (*model.OrderMatchResult, error) {
	g.Log().Info(ctx, "[OrderMatch] 开始执行订单匹配")

	result := &model.OrderMatchResult{
		Errors: make([]string, 0),
	}

	// 清理上次未完成的匹配结果（match_status=0 表示上次回流中断的残留数据）
	deleted, _ := g.DB().Exec(ctx, "DELETE FROM match_result WHERE match_status = 0")
	delCnt, _ := deleted.RowsAffected()
	if delCnt > 0 {
		g.Log().Infof(ctx, "[OrderMatch] 清理上次未完成的匹配结果: %d条", delCnt)
	}

	// 获取所有有数据的平台
	platforms, err := g.DB().Model("match_orders").Fields("DISTINCT platform").All()
	if err != nil {
		return nil, fmt.Errorf("查询平台列表失败: %v", err)
	}

	for _, p := range platforms {
		platform := p["platform"].String()
		if platform == "" {
			continue
		}

		tableName := matchPlatformTableMap[platform]
		updateField := matchPlatformUpdateField[platform]
		if tableName == "" || updateField == "" {
			g.Log().Warningf(ctx, "[OrderMatch] 未知平台: %s，跳过", platform)
			continue
		}

		g.Log().Infof(ctx, "[OrderMatch] 开始匹配平台: %s, 生产表: %s, 匹配字段: %s", platform, tableName, updateField)

		// Step 1: 匹配 — 以 match_orders 为基础表，按平台筛选后关联 match_jingxuan_orders
		var sqlMatch string
		switch platform {
		case "douyin":
			sqlMatch = `
				INSERT IGNORE INTO match_result (platform, order_no, talent_id, talent_name, match_status, matched_table, created_at)
				SELECT mo.platform, mo.order_no, mj.talent_id, mj.talent_name, 0, '', NOW()
				FROM match_orders mo
				INNER JOIN match_jingxuan_orders mj ON mo.platform = mj.platform AND mo.order_no = mj.order_no
				WHERE mo.platform = 'douyin'
				  AND mo.talent_id = 0`
		case "kuaishou":
			sqlMatch = `
				INSERT IGNORE INTO match_result (platform, order_no, talent_id, talent_name, match_status, matched_table, created_at)
				SELECT mo.platform, mo.order_no, mj.talent_id, mj.talent_name, 0, '', NOW()
				FROM match_orders mo
				INNER JOIN match_jingxuan_orders mj ON mo.platform = mj.platform AND mo.order_no = mj.order_no
				WHERE mo.platform = 'kuaishou'
				  AND mo.order_status IN ('已发货', '待发货')
				  AND (mo.after_sale_status = '' OR mo.after_sale_status = '退款关闭' OR mo.after_sale_status IS NULL)
				  AND (mo.talent_id = '' OR mo.talent_id IS NULL OR mo.talent_id = '0')
				  AND (mo.product_name LIKE 'zl%%' OR mo.product_name LIKE 'qz%%'
				    OR mo.product_name LIKE 'z1%%' OR mo.product_name LIKE 'z2%%' OR mo.product_name LIKE 'z3%%'
				    OR mo.product_name LIKE 'zl1%%' OR mo.product_name LIKE 'zl2%%' OR mo.product_name LIKE 'zl3%%')`
		case "shipinhao":
			sqlMatch = `
				INSERT IGNORE INTO match_result (platform, order_no, talent_id, talent_name, match_status, matched_table, created_at)
				SELECT mo.platform, mo.order_no, mj.talent_id, mj.talent_name, 0, '', NOW()
				FROM match_orders mo
				INNER JOIN match_jingxuan_orders mj ON mo.platform = mj.platform AND mo.order_no = mj.order_no
				WHERE mo.platform = 'shipinhao'
				  AND mo.order_status IN ('已发货', '待发货', '已完成', '完成', '部分发货', '待收货')
				  AND (mo.after_sale_status = '无' OR mo.after_sale_status = '用户取消申请' OR mo.after_sale_status IS NULL OR mo.after_sale_status = '')
				  AND (mo.talent_id = '' OR mo.talent_id IS NULL OR mo.talent_id = '0')
				  AND mo.account_type = '带货机构'`
		default:
			g.Log().Warningf(ctx, "[OrderMatch] 未知平台匹配规则: %s，跳过", platform)
			continue
		}
		res, err := g.DB().Exec(ctx, sqlMatch)
		if err != nil {
			g.Log().Errorf(ctx, "[OrderMatch] Step1 匹配失败[%s]: %v", platform, err)
			return nil, fmt.Errorf("匹配失败[%s]: %v", platform, err)
		}
		matched, _ := res.RowsAffected()
		result.Total += int(matched)
		g.Log().Infof(ctx, "[OrderMatch] Step1 匹配写入结果表: %d条", matched)

		// Step 2: 回流到生产订单表（抖音用 main_order_no，快手/视频号用 order_no）
		// 生产表(order_info等)与match_result的字符集排序规则不同，需要COLLATE
		backfillTalentCond := "(t.talent_id = '' OR t.talent_id IS NULL OR t.talent_id = '0')"
		if platform == "shipinhao" {
			backfillTalentCond = "(t.talent_id = '' OR t.talent_id IS NULL OR t.talent_id = '0' OR t.account_type = '带货机构' OR t.flow_source = '带货机构机构推广')"
		}
		sqlBackfill := fmt.Sprintf(`
			UPDATE %s t
			INNER JOIN match_result mr ON t.%s COLLATE utf8mb4_general_ci = mr.order_no COLLATE utf8mb4_general_ci
			SET t.talent_id = mr.talent_id, t.talent_name = mr.talent_name, t.reflow = 1
			WHERE mr.match_status = 0 AND mr.platform = ?
			  AND %s`, tableName, updateField, backfillTalentCond)
		res2, err2 := g.DB().Exec(ctx, sqlBackfill, platform)
		if err2 != nil {
			g.Log().Errorf(ctx, "[OrderMatch] Step2 回流失败[%s]: %v", platform, err2)
			return nil, fmt.Errorf("回流失败[%s]: %v", platform, err2)
		}
		backfilled, _ := res2.RowsAffected()
		g.Log().Infof(ctx, "[OrderMatch] Step2 回流生产表: %d条", backfilled)

		// Step 3: 回流成功后，标记 match_result.match_status=1（已回填）
		sqlMarkResult := fmt.Sprintf(`
			UPDATE match_result mr
			SET mr.match_status = 1, mr.matched_table = '%s'
			WHERE mr.match_status = 0 AND mr.platform = ?
			  AND EXISTS (SELECT 1 FROM %s t WHERE t.%s COLLATE utf8mb4_general_ci = mr.order_no COLLATE utf8mb4_general_ci AND t.talent_id COLLATE utf8mb4_general_ci = mr.talent_id COLLATE utf8mb4_general_ci AND t.reflow = 1)`,
			tableName, tableName, updateField)
		res3, err3 := g.DB().Exec(ctx, sqlMarkResult, platform)
		if err3 != nil {
			g.Log().Errorf(ctx, "[OrderMatch] Step3 标记match_result失败[%s]: %v", platform, err3)
		} else {
			marked3, _ := res3.RowsAffected()
			g.Log().Infof(ctx, "[OrderMatch] Step3 标记match_result已回填: %d条", marked3)
		}

		// 统计未匹配数量（匹配到但回流不成功的）
		unmatched := int(matched) - int(backfilled)
		if unmatched < 0 {
			unmatched = 0
		}

		result.Matched += int(backfilled)
		result.Unmatched += unmatched

		g.Log().Infof(ctx, "[OrderMatch] 平台%s匹配完成: 匹配%d, 回填成功%d, 未匹配%d",
			platform, matched, backfilled, unmatched)
	}

	g.Log().Infof(ctx, "[OrderMatch] 全部匹配完成: 总计%d, 回填成功%d, 未匹配%d",
		result.Total, result.Matched, result.Unmatched)

	return result, nil
}

// AutoMatchShipinhaoPulledOrders 将微信小店 API 拉取的带货机构订单自动套用视频号匹配规则。
// 小店接口里带货机构订单的 talent_id 往往是机构账号，不是真正的推广带货者；
// 也有部分订单只返回预估佣金、不返回来源信息。这里统一同步到 match_orders 并置空 talent_id，
// 再用 match_jingxuan_orders 回填真实达人。
func (s *OrderMatchService) AutoMatchShipinhaoPulledOrders(ctx context.Context, startTime, endTime string, storeId int, timeType string) (*model.OrderMatchResult, error) {
	result := &model.OrderMatchResult{Errors: make([]string, 0)}
	timeField := "order_submit_time"
	if strings.EqualFold(strings.TrimSpace(timeType), "update") {
		timeField = "order_update_time"
	}

	candidateWhere, candidateArgs := buildShipinhaoAutoMatchWhere("o", timeField, startTime, endTime, storeId)
	stageSQL := fmt.Sprintf(`
		INSERT INTO match_orders (
			platform, order_no, shop_name, product_name, product_num, order_pay_amount,
			order_status, after_sale_status, talent_id, talent_name, account_type, pay_finish_time, status, created_at
		)
		SELECT
			'shipinhao', o.order_no, o.shop_name, o.product_name, o.product_num, o.order_pay_amount,
			o.order_status, IFNULL(o.after_sale_status, ''), '', '', '带货机构',
			IFNULL(DATE_FORMAT(o.pay_finish_time, '%%Y-%%m-%%d %%H:%%i:%%s'), ''), 0, NOW()
		FROM shipinhao_orders o
		WHERE %s
		ON DUPLICATE KEY UPDATE
			shop_name = VALUES(shop_name),
			product_name = VALUES(product_name),
			product_num = VALUES(product_num),
			order_pay_amount = VALUES(order_pay_amount),
			order_status = VALUES(order_status),
			after_sale_status = VALUES(after_sale_status),
			talent_id = '',
			talent_name = '',
			account_type = '带货机构',
			pay_finish_time = VALUES(pay_finish_time),
			status = 0`, candidateWhere)
	stageRes, err := g.DB().Exec(ctx, stageSQL, candidateArgs...)
	if err != nil {
		return result, fmt.Errorf("同步微信小店待匹配订单失败: %w", err)
	}
	staged, _ := stageRes.RowsAffected()
	g.Log().Infof(ctx, "[OrderMatch] 微信小店带货机构订单同步到匹配暂存表: %d", staged)

	existsWhere, existsArgs := buildShipinhaoAutoMatchWhere("o", timeField, startTime, endTime, storeId)
	matchSQL := fmt.Sprintf(`
		INSERT IGNORE INTO match_result (platform, order_no, talent_id, talent_name, match_status, matched_table, created_at)
		SELECT mo.platform, mo.order_no, mj.talent_id, mj.talent_name, 0, '', NOW()
		FROM match_orders mo
		INNER JOIN match_jingxuan_orders mj
			ON mo.platform = mj.platform AND mo.order_no = mj.order_no
		WHERE mo.platform = 'shipinhao'
		  AND mo.account_type = '带货机构'
		  AND mo.order_status IN ('已发货', '待发货', '已完成', '完成', '部分发货', '待收货')
		  AND (mo.after_sale_status = '无' OR mo.after_sale_status = '用户取消申请' OR mo.after_sale_status IS NULL OR mo.after_sale_status = '')
		  AND (mo.talent_id = '' OR mo.talent_id IS NULL OR mo.talent_id = '0')
		  AND EXISTS (
			SELECT 1 FROM shipinhao_orders o
			WHERE o.order_no COLLATE utf8mb4_general_ci = mo.order_no COLLATE utf8mb4_general_ci
			  AND %s
		  )`, existsWhere)
	matchRes, err := g.DB().Exec(ctx, matchSQL, existsArgs...)
	if err != nil {
		return result, fmt.Errorf("微信小店订单匹配失败: %w", err)
	}
	matched, _ := matchRes.RowsAffected()
	result.Total = int(matched)
	g.Log().Infof(ctx, "[OrderMatch] 微信小店匹配结果写入: %d", matched)

	backfillWhere, backfillArgs := buildShipinhaoAutoMatchWhere("t", timeField, startTime, endTime, storeId)
	backfillSQL := fmt.Sprintf(`
		UPDATE shipinhao_orders t
		INNER JOIN match_result mr
			ON t.order_no COLLATE utf8mb4_general_ci = mr.order_no COLLATE utf8mb4_general_ci
		SET t.talent_id = mr.talent_id,
			t.talent_name = mr.talent_name,
			t.reflow = 1
		WHERE mr.match_status = 0
		  AND mr.platform = 'shipinhao'
		  AND (t.talent_id = '' OR t.talent_id IS NULL OR t.talent_id = '0' OR t.account_type = '带货机构' OR t.flow_source = '带货机构机构推广')
		  AND %s`, backfillWhere)
	backfillRes, err := g.DB().Exec(ctx, backfillSQL, backfillArgs...)
	if err != nil {
		return result, fmt.Errorf("微信小店订单回填达人失败: %w", err)
	}
	backfilled, _ := backfillRes.RowsAffected()
	result.Matched = int(backfilled)
	g.Log().Infof(ctx, "[OrderMatch] 微信小店订单回填达人: %d", backfilled)

	markWhere, markArgs := buildShipinhaoAutoMatchWhere("t", timeField, startTime, endTime, storeId)
	markSQL := fmt.Sprintf(`
		UPDATE match_result mr
		SET mr.match_status = 1,
			mr.matched_table = 'shipinhao_orders'
		WHERE mr.match_status = 0
		  AND mr.platform = 'shipinhao'
		  AND EXISTS (
			SELECT 1 FROM shipinhao_orders t
			WHERE t.order_no COLLATE utf8mb4_general_ci = mr.order_no COLLATE utf8mb4_general_ci
			  AND t.talent_id COLLATE utf8mb4_general_ci = mr.talent_id COLLATE utf8mb4_general_ci
			  AND t.reflow = 1
			  AND %s
		  )`, markWhere)
	if markRes, err := g.DB().Exec(ctx, markSQL, markArgs...); err != nil {
		g.Log().Warningf(ctx, "[OrderMatch] 标记微信小店匹配结果失败: %v", err)
	} else if marked, _ := markRes.RowsAffected(); marked > 0 {
		g.Log().Infof(ctx, "[OrderMatch] 标记微信小店匹配结果已回填: %d", marked)
	}

	result.Unmatched = result.Total - result.Matched
	if result.Unmatched < 0 {
		result.Unmatched = 0
	}
	return result, nil
}

func buildShipinhaoAutoMatchWhere(alias, timeField, startTime, endTime string, storeId int) (string, []interface{}) {
	where := fmt.Sprintf(`
		%s.order_no IS NOT NULL AND %s.order_no != ''
		AND (
			%s.account_type = '带货机构'
			OR %s.flow_source = '带货机构机构推广'
			OR (
				(%s.talent_id IS NULL OR %s.talent_id = '' OR %s.talent_id = '0')
				AND CAST(JSON_UNQUOTE(JSON_EXTRACT(%s.raw_json, '$.order_detail.settle_info.predict_commission_fee')) AS DECIMAL(12,2)) > 0
			)
		)
		AND (
			%s.wechat_order_status IN (20, 21, 30, 100)
			OR %s.order_status IN ('已发货', '待发货', '已完成', '完成', '部分发货', '待收货')
		)
		AND (%s.after_sale_status IS NULL OR %s.after_sale_status = '' OR %s.after_sale_status IN ('无', '用户取消申请'))`,
		alias, alias,
		alias,
		alias,
		alias, alias, alias,
		alias,
		alias, alias,
		alias, alias, alias,
	)
	args := make([]interface{}, 0, 3)
	if strings.TrimSpace(startTime) != "" {
		where += fmt.Sprintf(`
		AND %s.%s >= ?`, alias, timeField)
		args = append(args, startTime)
	}
	if strings.TrimSpace(endTime) != "" {
		where += fmt.Sprintf(`
		AND %s.%s < ?`, alias, timeField)
		args = append(args, endTime)
	}
	if storeId > 0 {
		where += fmt.Sprintf(`
		AND EXISTS (
			SELECT 1 FROM store s
			WHERE s.id = ? AND s.store_id COLLATE utf8mb4_general_ci = %s.store_id COLLATE utf8mb4_general_ci
		)`, alias)
		args = append(args, storeId)
	}
	return where, args
}

// ============================================
// 列表查询
// ============================================

func (s *OrderMatchService) OrderList(ctx context.Context, page, pageSize int, keyword, platform string) ([]g.Map, int64, error) {
	return s.paginateTable(ctx, "match_orders", page, pageSize, keyword, platform, "order_no, shop_name, product_name, talent_id")
}

func (s *OrderMatchService) JingxuanList(ctx context.Context, page, pageSize int, keyword, platform string) ([]g.Map, int64, error) {
	return s.paginateTable(ctx, "match_jingxuan_orders", page, pageSize, keyword, platform, "order_no, talent_id, talent_name")
}

// Stats 匹配结果统计
func (s *OrderMatchService) Stats(ctx context.Context, platform string) (g.Map, error) {
	query := g.DB().Model("match_result")
	matchedQuery := g.DB().Model("match_result").Where("match_status", 1)
	if platform != "" {
		query = query.Where("platform", platform)
		matchedQuery = matchedQuery.Where("platform", platform)
	}
	total, _ := query.Count()
	matched, _ := matchedQuery.Count()
	unmatched := total - matched

	return g.Map{
		"total":     total,
		"matched":   matched,
		"unmatched": unmatched,
	}, nil
}

// Export 导出数据到 Excel，返回文件路径
func (s *OrderMatchService) Export(ctx context.Context, table, platform, keyword, matchStatus string) (string, error) {
	query := g.DB().Model(table)
	if platform != "" {
		query = query.Where("platform", platform)
	}
	if keyword != "" {
		if table == "match_result" {
			query = query.Where("order_no LIKE ?", "%"+keyword+"%")
		} else {
			query = query.Where("(order_no LIKE ? OR talent_id LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
		}
	}
	if matchStatus != "" && table == "match_result" {
		query = query.Where("match_status", matchStatus)
	}

	rows, err := query.OrderDesc("id").All()
	if err != nil {
		return "", fmt.Errorf("查询数据失败: %v", err)
	}

	// 定义各表的列
	type colDef struct {
		key   string
		title string
	}
	columns := map[string][]colDef{
		"match_orders": {
			{"platform", "平台"}, {"order_no", "订单号"}, {"shop_name", "店铺名称"},
			{"product_name", "商品名称"}, {"product_num", "数量"}, {"order_pay_amount", "付款金额"},
			{"order_status", "订单状态"}, {"talent_id", "达人ID"}, {"talent_name", "达人昵称"},
			{"pay_finish_time", "付款时间"},
		},
		"match_jingxuan_orders": {
			{"platform", "平台"}, {"order_no", "订单号"}, {"talent_id", "达人ID"},
			{"talent_name", "达人昵称"}, {"product_name", "商品名称"}, {"commission_rate", "佣金比例"},
		},
		"match_result": {
			{"platform", "平台"}, {"order_no", "订单号"}, {"talent_id", "达人ID"},
			{"talent_name", "达人昵称"}, {"match_status", "匹配状态"}, {"matched_table", "匹配表"},
		},
	}

	cols, ok := columns[table]
	if !ok {
		return "", fmt.Errorf("未知表: %s", table)
	}

	f := excelize.NewFile()
	sheet := "Sheet1"

	// 写表头
	for i, col := range cols {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col.title)
	}

	// 写数据
	platformMap := map[string]string{"douyin": "抖音", "kuaishou": "快手", "shipinhao": "视频号"}
	matchStatusMap := map[string]string{"0": "未匹配", "1": "已匹配"}
	for rowIdx, row := range rows {
		for colIdx, col := range cols {
			val := row[col.key].String()
			if col.key == "platform" {
				if v, ok := platformMap[val]; ok {
					val = v
				}
			}
			if col.key == "match_status" {
				if v, ok := matchStatusMap[val]; ok {
					val = v
				}
			}
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	// 保存文件
	fileName := fmt.Sprintf("%s_%s.xlsx", table, time.Now().Format("20060102150405"))
	filePath := filepath.Join("uploads", fileName)
	os.MkdirAll("uploads", 0755)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return filePath, nil
}

func (s *OrderMatchService) ResultList(ctx context.Context, page, pageSize int, matchStatus, platform, keyword string) ([]g.Map, int64, error) {
	query := g.DB().Model("match_result")
	totalQuery := g.DB().Model("match_result")

	if matchStatus != "" {
		query = query.Where("match_status", matchStatus)
		totalQuery = totalQuery.Where("match_status", matchStatus)
	}
	if platform != "" {
		query = query.Where("platform", platform)
		totalQuery = totalQuery.Where("platform", platform)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("order_no LIKE ?", like)
		totalQuery = totalQuery.Where("order_no LIKE ?", like)
	}

	total, err := totalQuery.Count()
	if err != nil {
		return nil, 0, err
	}

	rows, err := query.Page(page, pageSize).OrderDesc("id").All()
	if err != nil {
		return nil, 0, err
	}

	items := make([]g.Map, len(rows))
	for i, row := range rows {
		items[i] = g.Map{
			"id":           row["id"],
			"platform":     row["platform"],
			"orderNo":      row["order_no"],
			"talentId":     row["talent_id"],
			"talentName":   row["talent_name"],
			"matchStatus":  row["match_status"],
			"matchedTable": row["matched_table"],
			"createdAt":    row["created_at"],
		}
	}

	return items, int64(total), nil
}

// ============================================
// 辅助方法
// ============================================

// toCamelCase snake_case → camelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func (s *OrderMatchService) paginateTable(ctx context.Context, table string, page, pageSize int, keyword, platform, searchFields string) ([]g.Map, int64, error) {
	query := g.DB().Model(table)
	totalQuery := g.DB().Model(table)

	if platform != "" {
		query = query.Where("platform", platform)
		totalQuery = totalQuery.Where("platform", platform)
	}

	if keyword != "" {
		fields := strings.Split(searchFields, ",")
		conditions := make([]string, 0)
		params := make([]interface{}, 0)
		for _, f := range fields {
			conditions = append(conditions, strings.TrimSpace(f)+" LIKE ?")
			params = append(params, "%"+keyword+"%")
		}
		whereClause := "(" + strings.Join(conditions, " OR ") + ")"
		query = query.Where(whereClause, params...)
		totalQuery = totalQuery.Where(whereClause, params...)
	}

	total, err := totalQuery.Count()
	if err != nil {
		return nil, 0, err
	}

	rows, err := query.Page(page, pageSize).OrderDesc("id").All()
	if err != nil {
		return nil, 0, err
	}

	items := make([]g.Map, len(rows))
	for i, row := range rows {
		item := g.Map{}
		for k, v := range row {
			item[toCamelCase(k)] = v
		}
		items[i] = item
	}

	return items, int64(total), nil
}

func (s *OrderMatchService) batchInsertTable(ctx context.Context, table string, batch []g.Map) (int, int, error) {
	if len(batch) == 0 {
		return 0, 0, nil
	}

	// 收集字段名（取第一条的 key 顺序）
	keys := make([]string, 0)
	for k := range batch[0] {
		keys = append(keys, k)
	}

	// 构建批量 INSERT SQL
	placeholders := "(" + strings.Repeat("?,", len(keys)-1) + "?)"
	sql := fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES ", table, strings.Join(keys, ","))

	args := make([]interface{}, 0, len(batch)*len(keys))
	for _, data := range batch {
		sql += placeholders + ","
		for _, k := range keys {
			args = append(args, data[k])
		}
	}
	sql = sql[:len(sql)-1] // 去掉末尾逗号

	res, err := g.DB().Exec(ctx, sql, args...)
	if err != nil {
		return 0, 0, err
	}

	affected, _ := res.RowsAffected()
	success := int(affected)
	skip := len(batch) - success

	if skip > 0 {
		// 记录被跳过的订单号（唯一键冲突，已存在）
		exampleOrders := make([]string, 0, min(10, len(batch)))
		for _, data := range batch {
			if v, ok := data["order_no"]; ok {
				exampleOrders = append(exampleOrders, fmt.Sprintf("%v", v))
			}
			if len(exampleOrders) >= 10 {
				break
			}
		}
		g.Log().Infof(ctx, "[OrderMatch] 跳过%d条重复记录, table=%s, 本批订单号示例: %v", skip, table, exampleOrders)
	}

	return success, skip, nil
}
