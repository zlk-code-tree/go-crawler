package product_delProductMainPicVideo_request

import (
	"doudian.com/open/sdk_golang/api/product_delProductMainPicVideo/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductDelProductMainPicVideoRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductDelProductMainPicVideoParam 
}
func (c *ProductDelProductMainPicVideoRequest) GetUrlPath() string{
	return "/product/delProductMainPicVideo"
}


func New() *ProductDelProductMainPicVideoRequest{
	request := &ProductDelProductMainPicVideoRequest{
		Param: &ProductDelProductMainPicVideoParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductDelProductMainPicVideoRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_delProductMainPicVideo_response.ProductDelProductMainPicVideoResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_delProductMainPicVideo_response.ProductDelProductMainPicVideoResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductDelProductMainPicVideoRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductDelProductMainPicVideoRequest) GetParams() *ProductDelProductMainPicVideoParam{
	return c.Param
}


type ProductDelProductMainPicVideoParam struct {
	// 商品id
	ProductId int64 `json:"product_id"`
}
