package product_isv_scanClue_request

import (
	"doudian.com/open/sdk_golang/api/product_isv_scanClue/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductIsvScanClueRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductIsvScanClueParam 
}
func (c *ProductIsvScanClueRequest) GetUrlPath() string{
	return "/product/isv/scanClue"
}


func New() *ProductIsvScanClueRequest{
	request := &ProductIsvScanClueRequest{
		Param: &ProductIsvScanClueParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductIsvScanClueRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_isv_scanClue_response.ProductIsvScanClueResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_isv_scanClue_response.ProductIsvScanClueResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductIsvScanClueRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductIsvScanClueRequest) GetParams() *ProductIsvScanClueParam{
	return c.Param
}


type ProductIsvScanClueParam struct {
	// 线索Id（游标id）
	ClueId int64 `json:"clue_id"`
	// 渠道code，搜索热招：competitive，蓝海：query_none_less，行业趋势：industry_selected，低价：choice_low_price
	SourceChannelCode []string `json:"source_channel_code"`
	// 限制200
	PageSize int64 `json:"page_size"`
}
