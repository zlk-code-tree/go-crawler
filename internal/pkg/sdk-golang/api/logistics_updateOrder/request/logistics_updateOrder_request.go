package logistics_updateOrder_request

import (
	"doudian.com/open/sdk_golang/api/logistics_updateOrder/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsUpdateOrderRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsUpdateOrderParam 
}
func (c *LogisticsUpdateOrderRequest) GetUrlPath() string{
	return "/logistics/updateOrder"
}


func New() *LogisticsUpdateOrderRequest{
	request := &LogisticsUpdateOrderRequest{
		Param: &LogisticsUpdateOrderParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsUpdateOrderRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_updateOrder_response.LogisticsUpdateOrderResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_updateOrder_response.LogisticsUpdateOrderResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsUpdateOrderRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsUpdateOrderRequest) GetParams() *LogisticsUpdateOrderParam{
	return c.Param
}


type ReceiverInfo struct {
	// 收件人地址信息
	Address *Address `json:"address"`
	// 收件人联系信息
	Contact *Contact `json:"contact"`
}
type ItemsItem struct {
	// 商品名称
	ItemName string `json:"item_name"`
	// 商品规格
	ItemSpecs *string `json:"item_specs"`
	// 商品数量
	ItemCount int32 `json:"item_count"`
	// 单件商品体积（cm3）
	ItemVolume *int32 `json:"item_volume"`
	// 单件商品重量（g)
	ItemWeight *int32 `json:"item_weight"`
	// 单件总净重量（g）
	ItemNetWeight *int32 `json:"item_net_weight"`
}
type Warehouse struct {
	// 目前该字段无效，统一传false
	IsSumUp bool `json:"is_sum_up"`
	// 仓库订单号(丹鸟等仓发链路使用)
	WhOrderNo *string `json:"wh_order_no"`
}
type LogisticsUpdateOrderParam struct {
	// 寄件人信息
	SenderInfo *SenderInfo `json:"sender_info"`
	// 收件人信息
	ReceiverInfo *ReceiverInfo `json:"receiver_info"`
	// 物流服务商编码
	LogisticsCode string `json:"logistics_code"`
	// 运单号
	TrackNo string `json:"track_no"`
	// 商品明细列表
	Items []ItemsItem `json:"items"`
	// 备注
	Remark *string `json:"remark"`
	// 备用扩展字段
	Extra *string `json:"extra"`
	// 实际使用取号服务店铺user_id
	UserId *int64 `json:"user_id"`
	// 总体积 货物的总体积或长，宽，高 ；整数 单位cm
	Volume *string `json:"volume"`
	// /总重量 ；整数 用于与快递商有计抛信任协议的商家）单位克
	Weight *int64 `json:"weight"`
	// 仓、门店、总对总发货
	Warehouse *Warehouse `json:"warehouse"`
}
type Contact struct {
	// 寄件人姓名
	Name *string `json:"name"`
	// 寄件人固话（和mobile二选一）
	Phone *string `json:"phone"`
	// 寄件人移动电话（和phone二选一）
	Mobile *string `json:"mobile"`
}
type SenderInfo struct {
	// 寄件人联系信息
	Contact *Contact `json:"contact"`
}
type Address struct {
	// 国家编码（默认CHN，目前只有国内业务）
	CountryCode string `json:"country_code"`
	// 省名称
	ProvinceName string `json:"province_name"`
	// 市名称
	CityName string `json:"city_name"`
	// 区/县名称
	DistrictName string `json:"district_name"`
	// 街道名称。街道名称（street_name）和街道code（street_code），若传入时，需要一起传入。
	StreetName *string `json:"street_name"`
	// 剩余详细地址，支持密文
	DetailAddress *string `json:"detail_address"`
	// 省code编码
	ProvinceCode *string `json:"province_code"`
	// 市code编码
	CityCode *string `json:"city_code"`
	// 区code编码
	DistrictCode *string `json:"district_code"`
	// 街道code编码
	StreetCode *string `json:"street_code"`
}
