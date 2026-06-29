package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-crawler/internal/model"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

type ShopService struct{}

func NewShopService() *ShopService {
	return &ShopService{}
}

// List 查询账号列表
func (s *ShopService) List(ctx context.Context, page, pageSize int, keyword, platform string) ([]g.Map, int64, error) {
	query := "SELECT * FROM account WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (uid LIKE ? OR account_name LIKE ? OR platform_nickname LIKE ? OR operator LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey, searchKey, searchKey)
	}
	if platform != "" {
		query += " AND platform = ?"
		params = append(params, platform)
	}

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	params = append(params, pageSize, offset)

	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	totalQuery := "SELECT COUNT(*) as total FROM account WHERE 1=1"
	totalParams := []interface{}{}
	if keyword != "" {
		totalQuery += " AND (uid LIKE ? OR account_name LIKE ? OR platform_nickname LIKE ? OR operator LIKE ?)"
		searchKey := "%" + keyword + "%"
		totalParams = append(totalParams, searchKey, searchKey, searchKey, searchKey)
	}
	if platform != "" {
		totalQuery += " AND platform = ?"
		totalParams = append(totalParams, platform)
	}

	resultTotal, err := g.DB().GetOne(ctx, totalQuery, totalParams...)
	if err != nil {
		return nil, 0, err
	}
	total := resultTotal["total"].Int64()

	shops := make([]g.Map, len(result))
	for i, row := range result {
		shops[i] = g.Map{
			"id":                row["id"],
			"platform":          row["platform"],
			"uid":               row["uid"],
			"accountName":       row["account_name"],
			"platformNickname":  row["platform_nickname"],
			"operator":          row["operator"],
			"companyId":         row["company_id"],
			"departmentId":      row["department_id"],
			"groupType":         row["group_type"],
			"memberName":        row["member_name"],
			"talentImage":       row["talent_image"],
			"bindStatus":        row["bind_status"],
			"status":            row["status"],
			"createdAt":         row["created_at"],
			"updatedAt":         row["updated_at"],
		}
	}

	return shops, total, nil
}

// Detail 获取账号详情
func (s *ShopService) Detail(ctx context.Context, id int) (g.Map, error) {
	if id <= 0 {
		return nil, gerror.New("账号ID不能为空")
	}

	result, err := g.DB().Model("account").Where("id", id).One()
	if err != nil {
		return nil, err
	}

	if result.IsEmpty() {
		return nil, nil
	}

	shop := g.Map{
		"id":                result["id"],
		"platform":          result["platform"],
		"uid":               result["uid"],
		"accountName":       result["account_name"],
		"platformNickname":  result["platform_nickname"],
		"operator":          result["operator"],
		"companyId":         result["company_id"],
		"departmentId":      result["department_id"],
		"groupType":         result["group_type"],
		"memberName":        result["member_name"],
		"talentImage":       result["talent_image"],
		"bindStatus":        result["bind_status"],
		"status":            result["status"],
		"createdAt":         result["created_at"],
		"updatedAt":         result["updated_at"],
	}

	return shop, nil
}

// Save 保存账号
func (s *ShopService) Save(ctx context.Context, req *model.ShopSaveReq) (int64, error) {
	// 检查同一平台下 uid 是否已存在（排除自己）
	query := g.DB().Model("account").Where("platform", req.Platform).Where("uid", req.Uid)
	if req.Id > 0 {
		query = query.WhereNot("id", req.Id)
	}
	count, err := query.Count()
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, gerror.New("该平台下账号ID已存在")
	}

	data := g.Map{
		"platform":            req.Platform,
		"uid": req.Uid,
		"account_name":        req.AccountName,
		"platform_nickname":   req.PlatformNickname,
		"operator":            req.Operator,
	}

	if req.Id > 0 {
		data["updated_at"] = time.Now()
		_, err := g.DB().Model("account").Where("id", req.Id).Data(data).Update()
		return int64(req.Id), err
	} else {
		data["created_at"] = time.Now()
		data["updated_at"] = time.Now()
		id, err := g.DB().Model("account").Data(data).InsertAndGetId()
		return id, err
	}
}

