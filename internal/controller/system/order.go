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

type OrderController struct{}

func NewOrderController() *OrderController {
	return &OrderController{}
}

// OrderList 订单列表接口 GET /system/order/list
func (c *OrderController) OrderList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	orderStatus := r.Get("orderStatus").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	orderService := service.NewOrderService()
	orders, total, err := orderService.List(r.Context(), page, pageSize, keyword, shopName, orderStatus, startTime, endTime)
	if err != nil {
		g.Log().Error(r.Context(), "订单查询失败:", err)
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

// OrderExport 导出订单Excel接口 GET /system/order/export
func (c *OrderController) OrderExport(r *ghttp.Request) {
	keyword := r.Get("keyword").String()
	shopName := r.Get("shopName").String()
	orderStatus := r.Get("orderStatus").String()
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()

	orderService := service.NewOrderService()
	filename, err := orderService.ExportToExcel(r.Context(), keyword, shopName, orderStatus, startTime, endTime)
	if err != nil {
		g.Log().Error(r.Context(), "订单导出失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}

	// 读取文件并返回
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)

	// 延迟删除临时文件
	defer os.Remove(filename)
}

// OrderImport 导入订单Excel接口 POST /system/order/import
func (c *OrderController) OrderImport(r *ghttp.Request) {
	// 设置超时时间为10分钟，支持大文件导入
	r.SetCtxVar("timeout", 600)

	file := r.GetUploadFile("file")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请上传文件"})
		return
	}

	// 检查文件类型
	if filepath.Ext(file.Filename) != ".xlsx" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "只支持.xlsx格式的Excel文件"})
		return
	}

	// 检查文件大小（限制100MB，订单数据可能较大）
	if file.Size > 100*1024*1024 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "文件大小不能超过100MB"})
		return
	}

	// 打开上传的文件
	srcFile, err := file.Open()
	if err != nil {
		g.Log().Error(r.Context(), "打开上传文件失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "打开上传文件失败: " + err.Error()})
		return
	}
	defer srcFile.Close()

	// 创建目标文件
	workDir, _ := os.Getwd()
	uploadDir := filepath.Join(workDir, "uploads")
	os.MkdirAll(uploadDir, 0755)
	tempFile := filepath.Join(uploadDir, fmt.Sprintf("order_import_%d_%d_%d.xlsx", r.GetCtxVar("userId").Int64(), r.GetCtxVar("requestId").Int64(), time.Now().Unix()))

	// 检查目标路径是否是目录
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
	defer os.Remove(tempFile) // 导入完成后删除临时文件

	// 复制文件内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		g.Log().Error(r.Context(), "保存文件失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存文件失败: " + err.Error()})
		return
	}

	g.Log().Infof(r.Context(), "订单文件保存成功: %s", tempFile)

	// 导入数据（流式处理+批量入库）
	orderService := service.NewOrderService()
	result, err := orderService.ImportFromExcel(r.Context(), tempFile)
	if err != nil {
		g.Log().Error(r.Context(), "订单导入失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导入失败: " + err.Error()})
		return
	}

	g.Log().Infof(r.Context(), "订单导入完成: 成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount)

	// 返回详细的导入结果
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": result,
		"msg":  fmt.Sprintf("导入完成：成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount),
	})
}
