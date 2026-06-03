package product_auditAutoRectifyGrant_request

import (
	"doudian.com/open/sdk_golang/api/product_auditAutoRectifyGrant/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductAuditAutoRectifyGrantRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductAuditAutoRectifyGrantParam 
}
func (c *ProductAuditAutoRectifyGrantRequest) GetUrlPath() string{
	return "/product/auditAutoRectifyGrant"
}


func New() *ProductAuditAutoRectifyGrantRequest{
	request := &ProductAuditAutoRectifyGrantRequest{
		Param: &ProductAuditAutoRectifyGrantParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductAuditAutoRectifyGrantRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_auditAutoRectifyGrant_response.ProductAuditAutoRectifyGrantResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_auditAutoRectifyGrant_response.ProductAuditAutoRectifyGrantResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductAuditAutoRectifyGrantRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductAuditAutoRectifyGrantRequest) GetParams() *ProductAuditAutoRectifyGrantParam{
	return c.Param
}


type ProductAuditAutoRectifyGrantParam struct {
	// 1、同意授权；2、拒绝授权；3、待定（代表先不做决策）
	Operate int32 `json:"operate"`
}
