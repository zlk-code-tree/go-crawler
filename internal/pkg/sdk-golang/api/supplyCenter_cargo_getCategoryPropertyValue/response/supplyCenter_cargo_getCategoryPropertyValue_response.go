package supplyCenter_cargo_getCategoryPropertyValue_response

import (
	"doudian.com/open/sdk_golang/core"
)

type SupplyCenterCargoGetCategoryPropertyValueResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *SupplyCenterCargoGetCategoryPropertyValueData `json:"data"`
}
type PropertyValuesItem struct {
	// 属性值id
	PropertyValueId int64 `json:"property_value_id"`
	// 属性值名称
	PropertyValueName string `json:"property_value_name"`
	// 规格值图片
	ValuePic *ValuePic `json:"value_pic"`
}
type Property struct {
	// 属性id
	PropertyId int64 `json:"property_id"`
	// 属性值
	PropertyValues []PropertyValuesItem `json:"property_values"`
}
type SupplyCenterCargoGetCategoryPropertyValueData struct {
	// 属性
	Property *Property `json:"property"`
}
type ValuePic struct {
	// 图片，url链接或rgb内容
	PicValue string `json:"pic_value"`
	// 属性值图片类型 url或rgb
	PicType string `json:"pic_type"`
}
