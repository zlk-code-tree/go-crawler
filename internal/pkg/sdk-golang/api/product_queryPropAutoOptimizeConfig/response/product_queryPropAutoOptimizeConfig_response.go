package product_queryPropAutoOptimizeConfig_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductQueryPropAutoOptimizeConfigResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductQueryPropAutoOptimizeConfigData `json:"data"`
}
type ProductQueryPropAutoOptimizeConfigData struct {
	// 是否开启店铺属性自动优化
	IsAutoOptimize bool `json:"is_auto_optimize"`
}
