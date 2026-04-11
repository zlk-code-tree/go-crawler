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
