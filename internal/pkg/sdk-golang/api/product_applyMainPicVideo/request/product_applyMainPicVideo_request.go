package product_applyMainPicVideo_request

import (
	"doudian.com/open/sdk_golang/api/product_applyMainPicVideo/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductApplyMainPicVideoRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductApplyMainPicVideoParam 
}
func (c *ProductApplyMainPicVideoRequest) GetUrlPath() string{
	return "/product/applyMainPicVideo"
}


func New() *ProductApplyMainPicVideoRequest{
	request := &ProductApplyMainPicVideoRequest{
		Param: &ProductApplyMainPicVideoParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductApplyMainPicVideoRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_applyMainPicVideo_response.ProductApplyMainPicVideoResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_applyMainPicVideo_response.ProductApplyMainPicVideoResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductApplyMainPicVideoRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductApplyMainPicVideoRequest) GetParams() *ProductApplyMainPicVideoParam{
	return c.Param
}


type ProductApplyMainPicVideoParam struct {
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 主图视频vid，可以先通过https://op.jinritemai.com/docs/api-docs/69/1617接口上传视频，获取审核通过的视频素材ID进行传入
	MaterialVideoId string `json:"material_video_id"`
}
