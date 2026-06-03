package topup_orderQueryCUC_request

import (
	"doudian.com/open/sdk_golang/api/topup_orderQueryCUC/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type TopupOrderQueryCUCRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *TopupOrderQueryCUCParam 
}
func (c *TopupOrderQueryCUCRequest) GetUrlPath() string{
	return "/topup/orderQueryCUC"
}


func New() *TopupOrderQueryCUCRequest{
	request := &TopupOrderQueryCUCRequest{
		Param: &TopupOrderQueryCUCParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *TopupOrderQueryCUCRequest) Execute(accessToken *doudian_sdk.AccessToken) (*topup_orderQueryCUC_response.TopupOrderQueryCUCResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &topup_orderQueryCUC_response.TopupOrderQueryCUCResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *TopupOrderQueryCUCRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *TopupOrderQueryCUCRequest) GetParams() *TopupOrderQueryCUCParam{
	return c.Param
}


type TopupOrderQueryCUCParam struct {
	// 电商订单Id
	PayTradeID *string `json:"payTradeID"`
	// 查询流水号
	QryId *string `json:"qryId"`
	// 合作伙伴编号
	ParterNo *string `json:"parterNo"`
}
