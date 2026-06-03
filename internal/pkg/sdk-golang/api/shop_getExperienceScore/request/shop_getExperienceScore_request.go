package shop_getExperienceScore_request

import (
	"doudian.com/open/sdk_golang/api/shop_getExperienceScore/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ShopGetExperienceScoreRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ShopGetExperienceScoreParam 
}
func (c *ShopGetExperienceScoreRequest) GetUrlPath() string{
	return "/shop/getExperienceScore"
}


func New() *ShopGetExperienceScoreRequest{
	request := &ShopGetExperienceScoreRequest{
		Param: &ShopGetExperienceScoreParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ShopGetExperienceScoreRequest) Execute(accessToken *doudian_sdk.AccessToken) (*shop_getExperienceScore_response.ShopGetExperienceScoreResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &shop_getExperienceScore_response.ShopGetExperienceScoreResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ShopGetExperienceScoreRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ShopGetExperienceScoreRequest) GetParams() *ShopGetExperienceScoreParam{
	return c.Param
}


type ShopGetExperienceScoreParam struct {
}
