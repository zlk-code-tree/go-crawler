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

type GoodsController struct{}

func NewGoodsController() *GoodsController {
	return &GoodsController{}
}

// GoodsList 商品列表接口 GET /system/goods/list
func (c *GoodsController) GoodsList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()
	status := r.Get("status", 0).Int() // 0表示不限，1=正常，2=下架

	goodsService := service.NewGoodsService()
	goods, total, err := goodsService.List(r.Context(), page, pageSize, keyword, status)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"data":  goods,
		"msg":   "ok",
		"total": total,
	})
}

// GoodsDetail 商品详情接口 GET /system/goods/detail
func (c *GoodsController) GoodsDetail(r *ghttp.Request) {
	var req struct {
		Id int `json:"id" in:"query" v:"required" dc:"商品ID"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	goodsService := service.NewGoodsService()
	goods, err := goodsService.Detail(r.Context(), req.Id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	if goods == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": "商品不存在"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": goods, "msg": "ok"})
}

// GoodsSave 保存商品接口 POST /system/goods/save
func (c *GoodsController) GoodsSave(r *ghttp.Request) {
	var req model.GoodsSaveReq

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误: " + err.Error()})
		return
	}

	goodsService := service.NewGoodsService()
	id, err := goodsService.Save(r.Context(), &req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": id}, "msg": "ok"})
}

// GoodsDelete 删除商品接口 POST /system/goods/delete
func (c *GoodsController) GoodsDelete(r *ghttp.Request) {
	var req struct {
		Id int `json:"id" v:"required#商品ID不能为空"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	goodsService := service.NewGoodsService()
	err := goodsService.Delete(r.Context(), req.Id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// GoodsTrashList 回收站商品列表 GET /api/goods-trash-list
func (c *GoodsController) GoodsTrashList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()

	goodsService := service.NewGoodsService()
	goods, total, err := goodsService.TrashList(r.Context(), page, pageSize, keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": goods, "msg": "ok", "total": total})
}

// GoodsTrashRestore 恢复回收站商品 POST /api/goods-trash-restore
func (c *GoodsController) GoodsTrashRestore(r *ghttp.Request) {
	var req struct {
		Ids []int `json:"ids" v:"required#商品ID不能为空"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}
	goodsService := service.NewGoodsService()
	affected, err := goodsService.Restore(r.Context(), req.Ids)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"affected": affected}, "msg": fmt.Sprintf("恢复%d条", affected)})
}

// GoodsTrashDestroy 彻底删除回收站商品 POST /api/goods-trash-destroy
func (c *GoodsController) GoodsTrashDestroy(r *ghttp.Request) {
	var req struct {
		Ids []int `json:"ids" v:"required#商品ID不能为空"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}
	goodsService := service.NewGoodsService()
	affected, err := goodsService.Destroy(r.Context(), req.Ids)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"affected": affected}, "msg": fmt.Sprintf("彻底删除%d条", affected)})
}

// GoodsExport 导出商品Excel接口 GET /system/goods/export
func (c *GoodsController) GoodsExport(r *ghttp.Request) {
	keyword := r.Get("keyword").String()

	goodsService := service.NewGoodsService()
	filename, err := goodsService.ExportToExcel(r.Context(), keyword)
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

// GoodsImport 导入商品Excel接口 POST /system/goods/import
func (c *GoodsController) GoodsImport(r *ghttp.Request) {
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
		g.Log().Error(r.Context(), "打开上传文件失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "打开上传文件失败: " + err.Error()})
		return
	}
	defer srcFile.Close()

	// 创建目标文件
	workDir, _ := os.Getwd()
	uploadDir := filepath.Join(workDir, "uploads")
	os.MkdirAll(uploadDir, 0755)
	tempFile := filepath.Join(uploadDir, fmt.Sprintf("goods_import_%d_%d_%d.xlsx", r.GetCtxVar("userId").Int64(), r.GetCtxVar("requestId").Int64(), time.Now().Unix()))

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

	g.Log().Infof(r.Context(), "商品文件保存成功: %s", tempFile)

	// 导入数据（流式处理+批量入库）
	goodsService := service.NewGoodsService()
	fullReplace := r.Get("fullReplace").Bool()
	dryRun := r.Get("dryRun").Bool()
	result, err := goodsService.ImportFromExcel(r.Context(), tempFile, fullReplace, dryRun)
	if err != nil {
		g.Log().Error(r.Context(), "商品导入失败:", err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导入失败: " + err.Error()})
		return
	}

	g.Log().Infof(r.Context(), "商品导入完成: 成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount)

	// 返回详细的导入结果
	msg := fmt.Sprintf("导入完成：成功%d条，跳过%d条，失败%d条", result.SuccessCount, result.SkipCount, result.FailCount)
	if fullReplace {
		if dryRun {
			msg = fmt.Sprintf("预览：成功%d条，待删除%d条，跳过%d条", result.SuccessCount, result.PendingDeleteCount, result.SkipCount)
		} else {
			msg = fmt.Sprintf("导入完成：成功%d条，删除%d条，跳过%d条，失败%d条", result.SuccessCount, result.DeletedCount, result.SkipCount, result.FailCount)
		}
	}
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": result,
		"msg":  msg,
	})
}
