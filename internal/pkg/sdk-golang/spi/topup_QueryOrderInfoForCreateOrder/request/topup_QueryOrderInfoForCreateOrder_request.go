package topup_QueryOrderInfoForCreateOrder_request

import (
	"doudian.com/open/sdk_golang/core"
	"doudian.com/open/sdk_golang/spi/topup_QueryOrderInfoForCreateOrder/response"
	"doudian.com/open/sdk_golang/utils"
)

type TopupQueryOrderInfoForCreateOrderRequest struct {
	doudian_sdk.BaseDoudianOpSpiRequest
	Param *TopupQueryOrderInfoForCreateOrderParam 
	Response *topup_QueryOrderInfoForCreateOrder_response.TopupQueryOrderInfoForCreateOrderResponse 
}
func (c *TopupQueryOrderInfoForCreateOrderRequest) GetParamJsonObject() interface{}{
	return c.Param
}


func (c *TopupQueryOrderInfoForCreateOrderRequest) GetResponseObject() interface{}{
	return c.Response
}


func (c *TopupQueryOrderInfoForCreateOrderRequest) Execute() (interface{}, error){
	return c.GetClient().Request(c)
}


func (c *TopupQueryOrderInfoForCreateOrderRequest) ResponseJson() (string, error){
	responseObj, err := c.Execute()
	if err != nil {
		return "", err
	}
	return utils.MarshalNoErr(responseObj), nil
}


func New() *TopupQueryOrderInfoForCreateOrderRequest{
	
	request := new(TopupQueryOrderInfoForCreateOrderRequest)
	request.SetClient(doudian_sdk.DefaultDoudianOpSpiClient)
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetSpiParam(new(doudian_sdk.DoudianOpSpiParam))
	request.Param = new(TopupQueryOrderInfoForCreateOrderParam)
	response := new(topup_QueryOrderInfoForCreateOrder_response.TopupQueryOrderInfoForCreateOrderResponse)
	response.Data = new(topup_QueryOrderInfoForCreateOrder_response.TopupQueryOrderInfoForCreateOrderData)
	request.Response = response
	return request
}


type TopupQueryOrderInfoForCreateOrderParam struct {
	// 商家外部订单号
	OuterOrderId string `json:"outer_order_id"`
	// 商品id
	ProductIdList []int64 `json:"product_id_list"`
	// 商品skuId
	SkuIdList []int64 `json:"sku_id_list"`
	// 店铺id
	ShopId int64 `json:"shop_id"`
}
