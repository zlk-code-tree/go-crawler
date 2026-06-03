package order_logisticsAddSinglePack_request

import (
	"doudian.com/open/sdk_golang/api/order_logisticsAddSinglePack/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderLogisticsAddSinglePackRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderLogisticsAddSinglePackParam 
}
func (c *OrderLogisticsAddSinglePackRequest) GetUrlPath() string{
	return "/order/logisticsAddSinglePack"
}


func New() *OrderLogisticsAddSinglePackRequest{
	request := &OrderLogisticsAddSinglePackRequest{
		Param: &OrderLogisticsAddSinglePackParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderLogisticsAddSinglePackRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_logisticsAddSinglePack_response.OrderLogisticsAddSinglePackResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_logisticsAddSinglePack_response.OrderLogisticsAddSinglePackResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderLogisticsAddSinglePackRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderLogisticsAddSinglePackRequest) GetParams() *OrderLogisticsAddSinglePackParam{
	return c.Param
}


type SerialNosListItem struct {
	// 码的类型
	Key *string `json:"key"`
	// SN/69/IMEI码的值
	Value []string `json:"value"`
}
type ShippedOrderInfoItem struct {
	// 发货需要回传的SN/69/IMEI码信息
	SerialNosList []SerialNosListItem `json:"serial_nos_list"`
	// 需要发货的子订单号
	ShippedOrderId string `json:"shipped_order_id"`
	// 上述子订单的待发货数
	ShippedNum int64 `json:"shipped_num"`
	// 已废弃
	ShippedItemIds []string `json:"shipped_item_ids"`
}
type OrderSerialNumberItem struct {
	// 父订单号
	OrderId *string `json:"order_id"`
	// 商品序列号，单个序列号长度不能超过30位字符，其中手机序列号仅支持填写15～17位数字
	SerialNumberList []string `json:"serial_number_list"`
}
type OrderLogisticsAddSinglePackParam struct {
	// 父订单ID列表
	OrderIdList []string `json:"order_id_list"`
	// 需要发货的子订单信息
	ShippedOrderInfo []ShippedOrderInfoItem `json:"shipped_order_info"`
	// 运单号
	LogisticsCode string `json:"logistics_code"`
	// 物流公司名字
	Company *string `json:"company"`
	// 请求唯一标识，相同request_id多次请求，第一次请求成功后，后续的请求会触发幂等，会直接返回第一次请求成功的结果，不会实际触发发货。
	RequestId string `json:"request_id"`
	// 是否拒绝退款申请（true表示拒绝退款，并继续发货；不传或为false表示有退款需要处理，拒绝发货），is_refund_reject和is_reject_refund随机使用一个即可
	IsRejectRefund *bool `json:"is_reject_refund"`
	// 已废弃。物流公司ID。请使用company_code字段。
	LogisticsId *string `json:"logistics_id"`
	// 物流公司Code，由接口/order/logisticsCompanyLis查询物流公司列表获得，必填
	CompanyCode *string `json:"company_code"`
	// 发货地址id
	AddressId *string `json:"address_id"`
	// 订单序列号
	OrderSerialNumber []OrderSerialNumberItem `json:"order_serial_number"`
	// 门店ID
	StoreId *int64 `json:"store_id"`
	// 退货地址ID,通过地址库列表【/address/list】接口查询。当传入该字段时，可实现自动审核场景买家退货至商家指定退货地址：例如商家配置的售后小助手以及平台闪电退货规则，会获取发货时传入的after_sale_address_id地址id对应的地址给买家展示进行商品退回；
	AfterSaleAddressId *int64 `json:"after_sale_address_id"`
}