// Stats 账号平台统计
func (s *ShopService) Stats(ctx context.Context) (g.Map, error) {
	rows, err := g.DB().GetAll(ctx, "SELECT platform, COUNT(*) as cnt FROM account GROUP BY platform")
	if err != nil {
		return nil, err
	}

	stats := g.Map{"total": 0, "douyin": 0, "kuaishou": 0, "shipinhao": 0}
	for _, row := range rows {
		platform := row["platform"].String()
		cnt := row["cnt"].Int()
		stats[platform] = cnt
		stats["total"] = stats["total"].(int) + cnt
	}

	return stats, nil
}

// Delete 删除账号
func (s *ShopService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return gerror.New("账号ID不能为空")
	}

	_, err := g.DB().Model("account").Where("id", id).Delete()
	return err
}

// ExportTemplate 生成店铺导入模版
func (s *ShopService) ExportTemplate(ctx context.Context) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "账号信息"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 表头（与导出一致）
	headers := []string{"平台", "达人ID", "账号名称", "平台昵称", "运营者"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 表头样式
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// 示例数据（提示用户填写格式）
	sampleRow := []string{"douyin", "123456789", "示例账号名称", "示例昵称", "张三"}
	for i, val := range sampleRow {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, val)
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 15)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 20)
	f.SetColWidth(sheetName, "D", "D", 18)
	f.SetColWidth(sheetName, "E", "E", 12)

	// 添加"填写说明"工作表
	helpSheet := "填写说明"
	f.NewSheet(helpSheet)
	f.SetCellValue(helpSheet, "A1", "字段说明")
	f.SetCellValue(helpSheet, "A2", "【平台】必填，支持：douyin/抖音、kuaishou/快手、shipinhao/视频号")
	f.SetCellValue(helpSheet, "A3", "【达人ID】抖音必填，快手、视频号选填")
	f.SetCellValue(helpSheet, "A4", `【账号名称】必填，对比视频号平台："账号名称" 快手平台："账号名称"`)

	f.SetCellValue(helpSheet, "A5", "【运营者】必填，运营负责人姓名")
	f.SetCellValue(helpSheet, "A6", "【平台昵称】选填，如抖音号、快手号")
	f.SetCellValue(helpSheet, "A8", "注意：达人ID在同一平台下不能重复")
	helpStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
	f.SetCellStyle(helpSheet, "A1", "A1", helpStyle)
	f.SetColWidth(helpSheet, "A", "A", 60)

	// 删除默认的 Sheet1
	f.DeleteSheet("Sheet1")

	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, "shop_import_template.xlsx")
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存模版文件失败：%w", err)
	}
	return filename, nil
}

// ExportToExcel 导出账号数据到Excel
func (s *ShopService) ExportToExcel(ctx context.Context, keyword string) (string, error) {
	// 查询所有符合条件的店铺
	query := "SELECT * FROM account WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (uid LIKE ? OR account_name LIKE ? OR platform_nickname LIKE ? OR operator LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey, searchKey, searchKey)
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

	sheetName := "账号信息"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"平台", "达人ID", "账号名称", "平台昵称", "运营者"}
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
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row["platform"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row["uid"])
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), row["account_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), row["platform_nickname"])
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), row["operator"])
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 25)
	f.SetColWidth(sheetName, "C", "C", 25)
	f.SetColWidth(sheetName, "D", "D", 20)
	f.SetColWidth(sheetName, "E", "E", 15)

	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, fmt.Sprintf("shop_export_%s.xlsx", time.Now().Format("20060102150405")))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}

	return filename, nil
}

// ImportResult 导入结果
type ImportResult struct {
	TotalRows    int      `json:"totalRows"`    // 总行数
	SuccessCount int      `json:"successCount"` // 成功导入数
	SkipCount    int      `json:"skipCount"`    // 跳过数
	FailCount    int      `json:"failCount"`    // 失败数
	Errors       []string `json:"errors"`       // 错误信息列表
}

