package topup_CreateOrderPreCheck_request

import (
	"doudian.com/open/sdk_golang/core"
	"doudian.com/open/sdk_golang/spi/topup_CreateOrderPreCheck/response"
	"doudian.com/open/sdk_golang/utils"
)

type TopupCreateOrderPreCheckRequest struct {
	doudian_sdk.BaseDoudianOpSpiRequest
	Param *TopupCreateOrderPreCheckParam 
	Response *topup_CreateOrderPreCheck_response.TopupCreateOrderPreCheckResponse 
}
func (c *TopupCreateOrderPreCheckRequest) GetParamJsonObject() interface{}{
	return c.Param
}


func (c *TopupCreateOrderPreCheckRequest) GetResponseObject() interface{}{
	return c.Response
}


func (c *TopupCreateOrderPreCheckRequest) Execute() (interface{}, error){
	return c.GetClient().Request(c)
}


func (c *TopupCreateOrderPreCheckRequest) ResponseJson() (string, error){
	responseObj, err := c.Execute()
	if err != nil {
		return "", err
	}
	return utils.MarshalNoErr(responseObj), nil
}


func New() *TopupCreateOrderPreCheckRequest{
	
	request := new(TopupCreateOrderPreCheckRequest)
	request.SetClient(doudian_sdk.DefaultDoudianOpSpiClient)
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetSpiParam(new(doudian_sdk.DoudianOpSpiParam))
	request.Param = new(TopupCreateOrderPreCheckParam)
	response := new(topup_CreateOrderPreCheck_response.TopupCreateOrderPreCheckResponse)
	response.Data = new(topup_CreateOrderPreCheck_response.TopupCreateOrderPreCheckData)
	request.Response = response
	return request
}


type TopupCreateOrderPreCheckParam struct {
	// 商品id列表
	ProductIdList []int64 `json:"product_id_list"`
	// skuId列表
	SkuIdList []int64 `json:"sku_id_list"`
	// 账号信息(这里只做参考，不同账号模板结构不同，详细结构下单捞日志看看，这个字段为空不知道怎么取去看下开平接入文档加密字段该怎么处理)，结构中的账号类型字段跟通知接口的账号类型字段含义不同（这里并不是账号类型名称，是账号类型！！！）
	AccountInfo string `json:"account_info"`
	Encrypt_account_info string `json:"encrypt_account_info"`
	// 商品id进行处理后的结果，用来解密账号id
	ProductKey string `json:"product_key"`
	// 商品编码
	CodeList []string `json:"code_list"`
	// 店铺id
	ShopId int64 `json:"shop_id"`
}
