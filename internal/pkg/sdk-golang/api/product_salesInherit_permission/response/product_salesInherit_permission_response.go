package product_salesInherit_permission_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductSalesInheritPermissionResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductSalesInheritPermissionData `json:"data"`
}
type ProductSalesInheritPermissionData struct {
	// true-有权限；false-无权限
	Success bool `json:"success"`
}
