package open_getAuthInfo_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OpenGetAuthInfoResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OpenGetAuthInfoData `json:"data"`
}
type OpenGetAuthInfoData struct {
	// 授权状态，1为正常。枚举为0:待商家确认，1:正常，2:取消，3:过期。授权状态为正常时，需额外判断auth_end_time
	Status int32 `json:"status"`
	// 授权生效时间
	AuthSuccessTime int64 `json:"auth_success_time"`
	// 授权截止时间
	AuthEndTime int64 `json:"auth_end_time"`
	// 授权主体ID
	AuthId string `json:"auth_id"`
	// 授权最近一次更新时间
	AuthUpdateTime int64 `json:"auth_update_time"`
}