// ImportFromExcel 从Excel导入账号数据（优化版：流式处理+批量入库）
// 平台信息从Excel的"平台"列读取
func (s *ShopService) ImportFromExcel(ctx context.Context, filePath string) (*ImportResult, error) {
	g.Log().Infof(ctx, "开始导入账号文件: %s", filePath)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		g.Log().Errorf(ctx, "打开Excel文件失败: %s, 错误: %v", filePath, err)
		// 返回更详细的错误信息，包含文件路径
		return nil, fmt.Errorf("打开Excel文件失败(filePath=%s): %w", filePath, err)
	}
	defer f.Close()

	// 获取第一个工作表
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		g.Log().Error(ctx, "Excel文件中没有工作表")
		return nil, gerror.New("Excel文件中没有工作表")
	}

	// 使用流式读取，避免一次性加载整个文件到内存
	rows, err := f.Rows(sheets[0])
	if err != nil {
		g.Log().Errorf(ctx, "读取Excel数据失败: %v", err)
		return nil, fmt.Errorf("读取Excel数据失败：%w", err)
	}
	defer rows.Close()

	result := &ImportResult{
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

	// 验证表头 - 支持中英文表头
	headerMapping := map[string]string{
		"platform":            "platform",
		"平台":                  "platform",
		"达人ID":                "uid",
		"account_name":        "account_name",
		"抖音名称":                "account_name",
		"抖音号名称":               "account_name",
		"账号名称":                "account_name",
		"platform_nickname":   "platform_nickname",
		"抖音ID":               "platform_nickname",
		"抖音号":                 "platform_nickname",
		"平台昵称":                "platform_nickname",
		"operator":            "operator",
		"运营者":                 "operator",
		"运营负责人":               "operator",
		"运营组长":                "operator",
	}

	// 打印读取到的表头
	g.Log().Infof(ctx, "Excel表头: %v", headers)

	// 检查必要的表头是否存在（支持中英文）
	requiredHeaders := []string{"platform", "account_name", "operator"}
	headerIndex := make(map[string]int) // 记录每个字段在Excel中的列索引

	for _, required := range requiredHeaders {
		found := false
		for i, h := range headers {
			h = strings.TrimSpace(h) // 去除空格
			if mapped, ok := headerMapping[h]; ok && mapped == required {
				headerIndex[required] = i
				found = true
				break
			}
		}
		if !found {
			// 改进错误提示：显示实际读取到的表头
			g.Log().Errorf(ctx, "缺少必要的列: %s, 实际读取到的表头: %v", required, headers)
			return nil, gerror.New(fmt.Sprintf("缺少必要的列: %s\n实际读取到的表头: %v\n支持的中文列名：平台、账号名称、运营者", required, headers))
		}
	}

	// 可选列的映射
	optionalHeaders := []string{"uid", "platform_nickname"}
	for _, opt := range optionalHeaders {
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if mapped, ok := headerMapping[h]; ok && mapped == opt {
				headerIndex[opt] = i
				break
			}
		}
	}

	// 批量处理配置
	const batchSize = 1000
	batch := make([]g.Map, 0, batchSize)
	rowNum := 1 // 从第2行开始（第1行是表头）

	// 预加载所有已存在的 platform+account_name 到内存，避免每行都查询数据库
	existingKeys := make(map[string]bool)
	existingRows, _ := g.DB().Model("account").Fields("platform, account_name").All()
	for _, row := range existingRows {
		key := row["platform"].String() + "|" + row["account_name"].String()
		existingKeys[key] = true
	}

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

		// 获取各列的值（使用 headerIndex 动态获取）
		getCol := func(key string) string {
			if idx, ok := headerIndex[key]; ok && idx < len(cols) {
				return strings.TrimSpace(cols[idx])
			}
			return ""
		}

		platform := getCol("platform")
		platformAccountId := getCol("uid")
		accountName := getCol("account_name")
		operator := getCol("operator")
		platformNickname := getCol("platform_nickname")

		// 跳过空行
		if platformAccountId == "" && accountName == "" {
			result.SkipCount++
			continue
		}

		// 验证必填字段
		if platform == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: platform(平台)不能为空", rowNum))
			result.FailCount++
			continue
		}
		// 统一平台值为英文标识
		platform = normalizePlatform(platform)
		if accountName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: account_name(账号名称)不能为空", rowNum))
			result.FailCount++
			continue
		}
		if operator == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: operator(运营者/运营组长)不能为空", rowNum))
			result.FailCount++
			continue
		}

		// 检查同一平台下 account_name 是否已存在
		key := platform + "|" + accountName
		if existingKeys[key] {
			result.SkipCount++
			continue
		}
		existingKeys[key] = true // 标记为已存在，避免同一批次内重复

		// 添加到批次
		data := g.Map{
			"platform":            platform,
			"account_name":        accountName,
			"platform_nickname":   platformNickname,
			"operator":            operator,
			"created_at":          time.Now(),
			"updated_at":          time.Now(),
		}
		if platformAccountId != "" {
			data["uid"] = platformAccountId
		}
		batch = append(batch, data)

		// 达到批次大小，执行批量插入
		if len(batch) >= batchSize {
			success, skip, err := s.batchInsert(ctx, batch)
			result.SuccessCount += success
			result.SkipCount += skip
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("批量插入失败: %v", err))
				result.FailCount += len(batch) - success - skip
			}
			batch = batch[:0] // 清空批次
		}
	}

	// 处理剩余的数据
	if len(batch) > 0 {
		success, skip, err := s.batchInsert(ctx, batch)
		result.SuccessCount += success
		result.SkipCount += skip
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("批量插入失败: %v", err))
			result.FailCount += len(batch) - success - skip
		}
	}

	// 限制错误信息数量，避免返回过大
	if len(result.Errors) > 100 {
		result.Errors = append(result.Errors[:100], fmt.Sprintf("... 还有 %d 条错误信息未显示", len(result.Errors)-100))
	}

	return result, nil
}

