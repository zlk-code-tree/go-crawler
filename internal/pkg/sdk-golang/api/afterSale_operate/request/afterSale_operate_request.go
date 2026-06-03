package afterSale_operate_request

import (
	"doudian.com/open/sdk_golang/api/afterSale_operate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AfterSaleOperateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AfterSaleOperateParam 
}
func (c *AfterSaleOperateRequest) GetUrlPath() string{
	return "/afterSale/operate"
}


func New() *AfterSaleOperateRequest{
	request := &AfterSaleOperateRequest{
		Param: &AfterSaleOperateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AfterSaleOperateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*afterSale_operate_response.AfterSaleOperateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &afterSale_operate_response.AfterSaleOperateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AfterSaleOperateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AfterSaleOperateRequest) GetParams() *AfterSaleOperateParam{
	return c.Param
}


type EvidenceItem struct {
	// 凭证类型，1:图片，2:视频(暂不支持)，3:音频(暂不支持展示)，4:文字(暂不支持展示)。“用户可见备注”为remark字段
	Type *int32 `json:"type"`
	// 凭证url
	Url *string `json:"url"`
	// 凭证描述
	Desc *string `json:"desc"`
}
type AfterSaleAddressDetail struct {
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段省；使用【/address/getProvince】接口获取
	ProvinceName *string `json:"province_name"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段市；需要使用【/address/getAreasByProvince】接口响应参数获取
	CityName *string `json:"city_name"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段区；需要使用【/address/getAreasByProvince】接口响应参数获取
	TownName *string `json:"town_name"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段街道名称
	StreetName *string `json:"street_name"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段地址详情
	Detail *string `json:"detail"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段收件人
	UserName *string `json:"user_name"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段联系电话，支持手机号和固定电话；固定电话需要传入区号。注意：区号和号码之间一定要传入“-”传值示例：0571-1234567；否则会报错电话号码不合法。
	Mobile *string `json:"mobile"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段省id；使用【/address/getProvince】接口获取
	ProvinceId *int64 `json:"province_id"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段市id；需要使用【/address/getAreasByProvince】接口响应参数获取
	CityId *int64 `json:"city_id"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段区id；需要使用【/address/getAreasByProvince】接口响应参数获取
	TownId *int64 `json:"town_id"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段街道id；需要使用【/address/getAreasByProvince】接口响应参数获取
	StreetId *int64 `json:"street_id"`
}
type Logistics struct {
	// 物流公司code，使用【/order/logisticsCompanyList】接口获取；type=311仅换货时需要填入
	CompanyCode *string `json:"company_code"`
	// 物流单号（快递单号），仅type=311仅换货时需要填入
	LogisticsCode *string `json:"logistics_code"`
	// 商家收件地址id（推荐使用），须通过【/address/list】获取【address_id】填入。商家在同意换货/退货时，可以传入该地址id，对应的地址会买家展示进行商品退回；
	ReceiverAddressId *int64 `json:"receiver_address_id"`
	// 已废弃字段（废弃时间2022-06-10）发件地址id
	SenderAddressId *int64 `json:"sender_address_id"`
	// 【该字段将在2023年8月31日下线】，请尽快切换使用receiver_address_id字段，商家收货地址详情；
	AfterSaleAddressDetail *AfterSaleAddressDetail `json:"after_sale_address_detail"`
}
type ItemsItem struct {
	// 维修类型（1-原商品维修；2-商品换新）
	RepairType *int64 `json:"repair_type"`
	// 售后单号
	AftersaleId string `json:"aftersale_id"`
	// 操作原因，拒绝操作必填
	Reason *string `json:"reason"`
	// 操作评论，拒绝操作必填
	Remark *string `json:"remark"`
	// 操作凭证编码，当【afterSale/rejectReasonCodeList】接口响应参数evidence_need=Y时，该参数必填；使用【afterSale/rejectReasonCodeList】接口获取响应参数中的reject_reason_code字段。
	Evidence []EvidenceItem `json:"evidence"`
	// 同意退货/同意换货物流信息，当前type=101，301，311是该集合需要传值。
	Logistics *Logistics `json:"logistics"`
	// 售后拒绝原因码，拒绝时必填填。通过/afterSale/rejectReasonCodeList获取
	RejectReasonCode *int64 `json:"reject_reason_code"`
	// 用于校验售后单版本是不是最新的版本，防止售后单变更且open侧审核通过导致资损。不传就不校验，传入后就会校验isv传入的售后单版本是不是最新的版本。 需要使用【/afterSale/Detail】接口返回的update_time字段或使用售后消息推送中的update_time字段
	UpdateTime *int64 `json:"update_time"`
}
type AfterSaleOperateParam struct {
	// 操作类型；枚举值说明文档：https://op.jinritemai.com/docs/question-docs/93/2752
	Type int32 `json:"type"`
	// 操作售后详情
	Items []ItemsItem `json:"items"`
	// 门店ID；抖超小时达业务使用；
	StoreId *int64 `json:"store_id"`
}
