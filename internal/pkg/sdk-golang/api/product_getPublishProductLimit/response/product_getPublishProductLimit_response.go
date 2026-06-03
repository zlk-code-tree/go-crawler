package product_getPublishProductLimit_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetPublishProductLimitResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetPublishProductLimitData `json:"data"`
}
type ProductLimitResultItem struct {
	// open店铺发品限制, 1-天级发品限制，2-一级类目在架商品数限制
	LimitType int64 `json:"limit_type"`
	// 是否有限制
	HasLimit bool `json:"has_limit"`
	// 是否命中限制
	HitLimit bool `json:"hit_limit"`
	// 剩余数量
	RemainCount int64 `json:"remain_count"`
}
type ProductGetPublishProductLimitData struct {
	// 返回校验结果列表
	ProductLimitResult []ProductLimitResultItem `json:"product_limit_result"`
}
