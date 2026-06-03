package product_qualityDetail_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductQualityDetailResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductQualityDetailData `json:"data"`
}
type FieldProblemItem struct {
	// 字段key，category-类目、props-属性、product_name-标题、pic-主图、desc_pic-详情图片、desc_text-详情文字
	FieldKey string `json:"field_key"`
	// 字段名
	FieldName string `json:"field_name"`
	// 问题编码
	ProblemKey int64 `json:"problem_key"`
	// 问题名
	ProblemName string `json:"problem_name"`
	// 问题建议
	Suggestion string `json:"suggestion"`
	// 问题类型编码
	ProblemType int64 `json:"problem_type"`
}
type QualityScore struct {
	// 质量分版本
	Version string `json:"version"`
	// 质量分等级
	Level string `json:"level"`
	// 质量分数值
	Score int64 `json:"score"`
}
type ProductQualityDetailData struct {
	// 可优化问题项
	FieldProblem []FieldProblemItem `json:"field_problem"`
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 商品名字
	ProductName string `json:"product_name"`
	// 质量分
	QualityScore *QualityScore `json:"quality_score"`
}
