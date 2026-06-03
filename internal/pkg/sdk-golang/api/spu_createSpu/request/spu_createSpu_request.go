package spu_createSpu_request

import (
	"doudian.com/open/sdk_golang/api/spu_createSpu/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SpuCreateSpuRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SpuCreateSpuParam 
}
func (c *SpuCreateSpuRequest) GetUrlPath() string{
	return "/spu/createSpu"
}


func New() *SpuCreateSpuRequest{
	request := &SpuCreateSpuRequest{
		Param: &SpuCreateSpuParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SpuCreateSpuRequest) Execute(accessToken *doudian_sdk.AccessToken) (*spu_createSpu_response.SpuCreateSpuResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &spu_createSpu_response.SpuCreateSpuResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SpuCreateSpuRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SpuCreateSpuRequest) GetParams() *SpuCreateSpuParam{
	return c.Param
}


type ContentItem struct {
	// 素材内容（/spu/batchUploadImg返回的URL）
	Content string `json:"content"`
	// 文件的名称
	Name string `json:"name"`
}
type MaterialsItem struct {
	// 素材的实际内容
	Content []ContentItem `json:"content"`
	// 素材类型名称
	Name string `json:"name"`
	// 描述信息
	Description string `json:"description"`
	// 素材类型，枚举值：1-主图
	Type int32 `json:"type"`
}
type ProofsItem struct {
	// 描述信息
	Description string `json:"description"`
	// 证明信息类型，枚举值：2-实物图、6-版权页
	Name string `json:"name"`
	// 证明材料的实际内容
	Content []ContentItem `json:"content"`
	// 证明信息（类型：实物图、版权页）
	Type int32 `json:"type"`
}
type BarcodeImageItem struct {
	// 条码图链接
	Content *string `json:"content"`
	// 条码图名称
	Name *string `json:"name"`
}
type BarcodesItem struct {
	// 条码值
	Barcode *string `json:"barcode"`
	// 条码类型，1-69码
	BarcodeType *int32 `json:"barcode_type"`
	// 条码图
	BarcodeImage []BarcodeImageItem `json:"barcode_image"`
}
type SpuCreateSpuParam struct {
	// 叶子类目ID
	CategoryLeafId int64 `json:"category_leaf_id"`
	// SPU图片（已废弃，请采用materials字段）
	SpuImages []string `json:"spu_images"`
	// 属性信息（/spu/getSpuRule接口可获取）
	PropertyInfos []PropertyInfosItem `json:"property_infos"`
	// SPU实物图（已废弃，请采用proofs字段）
	SpuActualImages []string `json:"spu_actual_images"`
	// SPU素材
	Materials []MaterialsItem `json:"materials"`
	// 证明信息
	Proofs []ProofsItem `json:"proofs"`
	// 条码
	Barcodes []BarcodesItem `json:"barcodes"`
	// 0：门捷列夫，1：标准类目2.0
	Channel *int32 `json:"channel"`
}
type ValuesItem struct {
	// 属性值
	ValueName string `json:"value_name"`
	// 属性值ID
	ValueId int64 `json:"value_id"`
	// 0：门捷列夫，1：标准类目2.0
	Channel *int32 `json:"channel"`
}
type PropertyInfosItem struct {
	// 属性值
	Values []ValuesItem `json:"values"`
	// 属性ID
	PropertyId int64 `json:"property_id"`
}
