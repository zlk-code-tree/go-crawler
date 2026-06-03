package logistics_registerPackageRoute_request

import (
	"doudian.com/open/sdk_golang/api/logistics_registerPackageRoute/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsRegisterPackageRouteRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsRegisterPackageRouteParam 
}
func (c *LogisticsRegisterPackageRouteRequest) GetUrlPath() string{
	return "/logistics/registerPackageRoute"
}


func New() *LogisticsRegisterPackageRouteRequest{
	request := &LogisticsRegisterPackageRouteRequest{
		Param: &LogisticsRegisterPackageRouteParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsRegisterPackageRouteRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_registerPackageRoute_response.LogisticsRegisterPackageRouteResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_registerPackageRoute_response.LogisticsRegisterPackageRouteResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsRegisterPackageRouteRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsRegisterPackageRouteRequest) GetParams() *LogisticsRegisterPackageRouteParam{
	return c.Param
}


type Contact struct {
	// 姓名
	Name string `json:"name"`
	// 手机号
	Phone string `json:"phone"`
	// 邮箱
	Email *string `json:"email"`
}
type Street struct {
	// 街道
	Name string `json:"name"`
	// code
	Code *string `json:"code"`
}
type Address struct {
	// 省
	Province *Province `json:"province"`
	// 市
	City *City `json:"city"`
	// 区
	Town *Town `json:"town"`
	// 街道
	Street *Street `json:"street"`
	// 详细地址
	Detail string `json:"detail"`
}
type Receiver struct {
	// 联系人
	Contact *Contact `json:"contact"`
	// 地址
	Address *Address `json:"address"`
}
type CargoListItem struct {
	// 名称
	Name *string `json:"name"`
	// 质量
	Quantity *int32 `json:"quantity"`
	// 体积
	Volume *int32 `json:"volume"`
	// 总重
	TotalWeight int32 `json:"total_weight"`
	// 净重
	TotalNetWeight int32 `json:"total_net_weight"`
	// 单位
	Unit *string `json:"unit"`
}
type Province struct {
	// 省
	Name string `json:"name"`
	// code
	Code *string `json:"code"`
}
type City struct {
	// 市
	Name string `json:"name"`
	// code
	Code *string `json:"code"`
}
type Town struct {
	// 区
	Name string `json:"name"`
	// code
	Code *string `json:"code"`
}
type Sender struct {
	// 联系人
	Contact *Contact `json:"contact"`
	// 地址
	Address *Address `json:"address"`
}
type LogisticsRegisterPackageRouteParam struct {
	// 物流商
	Express string `json:"express"`
	// 运单号
	TrackNo string `json:"track_no"`
	// 外部单据id
	OuterOrderId *string `json:"outer_order_id"`
	// 外部子单据id
	OuterSubOrderId *string `json:"outer_sub_order_id"`
	// 回调url
	CallbackUrl string `json:"callback_url"`
	// 收件人
	Receiver *Receiver `json:"receiver"`
	// 寄件人
	Sender *Sender `json:"sender"`
	// 货品列表
	CargoList []CargoListItem `json:"cargo_list"`
	// 拓展
	Extend *map[string]string `json:"extend"`
}
