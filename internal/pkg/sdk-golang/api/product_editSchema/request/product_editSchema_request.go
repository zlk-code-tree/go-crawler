package product_editSchema_request

import (
	"doudian.com/open/sdk_golang/api/product_editSchema/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductEditSchemaRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductEditSchemaParam 
}
func (c *ProductEditSchemaRequest) GetUrlPath() string{
	return "/product/editSchema"
}


func New() *ProductEditSchemaRequest{
	request := &ProductEditSchemaRequest{
		Param: &ProductEditSchemaParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductEditSchemaRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_editSchema_response.ProductEditSchemaResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_editSchema_response.ProductEditSchemaResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductEditSchemaRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductEditSchemaRequest) GetParams() *ProductEditSchemaParam{
	return c.Param
}


type ProductEditSchemaParam struct {
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 是否提审 true提审、false保存草稿
	Commit bool `json:"commit"`
	// 商品发布token
	Token string `json:"token"`
	// schema数据
	Model string `json:"model"`
	// 预测记录id，类目预测记录key=category_recommend_id
	RecommendIds *map[string]string `json:"recommend_ids"`
}
