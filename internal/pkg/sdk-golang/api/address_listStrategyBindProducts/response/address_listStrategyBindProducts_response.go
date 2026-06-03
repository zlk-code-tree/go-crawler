package address_listStrategyBindProducts_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AddressListStrategyBindProductsResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *AddressListStrategyBindProductsData `json:"data"`
}
type ProductListItem struct {
	// 商品id
	ProductId string `json:"product_id"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 商品图片
	Image string `json:"image"`
}
type AddressListStrategyBindProductsData struct {
	// 绑定商品总数
	Total int64 `json:"total"`
	// 当前页码;
	Page int64 `json:"page"`
	// 每页数量;
	PageSize int64 `json:"page_size"`
	// 绑定商品列表;
	ProductList []ProductListItem `json:"product_list"`
}