// batchInsert 批量插入数据，返回（成功数，跳过数，错误）
func (s *ShopService) batchInsert(ctx context.Context, batch []g.Map) (int, int, error) {
	if len(batch) == 0 {
		return 0, 0, nil
	}
	success, skip := 0, 0
	for _, data := range batch {
		_, err := g.DB().Model("account").Ctx(ctx).Data(data).Insert()
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062") {
				skip++
				continue
			}
			return success, skip, err
		}
		success++
	}
	return success, skip, nil
}

// normalizePlatform 将中文平台名统一转为英文标识
func normalizePlatform(p string) string {
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

// ==================== 千川广告主映射 ====================

// ListQianchuanMappings 查询千川映射列表
func (s *ShopService) ListQianchuanMappings(ctx context.Context, page, pageSize int, accountId int, companyId int64) ([]g.Map, int64, error) {
	query := `SELECT aqm.*, a.account_name, a.platform, a.uid, sc.name AS company_name
		FROM account_qianchuan_mapping aqm
		LEFT JOIN account a ON aqm.account_id = a.id
		LEFT JOIN sys_company sc ON aqm.company_id = sc.id
		WHERE 1=1`
	params := []interface{}{}

	if accountId > 0 {
		query += " AND aqm.account_id = ?"
		params = append(params, accountId)
	}
	if companyId > 0 {
		query += " AND aqm.company_id = ?"
		params = append(params, companyId)
	}

	// 总数
	countQuery := "SELECT COUNT(*) as total FROM account_qianchuan_mapping aqm WHERE 1=1"
	countParams := []interface{}{}
	if accountId > 0 {
		countQuery += " AND aqm.account_id = ?"
		countParams = append(countParams, accountId)
	}
	if companyId > 0 {
		countQuery += " AND aqm.company_id = ?"
		countParams = append(countParams, companyId)
	}
	totalRow, _ := g.DB().GetOne(ctx, countQuery, countParams...)
	total := totalRow["total"].Int64()

	query += " ORDER BY aqm.id DESC LIMIT ? OFFSET ?"
	params = append(params, pageSize, (page-1)*pageSize)

	rows, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}

	result := make([]g.Map, len(rows))
	for i, row := range rows {
		result[i] = g.Map{
			"id":                     row["id"],
			"accountId":              row["account_id"],
			"companyId":              row["company_id"],
			"qianchuanAdvertiserId":  row["qianchuan_advertiser_id"],
			"qianchuanAdvertiserName": row["qianchuan_advertiser_name"],
			"accountName":            row["account_name"],
			"platform":               row["platform"],
			"uid":                    row["uid"],
			"companyName":            row["company_name"],
			"createdAt":              row["created_at"],
		}
	}
	return result, total, nil
}

