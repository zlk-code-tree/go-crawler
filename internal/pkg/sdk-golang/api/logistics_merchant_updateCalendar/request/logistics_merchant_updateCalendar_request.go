package logistics_merchant_updateCalendar_request

import (
	"doudian.com/open/sdk_golang/api/logistics_merchant_updateCalendar/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsMerchantUpdateCalendarRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsMerchantUpdateCalendarParam 
}
func (c *LogisticsMerchantUpdateCalendarRequest) GetUrlPath() string{
	return "/logistics/merchant/updateCalendar"
}


func New() *LogisticsMerchantUpdateCalendarRequest{
	request := &LogisticsMerchantUpdateCalendarRequest{
		Param: &LogisticsMerchantUpdateCalendarParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsMerchantUpdateCalendarRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_merchant_updateCalendar_response.LogisticsMerchantUpdateCalendarResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_merchant_updateCalendar_response.LogisticsMerchantUpdateCalendarResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsMerchantUpdateCalendarRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsMerchantUpdateCalendarRequest) GetParams() *LogisticsMerchantUpdateCalendarParam{
	return c.Param
}


type MerchantCalendarListItem struct {
	// 区县code编码，000000表示默认编码，匹配所有地址
	DistrictCode *string `json:"district_code"`
	// 日期列表，列表中的每个元素格式为yyyy-MM-dd
	DateList []string `json:"date_list"`
}
type LogisticsMerchantUpdateCalendarParam struct {
	// 商家名称
	MerchantName *string `json:"merchant_name"`
	// 商家取件日历列表，最多同时上传100个区县的取件日历
	MerchantCalendarList []MerchantCalendarListItem `json:"merchant_calendar_list"`
}
