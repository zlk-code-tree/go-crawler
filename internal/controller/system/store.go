package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type StoreController struct{}

func NewStoreController() *StoreController {
	return &StoreController{}
}

// List 店铺列表 GET /api/store/list
func (c *StoreController) List(r *ghttp.Request) {
	var req struct {
		Page     int    `json:"page" in:"query" d:"1"`
		PageSize int    `json:"pageSize" in:"query" d:"20"`
		Keyword  string `json:"keyword" in:"query"`
		Platform string `json:"platform" in:"query"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": err.Error()})
		return
	}

	storeService := service.NewStoreService()
	stores, total, err := storeService.List(r.Context(), req.Page, req.PageSize, req.Keyword, req.Platform)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"data":  g.Map{"list": stores, "total": total},
		"msg":   "ok",
	})
}

// Detail 店铺详情 GET /api/store/detail
func (c *StoreController) Detail(r *ghttp.Request) {
	var req struct {
		Id int `json:"id" in:"query" v:"required"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": err.Error()})
		return
	}

	storeService := service.NewStoreService()
	store, err := storeService.Detail(r.Context(), req.Id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	if store == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": "店铺不存在"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": store, "msg": "ok"})
}

// Save 保存店铺 POST /api/store/save
func (c *StoreController) Save(r *ghttp.Request) {
	var req model.StoreSaveReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": err.Error()})
		return
	}

	storeService := service.NewStoreService()
	id, err := storeService.Save(r.Context(), &req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": id}, "msg": "ok"})
}

// Delete 删除店铺 POST /api/store/delete
func (c *StoreController) Delete(r *ghttp.Request) {
	var req struct {
		Id int `json:"id" v:"required#店铺ID不能为空"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": err.Error()})
		return
	}

	storeService := service.NewStoreService()
	if err := storeService.Delete(r.Context(), req.Id); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// Stats 店铺统计 GET /api/store/stats
func (c *StoreController) Stats(r *ghttp.Request) {
	storeService := service.NewStoreService()
	stats, err := storeService.Stats(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": stats, "msg": "ok"})
}

// Export 导出店铺 GET /api/store/export
func (c *StoreController) Export(r *ghttp.Request) {
	keyword := r.Get("keyword").String()
	platform := r.Get("platform").String()

	storeService := service.NewStoreService()
	filename, err := storeService.ExportToExcel(r.Context(), keyword, platform)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)

	defer os.Remove(filename)
}

// Import 导入店铺 POST /api/store/import
func (c *StoreController) Import(r *ghttp.Request) {
	file := r.GetUploadFile("file")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请上传文件"})
		return
	}

	if filepath.Ext(file.Filename) != ".xlsx" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "只支持.xlsx格式文件"})
		return
	}

	srcFile, err := file.Open()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "打开上传文件失败: " + err.Error()})
		return
	}
	defer srcFile.Close()

	workDir, _ := os.Getwd()
	uploadDir := filepath.Join(workDir, "uploads")
	os.MkdirAll(uploadDir, 0755)
	tempFile := filepath.Join(uploadDir, fmt.Sprintf("store_import_%d.xlsx", time.Now().Unix()))

	dstFile, err := os.Create(tempFile)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "创建临时文件失败: " + err.Error()})
		return
	}
	defer dstFile.Close()
	defer os.Remove(tempFile)

	io.Copy(dstFile, srcFile)

	storeService := service.NewStoreService()
	result, err := storeService.ImportFromExcel(r.Context(), tempFile)
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

// ExportTemplate 下载店铺导入模板 GET /api/store/export-template
func (c *StoreController) ExportTemplate(r *ghttp.Request) {
	storeService := service.NewStoreService()
	filename, err := storeService.ExportTemplate(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "生成模板失败: " + err.Error()})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)

	defer os.Remove(filename)
}
