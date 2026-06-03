package order_addOrderRemark_request

import (
	"doudian.com/open/sdk_golang/api/order_addOrderRemark/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderAddOrderRemarkRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderAddOrderRemarkParam 
}
func (c *OrderAddOrderRemarkRequest) GetUrlPath() string{
	return "/order/addOrderRemark"
}


func New() *OrderAddOrderRemarkRequest{
	request := &OrderAddOrderRemarkRequest{
		Param: &OrderAddOrderRemarkParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderAddOrderRemarkRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_addOrderRemark_response.OrderAddOrderRemarkResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_addOrderRemark_response.OrderAddOrderRemarkResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderAddOrderRemarkRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderAddOrderRemarkRequest) GetParams() *OrderAddOrderRemarkParam{
	return c.Param
}


type OrderAddOrderRemarkParam struct {
	// 店铺订单ID。
	OrderId string `json:"order_id"`
	// 备注内容，必填选项。如果不填会导致订单的备注被空值覆盖，最大不得超过500个字符。
	Remark string `json:"remark"`
	// 是否加旗标，不填则默认为false，不会修改旗帜颜色。 is_add_star=true，需要加旗帜或者覆盖原来的旗帜颜色；is_add_star=false，不添加旗帜，不会覆盖原来的旗帜颜色。
	IsAddStar *string `json:"is_add_star"`
	// 标星等级，范围0～5 0为灰色旗标，5为红色旗标，数字越大颜色越深 0灰 1紫 2青 3绿 4橙 5红
	Star *string `json:"star"`
}
