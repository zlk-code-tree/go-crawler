package product_getCategoryPropertyValue_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetCategoryPropertyValueResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetCategoryPropertyValueData `json:"data"`
}
type ProductGetCategoryPropertyValueData struct {
	// 属性
	Property *Property `json:"property"`
}
type ValuePic struct {
	// 图片，url链接或rgb内容
	PicValue string `json:"pic_value"`
	// 属性值图片类型url或rgb
	PicType string `json:"pic_type"`
}
type PropertyValuesItem struct {
	// 规格值图片
	ValuePic *ValuePic `json:"value_pic"`
	// 属性值名称
	PropertyValueName string `json:"property_value_name"`
	// 属性值id
	PropertyValueId int64 `json:"property_value_id"`
}
type Property struct {
	// 属性值
	PropertyValues []PropertyValuesItem `json:"property_values"`
	// 属性id
	PropertyId int64 `json:"property_id"`
}
