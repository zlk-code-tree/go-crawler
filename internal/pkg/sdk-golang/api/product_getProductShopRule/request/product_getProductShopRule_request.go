package product_getProductShopRule_request

import (
	"doudian.com/open/sdk_golang/api/product_getProductShopRule/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductGetProductShopRuleRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductGetProductShopRuleParam 
}
func (c *ProductGetProductShopRuleRequest) GetUrlPath() string{
	return "/product/getProductShopRule"
}


func New() *ProductGetProductShopRuleRequest{
	request := &ProductGetProductShopRuleRequest{
		Param: &ProductGetProductShopRuleParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductGetProductShopRuleRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_getProductShopRule_response.ProductGetProductShopRuleResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_getProductShopRule_response.ProductGetProductShopRuleResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductGetProductShopRuleRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductGetProductShopRuleRequest) GetParams() *ProductGetProductShopRuleParam{
	return c.Param
}


type ProductGetProductShopRuleParam struct {
	// 驳回自动整改:audit_auto_rectify
	RuleTypes []string `json:"rule_types"`
}
