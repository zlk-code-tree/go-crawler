package product_salesInherit_submit_request

import (
	"doudian.com/open/sdk_golang/api/product_salesInherit_submit/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductSalesInheritSubmitRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductSalesInheritSubmitParam 
}
func (c *ProductSalesInheritSubmitRequest) GetUrlPath() string{
	return "/product/salesInherit/submit"
}


func New() *ProductSalesInheritSubmitRequest{
	request := &ProductSalesInheritSubmitRequest{
		Param: &ProductSalesInheritSubmitParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductSalesInheritSubmitRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_salesInherit_submit_response.ProductSalesInheritSubmitResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_salesInherit_submit_response.ProductSalesInheritSubmitResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductSalesInheritSubmitRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductSalesInheritSubmitRequest) GetParams() *ProductSalesInheritSubmitParam{
	return c.Param
}


type ProductSalesInheritSubmitParam struct {
	// 提报记录列表（不超过20条）
	SubmitRecords []SubmitRecordsItem `json:"submit_records"`
}
type SubmitRecordsItem struct {
	// 站外平台类型；1-淘宝；
	PlatformId int64 `json:"platform_id"`
	// 抖店店铺商品id
	ProductId int64 `json:"product_id"`
	// 同主体站外店铺商品id，
	OutsideProductId int64 `json:"outside_product_id"`
	// 站外商品30天内销量；最小销量上传销量不能低于100个
	SubmitSalesNum int64 `json:"submit_sales_num"`
}
