package order_getCrossBorderFulfillInfo_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderGetCrossBorderFulfillInfoResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderGetCrossBorderFulfillInfoData `json:"data"`
}
type OrderGetCrossBorderFulfillInfoData struct {
	// 主订单号
	ShopOrderId string `json:"shop_order_id"`
	// 物流公司名称
	Company string `json:"company"`
	// 物流公司Code
	CompanyCode string `json:"company_code"`
	// 商品单id列表
	SkuOrderIds []string `json:"sku_order_ids"`
	// 物流单号
	LogisticsCode string `json:"logistics_code"`
	// 电子面单链接
	WayBillUrl string `json:"way_bill_url"`
	// 大头笔
	Mark string `json:"mark"`
	// 商品件数
	ProductNum int64 `json:"product_num"`
}