// BindQianchuanAdvertiser 绑定千川广告主
func (s *ShopService) BindQianchuanAdvertiser(ctx context.Context, accountId int, companyId int64, advertiserId int64, advertiserName string) (int64, error) {
	// 检查账号是否存在
	account, _ := g.DB().Model("account").Where("id", accountId).One()
	if account.IsEmpty() {
		return 0, gerror.New("账号不存在")
	}

	// 检查是否已绑定（同一账号+公司组合唯一）
	existing, _ := g.DB().Model("account_qianchuan_mapping").
		Where("account_id", accountId).Where("company_id", companyId).One()
	if !existing.IsEmpty() {
		// 更新
		id := existing["id"].Int64()
		_, err := g.DB().Model("account_qianchuan_mapping").Where("id", id).Data(g.Map{
			"qianchuan_advertiser_id":   advertiserId,
			"qianchuan_advertiser_name": advertiserName,
			"updated_at":                time.Now(),
		}).Update()
		return id, err
	}

	// 新增
	data := g.Map{
		"account_id":                accountId,
		"company_id":                companyId,
		"qianchuan_advertiser_id":   advertiserId,
		"qianchuan_advertiser_name": advertiserName,
		"created_at":                time.Now(),
		"updated_at":                time.Now(),
	}
	id, err := g.DB().Model("account_qianchuan_mapping").Ctx(ctx).Data(data).InsertAndGetId()
	return id, err
}

// UnbindQianchuanAdvertiser 解绑千川广告主
func (s *ShopService) UnbindQianchuanAdvertiser(ctx context.Context, id int64) error {
	_, err := g.DB().Model("account_qianchuan_mapping").Where("id", id).Delete()
	return err
}

// GetAvailableQianchuanAdvertisers 获取当前OAuth授权下可用的千川广告主列表
func (s *ShopService) GetAvailableQianchuanAdvertisers(ctx context.Context) ([]g.Map, error) {
	// 从 qianchuan_token 获取最新的 advertiser_ids
	row, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil || row.IsEmpty() {
		return nil, gerror.New("未找到千川授权信息，请先完成巨量千川OAuth授权")
	}

	idsJSON := row["advertiser_ids"].String()
	if idsJSON == "" {
		return nil, gerror.New("千川授权未包含广告主信息")
	}

	var ids []int64
	json.Unmarshal([]byte(idsJSON), &ids)

	// 从 qianchuan_advertiser_daily_report 获取广告主名称（取最新数据）
	// 如果 report 表没有数据（比如 token 过期导致拉取失败），仍然返回 ID 列表
	rows, err := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
		Fields("DISTINCT advertiser_id, advertiser_name").
		WhereIn("advertiser_id", ids).
		All()
	if err != nil {
		// report 表查不出数据时，构造只有 ID 的列表
		rows = nil
	}

	// 构建 advertiser_id → advertiser_name 映射
	nameMap := make(map[int64]string)
	for _, row := range rows {
		nameMap[row["advertiser_id"].Int64()] = row["advertiser_name"].String()
	}

	// 如果 report 表没有某些广告主的名称，也加入列表（名称为空）
	var advertisers []g.Map
	seenIds := make(map[int64]bool)
	for _, row := range rows {
		advId := row["advertiser_id"].Int64()
		seenIds[advId] = true
		advertisers = append(advertisers, g.Map{
			"advertiserId":   advId,
			"advertiserName": row["advertiser_name"],
		})
	}
	for _, id := range ids {
		if !seenIds[id] {
			advertisers = append(advertisers, g.Map{
				"advertiserId":   id,
				"advertiserName": "",
			})
		}
	}

	// 构建已绑定的映射
	bindings, _ := g.DB().Model("account_qianchuan_mapping").Ctx(ctx).All()
	boundMap := make(map[int64]g.Map)
	for _, b := range bindings {
		advId := b["qianchuan_advertiser_id"].Int64()
		boundMap[advId] = g.Map{
			"accountId": b["account_id"],
			"companyId": b["company_id"],
		}
	}

	result := make([]g.Map, len(advertisers))
	for i, adv := range advertisers {
		advId := g.NewVar(adv["advertiserId"]).Int64()
		item := g.Map{
			"advertiserId":   advId,
			"advertiserName": adv["advertiserName"],
			"bound":          false,
			"accountId":      0,
			"companyId":      int64(0),
		}
		if bound, ok := boundMap[advId]; ok {
			item["bound"] = true
			item["accountId"] = bound["accountId"]
			item["companyId"] = bound["companyId"]
		}
		result[i] = item
	}

	return result, nil
}

