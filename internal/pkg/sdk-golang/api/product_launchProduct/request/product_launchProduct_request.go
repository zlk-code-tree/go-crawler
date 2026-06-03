package product_launchProduct_request

import (
	"doudian.com/open/sdk_golang/api/product_launchProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductLaunchProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductLaunchProductParam 
}
func (c *ProductLaunchProductRequest) GetUrlPath() string{
	return "/product/launchProduct"
}


func New() *ProductLaunchProductRequest{
	request := &ProductLaunchProductRequest{
		Param: &ProductLaunchProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductLaunchProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_launchProduct_response.ProductLaunchProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_launchProduct_response.ProductLaunchProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductLaunchProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductLaunchProductRequest) GetParams() *ProductLaunchProductParam{
	return c.Param
}


type ProductLaunchProductParam struct {
	// 商品id
	ProductId int64 `json:"product_id"`
	// 预测记录id，类目预测记录key=category_recommend_id
	RecommendIds *map[string]string `json:"recommend_ids"`
}
