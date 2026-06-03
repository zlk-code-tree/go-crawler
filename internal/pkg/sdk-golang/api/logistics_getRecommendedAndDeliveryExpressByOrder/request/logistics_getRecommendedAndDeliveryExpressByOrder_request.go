package logistics_getRecommendedAndDeliveryExpressByOrder_request

import (
	"doudian.com/open/sdk_golang/api/logistics_getRecommendedAndDeliveryExpressByOrder/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsGetRecommendedAndDeliveryExpressByOrderRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsGetRecommendedAndDeliveryExpressByOrderParam 
}
func (c *LogisticsGetRecommendedAndDeliveryExpressByOrderRequest) GetUrlPath() string{
	return "/logistics/getRecommendedAndDeliveryExpressByOrder"
}


func New() *LogisticsGetRecommendedAndDeliveryExpressByOrderRequest{
	request := &LogisticsGetRecommendedAndDeliveryExpressByOrderRequest{
		Param: &LogisticsGetRecommendedAndDeliveryExpressByOrderParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsGetRecommendedAndDeliveryExpressByOrderRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_getRecommendedAndDeliveryExpressByOrder_response.LogisticsGetRecommendedAndDeliveryExpressByOrderResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_getRecommendedAndDeliveryExpressByOrder_response.LogisticsGetRecommendedAndDeliveryExpressByOrderResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsGetRecommendedAndDeliveryExpressByOrderRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsGetRecommendedAndDeliveryExpressByOrderRequest) GetParams() *LogisticsGetRecommendedAndDeliveryExpressByOrderParam{
	return c.Param
}


type LogisticsGetRecommendedAndDeliveryExpressByOrderParam struct {
	// 发货地址
	SenderAddress *SenderAddress `json:"sender_address"`
	// 订单信息列表
	OrderInfoList []OrderInfoListItem `json:"order_info_list"`
	// 订单平台来源编码，不传默认抖店
	OrderChannel *string `json:"order_channel"`
}
type SenderAddress struct {
	// 1
	StreetCode *string `json:"street_code"`
	// 1
	AddressId *int64 `json:"address_id"`
	// 详细地址，如果没有则传空字符串即可
	DetailAddress string `json:"detail_address"`
	// 省
	ProvinceName string `json:"province_name"`
	// 区
	DistrictName string `json:"district_name"`
	// 1
	CountryCode *string `json:"country_code"`
	// 1
	CityCode *string `json:"city_code"`
	// 1
	DistrictCode *string `json:"district_code"`
	// 1
	ZipCode *string `json:"zip_code"`
	// 市
	CityName string `json:"city_name"`
	// 街道
	StreetName *string `json:"street_name"`
	// 1
	ProvinceCode *string `json:"province_code"`
}
type OrderInfoListItem struct {
	// 产品类型，可不传
	ProductType *string `json:"product_type"`
	// 订单号
	OrderId string `json:"order_id"`
	// 物流商集合，不传默认：yuantong、zhongtong、yunda、shunfeng、jd、jtexpress、shentong
	ExpressList []string `json:"express_list"`
}
