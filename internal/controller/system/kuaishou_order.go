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

type KuaishouOrderController struct{}

func NewKuaishouOrderController() *KuaishouOrderController {
	return &KuaishouOrderController{}
}

// KuaishouOrderList 快手订单列表接口 GET /api/kuaishou-order-list
func (c *KuaishouOrderController) KuaishouOrderList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	orderStatus := r.Get("orderStatus").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	kuaishouOrderService := service.NewKuaishouOrderService()
	orders, total, err := kuaishouOrderService.List(r.Context(), page, pageSize, keyword, shopName, orderStatus, startTime, endTime)
	if err != nil {
		g.Log().Error(r.Context(), "快手订单查询失败:", err)
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

// KuaishouOrderExport 导出快手订单Excel接口 GET /api/kuaishou-order-export
func (c *KuaishouOrderController) KuaishouOrderExport(r *ghttp.Request) {
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	orderStatus := r.Get("orderStatus").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	kuaishouOrderService := service.NewKuaishouOrderService()
	filename, err := kuaishouOrderService.ExportToExcel(r.Context(), keyword, shopName, orderStatus, startTime, endTime)
	if err != nil {
		g.Log().Error(r.Context(), "快手订单导出失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)

	defer os.Remove(filename)
}

// KuaishouOrderImport 导入快手订单Excel接口 POST /api/kuaishou-order-import
func (c *KuaishouOrderController) KuaishouOrderImport(r *ghttp.Request) {
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
	tempFile := filepath.Join(uploadDir, fmt.Sprintf("kuaishou_order_import_%d_%d_%d.xlsx", r.GetCtxVar("userId").Int64(), r.GetCtxVar("requestId").Int64(), time.Now().Unix()))

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

	g.Log().Infof(r.Context(), "快手订单文件保存成功: %s", tempFile)

	kuaishouOrderService := service.NewKuaishouOrderService()
	result, err := kuaishouOrderService.ImportFromExcel(r.Context(), tempFile)
	if err != nil {
		g.Log().Error(r.Context(), "快手订单导入失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导入失败: " + err.Error()})
		return
	}

	g.Log().Infof(r.Context(), "快手订单导入完成: 成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount)

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": result,
		"msg":  fmt.Sprintf("导入完成：成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount),
	})
}
