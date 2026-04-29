package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

type ShopService struct{}

func NewShopService() *ShopService {
	return &ShopService{}
}

// List 查询店铺列表
func (s *ShopService) List(ctx context.Context, page, pageSize int, keyword string) ([]g.Map, int64, error) {
	query := "SELECT * FROM shop WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (uid LIKE ? OR douyin_name LIKE ? OR douyin_id LIKE ? OR operation_leader LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey, searchKey, searchKey)
	}

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	params = append(params, pageSize, offset)

	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	totalQuery := "SELECT COUNT(*) as total FROM shop WHERE 1=1"
	totalParams := []interface{}{}
	if keyword != "" {
		totalQuery += " AND (uid LIKE ? OR douyin_name LIKE ? OR douyin_id LIKE ? OR operation_leader LIKE ?)"
		searchKey := "%" + keyword + "%"
		totalParams = append(totalParams, searchKey, searchKey, searchKey, searchKey)
	}

	resultTotal, err := g.DB().GetOne(ctx, totalQuery, totalParams...)
	if err != nil {
		return nil, 0, err
	}
	total := resultTotal["total"].Int64()

	shops := make([]g.Map, len(result))
	for i, row := range result {
		shops[i] = g.Map{
			"id":               row["id"],
			"uid":              row["uid"],
			"douyinName":       row["douyin_name"],
			"douyinId":         row["douyin_id"],
			"operationLeader":  row["operation_leader"],
			"accountOwnership": row["account_ownership"],
			"shopPlatform":     row["shop_platform"],
			"createdAt":        row["created_at"],
			"updatedAt":        row["updated_at"],
		}
	}

	return shops, total, nil
}

// Detail 获取店铺详情
func (s *ShopService) Detail(ctx context.Context, id int) (g.Map, error) {
	if id <= 0 {
		return nil, gerror.New("店铺ID不能为空")
	}

	result, err := g.DB().Model("shop").Where("id", id).One()
	if err != nil {
		return nil, err
	}

	if result.IsEmpty() {
		return nil, nil
	}

	shop := g.Map{
		"id":               result["id"],
		"uid":              result["uid"],
		"douyinName":       result["douyin_name"],
		"douyinId":         result["douyin_id"],
		"operationLeader":  result["operation_leader"],
		"accountOwnership": result["account_ownership"],
		"shopPlatform":     result["shop_platform"],
		"createdAt":        result["created_at"],
		"updatedAt":        result["updated_at"],
	}

	return shop, nil
}

// Save 保存店铺
func (s *ShopService) Save(ctx context.Context, req *model.ShopSaveReq) (int64, error) {
	// 检查UID是否已存在（排除自己）
	query := g.DB().Model("shop").Where("uid", req.Uid)
	if req.Id > 0 {
		query = query.WhereNot("id", req.Id)
	}
	count, err := query.Count()
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, gerror.New("店铺唯一标识已存在")
	}

	data := g.Map{
		"uid":               req.Uid,
		"douyin_name":       req.DouyinName,
		"douyin_id":         req.DouyinId,
		"operation_leader":  req.OperationLeader,
		"account_ownership": req.AccountOwnership,
		"shop_platform":     req.ShopPlatform,
	}

	if req.Id > 0 {
		data["updated_at"] = time.Now()
		_, err := g.DB().Model("shop").Where("id", req.Id).Data(data).Update()
		return int64(req.Id), err
	} else {
		data["created_at"] = time.Now()
		data["updated_at"] = time.Now()
		id, err := g.DB().Model("shop").Data(data).InsertAndGetId()
		return id, err
	}
}

// Delete 删除店铺
func (s *ShopService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return gerror.New("店铺ID不能为空")
	}

	_, err := g.DB().Model("shop").Where("id", id).Delete()
	return err
}

// ExportToExcel 导出店铺数据到Excel
func (s *ShopService) ExportToExcel(ctx context.Context, keyword string) (string, error) {
	// 查询所有符合条件的店铺
	query := "SELECT * FROM shop WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (uid LIKE ? OR douyin_name LIKE ? OR douyin_id LIKE ? OR operation_leader LIKE ?)"
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

	sheetName := "店铺信息"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"uid", "douyin_name", "douyin_id", "operation_leader", "account_ownership", "shop_platform"}
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
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row["uid"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row["douyin_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), row["douyin_id"])
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), row["operation_leader"])
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), row["account_ownership"])
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), row["shop_platform"])
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 25)
	f.SetColWidth(sheetName, "C", "C", 20)
	f.SetColWidth(sheetName, "D", "D", 15)
	f.SetColWidth(sheetName, "E", "E", 20)
	f.SetColWidth(sheetName, "F", "F", 15)

	// 保存文件
	exportDir := "./exports"
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

