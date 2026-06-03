package sms_send_request

import (
	"doudian.com/open/sdk_golang/api/sms_send/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SmsSendRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SmsSendParam 
}
func (c *SmsSendRequest) GetUrlPath() string{
	return "/sms/send"
}


func New() *SmsSendRequest{
	request := &SmsSendRequest{
		Param: &SmsSendParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SmsSendRequest) Execute(accessToken *doudian_sdk.AccessToken) (*sms_send_response.SmsSendResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &sms_send_response.SmsSendResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SmsSendRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SmsSendRequest) GetParams() *SmsSendParam{
	return c.Param
}


type SmsTestVerification struct {
	// 明文手机号
	PostTel *string `json:"post_tel"`
	// 验证码
	Code *string `json:"code"`
}
type SmsSendParam struct {
	// 短信发送渠道，主要做资源隔离
	SmsAccount string `json:"sms_account"`
	// 签名
	Sign string `json:"sign"`
	// 短信模版id
	TemplateId string `json:"template_id"`
	// 短信模板占位符要替换的值
	TemplateParam string `json:"template_param"`
	// 透传字段，回执的时候原样返回给调用方，最大长度512字符
	Tag *string `json:"tag"`
	// 不支持手机号明文，仅支持手机号密文。同时传outbound_id和post_tel，以post_tel为准，不能同时为空
	PostTel *string `json:"post_tel"`
	// 用户自定义扩展码，仅当允许自定义扩展码的时候生效
	UserExtCode *string `json:"user_ext_code"`
	// 外呼id，由/member/getOutboundId接口获得，/member/getOutboundId当前还未开放，outbound_id暂不支持使用
	OutboundId *string `json:"outbound_id"`
	// 短链ID，需要链接统计效果必传，并且tag需要以map[string]interface{}的json形式传入，并且带上"link_id":""
	LinkId *string `json:"link_id"`
	// 小程序短链批次ID，需要链接统计效果必传，并且tag需要以map[string]interface{}的json形式传入，并且带上"mini_app_link_id":""
	MiniAppLinkId *string `json:"mini_app_link_id"`
	// 抖音openID
	DouyinOpenId *string `json:"douyin_open_id"`
	// 测试发送验证码
	SmsTestVerification *SmsTestVerification `json:"sms_test_verification"`
	// 订单id（店铺单）
	OrderId *string `json:"order_id"`
	// 售后单id
	AfterSaleId *string `json:"after_sale_id"`
}
