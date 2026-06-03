package product_getProductUpdateRule_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetProductUpdateRuleResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetProductUpdateRuleData `json:"data"`
}
type ReferencePriceRule struct {
	// true表示可填写参考价，false表示不可填写参考价
	IsSupport bool `json:"is_support"`
	// true表示必填参考价，false表示不必填参考价
	IsRequired bool `json:"is_required"`
	// 参考价凭证类型枚举值与对应的显示名称，比如{     "1" : "厂商建议零售价",     "2" : "吊牌价",     "3" : "定价",     "4" : "官网零售价" }
	CertificateTypes map[int64]string `json:"certificate_types"`
	// 参考价最高高于最低SKU价格的倍数，比如最低SKU价格为1，此值为10，则参考价不可高于10
	LowerSkuPriceTimes int64 `json:"lower_sku_price_times"`
}
type MainImageThreeToFourRule struct {
	// 是否展示主图3:4信息
	IsShow bool `json:"is_show"`
	// 主图3:4信息是否必填
	MustInput bool `json:"must_input"`
}
type SelectedFulfillmentRule struct {
	// 支持c2b定制时可用的发货规则
	NormalRule *NormalRule_5_5 `json:"normal_rule"`
}
type TradeRule struct {
	// 自动充值相关的规则
	SupportAutoChargeRule *SupportAutoChargeRule `json:"support_auto_charge_rule"`
}
type SkuPictureRule struct {
	// sku图必填类型：1-必填且所有SKU必须填写；2-非必填，要么全部填写，要么全部为空，不允许部分填写
	RequireType int64 `json:"require_type"`
	// 是否支持设置SKU图
	Support bool `json:"support"`
}
type UnitsItem struct {
	// 度量衡单位id
	UnitId int64 `json:"unit_id"`
	// 度量衡单位名称
	UnitName string `json:"unit_name"`
}
type ValidateRule struct {
	// 数据类型，integer、float
	DataType string `json:"data_type"`
	// 小数位数，data_type为float时有值
	Precision int64 `json:"precision"`
	// 最小值
	Min float64 `json:"min"`
	// 最大值
	Max float64 `json:"max"`
	// 单位规则明细
	UnitRules []UnitRulesItem `json:"unit_rules"`
}
type MatchableRuleItem struct {
	// 资质规则生效的触发条件结构：属性id -> 属性值clause作用：当命中ruleClause中给出的全部属性对时，刷新相关资质的「必填/非必填」配置
	RuleClause map[int64]RuleClauseItem `json:"rule_clause"`
	// 资质规则触发时，受影响的类目属性结构：属性id -> 属性effect（包含属性是否必填，属性名是否需要被覆写）作用：当命中ruleClause中给出的全部属性对时，刷新相关资质的「必填/非必填」配置
	PropertyEffect map[int64]PropertyEffectItem `json:"property_effect"`
}
type QualificationRuleItem struct {
	// 是否千川收取资质
	IsQianchuanNeed bool `json:"is_qianchuan_need"`
	// 是否必填
	IsRequired bool `json:"is_required"`
	// 资质要求描述文字
	TextList []string `json:"text_list"`
	// 资质名称
	Name string `json:"name"`
	// 资质ID
	Key string `json:"key"`
	// 资质规则和规则的效果
	MatchableRule []MatchableRuleItem `json:"matchable_rule"`
}
type TitleIllegalKeywordRuleItem struct {
	// 规则提示
	Tip string `json:"tip"`
	// 不允许出现的后缀关键词
	KeywordsInSuffix []string `json:"keywords_in_suffix"`
	// 不允许出现的前缀关键词
	KeywordsInPrefix []string `json:"keywords_in_prefix"`
	// 不允许出现的全局关键词
	Keywords []string `json:"keywords"`
}
type PropertyValuesItem struct {
	// 规格值id
	SellPropertyValueId int64 `json:"sell_property_value_id"`
	// 规格值名称
	SellPropertyValueName string `json:"sell_property_value_name"`
	// 规格值图片
	ValuePic *ValuePic `json:"value_pic"`
}
type RecommendNameRule struct {
	// 商品标题非法关键词规则
	TitleIllegalKeywordRule []TitleIllegalKeywordRuleItem `json:"title_illegal_keyword_rule"`
	// 品牌没有权限是否可以勾选
	PrefixBrandUnauthCanSel bool `json:"prefix_brand_unauth_can_sel"`
	// 前缀规则是否包含品牌
	PrefixPropHasBrand bool `json:"prefix_prop_has_brand"`
	// 标题长度规则
	TitleLimitRule *TitleLimitRule `json:"title_limit_rule"`
	// 当前类目id是否命中前缀推荐规则
	SatisfyPrefix bool `json:"satisfy_prefix"`
	// 命中规则的属性id详情
	PropertyIds []int64 `json:"property_ids"`
}
type OptionsItem_4 struct {
	// 售后天数选项列表
	Options []OptionsItem `json:"options"`
	// 三包服务类型名称
	Name string `json:"name"`
	// 三包服务类型值
	Value string `json:"value"`
}
type PriceRulesItem struct {
	// 是否可选，true表示可选，false表示不可选，如果不可选则表示商家没有设置过基础金价，需要前往抖店后台设置
	CanSelect bool `json:"can_select"`
	// 金价描述
	Desc string `json:"desc"`
	// 每克重黄金对应的价格，单位：分
	Value int64 `json:"value"`
	// 价格规则枚举：1是投资金2是首饰金
	Type int64 `json:"type"`
}
type HomeInstallServiceRule struct {
	// 上门安装可选模式
	Options []OptionsItem_4 `json:"options"`
	// 是否必选
	MustSelect bool `json:"must_select"`
}
type SkuRule_2_2 struct {
	// sku图填写规则
	SkuPictureRule *SkuPictureRule `json:"sku_picture_rule"`
	// 是否支持国补商品型号
	SupportSpecModelCode bool `json:"support_spec_model_code"`
	// 是否支持填写条形码
	SupportBarcode bool `json:"support_barcode"`
	// sku分类信息填写规则
	SkuClassificationRule *SkuClassificationRule `json:"sku_classification_rule"`
}
type PropertyEffectItem struct {
	// 类目属性名称是否需要被覆写成label_name（目前只有资质证号有这个逻辑）
	LabelName string `json:"label_name"`
	// 类目属性是否必填（目前只有生产企业名称和资质证号）: 0-必填, 1-非必填
	IsRequired int64 `json:"is_required"`
}
type PickUpMethodRule struct {
	// 虚拟商品提货方式规则
	VirtualGoodsRule *VirtualGoodsRule `json:"virtual_goods_rule"`
}
type StepRule struct {
	// 支持的时效列表
	MultiTimes []MultiTimesItem `json:"multi_times"`
	// 是否支持
	Support bool `json:"support"`
	// 阶梯现货部分延迟返货时间范围
	DelayOptions []int64 `json:"delay_options"`
	// 是特殊时间延迟发货，可忽略
	IsSpecialDelayOption bool `json:"is_special_delay_option"`
	// 阶梯发货阶梯部分发货时间最小值
	StepMinDelay int64 `json:"step_min_delay"`
	// 阶梯发货阶梯部分发货时间最大值
	StepMaxDelay int64 `json:"step_max_delay"`
	// 延迟发货规则
	DelayRule *DelayRule_4_4 `json:"delay_rule"`
}
type LargeProductRule struct {
	// 是否可选大件商品
	Enable bool `json:"enable"`
	// 是否必须大件商品
	MustSelect bool `json:"must_select"`
	// 可选项
	Options []OptionsItem_4_4 `json:"options"`
}
type RequiredSpecDetailsItem struct {
	// 规格项名称
	SellPropertyName string `json:"sell_property_name"`
	// 规格项id
	SellPropertyId int64 `json:"sell_property_id"`
	// 规格值选项
	PropertyValues []PropertyValuesItem `json:"property_values"`
	// 是否支持备注
	SupportRemark bool `json:"support_remark"`
	// 是否可以自定义规格值
	SupportDiy bool `json:"support_diy"`
	// 是否必填
	IsRequired bool `json:"is_required"`
	// 规格样式，cascader是为导航样式
	ValueDisplayStyle string `json:"value_display_style"`
	// 度量衡模版，可能有多个
	MeasureTemplates []MeasureTemplatesItem `json:"measure_templates"`
	// 是否需要二次查询规格值
	NeedPagingQueryValue bool `json:"need_paging_query_value"`
	// 导航属性
	NavigationProperties []NavigationPropertiesItem `json:"navigation_properties"`
	// 1
	PropertyDefaultDisplay bool `json:"property_default_display"`
}
type GoldPriceRule struct {
	// 价格规则信息，内有黄金的不同的单位价格以及枚举
	PriceRules []PriceRulesItem `json:"price_rules"`
	// 销售属性（规格）id，价格基于这个规格填写的数值按照公式进行计算
	SellPropertyId string `json:"sell_property_id"`
	// 是否必填，true表示必填，false表示非必填
	IsRequired bool `json:"is_required"`
	// 是否需要展示（可填），true表示需要展示（可填），false表示无需展示（不可填）
	IsShow bool `json:"is_show"`
}
type NormalRule_5_5 struct {
	// 描述信息
	Description string `json:"description"`
	// 可用的延迟发货天数
	DelayOptions []int64 `json:"delay_options"`
	// 不可用信息
	DenyDescription string `json:"deny_description"`
	// 是否可见
	Display string `json:"display"`
	// 是否支持
	Support string `json:"support"`
}
type SupportAutoChargeRule struct {
	// 是否支持自动充值
	Value bool `json:"value"`
	// 1
	Disabled bool `json:"disabled"`
}
type MultiTimesItem struct {
	// 时效描述
	TimeDesc string `json:"time_desc"`
	// 时效
	TimeType int64 `json:"time_type"`
}
type TimeSkuPresellWithNormalRule struct {
	// 是否支持
	Support bool `json:"support"`
	// 发货时效规格名称
	TimeSkuSpecName string `json:"time_sku_spec_name"`
	// 发货时效规格选项
	TimeSkuSpecDetial []TimeSkuSpecDetialItem `json:"time_sku_spec_detial"`
	// 预售门槛价，单位分
	MinPresellPrice int64 `json:"min_presell_price"`
}
type TitleLimitRule struct {
	// 标题最长长度
	MaxLength string `json:"max_length"`
	// 标题最短长度
	MinLength string `json:"min_length"`
}
type SupplyDayReturnSelectorItem struct {
	// 7天无理由选项名称
	Name string `json:"name"`
	// 7天无理由选项值
	Value string `json:"value"`
}
type TimeSkuPurePresellRule struct {
	// 是否支持
	Support bool `json:"support"`
	// 发货时效规格名称
	TimeSkuSpecName string `json:"time_sku_spec_name"`
	// 发货时效规格选项
	TimeSkuSpecDetial []TimeSkuSpecDetialItem `json:"time_sku_spec_detial"`
	// 预售门槛价，单位分
	MinPresellPrice int64 `json:"min_presell_price"`
}
type MeasureTemplatesItem struct {
	// 模版名称
	DisplayName string `json:"display_name"`
	// 度量衡模版id
	TemplateId int64 `json:"template_id"`
	// 度量衡模版模块内容，一个模版多个模块
	ValueModules []ValueModulesItem `json:"value_modules"`
}
type ExtraRule struct {
	// 上门安装服务规则
	HomeInstallServiceRule *HomeInstallServiceRule `json:"home_install_service_rule"`
	// c2b定制相关规则
	C2bCustomizeRule *C2bCustomizeRule `json:"c2b_customize_rule"`
}
type ComponentTemplateRule struct {
	// 尺码列配置
	SizeDimensions *SizeDimensions `json:"size_dimensions"`
	// 区间符号
	IntervalSizeSymbol []string `json:"interval_size_symbol"`
	// 维度下尺码范围
	SizeDimensionRange map[string]SizeDimensionRangeItem `json:"size_dimension_range"`
	// 当前类目支持的尺码模版子类型
	SupportTemplateSubType string `json:"support_template_sub_type"`
	// 是否展示尺码信息
	IsShow bool `json:"is_show"`
	// 尺码信息是否必填
	MustInput bool `json:"must_input"`
}
type OptionsItem struct {
	// 选项值
	Value string `json:"value"`
	// 选项名称
	Name string `json:"name"`
}
type NavigationPropertiesItem struct {
	// 导航属性id
	PropertyId int64 `json:"property_id"`
	// 导航属性名称
	PropertyName string `json:"property_name"`
}
type AppListItem struct {
	// 小程序名称
	Name string `json:"name"`
	// 小程序id
	Value string `json:"value"`
}
type SkuClassificationRule struct {
	// sku分类信息扩展项
	Options []OptionsItem_4_4 `json:"options"`
	// sku分类信息是否必填
	MustSelect bool `json:"must_select"`
	// 是否允许填写sku分类信息
	Enable bool `json:"enable"`
}
type SkuRule struct {
	// 最大绝对发货时间，秒级标准Unix时间戳
	AbsoluteMaxDate string `json:"absolute_max_date"`
	// 最小绝对发货时间，秒级标准Unix时间戳
	AbsoluteMinDate string `json:"absolute_min_date"`
	// 最大相对发货时间
	RelativeMaxDelay string `json:"relative_max_delay"`
	// sku发货时效是否支持范围输入
	SupportRangeOption bool `json:"support_range_option"`
	// 最小相对发货时间
	RelativeMinDelay string `json:"relative_min_delay"`
	// 是否支持sku发货时效
	Support bool `json:"support"`
}
type DeliveryAfterPayConfig struct {
	// 延迟发货时间最小值
	MinDeliveryDays int64 `json:"min_delivery_days"`
	// 延迟发货时间最大值
	MaxDeliveryDays int64 `json:"max_delivery_days"`
	// 最长预售结束时间
	MaxPresellEndDays int64 `json:"max_presell_end_days"`
	// 是否需要人审，可忽略
	NeedAudit bool `json:"need_audit"`
}
type DeliveryAfterPresellConfig struct {
	// 延迟发货时间最小值
	MinDeliveryDays int64 `json:"min_delivery_days"`
	// 延迟发货时间最大值
	MaxDeliveryDays int64 `json:"max_delivery_days"`
	// 最长预售结束时间
	MaxPresellEndDays int64 `json:"max_presell_end_days"`
	// 是否需要人审，可忽略
	NeedAudit bool `json:"need_audit"`
}
type ThreeGuarantees struct {
	// 三包服务类型，1-寄修，2-延保
	Options []OptionsItem_4 `json:"options"`
	// 是否必填
	MustSelect bool `json:"must_select"`
	// 是否支持三包服务
	Enable bool `json:"enable"`
}
type NormalRule struct {
	// sku发货时效规则
	SkuRule *SkuRule `json:"sku_rule"`
	// 特殊时间延迟发货规则
	DelayRule *DelayRule_4_4 `json:"delay_rule"`
	// 是否支持
	Support bool `json:"support"`
	// 发货时效选项，如当日发、次日发、48小时
	DelayOptions []int64 `json:"delay_options"`
	// 是否是特殊的时间发货，可忽略
	IsSpecialDelayOption bool `json:"is_special_delay_option"`
}
type ProductPresellRule struct {
	// 是否支持全款预售
	Support bool `json:"support"`
	// 预售门槛价
	MinPresellPrice int64 `json:"min_presell_price"`
	// 是否支持支付成功后发货
	SupportDeliveryAfterPay bool `json:"support_delivery_after_pay"`
	// 是否支持预售结束后发货
	SupportDeliveryAfterPresell bool `json:"support_delivery_after_presell"`
	// 支付结束后规则
	DeliveryAfterPayConfig *DeliveryAfterPayConfig `json:"delivery_after_pay_config"`
	// 预售结束后规则
	DeliveryAfterPresellConfig *DeliveryAfterPresellConfig `json:"delivery_after_presell_config"`
}
type AllergyReturnV2Rule struct {
	// 是否可选过敏包退
	Enable string `json:"enable"`
	// 是否必选过敏包退
	MustSelect string `json:"must_select"`
	// 可选项
	Options []OptionsItem_4_4 `json:"options"`
}
type SpuControlRule struct {
	// 是否支持spu发品
	SupportSpuProduct bool `json:"support_spu_product"`
	// 0不管控 1弱管控 2强管控
	ControlType int64 `json:"control_type"`
	// 是否支持新建spu
	SupportCreateSpu bool `json:"support_create_spu"`
	// 是否支持spu纠错
	SupportRectifySpu bool `json:"support_rectify_spu"`
	// 是否支持spu举报
	SupportReportSpu bool `json:"support_report_spu"`
	// 是否spu免审
	NoNeedAuditSpu bool `json:"no_need_audit_spu"`
}
type ValueModulesItem struct {
	// 模块id
	ModuleId int64 `json:"module_id"`
	// 前缀
	Prefix string `json:"prefix"`
	// 后缀
	Suffix string `json:"suffix"`
	// 输入类型：枚举 enum, 输入 input， enum_diy 枚举可输入
	InputType string `json:"input_type"`
	// 度量衡值选项
	Values []ValuesItem `json:"values"`
	// 度量衡单位
	Units []UnitsItem `json:"units"`
	// 度量衡规则
	ValidateRule *ValidateRule `json:"validate_rule"`
}
type ProductSpecRule struct {
	// 为true时，规格图要么全不填，要么全填
	AllSpecPicRequired bool `json:"all_spec_pic_required"`
	// 是否支持规格项顺序调整，true表示支持
	SupportPropertySequenceVariable bool `json:"support_property_sequence_variable"`
	// 【废弃字段,请勿依赖】totally_use_given_spec：完全使用系统下发规格；partly_use_given_spec部分使用系统下发规格；not_use_given_spec：不管控规格;"":为空时表示不管控规格
	SpecRuleCode string `json:"spec_rule_code"`
	// 商品规格列表
	RequiredSpecDetails []RequiredSpecDetailsItem `json:"required_spec_details"`
	// 最大可支持的规格层级数量
	MaxSpecNumLimit int64 `json:"max_spec_num_limit"`
	// sku组合数量上限
	SpecCombinationLimit int64 `json:"spec_combination_limit"`
	// 单个规格的规格值数量上限
	SpecSingleLimit int64 `json:"spec_single_limit"`
	// 是否支持规格项自定义
	SupportPropertyDiy bool `json:"support_property_diy"`
}
type SizeDimensions struct {
	// 有用列
	UsefulDimensions []string `json:"useful_dimensions"`
	// 其他可用列
	OtherDimensions []string `json:"other_dimensions"`
}
type DelayRule struct {
	// 支持的配置类型1：代表时间点，此时对应的config_value的值为支持的最晚发货的秒级时间戳2：代表相对时间，此时对应的config_value的值表示支付后几天发货；比如config_value=3表示支付后三天发货
	ConfigType int64 `json:"config_type"`
	// 与规则配置类型对应的配置值
	ConfigValue int64 `json:"config_value"`
	// 规则开始时间
	StartTime int64 `json:"start_time"`
	// 规则结束时间
	EndTime int64 `json:"end_time"`
}
type SkuPresellRule struct {
	// 是否支持
	Support bool `json:"support"`
	// 预售门槛价
	MinPresellPrice int64 `json:"min_presell_price"`
	// 是否支持支付结束后发货
	SupportDeliveryAfterPay bool `json:"support_delivery_after_pay"`
	// 是否支持预售结束后发货
	SupportDeliveryAfterPresell bool `json:"support_delivery_after_presell"`
	// 支付结束后规则
	DeliveryAfterPayConfig *DeliveryAfterPayConfig `json:"delivery_after_pay_config"`
	// 预售结束后规则
	DeliveryAfterPresellConfig *DeliveryAfterPresellConfig `json:"delivery_after_presell_config"`
}
type ValuePic struct {
	// 规格值图片内容
	PicValue string `json:"pic_value"`
	// 规格值图片类型rgb和url
	PicType string `json:"pic_type"`
}
type SizeDimensionRangeItem struct {
	// 最小范围
	Max int64 `json:"max"`
	// 最大范围
	Min int64 `json:"min"`
}
type C2bCustomizeRule struct {
	// 支持c2b定制时可用的履约规则
	SelectedFulfillmentRule *SelectedFulfillmentRule `json:"selected_fulfillment_rule"`
	// 可选小程序
	AppList []AppListItem `json:"app_list"`
	// 支持c2b定制时可用的7天无理由
	SupplyDayReturnSelector []SupplyDayReturnSelectorItem `json:"supply_day_return_selector"`
	// 是否支持c2b定制
	Enable bool `json:"enable"`
}
type DelayRule_4_4 struct {
	// 配置类型
	ConfigType int32 `json:"config_type"`
	// 配置值
	ConfigValue int64 `json:"config_value"`
	// 商家配置的开始时间
	StartTime int64 `json:"start_time"`
	// 商家配置的结束时间
	EndTime int64 `json:"end_time"`
}
type TimeSkuSpecDetialItem struct {
	// 规格值
	SpecValue string `json:"spec_value"`
	// 是否预售时效
	IsPresellSpec bool `json:"is_presell_spec"`
	// product_time_spec_same_day 当日发/次日发 product_time_spec_next_day 次日发 product_time_spec_48h  48小时内发货 product_time_spec_5d  5天内发货 product_time_spec_15d  15天内发货 product_time_spec_45d  45天内发货
	SpecCode string `json:"spec_code"`
}
type FulfillmentRule struct {
	// 特殊时间延迟发货规则
	DelayRule *DelayRule `json:"delay_rule"`
	// 现货发货模式规则
	NormalRule *NormalRule `json:"normal_rule"`
	// 阶梯发货模式规则
	StepRule *StepRule `json:"step_rule"`
	// 全款预售发货模式规则
	ProductPresellRule *ProductPresellRule `json:"product_presell_rule"`
	// SKU预售发货模式规则
	SkuPresellRule *SkuPresellRule `json:"sku_presell_rule"`
	// 现货预售混合发货规则（现货预售混合）
	TimeSkuPresellWithNormalRule *TimeSkuPresellWithNormalRule `json:"time_sku_presell_with_normal_rule"`
	// 现货预售混合发货模式规则（现货预售混合）
	TimeSkuPurePresellRule *TimeSkuPurePresellRule `json:"time_sku_pure_presell_rule"`
}
type SupplyDayReturnRule struct {
	// true 支持七天无理由，false 不支持七天无理由
	Enable bool `json:"enable"`
	// 可选的无理由退货选项列表
	Options []OptionsItem_4_4 `json:"options"`
}
type ValuesItem struct {
	// 度量衡值
	ValueName string `json:"value_name"`
}
type UnitRulesItem struct {
	// 度量衡单位id
	UnitId int64 `json:"unit_id"`
	// 最小值
	Min float64 `json:"min"`
	// 最大值
	Max float64 `json:"max"`
}
type RuleClauseItem struct {
	// 1 - 相等，2 - 不相等， 3 - 包含， 4 - 不包含
	Operand int64 `json:"operand"`
	// 操作（值枚举："等于"、"不等于"）
	OperandStr string `json:"operand_str"`
	// 属性值ID列表
	PropertyValues []int64 `json:"property_values"`
}
type OptionsItem_4_4 struct {
	// sku分类对应举例
	Example string `json:"example"`
	// sku分类对应介绍
	Description string `json:"description"`
	// sku分类枚举信息
	Value string `json:"value"`
	// sku分类信息中文名称
	Name string `json:"name"`
}
type AfterSaleRule struct {
	// 三包服务承诺配置规则
	ThreeGuarantees *ThreeGuarantees `json:"three_guarantees"`
	// 过敏包退规则
	AllergyReturnV2Rule *AllergyReturnV2Rule `json:"allergy_return_v2_rule"`
	// 大件商品规则
	LargeProductRule *LargeProductRule `json:"large_product_rule"`
	// 无理由退货规则
	SupplyDayReturnRule *SupplyDayReturnRule `json:"supply_day_return_rule"`
}
type VirtualGoodsRule struct {
	// 是否可能设置订单页展示核销入口；当该字段为 true且提取方式为「使用电子凭证」或「充值直连」时，可以设置
	SupportShowCheckoutEntry bool `json:"support_show_checkout_entry"`
}
type ProductGetProductUpdateRuleData struct {
	// 履约规则
	FulfillmentRule *FulfillmentRule `json:"fulfillment_rule"`
	// 商品标题推荐规则
	RecommendNameRule *RecommendNameRule `json:"recommend_name_rule"`
	// 售后服务规则
	AfterSaleRule *AfterSaleRule `json:"after_sale_rule"`
	// 参考价相关规则
	ReferencePriceRule *ReferencePriceRule `json:"reference_price_rule"`
	// spu管控规则
	SpuControlRule *SpuControlRule `json:"spu_control_rule"`
	// 商品规格约束
	ProductSpecRule *ProductSpecRule `json:"product_spec_rule"`
	// 商品尺码模板配置规则
	ComponentTemplateRule *ComponentTemplateRule `json:"component_template_rule"`
	// 商品主图3:4配置规则
	MainImageThreeToFourRule *MainImageThreeToFourRule `json:"main_image_three_to_four_rule"`
	// 金价信息，计价金类目下sku价格可以按照公式进行计算（价格=克重*每克价格+加工费），本字段提供相关信息
	GoldPriceRule *GoldPriceRule `json:"gold_price_rule"`
	// 其他规则
	ExtraRule *ExtraRule `json:"extra_rule"`
	// 交易相关的规则
	TradeRule *TradeRule `json:"trade_rule"`
	// sku规则
	SkuRule *SkuRule_2_2 `json:"sku_rule"`
	// 资质规则，类目属性影响资质必填和资质属性必填
	QualificationRule []QualificationRuleItem `json:"qualification_rule"`
	// 提取方式规则
	PickUpMethodRule *PickUpMethodRule `json:"pick_up_method_rule"`
}
