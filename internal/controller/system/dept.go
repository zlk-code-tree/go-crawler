package system

import (
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// DeptController 部门管理相关接口
// 路径前缀：/system/dept

type DeptController struct{}

func NewDeptController() *DeptController {
	return &DeptController{}
}

// DeptTree 部门树接口 GET /system/dept/tree
func (c *DeptController) DeptTree(r *ghttp.Request) {
	// 调用服务层方法查询部门树
	deptService := service.NewDeptService()
	data, err := deptService.Tree(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": data, "msg": "ok"})
}

// DeptList 部门列表接口 GET /system/dept/list
func (c *DeptController) DeptList(r *ghttp.Request) {
	// 解析查询参数
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword").String()

	// 从上下文中获取公司ID
	companyId := r.GetCtxVar("companyId").Int64()

	// 调用服务层方法查询部门列表
	deptService := service.NewDeptService()
	depts, total, err := deptService.List(r.Context(), page, pageSize, keyword, companyId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"data":  depts,
		"msg":   "ok",
		"total": total,
	})
}

// DeptSave 保存部门接口 POST /system/dept/save
func (c *DeptController) DeptSave(r *ghttp.Request) {
	// 解析请求参数
	var req struct {
		Id       int64  `json:"id"`
		Name     string `json:"name"`
		ParentId int64  `json:"parentId"`
		Level    int    `json:"level"`
		Status   int    `json:"status"`
		Sort     int    `json:"sort"`
		Remark   string `json:"remark"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	// 从上下文中获取公司ID
	companyId := r.GetCtxVar("companyId").Int64()

	// 构建部门模型
	dept := &model.SysDept{
		Id:       req.Id,
		Name:     req.Name,
		ParentId: req.ParentId,
		Level:    req.Level,
		Status:   req.Status,
		Sort:     req.Sort,
		Remark:   req.Remark,
	}

	// 调用服务层方法保存部门
	deptService := service.NewDeptService()
	id, err := deptService.Save(r.Context(), dept, companyId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": id}, "msg": "ok"})
}

// DeptDelete 删除部门接口 POST /system/dept/delete
func (c *DeptController) DeptDelete(r *ghttp.Request) {
	// 解析请求参数
	var req struct {
		Id int64 `json:"id"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	// 调用服务层方法删除部门
	deptService := service.NewDeptService()
	err := deptService.Delete(r.Context(), req.Id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}
