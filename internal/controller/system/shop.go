package system

import (
	"fmt"
	"go-crawler/internal/model"
	"go-crawler/internal/service"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type ShopController struct{}

func NewShopController() *ShopController {
	return &ShopController{}
}

// ShopList 店铺列表接口 GET /system/shop/list
func (c *ShopController) ShopList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()

	shopService := service.NewShopService()
	shops, total, err := shopService.List(r.Context(), page, pageSize, keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"data":  shops,
		"msg":   "ok",
		"total": total,
	})
}

// ShopDetail 店铺详情接口 GET /system/shop/detail
func (c *ShopController) ShopDetail(r *ghttp.Request) {
	var req struct {
		Id int `json:"id" in:"query" v:"required" dc:"店铺ID"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	shopService := service.NewShopService()
	shop, err := shopService.Detail(r.Context(), req.Id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	if shop == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": "店铺不存在"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": shop, "msg": "ok"})
}

// ShopSave 保存店铺接口 POST /system/shop/save
func (c *ShopController) ShopSave(r *ghttp.Request) {
	var req model.ShopSaveReq

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误: " + err.Error()})
		return
	}

	shopService := service.NewShopService()
	id, err := shopService.Save(r.Context(), &req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": id}, "msg": "ok"})
}

// ShopDelete 删除店铺接口 POST /system/shop/delete
func (c *ShopController) ShopDelete(r *ghttp.Request) {
	var req struct {
		Id int `json:"id" v:"required#店铺ID不能为空"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	shopService := service.NewShopService()
	err := shopService.Delete(r.Context(), req.Id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// ShopExport 导出店铺Excel接口 GET /system/shop/export
func (c *ShopController) ShopExport(r *ghttp.Request) {
	keyword := r.Get("keyword").String()

	shopService := service.NewShopService()
	filename, err := shopService.ExportToExcel(r.Context(), keyword)
	if err != nil {
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

// ShopImport 导入店铺Excel接口 POST /system/shop/import
func (c *ShopController) ShopImport(r *ghttp.Request) {
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

	// 检查文件大小（限制50MB）
	if file.Size > 50*1024*1024 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "文件大小不能超过50MB"})
		return
	}

	// 打开上传的文件
	srcFile, err := file.Open()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "打开上传文件失败: " + err.Error()})
		return
	}
	defer srcFile.Close()

	// 创建目标文件
	workDir, _ := os.Getwd()
	uploadDir := filepath.Join(workDir, "uploads")
	os.MkdirAll(uploadDir, 0755)
	tempFile := filepath.Join(uploadDir, fmt.Sprintf("shop_import_%d_%d_%d.xlsx", r.GetCtxVar("userId").Int64(), r.GetCtxVar("requestId").Int64(), time.Now().Unix()))

	// 检查目标路径是否是目录
	if stat, err := os.Stat(tempFile); err == nil && stat.IsDir() {
		os.RemoveAll(tempFile)
	}

	dstFile, err := os.Create(tempFile)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "创建临时文件失败: " + err.Error()})
		return
	}
	defer dstFile.Close()
	defer os.Remove(tempFile) // 导入完成后删除临时文件

	// 复制文件内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存文件失败: " + err.Error()})
		return
	}

	// 导入数据（流式处理+批量入库）
	shopService := service.NewShopService()
	result, err := shopService.ImportFromExcel(r.Context(), tempFile)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导入失败: " + err.Error()})
		return
	}

	// 返回详细的导入结果
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": result,
		"msg":  fmt.Sprintf("导入完成：成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount),
	})
}