// ==================== 账号组织归属自动补全 ====================

// OrgAttributionResult 账号归属补全结果
type OrgAttributionResult struct {
	AccountId    int    `json:"accountId"`
	AccountName  string `json:"accountName"`
	OldCompanyId int64  `json:"oldCompanyId"`
	NewCompanyId int64  `json:"newCompanyId"`
	TraderId     int64  `json:"traderId"`
	TraderName   string `json:"traderName"`
	Updated      bool   `json:"updated"`
}

// SyncAccountOrgAttribution 通过 trader_account_relation → trader 反向补全 account 的 company_id
// 解决 account 表 company_id/department_id 全为 0 导致按公司/部门维度统计丢失的问题
// dryRun=true 只预览不写库
func (s *ShopService) SyncAccountOrgAttribution(ctx context.Context, dryRun bool) ([]OrgAttributionResult, int, error) {
	// 查询所有 account → trader 关联（一个 account 可能关联多个 trader，取 company_id 最大的一个）
	rows, err := g.DB().Model("account a").
		InnerJoin("trader_account_relation tar", "a.id = tar.account_id").
		InnerJoin("trader t", "tar.trader_id = t.id").
		Fields("a.id, a.account_name, a.company_id AS old_company_id, a.department_id AS old_dept_id, MAX(t.id) AS trader_id, MAX(t.name) AS trader_name, MAX(t.company_id) AS trader_company_id, MAX(t.group_id) AS trader_group_id").
		Group("a.id, a.account_name, a.company_id, a.department_id").
		All()
	if err != nil {
		return nil, 0, err
	}

	results := make([]OrgAttributionResult, 0, len(rows))
	updatedCount := 0
	for _, row := range rows {
		r := OrgAttributionResult{
			AccountId:    row["id"].Int(),
			AccountName:  row["account_name"].String(),
			OldCompanyId: row["old_company_id"].Int64(),
			NewCompanyId: row["trader_company_id"].Int64(),
			TraderId:     row["trader_id"].Int64(),
			TraderName:   row["trader_name"].String(),
		}

		// 只更新 company_id 为 0 或与 trader 不一致的
		if r.OldCompanyId == r.NewCompanyId && r.OldCompanyId != 0 {
			r.Updated = false
			results = append(results, r)
			continue
		}

		r.Updated = true
		if !dryRun {
			data := g.Map{"company_id": r.NewCompanyId}
			// trader.group_id 若有效则同时补全 department_id
			if groupId := row["trader_group_id"].Int64(); groupId > 0 {
				data["department_id"] = groupId
			}
			_, err := g.DB().Model("account").Ctx(ctx).Where("id", r.AccountId).Data(data).Update()
			if err != nil {
				g.Log().Warningf(ctx, "[SyncOrgAttribution] 更新账号%d失败: %v", r.AccountId, err)
				r.Updated = false
			}
		}
		if r.Updated {
			updatedCount++
		}
		results = append(results, r)
	}

	return results, updatedCount, nil
}

// ==================== 千川广告主自动绑定 ====================

// AutoBindResult 自动绑定单个广告主的结果
type AutoBindResult struct {
	QianchuanAdvertiserId   int64  `json:"qianchuanAdvertiserId"`
	QianchuanAdvertiserName string `json:"qianchuanAdvertiserName"`
	MatchedType             string `json:"matchedType"` // store / account / none
	MatchedStoreId          int    `json:"matchedStoreId"`
	MatchedStoreName        string `json:"matchedStoreName"`
	MatchedAccountId        int    `json:"matchedAccountId"`
	MatchedAccountName      string `json:"matchedAccountName"`
	CompanyId               int64  `json:"companyId"`
	Bound                   bool   `json:"bound"`     // 是否已绑定过
	NewBound                bool   `json:"newBound"`  // 本次新绑定
	SkipReason              string `json:"skipReason"`
}

