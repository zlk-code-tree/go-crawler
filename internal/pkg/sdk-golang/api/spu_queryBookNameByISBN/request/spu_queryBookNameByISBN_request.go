package spu_queryBookNameByISBN_request

import (
	"doudian.com/open/sdk_golang/api/spu_queryBookNameByISBN/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SpuQueryBookNameByISBNRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SpuQueryBookNameByISBNParam 
}
func (c *SpuQueryBookNameByISBNRequest) GetUrlPath() string{
	return "/spu/queryBookNameByISBN"
}


func New() *SpuQueryBookNameByISBNRequest{
	request := &SpuQueryBookNameByISBNRequest{
		Param: &SpuQueryBookNameByISBNParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SpuQueryBookNameByISBNRequest) Execute(accessToken *doudian_sdk.AccessToken) (*spu_queryBookNameByISBN_response.SpuQueryBookNameByISBNResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &spu_queryBookNameByISBN_response.SpuQueryBookNameByISBNResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SpuQueryBookNameByISBNRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SpuQueryBookNameByISBNRequest) GetParams() *SpuQueryBookNameByISBNParam{
	return c.Param
}


type SpuQueryBookNameByISBNParam struct {
	// 类目ID
	CategoryLeafId *int64 `json:"category_leaf_id"`
	// ISBN编号
	Isbn string `json:"isbn"`
	// 当前页数。默认从0开始
	PageNo int64 `json:"page_no"`
	// 页大小。范围1-100
	PageSize int64 `json:"page_size"`
	// 获取数据的类目版本。0:门捷列夫类目；1:标准类目
	Channel *int32 `json:"channel"`
}
