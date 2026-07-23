package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

type GoodsService struct{}

func NewGoodsService() *GoodsService {
	return &GoodsService{}
}

// List 查询商品列表
func (s *GoodsService) List(ctx context.Context, page, pageSize int, keyword string, status int) ([]g.Map, int64, error) {
	// 只查询状态1和2，3表示删除
	query := "SELECT * FROM goods WHERE status IN (1,2)"
	params := []interface{}{}

	// 按状态筛选（0表示不限）
	if status > 0 {
		query = "SELECT * FROM goods WHERE status = ?"
		params = append(params, status)
	}

	if keyword != "" {
		query += " AND (goods_name LIKE ? OR goods_code LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey)
	}

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	params = append(params, pageSize, offset)

	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	totalQuery := "SELECT COUNT(*) as total FROM goods WHERE status IN (1,2)"
	totalParams := []interface{}{}
	
	// 按状态筛选（0表示不限）
	if status > 0 {
		totalQuery = "SELECT COUNT(*) as total FROM goods WHERE status = ?"
		totalParams = append(totalParams, status)
	}
	
	if keyword != "" {
		totalQuery += " AND (goods_name LIKE ? OR goods_code LIKE ?)"
		searchKey := "%" + keyword + "%"
		totalParams = append(totalParams, searchKey, searchKey)
	}

	resultTotal, err := g.DB().GetOne(ctx, totalQuery, totalParams...)
	if err != nil {
		return nil, 0, err
	}
	total := resultTotal["total"].Int64()

	goods := make([]g.Map, len(result))
	for i, row := range result {
		goods[i] = g.Map{
			"id":         row["id"],
			"goodsName":  row["goods_name"],
			"goodsCode":  row["goods_code"],
			"basePrice":  row["base_price"],
			"status":     row["status"],
			"createTime": row["create_time"],
			"updateTime": row["update_time"],
		}
	}

	return goods, total, nil
}

// Detail 获取商品详情
func (s *GoodsService) Detail(ctx context.Context, id int) (g.Map, error) {
	if id <= 0 {
		return nil, gerror.New("商品ID不能为空")
	}

	result, err := g.DB().Model("goods").Where("id", id).Where("status != ?", 3).One()
	if err != nil {
		return nil, err
	}

	if result.IsEmpty() {
		return nil, nil
	}

	goods := g.Map{
		"id":         result["id"],
		"goodsName":  result["goods_name"],
		"goodsCode":  result["goods_code"],
		"basePrice":  result["base_price"],
		"status":     result["status"],
		"createTime": result["create_time"],
		"updateTime": result["update_time"],
	}

	return goods, nil
}

// Save 保存商品
func (s *GoodsService) Save(ctx context.Context, req *model.GoodsSaveReq) (int64, error) {
	// 检查商品编码是否已存在（排除自己）
	query := g.DB().Model("goods").Where("goods_code", req.GoodsCode).Where("status != ?", 3)
	if req.Id > 0 {
		query = query.WhereNot("id", req.Id)
	}
	count, err := query.Count()
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, gerror.New("商品编码已存在")
	}

	data := g.Map{
		"goods_name": req.GoodsName,
		"goods_code": req.GoodsCode,
		"base_price": req.BasePrice,
		"status":     req.Status,
	}

	if req.Id > 0 {
		data["update_time"] = time.Now()
		_, err := g.DB().Model("goods").Where("id", req.Id).Data(data).Update()
		return int64(req.Id), err
	} else {
		data["create_time"] = time.Now()
		data["update_time"] = time.Now()
		id, err := g.DB().Model("goods").Data(data).InsertAndGetId()
		return id, err
	}
}

// Delete 删除商品（软删除）
func (s *GoodsService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return gerror.New("商品ID不能为空")
	}

	_, err := g.DB().Model("goods").Where("id", id).Data(g.Map{
		"status":      3,
		"update_time": time.Now(),
	}).Update()
	return err
}

