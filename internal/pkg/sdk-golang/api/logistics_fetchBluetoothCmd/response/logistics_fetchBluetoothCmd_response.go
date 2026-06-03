package logistics_fetchBluetoothCmd_response

import (
	"doudian.com/open/sdk_golang/core"
)

type LogisticsFetchBluetoothCmdResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *LogisticsFetchBluetoothCmdData `json:"data"`
}
type ErrInfosItem struct {
	// 运单号
	TrackNo string `json:"track_no"`
	// 错误code
	ErrCode int32 `json:"err_code"`
	// 错误msg
	ErrMsg string `json:"err_msg"`
}
type LogisticsFetchBluetoothCmdData struct {
	// 运单的打印指令
	WaybillInfos []WaybillInfosItem `json:"waybill_infos"`
	// 错误信息
	ErrInfos []ErrInfosItem `json:"err_infos"`
}
type WaybillInfosItem struct {
	// 运单号
	TrackNo string `json:"track_no"`
	// 打印指令
	PrintData string `json:"print_data"`
}
