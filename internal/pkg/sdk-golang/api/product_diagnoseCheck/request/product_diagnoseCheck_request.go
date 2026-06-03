package product_diagnoseCheck_request

import (
	"doudian.com/open/sdk_golang/api/product_diagnoseCheck/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductDiagnoseCheckRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductDiagnoseCheckParam 
}
func (c *ProductDiagnoseCheckRequest) GetUrlPath() string{
	return "/product/diagnoseCheck"
}


func New() *ProductDiagnoseCheckRequest{
	request := &ProductDiagnoseCheckRequest{
		Param: &ProductDiagnoseCheckParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductDiagnoseCheckRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_diagnoseCheck_response.ProductDiagnoseCheckResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_diagnoseCheck_response.ProductDiagnoseCheckResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductDiagnoseCheckRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductDiagnoseCheckRequest) GetParams() *ProductDiagnoseCheckParam{
	return c.Param
}


type ProductFormatNewItem struct {
	// 属性值id
	Value *int64 `json:"value"`
	// 属性值名称
	Name *string `json:"name"`
	// 是否为商家自定义属性值，1为是，0为不是，默认为0
	DiyType *int64 `json:"diy_type"`
}
type MeasureInfo struct {
	// 标准化的公式模板ID
	TemplateId *int64 `json:"template_id"`
	// 度量衡值组
	Values []ValuesItem `json:"values"`
}
type SkuListItem struct {
	// 价格
	Price *int64 `json:"price"`
	// sku名称用属性项和属性值拼接组成，格式：规格项1:规格值1;规格项2:规格值2;
	Name *string `json:"name"`
	// 库存
	Stock *int64 `json:"stock"`
	// 规格图片
	ImageUrl *string `json:"image_url"`
	// skuid
	SkuId *int64 `json:"sku_id"`
	// 规格id列表
	SpecDetailIds []string `json:"spec_detail_ids"`
	// sku分类
	SkuClassificationType *string `json:"sku_classification_type"`
	// 销售属性
	SellProperties []SellPropertiesItem `json:"sell_properties"`
}
type ProductMainPicItem struct {
	// 图片url
	Url *string `json:"url"`
	// 图片类型
	Type *int64 `json:"type"`
	// 图片id
	PicId *int64 `json:"pic_id"`
}
type ProductMaterialMapItem struct {
	// 素材类型
	MediaType *int64 `json:"media_type"`
	// 素材图片url
	Urls []string `json:"urls"`
	// 视频id
	ResourceIds []string `json:"resource_ids"`
	// 素材ID
	MaterialId *int64 `json:"material_id"`
}
type ValuesItem struct {
	// 前缀：如日用
	Prefix *string `json:"prefix"`
	// 模块ID
	ModuleId *int64 `json:"module_id"`
	// 值：如10
	Value *string `json:"value"`
	// 单位ID
	UnitId *int64 `json:"unit_id"`
	// 单位名称：如抽
	UnitName *string `json:"unit_name"`
	// 后缀：如*
	Suffix *string `json:"suffix"`
}
type SellPropertiesItem struct {
	// 销售属性项名称
	PropertyName *string `json:"property_name"`
	// 销售属性值名字
	ValueName *string `json:"value_name"`
	// 度量衡信息
	MeasureInfo *MeasureInfo `json:"measure_info"`
}
type ComponentTemplate struct {
	// 模板类型: size_info(尺码表)
	TemplateType *string `json:"template_type"`
	// 指定模板类型: clothing、undies、shoes、children_clothing
	TemplateSubType *string `json:"template_sub_type"`
	// 模板名称
	TemplateName *string `json:"template_name"`
	// 商品组件编辑器源数据 json
	ComponentFrontData *string `json:"component_front_data"`
	// 商品组件数据 json
	ComponentData *string `json:"component_data"`
	// 是否可共享
	Shareable *bool `json:"shareable"`
}
type ProductDiagnoseCheckParam struct {
	// 属性实体
	ProductFormatNew *map[int64][]ProductFormatNewItem `json:"product_format_new"`
	// 短标题
	ShortProductName *string `json:"short_product_name"`
	// 规格名
	SpecName *string `json:"spec_name"`
	// 规格词
	SpecValue *string `json:"spec_value"`
	// 商品sku
	SkuList []SkuListItem `json:"sku_list"`
	// 质量分
	QualityScoreVersion *string `json:"quality_score_version"`
	// 主图信息，主图1：1和主图3：4
	ProductMainPic []ProductMainPicItem `json:"product_main_pic"`
	// 卡券使用条件
	PoiUseCondition *string `json:"poi_use_condition"`
	// 尺码模板id
	SizeInfoTemplateId *int64 `json:"size_info_template_id"`
	// 属性
	Props *map[string]string `json:"props"`
	// 0:普通商品，1:教育类商品 2:跨境商品
	BizKind *int64 `json:"biz_kind"`
	// 详情文字
	DescText *string `json:"desc_text"`
	// 高危词判断
	RiskProp *string `json:"risk_prop"`
	// 商品类型
	ProductType int64 `json:"product_type"`
	// 叶子类目ID
	Category int64 `json:"category"`
	// 主图
	Pic []string `json:"pic"`
	// 商品素材信息
	ProductMaterialMap *map[int64]ProductMaterialMapItem `json:"product_material_map"`
	// 商品id
	ProductId int64 `json:"product_id"`
	// 商品名称
	ProductName *string `json:"product_name"`
	// 业务类型
	ProductBiz *int64 `json:"product_biz"`
	// 尺码模板开关
	IsTemplateOpened *bool `json:"is_template_opened"`
	// 商品卡券领取
	PoiNotification *string `json:"poi_notification"`
	// 场景 1发布 2诊断
	CheckType *int64 `json:"check_type"`
	// 品牌ID
	BrandId *string `json:"brand_id"`
	// 详情图
	DescPic []string `json:"desc_pic"`
	// 是否跨境 0 否 1 是
	CrossBorder *int64 `json:"cross_border"`
	// 品牌关联ID
	BrandRelatedId *string `json:"brand_related_id"`
	// 尺码结构及详细数据
	ComponentTemplate *ComponentTemplate `json:"component_template"`
	// 商品推荐语
	RecommendRemark *string `json:"recommend_remark"`
	// 是否仅需要基础分相关诊断项
	OnlyBaseScore *int64 `json:"only_base_score"`
}