// AutoBindQianchuanAdvertisers 自动绑定千川广告主到 store/account
// 策略：
//  1. 从 qianchuan_token 取出所有千川广告主 ID
//  2. 调用千川 advertiser/info API 批量获取广告主名称（token 过期则名称为空，全部标记为 none）
//  3. 优先用名称匹配 store 表 store_name（店铺名通常与千川广告主名一致或互含）
//  4. store 未匹配上的，再用 account 表 account_name 匹配
//  5. 匹配成功的写入 account_qianchuan_mapping（已存在则跳过）
//  6. dryRun=true 时只返回匹配结果不写库，供人工复核
func (s *ShopService) AutoBindQianchuanAdvertisers(ctx context.Context, dryRun bool) ([]AutoBindResult, int, int, error) {
	// 1. 取千川广告主 ID 列表 + access_token
	tokenRow, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil || tokenRow.IsEmpty() {
		return nil, 0, 0, gerror.New("未找到千川授权信息，请先完成巨量千川OAuth授权")
	}
	accessToken := tokenRow["access_token"].String()
	idsJSON := tokenRow["advertiser_ids"].String()
	if idsJSON == "" {
		return nil, 0, 0, gerror.New("千川授权未包含广告主信息，请先调用 /api/qianchuan-poller/sync-advertisers 同步")
	}
	var advIDs []int64
	if err := json.Unmarshal([]byte(idsJSON), &advIDs); err != nil || len(advIDs) == 0 {
		return nil, 0, 0, gerror.New("千川广告主ID列表解析失败或为空")
	}

	// 2. 批量获取广告主名称（千川 advertiser/info，每批最多 100 个）
	nameMap := s.fetchAdvertiserNames(ctx, accessToken, advIDs)
	g.Log().Infof(ctx, "[AutoBind] 千川广告主共%d个，成功获取名称%d个", len(advIDs), len(nameMap))

	// 3. 预加载 store 和 account（抖音平台）
	stores, _ := g.DB().Model("store").Ctx(ctx).Where("platform", "douyin").All()
	accounts, _ := g.DB().Model("account").Ctx(ctx).Where("platform", "douyin").All()

	// 已绑定记录，避免重复
	boundRows, _ := g.DB().Model("account_qianchuan_mapping").Ctx(ctx).All()
	boundSet := make(map[int64]bool, len(boundRows))
	for _, b := range boundRows {
		boundSet[b["qianchuan_advertiser_id"].Int64()] = true
	}

	results := make([]AutoBindResult, 0, len(advIDs))
	newBoundCount := 0

	for _, advID := range advIDs {
		r := AutoBindResult{
			QianchuanAdvertiserId:   advID,
			QianchuanAdvertiserName: nameMap[advID],
		}
		if boundSet[advID] {
			r.Bound = true
			r.SkipReason = "已绑定，跳过"
			results = append(results, r)
			continue
		}
		if r.QianchuanAdvertiserName == "" {
			r.MatchedType = "none"
			r.SkipReason = "无法获取广告主名称，无法自动匹配（请检查token是否过期）"
			results = append(results, r)
			continue
		}

		// 4. 优先匹配 store
		if matched := matchByName(r.QianchuanAdvertiserName, stores, "store_name", "id"); matched != nil {
			r.MatchedType = "store"
			r.MatchedStoreId = matched["id"].Int()
			r.MatchedStoreName = matched["store_name"].String()
			// store 没有 company_id，用 0 占位，后续可人工补
			r.CompanyId = 0
			// 尝试从 account 表找同名账号补全 account_id/company_id
			for _, acc := range accounts {
				if acc["account_name"].String() == r.MatchedStoreName {
					r.MatchedAccountId = acc["id"].Int()
					r.MatchedAccountName = acc["account_name"].String()
					r.CompanyId = acc["company_id"].Int64()
					break
				}
			}
			if !dryRun {
				if _, err := s.BindQianchuanAdvertiser(ctx, r.MatchedAccountId, r.CompanyId, advID, r.QianchuanAdvertiserName); err == nil {
					r.NewBound = true
					newBoundCount++
				} else {
					r.SkipReason = "写入映射失败: " + err.Error()
				}
			} else {
				r.NewBound = true
				newBoundCount++
			}
			results = append(results, r)
			continue
		}

		// 5. store 未匹配，尝试 account
		if matched := matchByName(r.QianchuanAdvertiserName, accounts, "account_name", "id"); matched != nil {
			r.MatchedType = "account"
			r.MatchedAccountId = matched["id"].Int()
			r.MatchedAccountName = matched["account_name"].String()
			r.CompanyId = matched["company_id"].Int64()
			if !dryRun {
				if _, err := s.BindQianchuanAdvertiser(ctx, r.MatchedAccountId, r.CompanyId, advID, r.QianchuanAdvertiserName); err == nil {
					r.NewBound = true
					newBoundCount++
				} else {
					r.SkipReason = "写入映射失败: " + err.Error()
				}
			} else {
				r.NewBound = true
				newBoundCount++
			}
			results = append(results, r)
			continue
		}

		r.MatchedType = "none"
		r.SkipReason = "名称未匹配到任何 store 或 account"
		results = append(results, r)
	}

	return results, len(advIDs), newBoundCount, nil
}