// ExportToExcel 导出商品数据到Excel
func (s *GoodsService) ExportToExcel(ctx context.Context, keyword string) (string, error) {
	// 查询所有符合条件的商品
	query := "SELECT * FROM goods WHERE status != 3"
	params := []interface{}{}

	if keyword != "" {
		query += " AND (goods_name LIKE ? OR goods_code LIKE ?)"
		searchKey := "%" + keyword + "%"
		params = append(params, searchKey, searchKey)
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

	sheetName := "商品信息"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"商品名称", "商品编码", "商品底价", "状态"}
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
	statusMap := map[int]string{1: "正常", 2: "下架", 3: "删除"}
	for i, row := range result {
		rowNum := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row["goods_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row["goods_code"])
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), row["base_price"])
		status := row["status"].Int()
		statusStr := statusMap[status]
		if statusStr == "" {
			statusStr = "未知"
		}
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), statusStr)
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 30)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 15)
	f.SetColWidth(sheetName, "D", "D", 10)

	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, fmt.Sprintf("goods_export_%s.xlsx", time.Now().Format("20060102150405")))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}

	return filename, nil
}

// ImportResult 导入结果
type GoodsImportResult struct {
	TotalRows            int      `json:"totalRows"`
	SuccessCount         int      `json:"successCount"`
	SkipCount            int      `json:"skipCount"`
	FailCount            int      `json:"failCount"`
	Errors               []string `json:"errors"`
	PendingDeleteCount   int      `json:"pendingDeleteCount"`             // 全量覆盖待删除数（dryRun 预览）
	PendingDeleteSamples []string `json:"pendingDeleteSamples,omitempty"` // 待删除商品示例（dryRun 预览）
	DeletedCount         int      `json:"deletedCount"`                   // 全量覆盖实际软删除数
}

