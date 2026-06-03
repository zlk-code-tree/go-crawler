package product_getCascadeValue_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetCascadeValueResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetCascadeValueData `json:"data"`
}
type PropertyValuesItem struct {
	// 属性值id
	PropertyValueId int64 `json:"property_value_id"`
	// 属性值名称
	PropertyValueName string `json:"property_value_name"`
	// 级联id
	CascadeId int64 `json:"cascade_id"`
}
type PropertiesItem struct {
	// 属性id
	PropertyId int64 `json:"property_id"`
	// 属性内容
	PropertyValues []PropertyValuesItem `json:"property_values"`
	// 是否有下级属性
	HasSubProperty bool `json:"has_sub_property"`
}
type ProductGetCascadeValueData struct {
	// 级联属性
	Properties []PropertiesItem `json:"properties"`
}
