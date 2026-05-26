package system

import (
	"fmt"
	"go-crawler/internal/service"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type OrderMatchController struct{}

func NewOrderMatchController() *OrderMatchController {
	return &OrderMatchController{}
}

// OrderMatchImportOrders 上传普通订单 POST /api/order-match/import-orders
func (c *OrderMatchController) OrderMatchImportOrders(r *ghttp.Request) {
	platform := r.Get("platform").String()
	if platform == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请选择平台"})
		return
	}

	file := r.GetUploadFile("file")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请上传文件"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".xlsx" && ext != ".xls" && ext != ".csv" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "只支持 .xlsx、.xls、.csv 格式"})
		return
	}

	tempFile, err := c.saveTempFile(r, file, "match_orders")
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}
	defer os.Remove(tempFile)

	svc := service.NewOrderMatchService()
	result, err := svc.ImportOrders(r.Context(), tempFile, platform)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导入失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": result,
		"msg":  fmt.Sprintf("导入完成：成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount),
	})
}

// OrderMatchImportJingxuan 上传精选联盟订单 POST /api/order-match/import-jingxuan
func (c *OrderMatchController) OrderMatchImportJingxuan(r *ghttp.Request) {
	platform := r.Get("platform").String()
	if platform == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请选择平台"})
		return
	}

	file := r.GetUploadFile("file")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请上传文件"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".xlsx" && ext != ".xls" && ext != ".csv" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "只支持 .xlsx、.xls、.csv 格式"})
		return
	}

	tempFile, err := c.saveTempFile(r, file, "match_jingxuan")
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}
	defer os.Remove(tempFile)

	svc := service.NewOrderMatchService()
	result, err := svc.ImportJingxuan(r.Context(), tempFile, platform)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导入失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": result,
		"msg":  fmt.Sprintf("导入完成：成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount),
	})
}

// OrderMatchRun 触发匹配 POST /api/order-match/run
func (c *OrderMatchController) OrderMatchRun(r *ghttp.Request) {
	svc := service.NewOrderMatchService()
	result, err := svc.MatchOrders(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "匹配失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": result, "msg": "匹配完成"})
}

// OrderMatchStats 匹配结果统计 GET /api/order-match/stats
func (c *OrderMatchController) OrderMatchStats(r *ghttp.Request) {
	platform := r.Get("platform").String()
	svc := service.NewOrderMatchService()
	data, err := svc.Stats(r.Context(), platform)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": data, "msg": "ok"})
}

// OrderMatchOrderList 普通订单列表 GET /api/order-match/order-list
func (c *OrderMatchController) OrderMatchOrderList(r *ghttp.Request) {
	page := r.Get("pageNum", r.Get("page", 1)).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()
	platform := r.Get("platform").String()

	svc := service.NewOrderMatchService()
	items, total, err := svc.OrderList(r.Context(), page, pageSize, keyword, platform)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"list": items, "total": total}, "msg": "ok"})
}

// OrderMatchJingxuanList 精选联盟列表 GET /api/order-match/jingxuan-list
func (c *OrderMatchController) OrderMatchJingxuanList(r *ghttp.Request) {
	page := r.Get("pageNum", r.Get("page", 1)).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()
	platform := r.Get("platform").String()

	svc := service.NewOrderMatchService()
	items, total, err := svc.JingxuanList(r.Context(), page, pageSize, keyword, platform)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"list": items, "total": total}, "msg": "ok"})
}

// OrderMatchResultList 匹配结果列表 GET /api/order-match/result-list
func (c *OrderMatchController) OrderMatchResultList(r *ghttp.Request) {
	page := r.Get("pageNum", r.Get("page", 1)).Int()
	pageSize := r.Get("pageSize", 10).Int()
	matchStatus := r.Get("matchStatus").String()
	platform := r.Get("platform").String()
	keyword := r.Get("keyword").String()

	svc := service.NewOrderMatchService()
	items, total, err := svc.ResultList(r.Context(), page, pageSize, matchStatus, platform, keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"list": items, "total": total}, "msg": "ok"})
}

// OrderMatchExport 导出数据 GET /api/order-match/export
func (c *OrderMatchController) OrderMatchExport(r *ghttp.Request) {
	table := r.Get("table").String()
	platform := r.Get("platform").String()
	keyword := r.Get("keyword").String()
	matchStatus := r.Get("matchStatus").String()

	if table == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "缺少table参数"})
		return
	}

	svc := service.NewOrderMatchService()
	filePath, err := svc.Export(r.Context(), table, platform, keyword, matchStatus)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}
	defer os.Remove(filePath)

	r.Response.ServeFileDownload(filePath)
}

// saveTempFile 保存上传文件到临时目录
func (c *OrderMatchController) saveTempFile(r *ghttp.Request, file *ghttp.UploadFile, prefix string) (string, error) {
	srcFile, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("打开上传文件失败: %v", err)
	}
	defer srcFile.Close()

	workDir, _ := os.Getwd()
	uploadDir := filepath.Join(workDir, "uploads")
	os.MkdirAll(uploadDir, 0755)
	tempFile := filepath.Join(uploadDir, fmt.Sprintf("%s_%d%s", prefix, time.Now().UnixNano(), filepath.Ext(file.Filename)))

	dstFile, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return tempFile, nil
}
