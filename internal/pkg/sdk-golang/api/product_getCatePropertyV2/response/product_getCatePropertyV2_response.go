package product_getCatePropertyV2_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetCatePropertyV2Response struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetCatePropertyV2Data `json:"data"`
}
type InputRule struct {
	// 度量衡间校验规则
	DependencyCheckRules []DependencyCheckRulesItem `json:"dependency_check_rules"`
	// 全局校验规则
	GlobalCheckRules []GlobalCheckRulesItem `json:"global_check_rules"`
}
type UnitsItem struct {
	// 单位名称
	UnitName string `json:"unit_name"`
	// 单位id
	UnitId int64 `json:"unit_id"`
}
type MeasureTemplatesItem struct {
	// 材质度量衡填写规则
	InputRule *InputRule `json:"input_rule"`
	// 模版名称
	DisplayName string `json:"display_name"`
	// 度量衡模版下的模块，可为多个，一般情况为一个，一般情况一个模块就是一个值+单位形式
	ValueModules []ValueModulesItem `json:"value_modules"`
	// 度量衡模版id
	TemplateId int64 `json:"template_id"`
}
type DependencyCheckRulesItem struct {
	// 验证规则
	RuleConstraint *RuleConstraint `json:"rule_constraint"`
	// 触发条件
	RuleCondition *RuleCondition `json:"rule_condition"`
	// 度量衡模版id
	TemplateId int64 `json:"template_id"`
}
type UnitRulesItem struct {
	// 最大值
	Max float64 `json:"max"`
	// 最小值
	Min float64 `json:"min"`
	// 单位id
	UnitId int64 `json:"unit_id"`
}
type PropertyPicRule struct {
	// 是否支持上传属性图
	Available bool `json:"available"`
	// 属性图是否必填
	Required bool `json:"required"`
}
type RuleCondition struct {
	// 预期值
	Value string `json:"value"`
	// 运算符
	Operator string `json:"operator"`
	// 度量衡模块id
	ModuleId int64 `json:"module_id"`
}
type ValidateRule struct {
	// 不同单位下的值约束
	UnitRules []UnitRulesItem `json:"unit_rules"`
	// date_type = date 时，有值，如YYYY-MM-DD
	TimeFormat string `json:"time_format"`
	// 最大值
	Max float64 `json:"max"`
	// 最小值
	Min float64 `json:"min"`
	// 小数位数，当data_type为float时有值
	Precision int64 `json:"precision"`
	// 填写值的数据格式：integer, float, date
	DataType string `json:"data_type"`
}
type ValueModulesItem struct {
	// 扩展信息
	Extra map[string]string `json:"extra"`
	// 模块是否必填，为false时表示这个模块可以不填
	ValueRequired bool `json:"value_required"`
	// 填写规则
	ValidateRule *ValidateRule `json:"validate_rule"`
	// 可选单位，可为空
	Units []UnitsItem `json:"units"`
	// 值的可选项，如材质属性可能会有：棉、麻等选项
	Values []ValuesItem `json:"values"`
	// 输入类型： 枚举、输入、枚举可输入： enum, input, enum_diy
	InputType string `json:"input_type"`
	// 后缀
	Suffix string `json:"suffix"`
	// 前缀
	Prefix string `json:"prefix"`
	// 模块id
	ModuleId int64 `json:"module_id"`
}
type DataItem struct {
	// 属性图规则
	PropertyPicRule *PropertyPicRule `json:"property_pic_rule"`
	// 属性feature类型列表
	FeatureTypeList []int64 `json:"feature_type_list"`
	// 度量衡模版，可为多个，一般情况为一个
	MeasureTemplates []MeasureTemplatesItem `json:"measure_templates"`
	// 是否有下级级联属性
	HasSubProperty bool `json:"has_sub_property"`
	// 【已废弃】老类目id，请使用category_id字段
	Cid int64 `json:"cid"`
	// 属性名称
	PropertyName string `json:"property_name"`
	// 属性可选值列表，为空时需要手动填写
	Options []OptionsItem `json:"options"`
	// 1：创建商品时该属性字段必填 0：创建商品时该属性字段选填
	Required int64 `json:"required"`
	// 属性状态，0：有效，1：失效
	Status int64 `json:"status"`
	// 输入text、单选select、多选multi_select、时间戳timestamp、时间段timerange
	Type string `json:"type"`
	// 新版类目id（优先使用）
	CategoryId int64 `json:"category_id"`
	// 最大多选数
	MultiSelectMax int64 `json:"multi_select_max"`
	// 属性类型，0 绑定属性 1关键属性 2售卖属性 3 商品属性
	PropertyType int64 `json:"property_type"`
	// 属性id
	PropertyId int64 `json:"property_id"`
	// 属性顺序
	Sequence int64 `json:"sequence"`
	// 关系id,SPU类目使用，表示自己的上一个关键属性ID
	RelationId int64 `json:"relation_id"`
	// 商品属性是否有支持商家自定义，1=支持自定义，0=不支持自定义。 使用场景：当开发者传入自定义的属性值时需在创建或更新商品接口的属性时，需把【product_format_new】字段中传入diy_type=1并且value=0；
	DiyType int64 `json:"diy_type"`
	// 0:非重要属性，1:重要属性
	ImportantType int64 `json:"important_type"`
}
type RuleConstraint struct {
	// 报错文案
	ErrMsg string `json:"err_msg"`
	// 预期值
	Value string `json:"value"`
	// 分组函数
	GroupFunc string `json:"group_func"`
	// 运算符
	Operator string `json:"operator"`
	// 度量衡模版id
	ModuleId int64 `json:"module_id"`
}
type GlobalCheckRulesItem struct {
	// 报错文案
	ErrMsg string `json:"err_msg"`
	// 运算符
	Operator string `json:"operator"`
	// 预期值
	TargetValue string `json:"target_value"`
	// 度量衡模块id
	ModuleId int64 `json:"module_id"`
	// 模板id
	TemplateId int64 `json:"template_id"`
}
type ValuesItem struct {
	// 值名称
	ValueName string `json:"value_name"`
}
type OptionsItem struct {
	// 可选值名称
	Name string `json:"name"`
	// 可选值
	Value string `json:"value"`
	// 值的id
	ValueId int64 `json:"value_id"`
	// 属性值顺序
	Sequence int64 `json:"sequence"`
}
type ProductGetCatePropertyV2Data struct {
	// 返回参数列表
	Data []DataItem `json:"data"`
	// 模板类型，SPU为1，结构化为2，默认为0（无特殊需求忽略该字段即可）
	TplType int64 `json:"tpl_type"`
}
