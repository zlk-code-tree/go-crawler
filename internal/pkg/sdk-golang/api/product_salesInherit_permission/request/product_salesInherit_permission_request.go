package product_salesInherit_permission_request

import (
	"doudian.com/open/sdk_golang/api/product_salesInherit_permission/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductSalesInheritPermissionRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductSalesInheritPermissionParam 
}
func (c *ProductSalesInheritPermissionRequest) GetUrlPath() string{
	return "/product/salesInherit/permission"
}


func New() *ProductSalesInheritPermissionRequest{
	request := &ProductSalesInheritPermissionRequest{
		Param: &ProductSalesInheritPermissionParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductSalesInheritPermissionRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_salesInherit_permission_response.ProductSalesInheritPermissionResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_salesInherit_permission_response.ProductSalesInheritPermissionResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductSalesInheritPermissionRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductSalesInheritPermissionRequest) GetParams() *ProductSalesInheritPermissionParam{
	return c.Param
}


type ProductSalesInheritPermissionParam struct {
}
