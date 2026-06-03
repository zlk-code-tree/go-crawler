package spu_QuerySpu_response

import (
	"doudian.com/open/sdk_golang/core"
)

type SpuQuerySpuResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *SpuQuerySpuData `json:"data"`
}
type PropertyValuesItem struct {
	// 属性值ID
	ValueId int64 `json:"valueId"`
	// 属性值名称
	ValueName string `json:"valueName"`
}
type SpuPropertyInfosItem struct {
	// 属性项ID
	PropertyId int64 `json:"propertyId"`
	// 属性项名称
	PropertyName string `json:"propertyName"`
	// 属性值信息
	PropertyValues []PropertyValuesItem `json:"propertyValues"`
	// 属性类型：0-绑定属性 1-关键属性 2-销售属性
	PropertyType int64 `json:"propertyType"`
}
type BarcodesItem struct {
	// 条码值
	Barcode string `json:"barcode"`
}
type ContentItem struct {
	// 内容
	Content string `json:"content"`
}
type MaterialsItem struct {
	// 类型: 1-主图
	Type int64 `json:"type"`
	// 内容
	Content []ContentItem `json:"content"`
}
type SpuListItem struct {
	// SPUID
	SpuId int64 `json:"spuId"`
	// SPU名称
	SpuName string `json:"spuName"`
	// 类目ID
	CategoryId int64 `json:"categoryId"`
	// 品牌ID
	BrandId int64 `json:"brandId"`
	// SPU状态
	Status int64 `json:"status"`
	// 属性信息
	SpuPropertyInfos []SpuPropertyInfosItem `json:"spuPropertyInfos"`
	// 条码
	Barcodes []BarcodesItem `json:"barcodes"`
	// 素材
	Materials []MaterialsItem `json:"materials"`
}
type SpuQuerySpuData struct {
	// 查询到的总数据量
	Total int64 `json:"total"`
	// SPU信息
	SpuList []SpuListItem `json:"spuList"`
}
