package system

import (
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type OperationLogController struct{}

func NewOperationLogController() *OperationLogController {
	return &OperationLogController{}
}

func (c *OperationLogController) OperationLogList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword", "").String()

	rows, total, err := service.NewOperationLogService().GetList(r.Context(), page, pageSize, keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取操作日志失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"data":  rows,
		"msg":   "ok",
		"total": total,
	})
}

func (c *OperationLogController) OperationLogDetail(r *ghttp.Request) {
	id := r.Get("id").Int64()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}

	detail, err := service.NewOperationLogService().GetDetail(r.Context(), id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取日志详情失败"})
		return
	}

	if detail == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": "日志不存在"})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": detail,
		"msg":  "ok",
	})
}
