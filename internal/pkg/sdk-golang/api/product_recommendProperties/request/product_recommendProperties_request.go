package product_recommendProperties_request

import (
	"doudian.com/open/sdk_golang/api/product_recommendProperties/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductRecommendPropertiesRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductRecommendPropertiesParam 
}
func (c *ProductRecommendPropertiesRequest) GetUrlPath() string{
	return "/product/recommendProperties"
}


func New() *ProductRecommendPropertiesRequest{
	request := &ProductRecommendPropertiesRequest{
		Param: &ProductRecommendPropertiesParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductRecommendPropertiesRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_recommendProperties_response.ProductRecommendPropertiesResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_recommendProperties_response.ProductRecommendPropertiesResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductRecommendPropertiesRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductRecommendPropertiesRequest) GetParams() *ProductRecommendPropertiesParam{
	return c.Param
}


type ProductFormatItem struct {
	// 属性值id
	Value *int64 `json:"value"`
	// 属性值名称
	Name *string `json:"name"`
	// 是否为商家自定义属性值，1为是，0为不是
	DiyType *int64 `json:"diy_type"`
	// 度量衡信息，对本接口暂时不需要传
	MeasureInfo *MeasureInfo `json:"measure_info"`
}
type ProductRecommendPropertiesParam struct {
	// 叶子类目ID
	CategoryId int64 `json:"category_id"`
	// 素材ID列表，用于属性预测。material_ids与img_urls选传其中一个即可。
	MaterialIds []string `json:"material_ids"`
	// 图片URL列表，用于属性预测。material_ids与img_urls选传其中一个即可。
	ImgUrls []string `json:"img_urls"`
	// 商品标题
	Name *string `json:"name"`
	// 商家自填属性；传入可以触发对自填属性值的判准
	ProductFormat *map[int64][]ProductFormatItem `json:"product_format"`
}
type ValuesItem struct {
	// 自定义度量衡扩展信息
	Extra *map[string]string `json:"extra"`
	// 模块id
	ModuleId *int64 `json:"module_id"`
	// 值内容
	Value *string `json:"value"`
	// 度量衡单位id
	UnitId *int64 `json:"unit_id"`
	// 度量衡单位名称
	UnitName *string `json:"unit_name"`
	// 前缀
	Prefix *string `json:"prefix"`
	// 后缀
	Suffix *string `json:"suffix"`
	// 是否必填
	ValueRequired *bool `json:"value_required"`
}
type MeasureInfo struct {
	// 度量衡模板id
	TemplateId *int64 `json:"template_id"`
	// 度量衡内容
	Values []ValuesItem `json:"values"`
	// 自定义度量衡扩展信息
	Extra *map[string]string `json:"extra"`
}
