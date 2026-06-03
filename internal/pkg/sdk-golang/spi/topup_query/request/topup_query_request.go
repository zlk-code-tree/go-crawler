package topup_query_request

import (
	"doudian.com/open/sdk_golang/core"
	"doudian.com/open/sdk_golang/spi/topup_query/response"
	"doudian.com/open/sdk_golang/utils"
)

type TopupQueryRequest struct {
	doudian_sdk.BaseDoudianOpSpiRequest
	Param *TopupQueryParam 
	Response *topup_query_response.TopupQueryResponse 
}
func (c *TopupQueryRequest) GetParamJsonObject() interface{}{
	return c.Param
}


func (c *TopupQueryRequest) GetResponseObject() interface{}{
	return c.Response
}


func (c *TopupQueryRequest) Execute() (interface{}, error){
	return c.GetClient().Request(c)
}


func (c *TopupQueryRequest) ResponseJson() (string, error){
	responseObj, err := c.Execute()
	if err != nil {
		return "", err
	}
	return utils.MarshalNoErr(responseObj), nil
}


func New() *TopupQueryRequest{
	
	request := new(TopupQueryRequest)
	request.SetClient(doudian_sdk.DefaultDoudianOpSpiClient)
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetSpiParam(new(doudian_sdk.DoudianOpSpiParam))
	request.Param = new(TopupQueryParam)
	response := new(topup_query_response.TopupQueryResponse)
	response.Data = new(topup_query_response.TopupQueryData)
	request.Response = response
	return request
}


type TopupQueryParam struct {
	// 充值平台交易单号
	TradeOrderNo *string `json:"trade_order_no"`
	// 业务类型
	TopupBiz *string `json:"topup_biz"`
	// 商家店铺id
	ShopId *string `json:"shop_id"`
	// 外部订单号，特殊充值场景才需要，譬如腾讯充值
	OuterOrderId string `json:"outer_order_id"`
}
