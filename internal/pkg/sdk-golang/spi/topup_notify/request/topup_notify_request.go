package topup_notify_request

import (
	"doudian.com/open/sdk_golang/core"
	"doudian.com/open/sdk_golang/spi/topup_notify/response"
	"doudian.com/open/sdk_golang/utils"
)

type TopupNotifyRequest struct {
	doudian_sdk.BaseDoudianOpSpiRequest
	Param *TopupNotifyParam 
	Response *topup_notify_response.TopupNotifyResponse 
}
func (c *TopupNotifyRequest) GetParamJsonObject() interface{}{
	return c.Param
}


func (c *TopupNotifyRequest) GetResponseObject() interface{}{
	return c.Response
}


func (c *TopupNotifyRequest) Execute() (interface{}, error){
	return c.GetClient().Request(c)
}


func (c *TopupNotifyRequest) ResponseJson() (string, error){
	responseObj, err := c.Execute()
	if err != nil {
		return "", err
	}
	return utils.MarshalNoErr(responseObj), nil
}


func New() *TopupNotifyRequest{
	
	request := new(TopupNotifyRequest)
	request.SetClient(doudian_sdk.DefaultDoudianOpSpiClient)
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetSpiParam(new(doudian_sdk.DoudianOpSpiParam))
	request.Param = new(TopupNotifyParam)
	response := new(topup_notify_response.TopupNotifyResponse)
	response.Data = new(topup_notify_response.TopupNotifyData)
	request.Response = response
	return request
}


type AccountListItem struct {
	// 腾讯充值参数用户相关信息
	AccountUt string `json:"account_ut"`
	// 充值账号
	AccountVal *string `json:"account_val"`
	Encrypt_account_val *string `json:"encrypt_account_val"`
	// 账号类型
	AccountType *string `json:"account_type"`
	// 账号名称
	AccountName *string `json:"account_name"`
}
type TopupNotifyParam struct {
	// 充值平台交易单号
	TradeOrderNo *string `json:"trade_order_no"`
	// 业务类型，充值请求参数原样返回即可 （全部业务类型参见充值对接文档）
	TopupBiz *string `json:"topup_biz"`
	// 整个充值流程的计时起点 格式（yyyyMMddHHmmss）
	TimeStart *string `json:"time_start"`
	// 充值请求截止时延，单位为秒，以time_start为起点
	TimeLimit *string `json:"time_limit"`
	// 购买数量
	BuyNum *string `json:"buy_num"`
	// 单个商品充值面额。注：总充值金额=buyNum*amountUnit，单位分 流量充值为商品价格"
	AmountUnit *string `json:"amount_unit"`
	// 在电商平台商品的skuId
	SkuId *string `json:"sku_id"`
	// 商家店铺id
	ShopId *string `json:"shop_id"`
	// 充值账号列表
	AccountList []AccountListItem `json:"account_list"`
	// 商品编码（抖店后台）
	Code *string `json:"code"`
	// 支付金额
	PayAmount *string `json:"pay_amount"`
	// 用户唯一id，同一买家由相同应用（AppKey）获取时唯一
	DoudianOpenId string `json:"doudian_open_id"`
	// 外部商家订单号，普通充值不需要该字段，特殊充值场景才有，譬如腾讯
	OuterOrderId string `json:"outer_order_id"`
}
