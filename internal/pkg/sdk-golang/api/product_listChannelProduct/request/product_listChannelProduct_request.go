package product_listChannelProduct_request

import (
	"doudian.com/open/sdk_golang/api/product_listChannelProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductListChannelProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductListChannelProductParam 
}
func (c *ProductListChannelProductRequest) GetUrlPath() string{
	return "/product/listChannelProduct"
}


func New() *ProductListChannelProductRequest{
	request := &ProductListChannelProductRequest{
		Param: &ProductListChannelProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductListChannelProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_listChannelProduct_response.ProductListChannelProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_listChannelProduct_response.ProductListChannelProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductListChannelProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductListChannelProductRequest) GetParams() *ProductListChannelProductParam{
	return c.Param
}


type ProductListChannelProductParam struct {
	// 主商品ID。不需要此查询条件可传0
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId []int64 `json:"channel_id"`
	// 渠道类型：3商家自卖，4达人带货
	ChannelType []int64 `json:"channel_type"`
	// 绑定的达人ID
	KolId []int64 `json:"kol_id"`
	// 渠道品状态：0下架，2上架，3封禁
	Status *int64 `json:"status"`
	// 有效期开始时间
	StartTime *string `json:"start_time"`
	// 有效期结束时间
	EndTime *string `json:"end_time"`
	// 第几页，从1开始
	Page int64 `json:"page"`
	// 每页数量，默认20，最多50
	Size int64 `json:"size"`
}
