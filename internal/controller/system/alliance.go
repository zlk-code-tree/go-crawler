package system

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/xuri/excelize/v2"
)

type AllianceController struct{}

func NewAllianceController() *AllianceController {
	return &AllianceController{}
}

// AllianceList 精选联盟订单列表 GET /api/order/alliance-list
func (c *AllianceController) AllianceList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 20).Int()
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	authorName := r.Get("authorName").String()
	authorId := r.Get("authorId").String()
	orderStatus := r.Get("orderStatus").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	list, total, err := service.NewAllianceService().List(r.Context(), page, pageSize, keyword, shopName, authorName, authorId, orderStatus, startTime, endTime)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": list, "total": total, "msg": "ok"})
}

// AllianceFillBuyinId 遍历匹配达人UID POST /api/order/alliance-fill-buyin
func (c *AllianceController) AllianceFillBuyinId(r *ghttp.Request) {
	updated, err := service.NewAllianceService().FillBuyinId(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "匹配失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": fmt.Sprintf("匹配完成，更新 %d 条", updated)})
}

// AllianceExport 导出精选联盟订单 GET /api/order/alliance-export
func (c *AllianceController) AllianceExport(r *ghttp.Request) {
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	authorName := r.Get("authorName").String()
	authorId := r.Get("authorId").String()
	orderStatus := r.Get("orderStatus").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	filename, err := service.NewAllianceService().Export(r.Context(), keyword, shopName, authorName, authorId, orderStatus, startTime, endTime)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)
	defer os.Remove(filename)
}

func formatTime(val interface{}) string {
	if val == nil {
		return ""
	}
	if t, ok := val.(time.Time); ok {
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02 15:04:05")
	}
	return ""
}

func newExcelFile(sheetName string, headers []string, colWidths []float64) (*excelize.File, error) {
	f := excelize.NewFile()
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetSheetName("Sheet1", sheetName)
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, w)
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}
	f.SetRowStyle(sheetName, 1, 1, style)
	return f, nil
}
