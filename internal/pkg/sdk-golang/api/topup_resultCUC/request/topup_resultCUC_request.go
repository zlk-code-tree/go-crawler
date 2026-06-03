package topup_resultCUC_request

import (
	"doudian.com/open/sdk_golang/api/topup_resultCUC/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type TopupResultCUCRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *TopupResultCUCParam 
}
func (c *TopupResultCUCRequest) GetUrlPath() string{
	return "/topup/resultCUC"
}


func New() *TopupResultCUCRequest{
	request := &TopupResultCUCRequest{
		Param: &TopupResultCUCParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *TopupResultCUCRequest) Execute(accessToken *doudian_sdk.AccessToken) (*topup_resultCUC_response.TopupResultCUCResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &topup_resultCUC_response.TopupResultCUCResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *TopupResultCUCRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *TopupResultCUCRequest) GetParams() *TopupResultCUCParam{
	return c.Param
}


type TopupResultCUCParam struct {
	// 交费号码（不含区号）
	PhoneNo *string `json:"phoneNo"`
	// 请求时间
	RequestTime *string `json:"requestTime"`
	// 返回代码
	Result *string `json:"result"`
	// 返回错误描述
	ResultDesc *string `json:"resultDesc"`
	// 联通侧订单号
	OrderNo *string `json:"orderNo"`
	// 支付完成时间
	PayTime *string `json:"payTime"`
	// 合作伙伴编号
	ParterNo *string `json:"parterNo"`
	// 交易流水
	TradeID *string `json:"tradeID"`
	// 区号（固话和宽带必填）
	AreaCode *string `json:"areaCode"`
	// 实际交费金额（分)
	Fee *string `json:"fee"`
	// 接入渠道,默认值：01
	MediaType *string `json:"mediaType"`
	// 订单完成时间 YYYYMMDDHH24MISS
	CompleteTime *string `json:"complete_time"`
	// 联通支付机构流水号
	PayNo *string `json:"payNo"`
	// 号码类型
	PhoneType *string `json:"phoneType"`
}
