package product_recommendProperties_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductRecommendPropertiesResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductRecommendPropertiesData `json:"data"`
}
type PropertyValuesItem struct {
	// 属性值ID
	ValueId int64 `json:"value_id"`
	// 属性值名称
	ValueName string `json:"value_name"`
}
type PropertiesItem struct {
	// 属性项名称
	PropertyName string `json:"property_name"`
	// 推荐的属性值列表
	PropertyValues []PropertyValuesItem `json:"property_values"`
	// 属性推荐来源：SimilarProductRecommend表示同款属性推荐DescPredictRecommend表示来自信息实时预测
	RecommendSource string `json:"recommend_source"`
	// 属性项ID
	PropertyId int64 `json:"property_id"`
}
type CheckResultItem struct {
	// 属性id
	PropertyId int64 `json:"property_id"`
	// 判准正确
	IsRight bool `json:"is_right"`
}
type ProductRecommendPropertiesData struct {
	// 推荐的属性列表
	Properties []PropertiesItem `json:"properties"`
	// 属性判准结果；如果存在判准结果，且is_right为false，可以在properties找到对应建议结果
	CheckResult map[int64]CheckResultItem `json:"check_result"`
}
