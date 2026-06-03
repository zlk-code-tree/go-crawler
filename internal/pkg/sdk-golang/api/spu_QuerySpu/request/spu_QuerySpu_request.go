package spu_QuerySpu_request

import (
	"doudian.com/open/sdk_golang/api/spu_QuerySpu/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SpuQuerySpuRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SpuQuerySpuParam 
}
func (c *SpuQuerySpuRequest) GetUrlPath() string{
	return "/spu/QuerySpu"
}


func New() *SpuQuerySpuRequest{
	request := &SpuQuerySpuRequest{
		Param: &SpuQuerySpuParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SpuQuerySpuRequest) Execute(accessToken *doudian_sdk.AccessToken) (*spu_QuerySpu_response.SpuQuerySpuResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &spu_QuerySpu_response.SpuQuerySpuResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SpuQuerySpuRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SpuQuerySpuRequest) GetParams() *SpuQuerySpuParam{
	return c.Param
}


type Properties struct {
	// 【选填】产品名称
	ProductName *string `json:"product_name"`
	// 【选填】CCC证书编号
	Ccc *string `json:"ccc"`
	// 【选填】ISBN
	Isbn *string `json:"isbn"`
	// 【选填】型号
	Model *string `json:"model"`
	// 【选填】书名
	BookName *string `json:"book_name"`
	// 【选填】货号
	GoodsNumber *string `json:"goods_number"`
	// 【选填】品牌
	BrandId *int64 `json:"brand_id"`
	// 【选填】系列
	Series *string `json:"series"`
	// 【选填】备案/批准文号
	RegistrationApproval *string `json:"registration_approval"`
}
type SpuQuerySpuParam struct {
	// 【必填】从第0页开始
	Page int64 `json:"page"`
	// 【必填】最大值为20
	Size int64 `json:"size"`
	// 【选填】叶子类目ID
	CategoryId *int64 `json:"category_id"`
	// 【选填】属性
	Properties *Properties `json:"properties"`
	// 【选填】SPU状态：1-上线，2-下线，3-审核中，4-审核不通过
	Status []int32 `json:"status"`
	// 0：门捷列夫，1：标准类目2.0
	Channel *int32 `json:"channel"`
}
