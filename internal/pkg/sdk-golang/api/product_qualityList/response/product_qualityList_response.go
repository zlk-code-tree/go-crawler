package product_qualityList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductQualityListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductQualityListData `json:"data"`
}
type ProblemTypeDistributionItem struct {
	// 待优化问题类型
	TypeName string `json:"type_name"`
	// 问题数量
	Num int64 `json:"num"`
}
type QualityListItem struct {
	// 可优化问题项
	FieldProblem []FieldProblemItem `json:"field_problem"`
	// 质量分数据
	QualityScore *QualityScore `json:"quality_score"`
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 商品名字
	ProductName string `json:"product_name"`
	// 待优化问题数量
	ProblemNumToImprove int64 `json:"problem_num_to_improve"`
	// 待优化问题分布列表，废弃不再用
	ProblemTypeDistribution []ProblemTypeDistributionItem `json:"problem_type_distribution"`
	// 商品是否达标，1达标，2不达标
	MeetStandard int64 `json:"meet_standard"`
	// 商品基础分，废弃不再用
	BaseScore int64 `json:"base_score"`
}
type ProductQualityListData struct {
	// 商品质量列表
	QualityList []QualityListItem `json:"quality_list"`
	// 店铺待优化商品总量
	Total int64 `json:"total"`
}
type FieldProblemItem struct {
	// 问题建议
	Suggestion string `json:"suggestion"`
	// 问题名
	ProblemName string `json:"problem_name"`
	// 问题编码
	ProblemKey int64 `json:"problem_key"`
	// 字段名
	FieldName string `json:"field_name"`
	// 字段key，category-类目、props-属性、product_name-标题、pic-主图、desc_pic-详情图片、desc_text-详情文字
	FieldKey string `json:"field_key"`
}
type QualityScore struct {
	// 质量分版本
	Version string `json:"version"`
	// 质量分等级
	Level string `json:"level"`
	// 质量分
	Score int64 `json:"score"`
}
