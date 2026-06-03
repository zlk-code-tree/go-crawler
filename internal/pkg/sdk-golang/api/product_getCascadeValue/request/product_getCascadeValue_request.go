package product_getCascadeValue_request

import (
	"doudian.com/open/sdk_golang/api/product_getCascadeValue/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductGetCascadeValueRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductGetCascadeValueParam 
}
func (c *ProductGetCascadeValueRequest) GetUrlPath() string{
	return "/product/getCascadeValue"
}


func New() *ProductGetCascadeValueRequest{
	request := &ProductGetCascadeValueRequest{
		Param: &ProductGetCascadeValueParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductGetCascadeValueRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_getCascadeValue_response.ProductGetCascadeValueResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_getCascadeValue_response.ProductGetCascadeValueResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductGetCascadeValueRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductGetCascadeValueRequest) GetParams() *ProductGetCascadeValueParam{
	return c.Param
}


type CascadeInfoItem struct {
	// 属性值id
	ValueId int64 `json:"value_id"`
	// 属性值名称
	ValueName string `json:"value_name"`
	// 级联id，第一次传0，后续查询使用上一次的查询结果中的cascade_id
	CascadeId int64 `json:"cascade_id"`
}
type ProductGetCascadeValueParam struct {
	// 类目id
	CategoryId int64 `json:"category_id"`
	// 属性id
	PropertyId int64 `json:"property_id"`
	// 级联参数，传当前属性属性值
	CascadeInfo []CascadeInfoItem `json:"cascade_info"`
}
