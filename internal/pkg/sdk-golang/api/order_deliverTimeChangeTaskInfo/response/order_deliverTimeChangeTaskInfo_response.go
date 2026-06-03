package order_deliverTimeChangeTaskInfo_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderDeliverTimeChangeTaskInfoResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderDeliverTimeChangeTaskInfoData `json:"data"`
}
type OrderDeliverTimeChangeTaskInfoData struct {
	// 任务单ID
	TaskId string `json:"task_id"`
	// 修改前发货时间
	OldTime int64 `json:"old_time"`
	// 修改后发货时间
	NewTime int64 `json:"new_time"`
	// 任务状态，OPEN-未处理 PASS-已处理 REJECT-已拒绝
	Status string `json:"status"`
	// 审核意见，0同意，>0 拒绝code
	RejectCode int64 `json:"reject_code"`
	// 审核拒绝原因
	RejectReason string `json:"reject_reason"`
	// 任务创建时间
	CreateTime int64 `json:"create_time"`
	// 任务更新时间
	UpdateTime int64 `json:"update_time"`
}
