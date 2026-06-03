package product_queryAutoSizeTemplateOptimizeConfig_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductQueryAutoSizeTemplateOptimizeConfigResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductQueryAutoSizeTemplateOptimizeConfigData `json:"data"`
}
type ProductQueryAutoSizeTemplateOptimizeConfigData struct {
	// 是否允许出尺码表自动优化弹窗
	EnableAutoOptimizePopup bool `json:"enable_auto_optimize_popup"`
}
