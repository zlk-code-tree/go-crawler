package v1

import "github.com/gogf/gf/v2/frame/g"

// --- Data APIs ---

type DataOverviewReq struct {
	g.Meta `path:"/data/overview" method:"get" tags:"Data" summary:"数据概览"`
}

type DataOverviewRes struct {
	g.Meta        `mime:"application/json"`
	TaskTotal     int         `json:"task_total" dc:"任务总数"`
	TaskRunning   int         `json:"task_running" dc:"运行中任务数"`
	TaskCompleted int         `json:"task_completed" dc:"已完成任务数"`
	RankingTotal  int         `json:"ranking_total" dc:"排行榜数据总数"`
	VideoTotal    int         `json:"video_total" dc:"视频分析总数"`
	TaskTrend     interface{} `json:"task_trend" dc:"任务趋势(近7天)"`
}

type DataRankingListReq struct {
	g.Meta    `path:"/data/ranking" method:"get" tags:"Data" summary:"商品榜单数据"`
	Page      int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize  int    `json:"pageSize" in:"query" d:"20" dc:"每页条数"`
	TaskId    string `json:"task_id" in:"query" dc:"任务ID"`
	Keyword   string `json:"keyword" in:"query" dc:"搜索关键词"`
	FirstList string `json:"first_list" in:"query" dc:"是否首次上榜：1-是"`
}

type DataRankingListRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"榜单数据"`
	Total    int         `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"pageSize" dc:"每页条数"`
}

type DataRankingStatsReq struct {
	g.Meta `path:"/data/ranking/stats" method:"get" tags:"Data" summary:"榜单统计"`
	TaskId string `json:"task_id" in:"query" dc:"任务ID"`
}

type DataRankingStatsRes struct {
	g.Meta         `mime:"application/json"`
	Total          int `json:"total" dc:"总数"`
	FirstListCount int `json:"first_list_count" dc:"首次上榜数"`
}

type DataVideoListReq struct {
	g.Meta    `path:"/data/video" method:"get" tags:"Data" summary:"视频分析数据"`
	Page      int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize  int    `json:"pageSize" in:"query" d:"20" dc:"每页条数"`
	TaskId    string `json:"task_id" in:"query" dc:"任务ID"`
	ProductId string `json:"product_id" in:"query" dc:"商品ID"`
	Keyword   string `json:"keyword" in:"query" dc:"搜索关键词"`
}

type DataVideoListRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"视频数据"`
	Total    int         `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"pageSize" dc:"每页条数"`
}

type DataVideoStatsReq struct {
	g.Meta `path:"/data/video/stats" method:"get" tags:"Data" summary:"视频统计"`
	TaskId string `json:"task_id" in:"query" dc:"任务ID"`
}

type DataVideoStatsRes struct {
	g.Meta `mime:"application/json"`
	Total  int `json:"total" dc:"总数"`
}

type DataExportReq struct {
	g.Meta `path:"/data/export/{taskId}" method:"get" tags:"Data" summary:"导出任务数据(CSV)"`
	TaskId string `json:"taskId" in:"path" v:"required" dc:"任务ID"`
}

type DataExportRes struct {
	g.Meta `mime:"text/csv"`
}

type DataTaskIdsReq struct {
	g.Meta `path:"/data/task_ids" method:"get" tags:"Data" summary:"获取所有任务ID"`
}

type DataTaskIdsRes struct {
	g.Meta `mime:"application/json"`
}
