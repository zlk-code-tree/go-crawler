package buyin_bindChannelProduct_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinBindChannelProductResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinBindChannelProductData `json:"data"`
}
type FailedItem struct {
	// 绑定失败的达人id，可能是uid或者buyinId，结合id_type字段使用
	Id int64 `json:"id"`
	// id字段的类型：0: 达人uid 1:buyinID
	IdType int64 `json:"id_type"`
	// 绑定失败的原因
	Reason string `json:"reason"`
}
type BuyinBindChannelProductData struct {
	// 成功绑定的数量
	Success int64 `json:"success"`
	// 绑定失败的帐号和原因
	Failed []FailedItem `json:"failed"`
}