// ImportFromExcel 从Excel导入商品数据
//
// 全量覆盖（fullReplace=true）：以 Excel 为准，库里活跃商品但不在 Excel 中的将被软删除（status=3）。
// dryRun=true：仅解析 Excel 并计算待删除数，不执行任何写入/删除，返回 PendingDeleteCount 预览。
func (s *GoodsService) ImportFromExcel(ctx context.Context, filePath string, fullReplace bool, dryRun bool) (*GoodsImportResult, error) {
	g.Log().Infof(ctx, "开始导入商品文件: %s fullReplace=%v dryRun=%v", filePath, fullReplace, dryRun)

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

	result := &GoodsImportResult{
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

	g.Log().Infof(ctx, "商品Excel表头: %v", headers)

	// 验证表头
	requiredHeaders := []string{"商品名称", "商品编码"}
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
		// 支持"商品单价"作为"商品底价"的别名
		if strings.TrimSpace(h) == "商品单价" {
			headerMap["商品底价"] = i
		}
	}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			g.Log().Errorf(ctx, "Excel表头缺少必填字段: %s, 实际表头: %v", required, headers)
			return nil, gerror.New(fmt.Sprintf("Excel表头缺少必填字段: %s\n实际读取到的表头: %v\n支持的列名：商品名称、商品编码、商品单价（或商品底价）", required, headers))
		}
	}

	// 批量处理配置
	const batchSize = 1000
	batch := make([]g.Map, 0, batchSize)
	rowNum := 1

	// 预加载已存在的商品（编码+名称+底价）
	type existingGoods struct {
		GoodsCode  string
		GoodsName  string
		BasePrice float64
	}
	existingGoodsMap := make(map[string]map[string]float64) // key: goods_code, value: map[goods_name]base_price
	existingRows, _ := g.DB().Model("goods").Where("status != ?", 3).Fields("goods_code, goods_name, base_price").All()
	for _, row := range existingRows {
		code := row["goods_code"].String()
		name := row["goods_name"].String()
		price := row["base_price"].Float64()
		if existingGoodsMap[code] == nil {
			existingGoodsMap[code] = make(map[string]float64)
		}
		existingGoodsMap[code][name] = price
	}

	// 流式读取并批量处理
	excelCodes := make(map[string]struct{}) // 本次 Excel 出现过的所有商品编码，用于全量覆盖判定
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
		if len(cols) < 2 || (getCol(cols, headerMap, "商品名称") == "" && getCol(cols, headerMap, "商品编码") == "") {
			result.SkipCount++
			continue
		}

		// 验证必填字段
		goodsName := getCol(cols, headerMap, "商品名称")
		goodsCode := getCol(cols, headerMap, "商品编码")

		if goodsName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 商品名称不能为空", rowNum))
			result.FailCount++
			continue
		}
		if goodsCode == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 商品编码不能为空", rowNum))
			result.FailCount++
			continue
		}

		// 解析底价（支持"商品底价"或"商品单价"）
		basePrice := 0.0
		priceHeader := "商品底价"
		if _, exists := headerMap["商品单价"]; exists {
			priceHeader = "商品单价"
		}
		if idx, exists := headerMap[priceHeader]; exists && idx < len(cols) {
			basePrice = parsePrice(getCol(cols, headerMap, priceHeader))
		}

		// 检查商品编码是否已存在
		excelCodes[goodsCode] = struct{}{}
		if dryRun {
			// dryRun：仅统计将新增/更新/跳过的数量，不写入
			if goodsMap, exists := existingGoodsMap[goodsCode]; exists {
				if existingPrice, nameExists := goodsMap[goodsName]; nameExists {
					if existingPrice == basePrice {
						result.SkipCount++
					} else {
						result.SuccessCount++
					}
				} else {
					result.SuccessCount++
				}
			} else {
				result.SuccessCount++
				existingGoodsMap[goodsCode] = map[string]float64{goodsName: basePrice}
			}
			continue
		}
		if goodsMap, exists := existingGoodsMap[goodsCode]; exists {
			if existingPrice, nameExists := goodsMap[goodsName]; nameExists {
				// 商品编码和名称都相同
				if existingPrice == basePrice {
					// 价格也相同，跳过
					result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 商品编码=%s, 商品名称=%s 已存在且价格相同，跳过", rowNum, goodsCode, goodsName))
					result.SkipCount++
					continue
				} else {
					// 价格不同，更新
					data := g.Map{
						"base_price":  basePrice,
						"update_time": time.Now(),
					}
					_, err := g.DB().Model("goods").Where("goods_code", goodsCode).Where("goods_name", goodsName).Data(data).Update()
					if err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 更新商品失败: %v", rowNum, err))
						result.FailCount++
					} else {
						result.SuccessCount++
						existingGoodsMap[goodsCode][goodsName] = basePrice
					}
					continue
				}
			} else {
				// 商品编码存在但名称不同，更新名称和价格
				data := g.Map{
					"goods_name":  goodsName,
					"base_price":  basePrice,
					"update_time": time.Now(),
				}
				_, err := g.DB().Model("goods").Where("goods_code", goodsCode).Data(data).Update()
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 更新商品失败: %v", rowNum, err))
					result.FailCount++
				} else {
					result.SuccessCount++
					// 更新内存：删除旧名称映射，添加新名称
					delete(existingGoodsMap[goodsCode], "")
					for oldName := range goodsMap {
						if oldName != goodsName {
							delete(goodsMap, oldName)
						}
					}
					existingGoodsMap[goodsCode][goodsName] = basePrice
				}
				continue
			}
		}

		// 商品编码不存在或名称不同，新增
		// 添加到批次
		data := g.Map{
			"goods_name":  goodsName,
			"goods_code":  goodsCode,
			"base_price":  basePrice,
			"status":      1,
			"create_time": time.Now(),
			"update_time": time.Now(),
		}
		batch = append(batch, data)
		// 更新内存
		if existingGoodsMap[goodsCode] == nil {
			existingGoodsMap[goodsCode] = make(map[string]float64)
		}
		existingGoodsMap[goodsCode][goodsName] = basePrice


		// 达到批次大小，执行批量插入
		if len(batch) >= batchSize {
			if err := s.batchInsert(ctx, batch); err != nil {
				errMsg := fmt.Sprintf("批量插入失败（约第%d-%d行）: %v", rowNum-batchSize+1, rowNum, err)
				result.Errors = append(result.Errors, errMsg)
				g.Log().Errorf(ctx, "商品导入%s", errMsg)
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
			g.Log().Errorf(ctx, "商品导入%s", errMsg)
			result.FailCount += len(batch)
		} else {
			g.Log().Infof(ctx, "批量插入成功：%d条（最后一批）", len(batch))
			result.SuccessCount += len(batch)
		}
	}

	// 全量覆盖：软删除库里活跃但不在 Excel 中的商品
	if fullReplace {
		pendingDeletes, pendingSamples := s.collectGoodsToDelete(ctx, excelCodes)
		result.PendingDeleteCount = len(pendingDeletes)
		result.PendingDeleteSamples = pendingSamples
		if !dryRun && len(pendingDeletes) > 0 {
			deleted, delErr := s.softDeleteGoods(ctx, pendingDeletes)
			if delErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("软删除多余商品失败: %v", delErr))
			} else {
				result.DeletedCount = deleted
			}
		}
	}

	// 记录所有错误信息到日志
	if len(result.Errors) > 0 {
		g.Log().Errorf(ctx, "商品导入失败详情（共%d条错误）:", len(result.Errors))
		for i, errMsg := range result.Errors {
			g.Log().Errorf(ctx, "  错误%d: %s", i+1, errMsg)
			if i >= 200 {
				g.Log().Errorf(ctx, "  ... 还有 %d 条错误信息请查看返回结果", len(result.Errors)-200)
				break
			}
		}
	}

	// 限制返回给前端的错误信息数量
	if len(result.Errors) > 50 {
		result.Errors = append(result.Errors[:50], fmt.Sprintf("... 还有 %d 条错误信息未显示，请查看日志获取完整信息", len(result.Errors)-50))
	}

	g.Log().Infof(ctx, "商品导入完成: 成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount)

	return result, nil
}

