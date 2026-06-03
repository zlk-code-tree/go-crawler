package order_uploadextrapackage_request

import (
	"doudian.com/open/sdk_golang/api/order_uploadextrapackage/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderUploadextrapackageRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderUploadextrapackageParam 
}
func (c *OrderUploadextrapackageRequest) GetUrlPath() string{
	return "/order/uploadextrapackage"
}


func New() *OrderUploadextrapackageRequest{
	request := &OrderUploadextrapackageRequest{
		Param: &OrderUploadextrapackageParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderUploadextrapackageRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_uploadextrapackage_response.OrderUploadextrapackageResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_uploadextrapackage_response.OrderUploadextrapackageResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderUploadextrapackageRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderUploadextrapackageRequest) GetParams() *OrderUploadextrapackageParam{
	return c.Param
}


type BundleProductListItem struct {
	// 关联的组合商品的件数
	Count *int64 `json:"count"`
	// 组合商品中子商品的SkuID
	SkuId *string `json:"sku_id"`
	// 组合商品中子商品的商品ID
	ProductId *string `json:"product_id"`
}
type ProductOrdersItem struct {
	// 组合商品信息列表
	BundleProductList []BundleProductListItem `json:"bundle_product_list"`
	// 选填。如果是组合商品，该字段为空，即使传递也不会被采用；如果是非组合商品，该字段必填，为大于0的整数。
	ProductCount int64 `json:"product_count"`
	// 商品单单号
	ProductOrderId string `json:"product_order_id"`
}
type OrderUploadextrapackageParam struct {
	// 店铺单号
	ShopOrderId string `json:"shop_order_id"`
	// 运单号关联的商品信息列表
	ProductOrders []ProductOrdersItem `json:"product_orders"`
	// 物流单号
	TrackingNo string `json:"tracking_no"`
	// 物流公司code
	CompanyCode string `json:"company_code"`
	// 商家在地址库设置的地址对应的ID，用于发货时指定发货的地址
	SendAddressId int64 `json:"send_address_id"`
	// 子母单情况下指定当前传递的单号对应的母单号，如果没有，该字段不需要传递，直接为空即可。
	ParentTrackingNo *string `json:"parent_tracking_no"`
	// 店铺对应的门店ID，如果没有门店，该字段不用传递。
	StoreId *int64 `json:"store_id"`
}
