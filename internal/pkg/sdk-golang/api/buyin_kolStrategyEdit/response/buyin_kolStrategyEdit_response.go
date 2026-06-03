package buyin_kolStrategyEdit_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinKolStrategyEditResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinKolStrategyEditData `json:"data"`
}
type FailedItem struct {
	// 达人百应ID
	BuyinId int64 `json:"buyin_id"`
	// 失败原因
	Reason string `json:"reason"`
}
type BuyinKolStrategyEditData struct {
	// 请求结果
	Success bool `json:"success"`
	// 失败的达人维度结果信息
	Failed []FailedItem `json:"failed"`
}
