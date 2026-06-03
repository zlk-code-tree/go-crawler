package topup_result_request

import (
	"doudian.com/open/sdk_golang/api/topup_result/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type TopupResultRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *TopupResultParam 
}
func (c *TopupResultRequest) GetUrlPath() string{
	return "/topup/result"
}


func New() *TopupResultRequest{
	request := &TopupResultRequest{
		Param: &TopupResultParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *TopupResultRequest) Execute(accessToken *doudian_sdk.AccessToken) (*topup_result_response.TopupResultResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &topup_result_response.TopupResultResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *TopupResultRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *TopupResultRequest) GetParams() *TopupResultParam{
	return c.Param
}


type TopupResultParam struct {
	// 订单号
	TradeOrderNo string `json:"trade_order_no"`
	// 充值业务
	TopupBiz string `json:"topup_biz"`
	// 商家充值单号
	SellerOrderNo string `json:"seller_order_no"`
	// 充值结果
	SellerOrderStatus string `json:"seller_order_status"`
	// 错误码
	ErrCode *string `json:"err_code"`
	// 错误信息
	ErrDesc *string `json:"err_desc"`
	// 去使用链接
	Url *string `json:"url"`
	// 去使用链接类型小程序：microapp普通链接：normal
	UrlType *string `json:"url_type"`
	// 充值失败错误码10001：手机号码无效10002：商家缺货10003 ： 命中运营商风控策略
	TopupFailureReasonCode *string `json:"topup_failure_reason_code"`
}
