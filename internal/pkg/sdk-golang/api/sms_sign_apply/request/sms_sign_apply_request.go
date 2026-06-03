package sms_sign_apply_request

import (
	"doudian.com/open/sdk_golang/api/sms_sign_apply/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SmsSignApplyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SmsSignApplyParam 
}
func (c *SmsSignApplyRequest) GetUrlPath() string{
	return "/sms/sign/apply"
}


func New() *SmsSignApplyRequest{
	request := &SmsSignApplyRequest{
		Param: &SmsSignApplyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SmsSignApplyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*sms_sign_apply_response.SmsSignApplyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &sms_sign_apply_response.SmsSignApplyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SmsSignApplyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SmsSignApplyRequest) GetParams() *SmsSignApplyParam{
	return c.Param
}


type SmsSignApplyParam struct {
	// 短信发送渠道，主要做资源隔离
	SmsAccount string `json:"sms_account"`
	// 签名
	Sign string `json:"sign"`
	// 版本号 v2则走新链路
	Version *string `json:"version"`
	// 是否使用isv专用消息组，为true且申请签名与抖店后台签名不一致时，将签名申请到isv专用消息组
	AutoUseIsvDedicatedMessageGroup *bool `json:"auto_use_isv_dedicated_message_group"`
}
