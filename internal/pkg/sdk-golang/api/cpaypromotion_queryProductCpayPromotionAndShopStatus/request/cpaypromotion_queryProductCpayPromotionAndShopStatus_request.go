package cpaypromotion_queryProductCpayPromotionAndShopStatus_request

import (
	"doudian.com/open/sdk_golang/api/cpaypromotion_queryProductCpayPromotionAndShopStatus/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CpaypromotionQueryProductCpayPromotionAndShopStatusRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CpaypromotionQueryProductCpayPromotionAndShopStatusParam 
}
func (c *CpaypromotionQueryProductCpayPromotionAndShopStatusRequest) GetUrlPath() string{
	return "/cpaypromotion/queryProductCpayPromotionAndShopStatus"
}


func New() *CpaypromotionQueryProductCpayPromotionAndShopStatusRequest{
	request := &CpaypromotionQueryProductCpayPromotionAndShopStatusRequest{
		Param: &CpaypromotionQueryProductCpayPromotionAndShopStatusParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CpaypromotionQueryProductCpayPromotionAndShopStatusRequest) Execute(accessToken *doudian_sdk.AccessToken) (*cpaypromotion_queryProductCpayPromotionAndShopStatus_response.CpaypromotionQueryProductCpayPromotionAndShopStatusResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &cpaypromotion_queryProductCpayPromotionAndShopStatus_response.CpaypromotionQueryProductCpayPromotionAndShopStatusResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CpaypromotionQueryProductCpayPromotionAndShopStatusRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CpaypromotionQueryProductCpayPromotionAndShopStatusRequest) GetParams() *CpaypromotionQueryProductCpayPromotionAndShopStatusParam{
	return c.Param
}


type CpaypromotionQueryProductCpayPromotionAndShopStatusParam struct {
	// 商品ID
	ProductId *string `json:"product_id"`
	// 商品类目
	ProductCategray *int64 `json:"product_categray"`
	// 商品价格(单位，分)
	ProductPrice *int64 `json:"product_price"`
}
