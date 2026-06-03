package spu_getSpuRule_response

import (
	"doudian.com/open/sdk_golang/core"
)

type SpuGetSpuRuleResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *SpuGetSpuRuleData `json:"data"`
}
type SpuMaterialRulesItem struct {
	// SPU素材的具体规则
	MaterialRule *MaterialRule `json:"material_rule"`
	// 素材类型枚举值：1-主图
	MaterialType int32 `json:"material_type"`
}
type ProofRule struct {
	// 证明材料名称
	Name string `json:"name"`
	// 是否必填，0:非必填，1:必填
	IsRequired int64 `json:"is_required"`
	// 最小数量
	MinNum int64 `json:"min_num"`
	// 最大数量
	MaxNum int64 `json:"max_num"`
	// 最大容量，单位为M
	MaxSize int64 `json:"max_size"`
	// 素材类型为图片时的尺寸比
	AspectRatio []string `json:"aspect_ratio"`
	// 素材类型为图片时的像素比
	Pixel []int64 `json:"pixel"`
	// 数据格式类型
	DataFormat []string `json:"data_format"`
}
type BarcodeRule struct {
	// 是否必填，0:非必填，1:必填
	IsRequired int64 `json:"is_required"`
	// 最小数量
	MinNum int64 `json:"min_num"`
	// 最大数量
	MaxNum int64 `json:"max_num"`
}
type SpuPropertyRuleItem struct {
	// 属性名
	PropertyName string `json:"property_name"`
	// 属性ID
	PropertyId int64 `json:"property_id"`
	// 是否必填，0:非必填，1:必填
	IsRequired int64 `json:"is_required"`
	// 输入类型 select:单选，multi_select:多选，text:输入框，timestamp:时间戳，timeframe:时间区间
	ValueType string `json:"value_type"`
	// 属性类型，0 绑定属性 1关键属性 2销售属性
	PropertyType int64 `json:"property_type"`
	// 是否支持枚举可输入
	DiyType int64 `json:"diy_type"`
	// 最大可选数量，多选时需要用
	MaxSelectNum int64 `json:"max_select_num"`
	// 属性值
	Values []ValuesItem `json:"values"`
}
type SpuActualImagesRule struct {
	// 是否必填
	IsRequired int64 `json:"is_required"`
	// 最大数量
	MaxNum int64 `json:"max_num"`
	// 最小数量
	MinNum int64 `json:"min_num"`
}
type MaterialRule struct {
	// 素材名称
	Name string `json:"name"`
	// 是否必填，0:非必填，1:必填
	IsRequired int64 `json:"is_required"`
	// 最小数量
	MinNum int64 `json:"min_num"`
	// 最大数量
	MaxNum int64 `json:"max_num"`
	// 最大容量，单位为M
	MaxSize int64 `json:"max_size"`
	// 素材类型为图片时的尺寸比
	AspectRatio []string `json:"aspect_ratio"`
	// 素材类型为图片时的像素比
	Pixel []int64 `json:"pixel"`
	// 数据格式类型，例如jpg、png等格式
	DataFormat []string `json:"data_format"`
}
type SpuProofRulesItem struct {
	// 实物证明类型，枚举值：2-实物图、3-说明书图片、4-说明书PDF、5-医疗器械注册证、6-版权页。其中说明书PDF和说明书图片要求二者必填其一
	ProofType int32 `json:"proof_type"`
	// SPU证明材料的具体规则
	ProofRule *ProofRule `json:"proof_rule"`
}
type BarcodeImageRule struct {
	// 数据格式类型
	DataFormat []string `json:"data_format"`
	// 是否必填，0:非必填，1:必填
	IsRequired int64 `json:"is_required"`
	// 最小数量
	MinNum int64 `json:"min_num"`
	// 最大数量
	MaxNum int64 `json:"max_num"`
}
type SpuBarcodeRule struct {
	// 条码值
	BarcodeRule *BarcodeRule `json:"barcode_rule"`
	// 条码图
	BarcodeImageRule *BarcodeImageRule `json:"barcode_image_rule"`
}
type SpuGetSpuRuleData struct {
	// SPU属性规则
	SpuPropertyRule []SpuPropertyRuleItem `json:"spu_property_rule"`
	// SPU图片规则(已废弃，请采用spu_material_rules字段)
	SpuImagesRule *SpuImagesRule `json:"spu_images_rule"`
	// SPU实物图规则（已废弃，请采用spu_proof_rules字段）
	SpuActualImagesRule *SpuActualImagesRule `json:"spu_actual_images_rule"`
	// 0-不管控，商家在该类目下发布商品时，不强制要求命中SPU。 - 1-弱管控，商家在该类目下发布商品时，强制要求命中SPU（即关键属性必须一致），但绑定属性不强制要求一致。 - 2-强管控，商家在该类目下发布商品时，强制要求命中SPU，且绑定属性必须一致。
	ControlType int64 `json:"control_type"`
	// - true 表示商家在该类目基于SPU发布商品，发商品的管控策略见管控类型。 - false 表示商家在该类目不需要基于SPU发布商品。不用发布SPU，直接发布商品即可。
	SupportSpuProduct bool `json:"support_spu_product"`
	// true-可以发布SPU，false-不能发布SPU
	SupportCreateSpu bool `json:"support_create_spu"`
	// SPU素材规则（素材类型：主图）
	SpuMaterialRules []SpuMaterialRulesItem `json:"spu_material_rules"`
	// 证明材料规则（证明材料类型：实物图、版权页）
	SpuProofRules []SpuProofRulesItem `json:"spu_proof_rules"`
	// 条码规则
	SpuBarcodeRule *SpuBarcodeRule `json:"spu_barcode_rule"`
}
type ValuesItem struct {
	// 属性值ID，对于输入框类型，填0即可
	ValueId int64 `json:"value_id"`
	// 属性值，属性值的中文
	ValueName string `json:"value_name"`
}
type SpuImagesRule struct {
	// 是否必填
	IsRequired int64 `json:"is_required"`
	// 最大数量
	MaxNum int64 `json:"max_num"`
	// 最小数量
	MinNum int64 `json:"min_num"`
}
