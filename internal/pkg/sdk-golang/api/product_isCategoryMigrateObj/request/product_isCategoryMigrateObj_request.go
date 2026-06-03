package product_isCategoryMigrateObj_request

import (
	"doudian.com/open/sdk_golang/api/product_isCategoryMigrateObj/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductIsCategoryMigrateObjRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductIsCategoryMigrateObjParam 
}
func (c *ProductIsCategoryMigrateObjRequest) GetUrlPath() string{
	return "/product/isCategoryMigrateObj"
}


func New() *ProductIsCategoryMigrateObjRequest{
	request := &ProductIsCategoryMigrateObjRequest{
		Param: &ProductIsCategoryMigrateObjParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductIsCategoryMigrateObjRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_isCategoryMigrateObj_response.ProductIsCategoryMigrateObjResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_isCategoryMigrateObj_response.ProductIsCategoryMigrateObjResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductIsCategoryMigrateObjRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductIsCategoryMigrateObjRequest) GetParams() *ProductIsCategoryMigrateObjParam{
	return c.Param
}


type ProductIsCategoryMigrateObjParam struct {
	// 类目id
	CategoryId *int64 `json:"category_id"`
}
