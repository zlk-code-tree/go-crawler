package product_publishPreCheck_request

import (
	"doudian.com/open/sdk_golang/api/product_publishPreCheck/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductPublishPreCheckRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductPublishPreCheckParam 
}
func (c *ProductPublishPreCheckRequest) GetUrlPath() string{
	return "/product/publishPreCheck"
}


func New() *ProductPublishPreCheckRequest{
	request := &ProductPublishPreCheckRequest{
		Param: &ProductPublishPreCheckParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductPublishPreCheckRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_publishPreCheck_response.ProductPublishPreCheckResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_publishPreCheck_response.ProductPublishPreCheckResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductPublishPreCheckRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductPublishPreCheckRequest) GetParams() *ProductPublishPreCheckParam{
	return c.Param
}


type ProductPublishPreCheckParam struct {
	// 校验类型，需要进行哪些校验，可以支持部分校验。店铺发品门槛校验：shop_publish_product_access，不需要传其他字段；店铺使用类目权限校验：shop_use_category_access，需要传category_id字段
	CheckTypes []string `json:"check_types"`
	// 叶子类目id
	CategoryId *int64 `json:"category_id"`
	// 外部商品编码，对应商品outer_product_id字段
	OutProductId *string `json:"out_product_id"`
}
