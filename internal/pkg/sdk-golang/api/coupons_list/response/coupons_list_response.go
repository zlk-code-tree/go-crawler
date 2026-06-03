package coupons_list_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CouponsListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data Data `json:"data"`
}
type CouponsListData struct {
	// 列表
	Data Data `json:"data"`
}
type DataItem struct {
	// 平台订单id
	OrderId string `json:"order_id"`
	// 三方商家的订单Id，如果有多个，使用;分割
	VerrifyOrderIds string `json:"verrify_order_ids"`
	// 卡券id
	CertId string `json:"cert_id"`
	// 外部卡券id
	OutCertId string `json:"out_cert_id"`
	// 卡号券码
	CertNo string `json:"cert_no"`
	// 有效开始时间
	ValidStart int64 `json:"valid_start"`
	// 有效结束时间
	ValidEnd int64 `json:"valid_end"`
	// 发券时间，0则用当前时间
	GrantTime int64 `json:"grant_time"`
	// 状态：1未激活 2未使用 3已过期
	Status int64 `json:"status"`
	// 可核销次数
	TotalNum int64 `json:"total_num"`
	// 已核销次数
	VerifyNum int64 `json:"verify_num"`
	// 剩余可用次数
	AvailableNum int64 `json:"available_num"`
}
type Data struct {
	// 列表
	Data []DataItem `json:"data"`
}
