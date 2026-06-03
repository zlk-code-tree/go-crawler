package supplyCenter_cargo_getCategoryPropertyValue_request

import (
	"doudian.com/open/sdk_golang/api/supplyCenter_cargo_getCategoryPropertyValue/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SupplyCenterCargoGetCategoryPropertyValueRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SupplyCenterCargoGetCategoryPropertyValueParam 
}
func (c *SupplyCenterCargoGetCategoryPropertyValueRequest) GetUrlPath() string{
	return "/supplyCenter/cargo/getCategoryPropertyValue"
}


func New() *SupplyCenterCargoGetCategoryPropertyValueRequest{
	request := &SupplyCenterCargoGetCategoryPropertyValueRequest{
		Param: &SupplyCenterCargoGetCategoryPropertyValueParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SupplyCenterCargoGetCategoryPropertyValueRequest) Execute(accessToken *doudian_sdk.AccessToken) (*supplyCenter_cargo_getCategoryPropertyValue_response.SupplyCenterCargoGetCategoryPropertyValueResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &supplyCenter_cargo_getCategoryPropertyValue_response.SupplyCenterCargoGetCategoryPropertyValueResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SupplyCenterCargoGetCategoryPropertyValueRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SupplyCenterCargoGetCategoryPropertyValueRequest) GetParams() *SupplyCenterCargoGetCategoryPropertyValueParam{
	return c.Param
}


type SupplyCenterCargoGetCategoryPropertyValueParam struct {
	// 类目id
	CategoryId int64 `json:"category_id"`
	// 属性项id
	PropertyId int64 `json:"property_id"`
}
