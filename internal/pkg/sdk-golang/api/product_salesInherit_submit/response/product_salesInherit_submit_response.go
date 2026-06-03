package product_salesInherit_submit_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductSalesInheritSubmitResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductSalesInheritSubmitData `json:"data"`
}
type FailedProductListItem struct {
	// 提报失败站内商品id
	ProductId int64 `json:"product_id"`
	// 失败原因描述
	FailedMsg string `json:"failed_msg"`
}
type ProductSalesInheritSubmitData struct {
	// 提报成功站内商品id列表
	SuccessProductIds []int64 `json:"success_product_ids"`
	// 提报失败商品列表
	FailedProductList []FailedProductListItem `json:"failed_product_list"`
}
