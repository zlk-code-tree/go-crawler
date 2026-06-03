package order_logisticsAdd_request

import (
	"doudian.com/open/sdk_golang/api/order_logisticsAdd/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderLogisticsAddRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderLogisticsAddParam 
}
func (c *OrderLogisticsAddRequest) GetUrlPath() string{
	return "/order/logisticsAdd"
}


func New() *OrderLogisticsAddRequest{
	request := &OrderLogisticsAddRequest{
		Param: &OrderLogisticsAddParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderLogisticsAddRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_logisticsAdd_response.OrderLogisticsAddResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_logisticsAdd_response.OrderLogisticsAddResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderLogisticsAddRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderLogisticsAddRequest) GetParams() *OrderLogisticsAddParam{
	return c.Param
}


type BatchAndTraceCodeListItem struct {
	// 溯源码
	TraceCode *string `json:"trace_code"`
	// 批次号
	BatchCode *string `json:"batch_code"`
}
type TraceInfoListItem struct {
	// 溯源码详情
	BatchAndTraceCodeList []BatchAndTraceCodeListItem `json:"batch_and_trace_code_list"`
	// sku单id
	SkuOrderId *string `json:"sku_order_id"`
}
type OrderLogisticsAddParam struct {
	// 订单ID
	OrderId string `json:"order_id"`
	// 【已废弃】物流公司ID。请使用company_code字段。
	LogisticsId *int64 `json:"logistics_id"`
	// 物流公司名称
	Company *string `json:"company"`
	// (必填)物流公司code,可从/order/logisticsCompanyList接口获取。
	CompanyCode *string `json:"company_code"`
	// (必填)快递单号
	LogisticsCode *string `json:"logistics_code"`
	// 是否拒绝退款申请（true表示拒绝退款，并继续发货；不传或为false表示有退款需要处理，拒绝发货），is_refund_reject和is_reject_refund随机使用一个即可
	IsRefundReject *bool `json:"is_refund_reject"`
	// 是否拒绝退款申请（true表示拒绝退款，并继续发货；不传或为false表示有退款需要处理，拒绝发货），is_refund_reject和is_reject_refund随机使用一个即可
	IsRejectRefund *bool `json:"is_reject_refund"`
	// 商品序列号，15-17位数字
	SerialNumberList []string `json:"serial_number_list"`
	// 发货地址ID,通过地址库列表【/address/list】接口查询
	AddressId *int64 `json:"address_id"`
	// 门店ID；抖超小时达店铺需要填写；
	StoreId *int64 `json:"store_id"`
	// 退货地址ID,通过地址库列表【/address/list】接口查询。当传入该字段时，可实现自动审核场景买家退货至商家指定退货地址：例如商家配置的售后小助手以及平台闪电退货规则，会获取发货时传入的after_sale_address_id地址id对应的地址给买家展示进行商品退回；
	AfterSaleAddressId *int64 `json:"after_sale_address_id"`
	// 挂在商品单上的信息，只是用于传递和商品单关联的信息，比如SN/69/IMEI码。目前接口还是按照店铺单维度推进发货状态，不会按照传递过来的product_order_id推进订单状态。比如，店铺单下有A、B、C三个商品单，其中A单需要SN/69/IMEI码，这时候只需要把A单关联的SN/69/IMEI传递过来即可。接口会把A、B、C三个商品单都推进到已发货状态。
	ProductOrders []ProductOrdersItem `json:"product_orders"`
	// 【注意！目前仅「即时零售」业务使用！！！】溯源码list，多个sku放一起，用内部的sku_order_id区分,举个例子，假设你有3个sku，你就创建一个list，长度为3，list里面每一个元素都是一个sku维度，用sku_order_id区分，其中，每一个sku的信息，里面是可以传多个批次和溯源码的，具体看你需要传几个，你就在batch_and_trace_code_list里面传几个；
	TraceInfoList []TraceInfoListItem `json:"trace_info_list"`
}
type SerialNoListItem struct {
	// SN/69/IMEI码的值。
	Value []string `json:"value"`
	// SN/69/IMEI码的类型
	Key *string `json:"key"`
}
type ProductOrdersItem struct {
	// 需要的SN/69/IMEI码列表。
	SerialNoList []SerialNoListItem `json:"serial_no_list"`
	// 商品单单号
	ProductOrderId *string `json:"product_order_id"`
}
