package company

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	v1 "go-crawler/api/hello/v1"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// CompanyList GET /api/system/company/list
func (c *Controller) CompanyList(ctx context.Context, req *v1.SystemCompanyListReq) (res *v1.SystemCompanyListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	m := g.DB().Model("sys_company").OrderAsc("sort").OrderAsc("id")
	if req.Keyword != "" {
		m = m.WhereOrLike("name", "%"+req.Keyword+"%").WhereOrLike("code", "%"+req.Keyword+"%")
	}
	if req.Status != "" {
		m = m.Where("status", req.Status)
	} else {
		m = m.Where("status", 1)
	}
	total, _ := m.Count()
	list, e := m.Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}

	// 收集所有 parent_id，批量查父公司名称
	parentIds := make([]int64, 0)
	for _, row := range list {
		pid := row["parent_id"].Int64()
		if pid > 0 {
			parentIds = append(parentIds, pid)
		}
	}
	parentNameMap := make(map[int64]string)
	if len(parentIds) > 0 {
		parents, _ := g.DB().Model("sys_company").Fields("id,name").WhereIn("id", parentIds).All()
		for _, p := range parents {
			parentNameMap[p["id"].Int64()] = p["name"].String()
		}
	}

	result := make([]g.Map, 0, len(list))
	for _, row := range list {
		item := row.Map()
		pid := row["parent_id"].Int64()
		item["parent_name"] = parentNameMap[pid] // pid=0 时 map 返回零值空字符串
		result = append(result, item)
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": result, "total": total, "page": page, "pageSize": pageSize,
	}})
	return
}

// CompanyCreate POST /api/system/company/create
func (c *Controller) CompanyCreate(ctx context.Context, req *v1.SystemCompanyCreateReq) (res *v1.SystemCompanyCreateRes, err error) {
	r := g.RequestFromCtx(ctx)
	count, _ := g.DB().Model("sys_company").Where("code", req.Code).Count()
	if count > 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "公司编码已存在", "data": nil})
		return
	}
	result, e := g.DB().Model("sys_company").Insert(g.Map{
		"name":       req.Name,
		"code":       req.Code,
		"parent_id":  req.ParentId,
		"level":      req.Level,
		"sort":       req.Sort,
		"remark":     req.Remark,
		"status":     1,
		"created_at": time.Now(),
	})
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	id, _ := result.LastInsertId()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"id": id}})
	return
}

// CompanyUpdate PUT /api/system/company/update
func (c *Controller) CompanyUpdate(ctx context.Context, req *v1.SystemCompanyUpdateReq) (res *v1.SystemCompanyUpdateRes, err error) {
	r := g.RequestFromCtx(ctx)
	update := g.Map{"remark": req.Remark}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.ParentId != nil {
		update["parent_id"] = *req.ParentId
	}
	if req.Level != nil {
		update["level"] = *req.Level
	}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	if req.Sort != nil {
		update["sort"] = *req.Sort
	}
	if _, e := g.DB().Model("sys_company").Where("id", req.Id).Update(update); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// CompanyDetail GET /api/system/company/detail
func (c *Controller) CompanyDetail(ctx context.Context, req *v1.SystemCompanyDetailReq) (res *v1.SystemCompanyDetailRes, err error) {
	r := g.RequestFromCtx(ctx)
	record, e := g.DB().Model("sys_company").Where("id", req.Id).One()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	if record.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 404, "message": "公司不存在", "data": nil})
		return
	}
	item := record.Map()
	pid := record["parent_id"].Int64()
	if pid > 0 {
		parent, _ := g.DB().Model("sys_company").Fields("name").Where("id", pid).Value()
		item["parent_name"] = parent.String()
	} else {
		item["parent_name"] = nil
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": item})
	return
}

// CompanyDelete DELETE /api/system/company/delete
func (c *Controller) CompanyDelete(ctx context.Context, req *v1.SystemCompanyDeleteReq) (res *v1.SystemCompanyDeleteRes, err error) {
	r := g.RequestFromCtx(ctx)
	// 检查是否有子公司
	childCount, _ := g.DB().Model("sys_company").Where("parent_id", req.Id).Where("status", 1).Count()
	if childCount > 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "该公司下存在子公司，无法删除", "data": nil})
		return
	}
	// 软删除
	if _, e := g.DB().Model("sys_company").Where("id", req.Id).Update(g.Map{"status": 0}); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}
