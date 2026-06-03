package product_diagnoseCheck_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductDiagnoseCheckResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductDiagnoseCheckData `json:"data"`
}
type ItemRcmdEffect struct {
	// 同行使用率
	PeerUsageRate string `json:"peer_usage_rate"`
	// 提升比例上限
	ImproveValueUpperBound string `json:"improve_value_upper_bound"`
	// 提升比例下限
	ImproveValueLowerBound string `json:"improve_value_lower_bound"`
	// 指标描述
	EffctDesc string `json:"effct_desc"`
	// 指标效果
	OptimizationInfo *OptimizationInfo `json:"optimization_info"`
}
type RecommendTagInfoListItem struct {
	// hot normal
	TagType string `json:"tag_type"`
	// 推荐词内容
	Name string `json:"name"`
}
type AiSuggestPropItem struct {
	// 无
	PropertyId int64 `json:"property_id"`
	// 无
	PropertyName string `json:"property_name"`
	// 无
	ValueId int64 `json:"value_id"`
	// 无
	ValueName string `json:"value_name"`
}
type ProductDiagnoseCheckData struct {
	// 质量分
	QualityScore *QualityScore `json:"quality_score"`
	// 必要图文
	DiagnoseEssentialMaterial *DiagnoseEssentialMaterial `json:"diagnose_essential_material"`
	// 质量分是否计算中
	QualityScoreCalculating bool `json:"quality_score_calculating"`
	// 字段问题
	FieldProblem []FieldProblemItem `json:"field_problem"`
	// 基础分
	BaseScore int64 `json:"base_score"`
	// 推荐值
	DiagnoseRecommend *DiagnoseRecommend `json:"diagnose_recommend"`
}
type QualityScore struct {
	// 质量分
	Score int64 `json:"score"`
	// 等级
	Level string `json:"level"`
	// 质量分版本
	Version string `json:"version"`
}
type EssentialMaterialInfoListItem struct {
	// 无
	MaterialSource string `json:"material_source"`
	// 无
	MaterialComponentId int64 `json:"material_component_id"`
	// 无
	Code int64 `json:"code"`
	// 无
	Url string `json:"url"`
}
type DiagnoseEssentialMaterial struct {
	// 必要图文
	EssentialMaterialInfoList []EssentialMaterialInfoListItem `json:"essential_material_info_list"`
}
type OptimizationInfo struct {
	// 无
	DefaultEventType []string `json:"default_event_type"`
	// 无
	OptimizationKey string `json:"optimization_key"`
	// 无
	OptimizationName string `json:"optimization_name"`
	// 无
	OptimizationStatus string `json:"optimization_status"`
	// 无
	UnavailableReason string `json:"unavailable_reason"`
	// 无
	DefaultIndexValue []string `json:"default_index_value"`
}
type FieldProblemItem struct {
	// 问题类型
	ProblemType int64 `json:"problem_type"`
	// 字段key
	FieldKey string `json:"field_key"`
	// problem对应的tab类型
	TabOptimizationType string `json:"tab_optimization_type"`
	// 诊断问题示例详情
	DetailLink string `json:"detail_link"`
	// 诊断问题定位
	ProblemPosition []string `json:"problem_position"`
	// 是否合规问题
	IsFatal bool `json:"is_fatal"`
	// 该诊断项是否是信息空缺导致的
	IsEmptyType bool `json:"is_empty_type"`
	// 是否支持app展示
	IsAppSupport bool `json:"is_app_support"`
	// 问题得分
	ProblemScore float64 `json:"problem_score"`
	// 问题名
	ProblemName string `json:"problem_name"`
	// 是否支持豁免
	IsSupportExemption bool `json:"is_support_exemption"`
	// 问题分类
	ProblemCategory string `json:"problem_category"`
	// 商品前端展示问题
	ProblemSimpleName string `json:"problem_simple_name"`
	// 诊断问题简介
	ProblemDesc string `json:"problem_desc"`
	// 优化类型
	OptimizationType int64 `json:"optimization_type"`
	// 问题扩展信息
	ProblemBizExtra *ProblemBizExtra `json:"problem_biz_extra"`
	// 是否是计分项
	IsQualityScore bool `json:"is_quality_score"`
	// 问题类型2024: 缺失模特上身效果图,请补充2025: 缺失商品平铺展示,请补充2026: 缺失商品局部/细节图展示,请补充488: 主图存在不良暗示问题1005: 商品无达人测评1002: 商品有山寨侵权风险1003: 商品品牌命中平台禁售管控604: 主图属于低质手机拍摄图606: 主图文字不规范602: 主图售卖主体缺失605: 主图存在压缩等问题603: 主图存在诱导点击嫌疑601: 主图含明显边框431: 缺少3:4主图455: 商品缺少规格图20001: 规格图重复425: 商品缺少主图视频或主图视频质量不合格703: 商品缺少讲解回放2011: 商品详情页缺少规格/尺寸2007: 商品详情页缺少清晰配料表2008: 商品详情页缺少清晰营养成分表803: 请丰富详情内容，不可仅为文字说明2005: 服饰商品详情页无清晰尺码表111: 标题可补充热搜词5002: 商品品牌属性[%s]与商品标题描述[%s]、详情描述[%s]不一致，请修改201: 类目选择需优化，建议选择类目：「%s」2002: 重要属性未全部填写
	ProblemKey int64 `json:"problem_key"`
	// 有问题的属性id
	PropIds []int64 `json:"prop_ids"`
	// 多值时的位置
	Index int64 `json:"index"`
	// 是否拦截发布
	IsBlock bool `json:"is_block"`
	// 是否基础分
	IsBaseScore bool `json:"is_base_score"`
	// 问题项所属二级优化类型
	SecondOptimizationType string `json:"second_optimization_type"`
	// 是否已忽略
	IsIgnored bool `json:"is_ignored"`
	// 诊断项对应的问题域 如主图、白底图
	DiagnoseTarget string `json:"diagnose_target"`
	// 诊断的字段类型：主图、标题、属性、短标题、主图视频、讲解回放、达人秀
	FieldName string `json:"field_name"`
	// 建议
	Suggestion string `json:"suggestion"`
	// 收益指标
	ItemRcmdEffect *ItemRcmdEffect `json:"item_rcmd_effect"`
	// 优化类型一级分类
	OptimizationObject string `json:"optimization_object"`
	// 优化类型二级分类
	OptimizationIssue string `json:"optimization_issue"`
	// 问题影响描述
	ProblemAffectDescribe []string `json:"problem_affect_describe"`
	// 问题分类
	JumpLink string `json:"jump_link"`
	// 问题项所属一级优化类型
	FirstOptimizationType string `json:"first_optimization_type"`
	// 是否支持忽略预警
	IsSupportIgnore bool `json:"is_support_ignore"`
}
type PropMapItem struct {
	// value不为0则推荐选择项
	Value int64 `json:"value"`
	// value为0则取name
	Name string `json:"name"`
}
type DiagnoseRecommend struct {
	// 字段key
	Name string `json:"name"`
	// key=属性id, value=推荐属性值
	PropMap map[int64][]PropMapItem `json:"prop_map"`
	// 无
	SuggestSeoTitleWords []string `json:"suggest_seo_title_words"`
	// 无
	RecommendTagInfoList []RecommendTagInfoListItem `json:"recommend_tag_info_list"`
	// 无
	AiSuggestProp []AiSuggestPropItem `json:"ai_suggest_prop"`
}
type ProblemBizExtra struct {
	// 存放填错属性项对应的文案，key为属性ID
	ConflictPropsDocsMap map[int64]string `json:"conflict_props_docs_map"`
	// 重复铺货商品
	SimilarProductList []int64 `json:"similar_product_list"`
	// 标题错别字
	TitleTypo string `json:"title_typo"`
}
