package spu_getSpuRule_request

import (
	"doudian.com/open/sdk_golang/api/spu_getSpuRule/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SpuGetSpuRuleRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SpuGetSpuRuleParam 
}
func (c *SpuGetSpuRuleRequest) GetUrlPath() string{
	return "/spu/getSpuRule"
}


func New() *SpuGetSpuRuleRequest{
	request := &SpuGetSpuRuleRequest{
		Param: &SpuGetSpuRuleParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SpuGetSpuRuleRequest) Execute(accessToken *doudian_sdk.AccessToken) (*spu_getSpuRule_response.SpuGetSpuRuleResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &spu_getSpuRule_response.SpuGetSpuRuleResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SpuGetSpuRuleRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SpuGetSpuRuleRequest) GetParams() *SpuGetSpuRuleParam{
	return c.Param
}


type SpuGetSpuRuleParam struct {
	// 类目ID
	CategoryId int64 `json:"category_id"`
	// 类目2.0渠道字段。0为获取门捷列夫类目下的规则。1为获取标准类目的规则。注意类目ID号段要与channel匹配，否则会报错。如23362为门捷列夫类目ID，channl需要为0.系统会根据category_id自身字段判断是门捷还是标准类目，但是channel优先级更高
	Channel *int32 `json:"channel"`
}
