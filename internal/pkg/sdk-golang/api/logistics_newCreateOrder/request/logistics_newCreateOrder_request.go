package logistics_newCreateOrder_request

import (
	"doudian.com/open/sdk_golang/api/logistics_newCreateOrder/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsNewCreateOrderRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsNewCreateOrderParam 
}
func (c *LogisticsNewCreateOrderRequest) GetUrlPath() string{
	return "/logistics/newCreateOrder"
}


func New() *LogisticsNewCreateOrderRequest{
	request := &LogisticsNewCreateOrderRequest{
		Param: &LogisticsNewCreateOrderParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsNewCreateOrderRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_newCreateOrder_response.LogisticsNewCreateOrderResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_newCreateOrder_response.LogisticsNewCreateOrderResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsNewCreateOrderRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsNewCreateOrderRequest) GetParams() *LogisticsNewCreateOrderParam{
	return c.Param
}


type EbillDeliveryInfo struct {
	// 枚举（OnlineShip）
	Scene *string `json:"scene"`
}
type LogisticsNewCreateOrderParam struct {
	// 寄件人信息
	SenderInfo *SenderInfo `json:"sender_info"`
	// 物流服务商编码
	LogisticsCode string `json:"logistics_code"`
	// 详细订单列表
	OrderInfos []OrderInfosItem `json:"order_infos"`
	// 共享账号场景下需传，代表实际使用取号服务的shop_id（需与order_id匹配）；若无法获取到该shop_id，value传值 -1
	UserId *int64 `json:"user_id"`
	// 派送要求（目前仅德邦支持）
	DeliveryReq *DeliveryReq `json:"delivery_req"`
	// 订单渠道来源编码，具体请参考[下单渠道来源编码表](https://bytedance.feishu.cn/sheets/shtcngIVwcJlgXLzWhEtKrmv7Af)，当order_id订单号为非抖音订单时必传
	OrderChannel *string `json:"order_channel"`
	// 发货场景结构体
	EbillDeliveryInfo *EbillDeliveryInfo `json:"ebill_delivery_info"`
}
type Contact struct {
	// 寄件人姓名
	Name string `json:"name"`
	// 寄件人固话（和mobile二选一）
	Phone *string `json:"phone"`
	// 寄件人移动电话（和phone二选一）
	Mobile *string `json:"mobile"`
}
type SenderInfo struct {
	// 寄件人地址信息（需要调用/logistics/listShopNetsite查询）
	Address *Address `json:"address"`
	// 寄件人联系信息
	Contact *Contact `json:"contact"`
}
type Address_4_4 struct {
	// 国家编码（默认中国CHN）
	CountryCode string `json:"country_code"`
	// 省名称
	ProvinceName string `json:"province_name"`
	// 市名称
	CityName string `json:"city_name"`
	// 区/县名称
	DistrictName string `json:"district_name"`
	// 街道名称
	StreetName *string `json:"street_name"`
	// 剩余详细地址，支持密文
	DetailAddress string `json:"detail_address"`
	// 省编码code
	ProvinceCode *string `json:"province_code"`
	// 市编码code
	CityCode *string `json:"city_code"`
	// 区编码code
	DistrictCode *string `json:"district_code"`
	// 街道编码code
	StreetCode *string `json:"street_code"`
}
type ReceiverInfo struct {
	// 收件人地址信息
	Address *Address_4_4 `json:"address"`
	// 收件人联系信息
	Contact *Contact `json:"contact"`
}
type Warehouse struct {
	// true 总对总门店发货
	IsSumUp bool `json:"is_sum_up"`
	// 仓库id编码
	WhCode *string `json:"wh_code"`
	// 仓库订单号(丹鸟等仓发链路使用)
	WhOrderNo *string `json:"wh_order_no"`
	// 发货方式，2-门店发货 3-仓库发货（不传默认为3）
	DeliveryType *string `json:"delivery_type"`
}
type NetInfo struct {
	// 物流服务商类型，直营/加盟
	Category *string `json:"category"`
	// 网点编码,当category为加盟类型时，该字段必填；为直营类型时可不传；对总模式该字段均为非必填
	NetCode *string `json:"net_code"`
	// 总对总账号 月结账号
	MonthlyAccount *string `json:"monthly_account"`
	// 总对总密码
	SecretKey *string `json:"secret_key"`
}
type DeliveryReq struct {
	// true
	IsCenterDelivery *bool `json:"is_center_delivery"`
	// true
	IsPickupSelf *bool `json:"is_pickup_self"`
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
	// 街道名称
	StreetName *string `json:"street_name"`
	// 剩余详细地址
	DetailAddress string `json:"detail_address"`
}
type ServiceListItem struct {
	// 增值服务类型
	ServiceCode *string `json:"service_code"`
	// 增值服务对应的value值，如果增值类型涉及金额会校验，单位：分
	ServiceValue *string `json:"service_value"`
}
type PodModelAddress struct {
	// 国家编码（默认CHN，目前只有国内业务）
	CountryCode string `json:"country_code"`
	// 省名称
	ProvinceName string `json:"province_name"`
	// 市名称
	CityName string `json:"city_name"`
	// 区/县名称
	DistrictName string `json:"district_name"`
	// 街道名称
	StreetName *string `json:"street_name"`
	// 剩余详细地址
	DetailAddress string `json:"detail_address"`
}
type ItemsItem struct {
	// 商品名称
	ItemName string `json:"item_name"`
	// 商品规格
	ItemSpecs *string `json:"item_specs"`
	// 商品数量
	ItemCount int32 `json:"item_count"`
	// 单件商品体积（cm3
	ItemVolume *int32 `json:"item_volume"`
	// 单件商品重量（g)
	ItemWeight *int32 `json:"item_weight"`
	// 单件总净重量（g）
	ItemNetWeight *int32 `json:"item_net_weight"`
}
type OrderInfosItem struct {
	// 订单号（合单情况下，isv只需传其中一个订单号，传入的订单号需去掉末尾后缀字母A)；非抖音订单需要传order_channel字段，长度仅支持32个字符，格式：数字、大小写字母及“-”、“/”2种分隔符的组合字符串，例如：Doudian-123456
	OrderId string `json:"order_id"`
	// 包裹id。拆包（一个订单获取多个独立的运单，非子母件）场景使用，只能传入15位以内的数字、字母和下划线，大小写敏感（即123A,123a为不同包裹id），仅第2个及以后的运单取号时传不同id。
	PackId *string `json:"pack_id"`
	// 增值服务列表，更多增值服务参考[抖音电商电子面单对接文档](https://bytedance.feishu.cn/wiki/wikcnNVdm3TlJutNCUlKmiVuwnf)
	ServiceList []ServiceListItem `json:"service_list"`
	// 快递产品类型，更多产品类型参考[抖音电商电子面单对接文档](https://bytedance.feishu.cn/wiki/wikcnNVdm3TlJutNCUlKmiVuwnf)
	ProductType *string `json:"product_type"`
	// 支付方式：1-寄付月结，2-寄付现结）若不传，默认为商家与物流商网点约定的支付方式
	PayMethod *int16 `json:"pay_method"`
	// 运费金额，单位为分
	PayAmount *int64 `json:"pay_amount"`
	// 回单寄回地址
	PodModelAddress *PodModelAddress `json:"pod_model_address"`
	// 收件人信息
	ReceiverInfo *ReceiverInfo `json:"receiver_info"`
	// 商品明细列表
	Items []ItemsItem `json:"items"`
	// 要求上门取件时间段
	SenderFetchTime *string `json:"sender_fetch_time"`
	// 是否返回签回单（签单返还）的运单号，支持以下值：1：要求 0：不要求\"
	IsSignBack *int16 `json:"is_sign_back"`
	// 订单备注
	Remark *string `json:"remark"`
	// 备用扩展字段（非必传字段，如果传值不可为"null",按照示例来传。）
	Extra *string `json:"extra"`
	// 子母件的总包裹数量，仅子母件场景使用，使用时包裹数量必须≥2，不传则默认为普通的一单一包裹运单。
	TotalPackCount *int32 `json:"total_pack_count"`
	// 商品总重量，单位：克（仅支持顺丰物流使用）
	TotalWeight *string `json:"total_weight"`
	// 商品总长，单位：cm（仅支持顺丰物流使用）
	TotalLength *string `json:"total_length"`
	// 商品总宽，单位：cm（仅支持顺丰物流使用）
	TotalWidth *string `json:"total_width"`
	// 商品总高，单位：cm（仅支持顺丰物流使用）
	TotalHeight *string `json:"total_height"`
	// 商品总体积，单位：cm3（仅支持顺丰物流使用）
	Volume *string `json:"volume"`
	// 仓、门店、总对总发货
	Warehouse *Warehouse `json:"warehouse"`
	// 总对总信息门店信息
	NetInfo *NetInfo `json:"net_info"`
	// 物料码
	ShippingCode *string `json:"shipping_code"`
	// 顺丰极效前置场景（必填）使用  2:极效前置单
	SpecialDeliveryTypeCode *string `json:"special_delivery_type_code"`
	// 顺丰极效前置场景（必填）使用   Y:若不支持则返回普通运单 N:若不支持则返回错误码
	SpecialDeliveryTypeValue *string `json:"special_delivery_type_value"`
	// 包裹总重量（g）（丹鸟专用，其余物流商取号不要传该字段）
	PackageWeight *int32 `json:"package_weight"`
	// 合单订单号列表
	CombineOrders []string `json:"combine_orders"`
}
