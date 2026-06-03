package product_addSchema_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductAddSchemaResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductAddSchemaData `json:"data"`
}
type SkuItem struct {
	// skuID
	SkuId int64 `json:"sku_id"`
	// 外部skuID
	OutSkuId int64 `json:"out_sku_id"`
	// 外部skuID
	OuterSkuId string `json:"outer_sku_id"`
	// 外部skuID
	Code string `json:"code"`
}
type ProductAddSchemaData struct {
	// 商品ID
	ProductId int64 `json:"product_id"`
	// sku信息
	Sku []SkuItem `json:"sku"`
}
