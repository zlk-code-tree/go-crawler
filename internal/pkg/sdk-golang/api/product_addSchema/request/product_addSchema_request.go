package product_addSchema_request

import (
	"doudian.com/open/sdk_golang/api/product_addSchema/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductAddSchemaRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductAddSchemaParam 
}
func (c *ProductAddSchemaRequest) GetUrlPath() string{
	return "/product/addSchema"
}


func New() *ProductAddSchemaRequest{
	request := &ProductAddSchemaRequest{
		Param: &ProductAddSchemaParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductAddSchemaRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_addSchema_response.ProductAddSchemaResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_addSchema_response.ProductAddSchemaResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductAddSchemaRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductAddSchemaRequest) GetParams() *ProductAddSchemaParam{
	return c.Param
}


type RecruitInfo struct {
	// 发品源
	RecruitSource *string `json:"recruit_source"`
	// 线索ID
	RecruitFollowId string `json:"recruit_follow_id"`
	// 招商类型
	RecruitType string `json:"recruit_type"`
}
type ProductAddSchemaParam struct {
	// 是否提审
	Commit bool `json:"commit"`
	// 商品发布提交token
	Token string `json:"token"`
	// schema数据、json协议
	Model string `json:"model"`
	// 1
	ProductId *int64 `json:"product_id"`
	// 招商信息
	RecruitInfo *RecruitInfo `json:"recruit_info"`
	// 预测记录id，类目预测记录key=category_recommend_id
	RecommendIds *map[string]string `json:"recommend_ids"`
}
