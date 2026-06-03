package logistics_pushOnlineServiceConfig_request

import (
	"doudian.com/open/sdk_golang/api/logistics_pushOnlineServiceConfig/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsPushOnlineServiceConfigRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsPushOnlineServiceConfigParam 
}
func (c *LogisticsPushOnlineServiceConfigRequest) GetUrlPath() string{
	return "/logistics/pushOnlineServiceConfig"
}


func New() *LogisticsPushOnlineServiceConfigRequest{
	request := &LogisticsPushOnlineServiceConfigRequest{
		Param: &LogisticsPushOnlineServiceConfigParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsPushOnlineServiceConfigRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_pushOnlineServiceConfig_response.LogisticsPushOnlineServiceConfigResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_pushOnlineServiceConfig_response.LogisticsPushOnlineServiceConfigResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsPushOnlineServiceConfigRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsPushOnlineServiceConfigRequest) GetParams() *LogisticsPushOnlineServiceConfigParam{
	return c.Param
}


type SendAddress struct {
	// 详细地址（暂时无效，可写死空字符串）
	DetailAddress string `json:"detail_address"`
	// 国家编码，写死CHN
	CountryCode string `json:"country_code"`
	// 地区名称
	DistrictName string `json:"district_name"`
	// 省名称
	ProvinceName string `json:"province_name"`
	// 街道名称
	StreetName string `json:"street_name"`
	// 城市名称
	CityName string `json:"city_name"`
}
type ReceAddress struct {
	// 详细地址（暂时无效，可写死空字符串）
	DetailAddress string `json:"detail_address"`
	// 国家编码，写死CHN
	CountryCode string `json:"country_code"`
	// 街道名称
	StreetName string `json:"street_name"`
	// 省名称
	ProvinceName string `json:"province_name"`
	// 城市名称
	CityName string `json:"city_name"`
	// 区县名称
	DistrictName string `json:"district_name"`
}
type RouteListItem struct {
	// 寄件地址
	SendAddress *SendAddress `json:"send_address"`
	// 收件地址
	ReceAddress *ReceAddress `json:"rece_address"`
}
type LogisticsPushOnlineServiceConfigParam struct {
	// 服务类型。0-增值服务；1-产品类型
	ServiceType *string `json:"service_type"`
	// 服务描述
	Desc *string `json:"desc"`
	// 是否自动开通。0-否；1-是
	IsAutoOpen *int32 `json:"is_auto_open"`
	// 操作人名称
	OperatorName *string `json:"operator_name"`
	// 操作人ID
	OperatorId *string `json:"operator_id"`
	// 服务名称（也是面单打印的名称）
	Name *string `json:"name"`
	// 服务code
	Code *string `json:"code"`
	// 线路限制
	RouteList []RouteListItem `json:"route_list"`
}
