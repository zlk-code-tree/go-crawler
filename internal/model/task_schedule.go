package model

// TaskLogEntry 一次任务执行的日志记录
type TaskLogEntry struct {
	Id            int64  `json:"id"`
	TaskName      string `json:"taskName"`
	TaskType      string `json:"taskType"`
	Status        int    `json:"status"` // 0运行中 1成功 2失败 3跳过
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	DurationMs    int    `json:"durationMs"`
	ApiUrl        string `json:"apiUrl"`
	RequestParams string `json:"requestParams"`
	ResponseCode  string `json:"responseCode"`
	ResponseBody  string `json:"responseBody"`
	ErrorMsg      string `json:"errorMsg"`
	CreatedAt     string `json:"createdAt"`
}

// TaskStatsItem 任务统计项
type TaskStatsItem struct {
	TaskType       string  `json:"taskType"`
	TaskName       string  `json:"taskName"`
	TotalCount     int     `json:"totalCount"`
	SuccessCount   int     `json:"successCount"`
	FailCount      int     `json:"failCount"`
	SkipCount      int     `json:"skipCount"`
	SuccessRate    float64 `json:"successRate"`
	LastStatus     int     `json:"lastStatus"`
	LastStatusDesc string  `json:"lastStatusDesc"`
	LastRunTime    string  `json:"lastRunTime"`
	AvgDurationMs  int     `json:"avgDurationMs"`
}

// TaskScheduleStats 概览统计
type TaskScheduleStats struct {
	TotalTasks   int `json:"totalTasks"`
	RunningCount int `json:"runningCount"`
	TodaySuccess int `json:"todaySuccess"`
	TodayFail    int `json:"todayFail"`
}
