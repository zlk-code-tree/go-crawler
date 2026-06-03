package token_create_request

import (
	"doudian.com/open/sdk_golang/api/token_create/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type TokenCreateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *TokenCreateParam 
}
func (c *TokenCreateRequest) GetUrlPath() string{
	return "/token/create"
}


func New() *TokenCreateRequest{
	request := &TokenCreateRequest{
		Param: &TokenCreateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *TokenCreateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*token_create_response.TokenCreateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &token_create_response.TokenCreateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *TokenCreateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *TokenCreateRequest) GetParams() *TokenCreateParam{
	return c.Param
}


type TokenCreateParam struct {
	// 授权码；参数必传，工具型应用: 传code值；自用型应用:传""
	Code *string `json:"code"`
	// 授权类型 ；【工具型应用:authorization_code  自用型应用:authorization_self】，如果自用型应用有授权code，传authorization_code
	GrantType string `json:"grant_type"`
	// 判断测试店铺标识 ，非必传，若新增测试店铺传1，若不是则不必传
	TestShop *string `json:"test_shop"`
	// 店铺ID，抖店自研应用使用。当auth_subject_type不为空时，该字段请勿传值，请将值传入到auth_id字段中
	ShopId *string `json:"shop_id"`
	// 授权id，配合auth_subject_type字段使用。当auth_subject_type不为空时，请使用auth_id字段传值，shop_id请勿使用。
	AuthId *string `json:"auth_id"`
	// 授权主体类型，配合auth_id字段使用，YunCang -云仓；WuLiuShang -物流商；WLGongYingShang -物流供应商；MiniApp -小程序；MCN-联盟MCN机构；DouKe-联盟抖客 ；Colonel-联盟团长；
	AuthSubjectType *string `json:"auth_subject_type"`
}
