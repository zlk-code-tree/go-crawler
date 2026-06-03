package sms_batchSend_request

import (
	"doudian.com/open/sdk_golang/api/sms_batchSend/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SmsBatchSendRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SmsBatchSendParam 
}
func (c *SmsBatchSendRequest) GetUrlPath() string{
	return "/sms/batchSend"
}


func New() *SmsBatchSendRequest{
	request := &SmsBatchSendRequest{
		Param: &SmsBatchSendParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SmsBatchSendRequest) Execute(accessToken *doudian_sdk.AccessToken) (*sms_batchSend_response.SmsBatchSendResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &sms_batchSend_response.SmsBatchSendResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SmsBatchSendRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SmsBatchSendRequest) GetParams() *SmsBatchSendParam{
	return c.Param
}


type SmsMessageListItem struct {
	// 订单id
	OrderId *string `json:"order_id"`
	// 抖音小程序openID
	DouyinOpenId *string `json:"douyin_open_id"`
	// 不支持手机号明文，仅支持手机号密文。同时传outbound_id和post_tel，以post_tel为准，不能同时为空
	PostTel *string `json:"post_tel"`
	// 外呼id，由/member/getOutboundId接口获取。/member/getOutboundId当前还未开放，outbound_id暂不支持使用
	OutboundId *string `json:"outbound_id"`
	// 参数
	TemplateParam string `json:"template_param"`
	// 售后单id
	AfterSaleId *string `json:"after_sale_id"`
}
type SmsBatchSendParam struct {
	// 短信发送渠道，主要做资源隔离
	SmsAccount string `json:"sms_account"`
	// 短信签名
	Sign string `json:"sign"`
	// 短信列表
	SmsMessageList []SmsMessageListItem `json:"sms_message_list"`
	// 短信模板id
	TemplateId string `json:"template_id"`
	// 透传字段，回执的时候原样返回给调用方，最大长度512字符
	Tag *string `json:"tag"`
	// 用户自定义扩展码，仅当允许自定义扩展码的时候生效
	UserExtCode *string `json:"user_ext_code"`
	// 需要链接统计效果时必传，且    2. tag需要以map[string]interface{}的json形式传入，并且带上"link_id":""
	LinkId *string `json:"link_id"`
	// 需要链接统计效果时必传，且    2. tag需要以map[string]interface{}的json形式传入，并且带上"mini_app_link_id":""
	MiniAppLinkId *string `json:"mini_app_link_id"`
}