// fetchAdvertiserNames 调用千川 advertiser/info 批量获取广告主名称
// 每批最多 100 个广告主ID，token 过期时返回空 map（不阻断流程）
func (s *ShopService) fetchAdvertiserNames(ctx context.Context, accessToken string, advIDs []int64) map[int64]string {
	nameMap := make(map[int64]string, len(advIDs))
	if accessToken == "" || len(advIDs) == 0 {
		return nameMap
	}

	const batchSize = 100
	client := g.Client()
	client.SetHeader("Access-Token", accessToken)

	for start := 0; start < len(advIDs); start += batchSize {
		end := start + batchSize
		if end > len(advIDs) {
			end = len(advIDs)
		}
		batch := advIDs[start:end]

		// 构造 advertiser_ids 参数（逗号分隔）
		idStrs := make([]string, len(batch))
		for i, id := range batch {
			idStrs[i] = fmt.Sprintf("%d", id)
		}
		apiURL := "https://api.oceanengine.com/open_api/2/advertiser/info/?advertiser_ids=" + strings.Join(idStrs, ",")
		resp, err := client.Get(ctx, apiURL)
		if err != nil {
			g.Log().Warningf(ctx, "[AutoBind] 获取广告主名称失败(batch %d-%d): %v", start, end, err)
			continue
		}
		body := resp.ReadAllString()
		resp.Close()

		var infoResp struct {
			Code *int64  `json:"code"`
			Msg  *string `json:"message"`
			Data []struct {
				Id   *int64  `json:"id"`
				Name *string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &infoResp); err != nil {
			g.Log().Warningf(ctx, "[AutoBind] 解析广告主名称响应失败: %v", err)
			continue
		}
		if infoResp.Code != nil && *infoResp.Code != 0 {
			g.Log().Warningf(ctx, "[AutoBind] advertiser/info 返回错误: code=%d msg=%s（token可能已过期）", *infoResp.Code, func() string { if infoResp.Msg != nil { return *infoResp.Msg }; return "" }())
			// token 过期等致命错误，停止后续批次
			break
		}
		for _, d := range infoResp.Data {
			if d.Id != nil && d.Name != nil {
				nameMap[*d.Id] = *d.Name
			}
		}
	}
	return nameMap
}

// matchByName 在记录集中按名称双向包含匹配
// nameField 为名称字段
func matchByName(target string, rows gdb.Result, nameField, _ string) gdb.Record {
	if target == "" || len(rows) == 0 {
		return nil
	}
	// 去除常见后缀做更宽松匹配：旗舰店/专营店/官方旗舰店
	targetClean := cleanShopName(target)
	for _, row := range rows {
		cand := row[nameField].String()
		if cand == "" {
			continue
		}
		if target == cand {
			return row
		}
		// 双向包含
		if strings.Contains(target, cand) || strings.Contains(cand, target) {
			return row
		}
		candClean := cleanShopName(cand)
		if targetClean != "" && candClean != "" && (targetClean == candClean ||
			strings.Contains(targetClean, candClean) || strings.Contains(candClean, targetClean)) {
			return row
		}
	}
	return nil
}

// cleanShopName 去除店铺名常见后缀，便于模糊匹配
func cleanShopName(name string) string {
	suffixes := []string{"官方旗舰店", "旗舰店", "专营店", "专卖店", "图书专营店", "进口图书专营店"}
	out := name
	for _, s := range suffixes {
		out = strings.TrimSuffix(out, s)
	}
	return strings.TrimSpace(out)
}
