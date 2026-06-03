package order_queryLogisticsCompanyList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderQueryLogisticsCompanyListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderQueryLogisticsCompanyListData `json:"data"`
}
type LogisticsCompanyListByShopItem struct {
	// 物流公司的code
	Code string `json:"code"`
	// 物流公司名称
	Name string `json:"name"`
}
type LogistcisCompanyListByIdsItem struct {
	// 订单能够使用的物流code
	Code string `json:"code"`
	// 订单能够使用的物流名称
	Name string `json:"name"`
}
type LogisticsCompanyListByOrderIdsItem struct {
	// 订单号
	OrderId string `json:"order_id"`
	// 除logistics_company_list_by_shop外，该订单号能够使用的物流公司列表
	LogistcisCompanyListByIds []LogistcisCompanyListByIdsItem `json:"logistcis_company_list_by_ids"`
}
type FailShopOrderIdAndReasonItem struct {
	// 订单不存在对应的错误码
	FailCode string `json:"fail_code"`
	// 失败原因
	FailReason string `json:"fail_reason"`
	// 本次请求中失败的订单号
	OrderIds []string `json:"order_ids"`
}
type OrderQueryLogisticsCompanyListData struct {
	// 店铺维度所有订单能够使用的物流公司列表。
	LogisticsCompanyListByShop []LogisticsCompanyListByShopItem `json:"logistics_company_list_by_shop"`
	// 除logistics_company_list_by_shop外，订单维度能够使用的物流公司列表
	LogisticsCompanyListByOrderIds []LogisticsCompanyListByOrderIdsItem `json:"logistics_company_list_by_order_ids"`
	// 本次请求中失败的失败的订单号以及原因列表
	FailShopOrderIdAndReason []FailShopOrderIdAndReasonItem `json:"fail_shop_order_id_and_reason"`
}
