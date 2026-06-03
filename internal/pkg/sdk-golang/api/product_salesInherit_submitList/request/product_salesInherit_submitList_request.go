package product_salesInherit_submitList_request

import (
	"doudian.com/open/sdk_golang/api/product_salesInherit_submitList/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductSalesInheritSubmitListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductSalesInheritSubmitListParam 
}
func (c *ProductSalesInheritSubmitListRequest) GetUrlPath() string{
	return "/product/salesInherit/submitList"
}


func New() *ProductSalesInheritSubmitListRequest{
	request := &ProductSalesInheritSubmitListRequest{
		Param: &ProductSalesInheritSubmitListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductSalesInheritSubmitListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_salesInherit_submitList_response.ProductSalesInheritSubmitListResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_salesInherit_submitList_response.ProductSalesInheritSubmitListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductSalesInheritSubmitListRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductSalesInheritSubmitListRequest) GetParams() *ProductSalesInheritSubmitListParam{
	return c.Param
}


type Page struct {
	// 当前页，从1起
	PageNo int32 `json:"page_no"`
	// 每页数量；最大50；
	PageSize int32 `json:"page_size"`
}
type ProductSalesInheritSubmitListParam struct {
	// 抖店店铺商品id列表,最大支持50条；
	ProductIds []int64 `json:"product_ids"`
	// 站外商品id列表（与站内商品id条件取）；最大支持50条；
	OutsideProductIds []int64 `json:"outside_product_ids"`
	// 分页参数
	Page *Page `json:"page"`
}