// ImportFromExcel 从Excel导入店铺数据（优化版：流式处理+批量入库）
func (s *ShopService) ImportFromExcel(ctx context.Context, filePath string) (*ImportResult, error) {
	g.Log().Infof(ctx, "开始导入店铺文件: %s", filePath)

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
		"uid":               "uid",
		"UID":               "uid",
		"店铺唯一标识":             "uid",
		"douyin_name":       "douyin_name",
		"抖音名称":               "douyin_name",
		"抖音号名称":              "douyin_name",
		"douyin_id":         "douyin_id",
		"抖音ID":              "douyin_id",
		"抖音号":                "douyin_id",
		"operation_leader":   "operation_leader",
		"运营负责人":              "operation_leader",
		"运营组长":               "operation_leader",
		"account_ownership": "account_ownership",
		"账号归属":               "account_ownership",
		"shop_platform":      "shop_platform",
		"店铺平台":               "shop_platform",
	}

	// 打印读取到的表头
	g.Log().Infof(ctx, "Excel表头: %v", headers)

	// 检查必要的表头是否存在（支持中英文）
	// 注意：shop_platform 在数据库中有默认值'抖音'，所以不是必须的
	requiredHeaders := []string{"uid", "douyin_name", "operation_leader"}
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
			return nil, gerror.New(fmt.Sprintf("缺少必要的列: %s\n实际读取到的表头: %v\n支持的中文列名：店铺唯一标识(UID)、抖音名称、运营组长", required, headers))
		}
	}

	// 可选列的映射（包括 shop_platform，数据库有默认值）
	optionalHeaders := []string{"douyin_id", "account_ownership", "shop_platform"}
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

	// 预加载所有已存在的UID到内存，避免每行都查询数据库
	existingUIDs := make(map[string]bool)
	existingRows, _ := g.DB().Model("shop").Fields("uid").All()
	for _, row := range existingRows {
		existingUIDs[row["uid"].String()] = true
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

		uid := getCol("uid")
		douyinName := getCol("douyin_name")
		operationLeader := getCol("operation_leader")
		shopPlatform := getCol("shop_platform")

		// 跳过空行
		if uid == "" && douyinName == "" {
			result.SkipCount++
			continue
		}

		// 验证必填字段
		if uid == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: uid(店铺唯一标识/UID)不能为空", rowNum))
			result.FailCount++
			continue
		}
		if douyinName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: douyin_name(抖音名称/抖音号名称)不能为空", rowNum))
			result.FailCount++
			continue
		}
		if operationLeader == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: operation_leader(运营负责人/运营组长)不能为空", rowNum))
			result.FailCount++
			continue
		}
		// shopPlatform 可选，如果为空则使用数据库默认值'抖音'
		if shopPlatform == "" {
			shopPlatform = "抖音"
		}

		// 检查UID是否已存在
		if existingUIDs[uid] {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: uid=%s 已存在", rowNum, uid))
			result.SkipCount++
			continue
		}

		// 添加到批次
		data := g.Map{
			"uid":               uid,
			"douyin_name":       douyinName,
			"douyin_id":         getCol("douyin_id"),
			"operation_leader":  operationLeader,
			"account_ownership": getCol("account_ownership"),
			"shop_platform":     shopPlatform,
			"created_at":        time.Now(),
			"updated_at":        time.Now(),
		}
		batch = append(batch, data)
		existingUIDs[uid] = true // 标记为已存在，避免同一批次内重复

		// 达到批次大小，执行批量插入
		if len(batch) >= batchSize {
			if err := s.batchInsert(ctx, batch); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("批量插入失败: %v", err))
				result.FailCount += len(batch)
			} else {
				result.SuccessCount += len(batch)
			}
			batch = batch[:0] // 清空批次
		}
	}

	// 处理剩余的数据
	if len(batch) > 0 {
		if err := s.batchInsert(ctx, batch); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("批量插入失败: %v", err))
			result.FailCount += len(batch)
		} else {
			result.SuccessCount += len(batch)
		}
	}

	// 限制错误信息数量，避免返回过大
	if len(result.Errors) > 100 {
		result.Errors = append(result.Errors[:100], fmt.Sprintf("... 还有 %d 条错误信息未显示", len(result.Errors)-100))
	}

	return result, nil
}

// batchInsert 批量插入数据
func (s *ShopService) batchInsert(ctx context.Context, batch []g.Map) error {
	if len(batch) == 0 {
		return nil
	}
	_, err := g.DB().Model("shop").Ctx(ctx).Data(batch).Insert()
	return err
}
