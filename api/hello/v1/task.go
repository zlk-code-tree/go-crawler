package v1

import "github.com/gogf/gf/v2/frame/g"

// --- Task APIs ---

type TaskListReq struct {
	g.Meta   `path:"/task/list" method:"get" tags:"Task" summary:"获取任务列表"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"20" dc:"每页条数"`
	Status   string `json:"status" in:"query" dc:"任务状态"`
	Keyword  string `json:"keyword" in:"query" dc:"搜索关键词"`
}

type TaskListRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"任务列表"`
	Total    int         `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"pageSize" dc:"每页条数"`
}

type TaskDetailReq struct {
	g.Meta `path:"/task/detail/{id}" method:"get" tags:"Task" summary:"获取任务详情"`
	Id     string `json:"id" in:"path" v:"required" dc:"任务ID"`
}

type TaskDetailRes struct {
	g.Meta     `mime:"application/json"`
	Task       interface{} `json:"task" dc:"任务信息"`
	Checkpoint interface{} `json:"checkpoint" dc:"检查点信息"`
}

type TaskCreateReq struct {
	g.Meta    `path:"/task/create" method:"post" tags:"Task" summary:"创建爬虫任务"`
	TargetUrl string `json:"target_url" v:"required" dc:"目标URL"`
	LoginType string `json:"login_type" d:"qrcode" dc:"登录方式：qrcode/password"`
}

type TaskCreateRes struct {
	g.Meta `mime:"application/json"`
	TaskId string `json:"task_id" dc:"任务ID"`
}

type TaskStopReq struct {
	g.Meta `path:"/task/stop/{id}" method:"post" tags:"Task" summary:"停止任务"`
	Id     string `json:"id" in:"path" v:"required" dc:"任务ID"`
}

type TaskStopRes struct {
	g.Meta `mime:"application/json"`
}

type TaskDeleteReq struct {
	g.Meta `path:"/task/delete/{id}" method:"delete" tags:"Task" summary:"删除任务"`
	Id     string `json:"id" in:"path" v:"required" dc:"任务ID"`
}

type TaskDeleteRes struct {
	g.Meta `mime:"application/json"`
}
