package logistics_merchant_confirmUpdateResult_request

import (
	"doudian.com/open/sdk_golang/api/logistics_merchant_confirmUpdateResult/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsMerchantConfirmUpdateResultRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsMerchantConfirmUpdateResultParam 
}
func (c *LogisticsMerchantConfirmUpdateResultRequest) GetUrlPath() string{
	return "/logistics/merchant/confirmUpdateResult"
}


func New() *LogisticsMerchantConfirmUpdateResultRequest{
	request := &LogisticsMerchantConfirmUpdateResultRequest{
		Param: &LogisticsMerchantConfirmUpdateResultParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsMerchantConfirmUpdateResultRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_merchant_confirmUpdateResult_response.LogisticsMerchantConfirmUpdateResultResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_merchant_confirmUpdateResult_response.LogisticsMerchantConfirmUpdateResultResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsMerchantConfirmUpdateResultRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsMerchantConfirmUpdateResultRequest) GetParams() *LogisticsMerchantConfirmUpdateResultParam{
	return c.Param
}


type Contact struct {
	// 电话号码
	Phone string `json:"phone"`
	// 姓名
	Name string `json:"name"`
}
type Address struct {
	// 市名称
	CityName string `json:"city_name"`
	// 街道编码
	StreetCode string `json:"street_code"`
	// 省编码
	ProvinceCode string `json:"province_code"`
	// 街道名称
	StreetName string `json:"street_name"`
	// 市编码
	CityCode string `json:"city_code"`
	// 区编码
	DistrictCode string `json:"district_code"`
	// 区名称
	DistrictName string `json:"district_name"`
	// 详细地址
	DetailAddress string `json:"detail_address"`
	// 省名称
	ProvinceName string `json:"province_name"`
}
type Sender struct {
	// 联系方式
	Contact *Contact `json:"contact"`
	// 地址
	Address *Address `json:"address"`
}
type LogisticsMerchantConfirmUpdateResultParam struct {
	// 商家名称
	MerchantName string `json:"merchant_name"`
	// 售后单号
	AfterSaleId string `json:"after_sale_id"`
	// 寄件人
	Sender *Sender `json:"sender"`
	// 开始取件时间
	StartFetchTime string `json:"start_fetch_time"`
	// 结束取件时间
	EndFetchTime string `json:"end_fetch_time"`
	// 请求id，用于幂等
	RequestId string `json:"request_id"`
	// 更新结果
	Result string `json:"result"`
}
