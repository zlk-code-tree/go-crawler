package product_launchProduct_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductLaunchProductResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductLaunchProductData `json:"data"`
}
type ProductLaunchProductData struct {
}
