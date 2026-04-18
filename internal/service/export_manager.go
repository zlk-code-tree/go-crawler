package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/os/glog"
	"github.com/xuri/excelize/v2"
)

// ExportManager 数据导出管理器
type ExportManager struct {
	exportDir string
}

// NewExportManager 创建导出管理器
func NewExportManager(exportDir string) *ExportManager {
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		panic(fmt.Sprintf("创建导出目录失败：%v", err))
	}
	return &ExportManager{
		exportDir: exportDir,
	}
}

// ExportToCSV 导出商品数据到 CSV
func (em *ExportManager) ExportToCSV(ctx context.Context, taskId string, products []model.TikTokRankingListInput) (string, error) {
	if len(products) == 0 {
		return "", fmt.Errorf("没有数据可导出")
	}

	filename := filepath.Join(em.exportDir, fmt.Sprintf("product_ranking_%s.csv", taskId))
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("创建 CSV 文件失败：%w", err)
	}
	defer file.Close()

	// 设置 UTF-8 BOM，防止 Excel 中文乱码
	file.WriteString("\xef\xbb\xbf")

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{
		"排名", "商品名称", "店铺名称", "短视频用户支付金额",
		"短视频点击次数", "短视频成交件数", "带货视频数",
		"首次上架时间", "最低到手价", "创建时间",
	}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("写入表头失败：%w", err)
	}

	// 写入数据
	for _, p := range products {
		record := []string{
			strconv.Itoa(p.Rank),
			p.ProductName,
			p.ShopName,
			p.PaymentAmount,
			p.ClickCount,
			p.SalesVolume,
			p.DeliveryVideoNum,
			p.FirstShelfTime,
			p.MinPrice,
			p.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("写入数据失败：%w", err)
		}
	}

	glog.Info(ctx, fmt.Sprintf("[%s] ✅ 导出 CSV 成功：%s (%d 条)", taskId, filename, len(products)))
	return filename, nil
}

// ExportToExcel 导出商品数据到 Excel
func (em *ExportManager) ExportToExcel(ctx context.Context, taskId string, products []model.TikTokRankingListInput) (string, error) {
	if len(products) == 0 {
		return "", fmt.Errorf("没有数据可导出")
	}

	filename := filepath.Join(em.exportDir, fmt.Sprintf("product_ranking_%s.xlsx", taskId))
	f := excelize.NewFile()
	defer f.Close()

	// 创建工作表
	sheetName := "商品榜单"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{
		"排名", "商品名称", "店铺名称", "短视频用户支付金额",
		"短视频点击次数", "短视频成交件数", "带货视频数",
		"首次上架时间", "最低到手价", "创建时间",
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
	for i, p := range products {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), p.Rank)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), p.ProductName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), p.ShopName)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), p.PaymentAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), p.ClickCount)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), p.SalesVolume)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), p.DeliveryVideoNum)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), p.FirstShelfTime)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), p.MinPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), p.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "B", 30)
	f.SetColWidth(sheetName, "C", "C", 20)
	f.SetColWidth(sheetName, "D", "I", 15)
	f.SetColWidth(sheetName, "J", "J", 20)

	// 设置自动筛选（使用兼容的 API）
	f.SetSheetRow(sheetName, "A1", &headers)

	// 保存文件
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存 Excel 文件失败：%w", err)
	}

	glog.Info(ctx, fmt.Sprintf("[%s] ✅ 导出 Excel 成功：%s (%d 条)", taskId, filename, len(products)))
	return filename, nil
}

// ExportVideoAnalysisToCSV 导出视频分析数据到 CSV
func (em *ExportManager) ExportVideoAnalysisToCSV(ctx context.Context, taskId string, videos []model.VideoAnalysisData) (string, error) {
	if len(videos) == 0 {
		return "", fmt.Errorf("没有数据可导出")
	}

	filename := filepath.Join(em.exportDir, fmt.Sprintf("video_analysis_%s.csv", taskId))
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("创建 CSV 文件失败：%w", err)
	}
	defer file.Close()

	file.WriteString("\xef\xbb\xbf")
	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"商品 ID", "商品名称", "视频 ID", "短视频信息", "用户支付金额",
		"商品曝光人数", "商品点击人数", "商品成交人数",
		"商品点击率", "商品点击 - 成交率", "商品曝光 - 成交率",
		"商品成交订单数", "商品成交件数", "创建时间",
	}
	writer.Write(headers)

	for _, v := range videos {
		record := []string{
			v.ProductId,
			v.ProductName,
			v.VideoId,
			v.VideoDesc,
			v.PaymentAmount,
			v.ExposeUserCount,
			v.ClickUserCount,
			v.ConvertUserCount,
			v.ClickRate,
			v.ClickConvertRate,
			v.ExposeConvertRate,
			v.OrderCount,
			v.SalesVolume,
			v.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		writer.Write(record)
	}

	glog.Info(ctx, fmt.Sprintf("[%s] ✅ 导出视频分析 CSV 成功：%s (%d 条)", taskId, filename, len(videos)))
	return filename, nil
}

// ExportAll 导出所有数据（商品 + 视频分析）
func (em *ExportManager) ExportAll(ctx context.Context, taskId string, products []model.TikTokRankingListInput, videos []model.VideoAnalysisData) (map[string]string, error) {
	results := make(map[string]string)

	// 导出商品数据
	if len(products) > 0 {
		csvFile, err := em.ExportToCSV(ctx, taskId, products)
		if err != nil {
			glog.Error(ctx, fmt.Sprintf("[%s] 导出商品 CSV 失败：%v", taskId, err))
		} else {
			results["product_csv"] = csvFile
		}

		excelFile, err := em.ExportToExcel(ctx, taskId, products)
		if err != nil {
			glog.Error(ctx, fmt.Sprintf("[%s] 导出商品 Excel 失败：%v", taskId, err))
		} else {
			results["product_excel"] = excelFile
		}
	}

	// 导出视频分析数据
	if len(videos) > 0 {
		csvFile, err := em.ExportVideoAnalysisToCSV(ctx, taskId, videos)
		if err != nil {
			glog.Error(ctx, fmt.Sprintf("[%s] 导出视频分析 CSV 失败：%v", taskId, err))
		} else {
			results["video_csv"] = csvFile
		}
	}

	return results, nil
}
