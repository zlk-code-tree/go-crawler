package promise_SaveTemplate_request

import (
	"doudian.com/open/sdk_golang/api/promise_SaveTemplate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type PromiseSaveTemplateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *PromiseSaveTemplateParam 
}
func (c *PromiseSaveTemplateRequest) GetUrlPath() string{
	return "/promise/SaveTemplate"
}


func New() *PromiseSaveTemplateRequest{
	request := &PromiseSaveTemplateRequest{
		Param: &PromiseSaveTemplateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *PromiseSaveTemplateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*promise_SaveTemplate_response.PromiseSaveTemplateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &promise_SaveTemplate_response.PromiseSaveTemplateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *PromiseSaveTemplateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *PromiseSaveTemplateRequest) GetParams() *PromiseSaveTemplateParam{
	return c.Param
}


type WarehouseInfo struct {
	// 仓id
	WarehouseId *string `json:"warehouse_id"`
	// 仓名称
	WarehouseName *string `json:"warehouse_name"`
}
type TemplateInfo struct {
	// 传0即可
	CShopId *int64 `json:"c_shop_id"`
	// 模板ID。创建时效规则模板时，该参数不填。修改时效规则模板时，该参数为对应时效规则模板的template_id。
	TemplateId *string `json:"template_id"`
	// 模板时效类型 1-承诺发货 2-承诺送达
	TimeType *int64 `json:"time_type"`
	// 是否支持预约
	IsSupportAppointment *bool `json:"is_support_appointment"`
	// 是否支持修改预约时间
	IsSupportModifyAppointment *bool `json:"is_support_modify_appointment"`
	// 状态 1-生效中；2-已停用；3-删除
	Status *int64 `json:"status"`
	// 模板名称（必填）
	TemplateName string `json:"template_name"`
	// 仓库信息。warehouse_id和warehouse_name可以同时不填。若不填，则该模板的生效范围为全店生效。如果填写了仓库信息，则该模板的生效范围为按仓生效。
	WarehouseInfo *WarehouseInfo `json:"warehouse_info"`
	// 允许用户预约的最大天数。目前支持15-90范围内的整数。例如，15表示用户最大可约天数为15天。
	SupportAppoinmentDay *int64 `json:"support_appoinment_day"`
	// 商家名称
	ShopName *string `json:"shop_name"`
	// 业务身份
	BizTag *string `json:"biz_tag"`
	// 支持修改预约时效的最小buffer时间。目前支持1-72范围内的整数。例如，12表示订单发货的12小时之前，允许用户修改订单发货时间。
	MinHourForModifyAppointment *int64 `json:"min_hour_for_modify_appointment"`
}
type CapacityItem struct {
	// 日期
	Date *string `json:"date"`
	// 容量
	Capacity *int64 `json:"capacity"`
	// 指定日期还是default
	IsDefault *bool `json:"is_default"`
}
type AddressListItem struct {
	// 省编码
	ProvinceCode *string `json:"province_code"`
	// 省名称
	ProvinceName *string `json:"province_name"`
	// 城市编码
	CityCode *string `json:"city_code"`
	// 城市名称
	CityName *string `json:"city_name"`
	// 地区编码
	DistrictCode *string `json:"district_code"`
	// 地区名称
	DistrictName *string `json:"district_name"`
	// 街道编码
	StreetCode *string `json:"street_code"`
	// 街道名称
	StreetName *string `json:"street_name"`
}
type WaveRuleConfigsItem struct {
	// 波次时效类型。1-承诺发货 2-承诺送达
	TimeType *int64 `json:"time_type"`
	// 波次id。如果是创建新的波次，则该参数不填。如果是修改波次，则该参数为对应的修改的波次的id。
	WaveId *string `json:"wave_id"`
	// 发货/送达时间: 天数。如果time_type = 1, wave_type = 0, 则可选0D/1D/2D。如果time_type = 2, wave_type = 0, 则可选0D/1D/2D/3D/4D/5D/6D/7D。
	Day *string `json:"day"`
	// 波次对应产能id。如果是创建新的波次，则该参数不填。如果是修改波次，则该参数为对应的修改的波次的产能id。
	DetailId *string `json:"detail_id"`
	// 可在这里配置制定日期的产能。如果is_default为true，则date和capacity都不填，表示常规产能。如果is_default为false，填写date和capacity，表示指定日期的产能。例子，表示2024-07-17产能为111。
	Capacity []CapacityItem `json:"capacity"`
	// 仓库信息。warehouse_id和warehouse_name可以同时不填。若不填，则该模板的生效范围为全店生效。如果填写了仓库信息，则该模板按仓生效。
	WarehouseInfo *WarehouseInfo `json:"warehouse_info"`
	// 截单时间：时分秒。（精确到秒）
	CutoffTimeHms *string `json:"cutoff_time_hms"`
	// 波次类型 0-默认波次 1-特殊类目波次 2-不可达地址波次
	WaveType *int32 `json:"wave_type"`
	// 地址列表。当time_type = 2时，可以填写address_list，表示承诺送达的地址列表。当time_type = 1时，该参数不填。
	AddressList []AddressListItem `json:"address_list"`
}
type ExtraInfo struct {
	// 是否同意了协议
	IsProtocolAccepted *bool `json:"is_protocol_accepted"`
}
type PromiseSaveTemplateParam struct {
	// 模板信息
	TemplateInfo *TemplateInfo `json:"template_info"`
	// 波次规则配置。
	WaveRuleConfigs []WaveRuleConfigsItem `json:"wave_rule_configs"`
	// 其他信息
	ExtraInfo *ExtraInfo `json:"extra_info"`
}
