package product_isCategoryMigrateObj_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductIsCategoryMigrateObjResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductIsCategoryMigrateObjData `json:"data"`
}
type ProductIsCategoryMigrateObjData struct {
	// 是否是类目2.0店铺
	ShopResult bool `json:"shop_result"`
	// 是否是类目2.0类目
	CategoryResult bool `json:"category_result"`
}
