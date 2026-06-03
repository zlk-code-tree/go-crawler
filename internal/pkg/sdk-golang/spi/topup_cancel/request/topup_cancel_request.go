package topup_cancel_request

import (
	"doudian.com/open/sdk_golang/core"
	"doudian.com/open/sdk_golang/spi/topup_cancel/response"
	"doudian.com/open/sdk_golang/utils"
)

type TopupCancelRequest struct {
	doudian_sdk.BaseDoudianOpSpiRequest
	Param *TopupCancelParam 
	Response *topup_cancel_response.TopupCancelResponse 
}
func (c *TopupCancelRequest) GetParamJsonObject() interface{}{
	return c.Param
}


func (c *TopupCancelRequest) GetResponseObject() interface{}{
	return c.Response
}


func (c *TopupCancelRequest) Execute() (interface{}, error){
	return c.GetClient().Request(c)
}


func (c *TopupCancelRequest) ResponseJson() (string, error){
	responseObj, err := c.Execute()
	if err != nil {
		return "", err
	}
	return utils.MarshalNoErr(responseObj), nil
}


func New() *TopupCancelRequest{
	
	request := new(TopupCancelRequest)
	request.SetClient(doudian_sdk.DefaultDoudianOpSpiClient)
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetSpiParam(new(doudian_sdk.DoudianOpSpiParam))
	request.Param = new(TopupCancelParam)
	response := new(topup_cancel_response.TopupCancelResponse)
	response.Data = new(topup_cancel_response.TopupCancelData)
	request.Response = response
	return request
}


type TopupCancelParam struct {
	// 电商平台交易单号
	TradeOrderNo *string `json:"trade_order_no"`
	// 业务类型
	TopupBiz *string `json:"topup_biz"`
	// 商家店铺id
	ShopId *string `json:"shop_id"`
	// 外部订单号，普通充值无需传，特殊充值场景才要，譬如腾讯充值
	OuterOrderId string `json:"outer_order_id"`
}
