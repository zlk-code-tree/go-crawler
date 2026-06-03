package product_getCatePropertyV2_request

import (
	"doudian.com/open/sdk_golang/api/product_getCatePropertyV2/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductGetCatePropertyV2Request struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductGetCatePropertyV2Param 
}
func (c *ProductGetCatePropertyV2Request) GetUrlPath() string{
	return "/product/getCatePropertyV2"
}


func New() *ProductGetCatePropertyV2Request{
	request := &ProductGetCatePropertyV2Request{
		Param: &ProductGetCatePropertyV2Param{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductGetCatePropertyV2Request) Execute(accessToken *doudian_sdk.AccessToken) (*product_getCatePropertyV2_response.ProductGetCatePropertyV2Response, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_getCatePropertyV2_response.ProductGetCatePropertyV2Response{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductGetCatePropertyV2Request) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductGetCatePropertyV2Request) GetParams() *ProductGetCatePropertyV2Param{
	return c.Param
}


type ValuesItem struct {
	// 自定义度量衡扩展信息
	Extra *map[string]string `json:"extra"`
	// 度量衡单位名称
	UnitName *string `json:"unit_name"`
	// 值内容
	Value *string `json:"value"`
	// 前缀
	Prefix *string `json:"prefix"`
	// 后缀
	Suffix *string `json:"suffix"`
	// 前缀id
	PrefixId *int64 `json:"prefix_id"`
	// 度量衡单位id
	UnitId *int64 `json:"unit_id"`
	// 后缀id
	SuffixId *int64 `json:"suffix_id"`
	// 模块id
	ModuleId *int64 `json:"module_id"`
	// 是否必填
	ValueRequired *bool `json:"value_required"`
	// 值id
	ValueId *int64 `json:"value_id"`
}
type MeasureInfo struct {
	// 度量衡模板id
	TemplateId *int64 `json:"template_id"`
	// 自定义度量衡扩展信息
	Extra *map[string]string `json:"extra"`
	// 度量衡内容
	Values []ValuesItem `json:"values"`
}
type ProductFormatNewItem struct {
	// 1
	Value *int64 `json:"value"`
	// 度量衡信息
	MeasureInfo *MeasureInfo `json:"measure_info"`
	// 1
	Name *string `json:"name"`
	// 是否为商家自定义属性值，1为是，0为不是
	DiyType *int64 `json:"diy_type"`
}
type ProductGetCatePropertyV2Param struct {
	// 叶子类目id 1、传category_leaf_id ，则不需要传first_cid、second_cid、third_cid这三个字段 2、如果没传category_leaf_id，走之前的逻辑，需要传first_cid、second_cid、third_cid这三个字段
	CategoryLeafId int64 `json:"category_leaf_id"`
	// 1
	ProductFormatNew *map[int64][]ProductFormatNewItem `json:"product_format_new"`
	// 1
	NeedProcessPropCascade *bool `json:"need_process_prop_cascade"`
}
