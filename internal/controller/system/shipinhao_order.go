package system

import (
	"fmt"
	"go-crawler/internal/service"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type ShipinhaoOrderController struct{}

func NewShipinhaoOrderController() *ShipinhaoOrderController {
	return &ShipinhaoOrderController{}
}

// ShipinhaoOrderList 视频号订单列表接口 GET /api/shipinhao-order-list
func (c *ShipinhaoOrderController) ShipinhaoOrderList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	orderStatus := r.Get("orderStatus").String()
	talentName := r.Get("talentName").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	shipinhaoOrderService := service.NewShipinhaoOrderService()
	orders, total, err := shipinhaoOrderService.List(r.Context(), page, pageSize, keyword, shopName, orderStatus, talentName, startTime, endTime)
	if err != nil {
		g.Log().Error(r.Context(), "视频号订单查询失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"data":  orders,
		"msg":   "ok",
		"total": total,
	})
}

// ShipinhaoOrderExport 导出视频号订单Excel接口 GET /api/shipinhao-order-export
func (c *ShipinhaoOrderController) ShipinhaoOrderExport(r *ghttp.Request) {
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	orderStatus := r.Get("orderStatus").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	shipinhaoOrderService := service.NewShipinhaoOrderService()
	filename, err := shipinhaoOrderService.ExportToExcel(r.Context(), keyword, shopName, orderStatus, startTime, endTime)
	if err != nil {
		g.Log().Error(r.Context(), "视频号订单导出失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)

	defer os.Remove(filename)
}

// ShipinhaoOrderImport 导入视频号订单Excel接口 POST /api/shipinhao-order-import
func (c *ShipinhaoOrderController) ShipinhaoOrderImport(r *ghttp.Request) {
	r.SetCtxVar("timeout", 600)

	file := r.GetUploadFile("file")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请上传文件"})
		return
	}

	if filepath.Ext(file.Filename) != ".xlsx" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "只支持.xlsx格式的Excel文件"})
		return
	}

	if file.Size > 100*1024*1024 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "文件大小不能超过100MB"})
		return
	}

	srcFile, err := file.Open()
	if err != nil {
		g.Log().Error(r.Context(), "打开上传文件失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "打开上传文件失败: " + err.Error()})
		return
	}
	defer srcFile.Close()

	workDir, _ := os.Getwd()
	uploadDir := filepath.Join(workDir, "uploads")
	os.MkdirAll(uploadDir, 0755)
	tempFile := filepath.Join(uploadDir, fmt.Sprintf("shipinhao_order_import_%d_%d_%d.xlsx", r.GetCtxVar("userId").Int64(), r.GetCtxVar("requestId").Int64(), time.Now().Unix()))

	if stat, err := os.Stat(tempFile); err == nil && stat.IsDir() {
		os.RemoveAll(tempFile)
	}

	dstFile, err := os.Create(tempFile)
	if err != nil {
		g.Log().Error(r.Context(), "创建临时文件失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "创建临时文件失败: " + err.Error()})
		return
	}
	defer dstFile.Close()
	defer os.Remove(tempFile)

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		g.Log().Error(r.Context(), "保存文件失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存文件失败: " + err.Error()})
		return
	}

	g.Log().Infof(r.Context(), "视频号订单文件保存成功: %s", tempFile)

	shipinhaoOrderService := service.NewShipinhaoOrderService()
	result, err := shipinhaoOrderService.ImportFromExcel(r.Context(), tempFile)
	if err != nil {
		g.Log().Error(r.Context(), "视频号订单导入失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导入失败: " + err.Error()})
		return
	}

	g.Log().Infof(r.Context(), "视频号订单导入完成: 成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount)

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": result,
		"msg":  fmt.Sprintf("导入完成：成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount),
	})
}
