package cpaypromotion_operateProductDiscount_request

import (
	"doudian.com/open/sdk_golang/api/cpaypromotion_operateProductDiscount/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CpaypromotionOperateProductDiscountRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CpaypromotionOperateProductDiscountParam 
}
func (c *CpaypromotionOperateProductDiscountRequest) GetUrlPath() string{
	return "/cpaypromotion/operateProductDiscount"
}


func New() *CpaypromotionOperateProductDiscountRequest{
	request := &CpaypromotionOperateProductDiscountRequest{
		Param: &CpaypromotionOperateProductDiscountParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CpaypromotionOperateProductDiscountRequest) Execute(accessToken *doudian_sdk.AccessToken) (*cpaypromotion_operateProductDiscount_response.CpaypromotionOperateProductDiscountResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &cpaypromotion_operateProductDiscount_response.CpaypromotionOperateProductDiscountResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CpaypromotionOperateProductDiscountRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CpaypromotionOperateProductDiscountRequest) GetParams() *CpaypromotionOperateProductDiscountParam{
	return c.Param
}


type SupportPeriodItem struct {
	// 实际费率
	ActualRatio *string `json:"actual_ratio"`
	// 期数（3，6，12），查询商品优惠接口中返回
	Period *string `json:"period"`
}
type CpaypromotionOperateProductDiscountParam struct {
	// 商家名称
	ShopName string `json:"shop_name"`
	// 商品ID
	ProductId *string `json:"product_id"`
	// 商品最高价格(多SKU)，单位：分
	ProductHighAmount *int64 `json:"product_high_amount"`
	// 使用的招商活动id
	PromotionAcitivityId string `json:"promotion_acitivity_id"`
	// 期数优惠信息，如果不填则表示该商品退出报名
	SupportPeriod []SupportPeriodItem `json:"support_period"`
	// 商品最低价格(多SKU)，单位：分
	ProductLowAmount *int64 `json:"product_low_amount"`
}