// collectGoodsToDelete 找出库里所有活跃商品中、不在本次 Excel codes 里的记录。
// 返回待删除 id 列表与示例商品名（前 10 个，用于前端预览）。
func (s *GoodsService) collectGoodsToDelete(ctx context.Context, excelCodes map[string]struct{}) ([]int64, []string) {
	rows, err := g.DB().Model("goods").Where("status != ?", 3).Fields("id, goods_code, goods_name").All()
	if err != nil {
		g.Log().Warningf(ctx, "[collectGoodsToDelete] 查询活跃商品失败: %v", err)
		return nil, nil
	}
	var ids []int64
	var samples []string
	for _, row := range rows {
		if _, ok := excelCodes[row["goods_code"].String()]; ok {
			continue
		}
		ids = append(ids, row["id"].Int64())
		if len(samples) < 10 {
			samples = append(samples, row["goods_name"].String())
		}
	}
	return ids, samples
}

// softDeleteGoods 软删除（status=3）指定 id 的商品，返回受影响行数。
func (s *GoodsService) softDeleteGoods(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := g.DB().Model("goods").WhereIn("id", ids).Data(g.Map{
		"status":      3,
		"update_time": time.Now(),
	}).Update()
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	return int(affected), nil
}

// batchInsert 批量插入商品数据
func (s *GoodsService) batchInsert(ctx context.Context, batch []g.Map) error {
	if len(batch) == 0 {
		return nil
	}
	_, err := g.DB().Model("goods").Ctx(ctx).Data(batch).Insert()
	return err
}

// 辅助函数：解析价格
func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0.0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return val
}
