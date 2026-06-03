package topup_refundNoticeCUC_request

import (
	"doudian.com/open/sdk_golang/api/topup_refundNoticeCUC/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type TopupRefundNoticeCUCRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *TopupRefundNoticeCUCParam 
}
func (c *TopupRefundNoticeCUCRequest) GetUrlPath() string{
	return "/topup/refundNoticeCUC"
}


func New() *TopupRefundNoticeCUCRequest{
	request := &TopupRefundNoticeCUCRequest{
		Param: &TopupRefundNoticeCUCParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *TopupRefundNoticeCUCRequest) Execute(accessToken *doudian_sdk.AccessToken) (*topup_refundNoticeCUC_response.TopupRefundNoticeCUCResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &topup_refundNoticeCUC_response.TopupRefundNoticeCUCResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *TopupRefundNoticeCUCRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *TopupRefundNoticeCUCRequest) GetParams() *TopupRefundNoticeCUCParam{
	return c.Param
}


type TopupRefundNoticeCUCParam struct {
	// 手机号
	PhoneNo *string `json:"phoneNo"`
	// 实际交费金额（分)
	Fee *string `json:"fee"`
	// 退款时间，格式为:YYYYMMDDHH24MISS
	RefundTime *string `json:"refundTime"`
	// 退款时间，格式：yyyyMMddHHmmss
	TimeStampT *string `json:"timeStampT"`
	// 交费失败通知流水
	NotifyID *string `json:"notifyID"`
	// 合作商编号
	ParterNo *string `json:"parterNo"`
	// 交费交易流水（异步交费接口tradeID）
	PayTradeID *string `json:"payTradeID"`
	// 区号
	AreaCode *string `json:"areaCode"`
}
