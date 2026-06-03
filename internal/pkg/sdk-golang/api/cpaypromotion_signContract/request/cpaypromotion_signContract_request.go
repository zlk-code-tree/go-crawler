package cpaypromotion_signContract_request

import (
	"doudian.com/open/sdk_golang/api/cpaypromotion_signContract/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CpaypromotionSignContractRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CpaypromotionSignContractParam 
}
func (c *CpaypromotionSignContractRequest) GetUrlPath() string{
	return "/cpaypromotion/signContract"
}


func New() *CpaypromotionSignContractRequest{
	request := &CpaypromotionSignContractRequest{
		Param: &CpaypromotionSignContractParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CpaypromotionSignContractRequest) Execute(accessToken *doudian_sdk.AccessToken) (*cpaypromotion_signContract_response.CpaypromotionSignContractResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &cpaypromotion_signContract_response.CpaypromotionSignContractResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CpaypromotionSignContractRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CpaypromotionSignContractRequest) GetParams() *CpaypromotionSignContractParam{
	return c.Param
}


type ContractListItem struct {
	// 合同连接 url
	ContractUrl string `json:"contract_url"`
	// 协议类型；0:月付协议；1:月付营销协议
	ContractType int32 `json:"contract_type"`
}
type CpaypromotionSignContractParam struct {
	// 商家店铺名称
	ShopName string `json:"shop_name"`
	// 要执行签约的协议列表
	ContractList []ContractListItem `json:"contract_list"`
}
