package shop_getExperienceScore_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ShopGetExperienceScoreResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ShopGetExperienceScoreData `json:"data"`
}
type QualityReturnRate struct {
	// 指标分数
	NodeScore float64 `json:"node_score"`
	// 加权指标分数
	NodeScoreWeight float64 `json:"node_score_weight"`
	// 指标值
	Score float64 `json:"score"`
}
type DeliveryAccomplishRate struct {
	// 加权分数
	NodeScoreWeight float64 `json:"node_score_weight"`
	// 指标分数
	NodeScore float64 `json:"node_score"`
	// 指标值
	Score float64 `json:"score"`
}
type AverageFeedbackTime struct {
	// 指标分数
	NodeScore float64 `json:"node_score"`
	// 加权分数
	NodeScoreWeight float64 `json:"node_score_weight"`
	// 指标值
	Score float64 `json:"score"`
}
type AftersaleDealReachRate struct {
	// 指标分数
	NodeScore float64 `json:"node_score"`
	// 指标值
	Score float64 `json:"score"`
	// 加权分数
	NodeScoreWeight float64 `json:"node_score_weight"`
}
type ShopGetExperienceScoreData struct {
	// 体验分总分
	SellerRatingInfo *SellerRatingInfo `json:"seller_rating_info"`
	// 商品综合评价
	ProdDsr *ProdDsr `json:"prod_dsr"`
	// 发货物流品退率
	LgtQreturnRate *LgtQreturnRate `json:"lgt_qreturn_rate"`
	// 商品品质退货率得分
	QualityReturnRate *QualityReturnRate `json:"quality_return_rate"`
	// 揽收时效达成率
	CollectOntimeRate *CollectOntimeRate `json:"collect_ontime_rate"`
	// 运单配送时效达成率
	DeliveryAccomplishRate *DeliveryAccomplishRate `json:"delivery_accomplish_rate"`
	// 飞鸽平均响应时长
	AverageFeedbackTime *AverageFeedbackTime `json:"average_feedback_time"`
	// 售后处理时长达成率
	AftersaleDealReachRate *AftersaleDealReachRate `json:"aftersale_deal_reach_rate"`
}
type LgtQreturnRate struct {
	// 指标分数
	NodeScore float64 `json:"node_score"`
	// 指标值
	Score float64 `json:"score"`
	// 加权指标分数
	NodeScoreWeight float64 `json:"node_score_weight"`
}
type CollectOntimeRate struct {
	// 指标分数
	NodeScore float64 `json:"node_score"`
	// 当前维度分数
	Score float64 `json:"score"`
	// 加权分数
	NodeScoreWeight float64 `json:"node_score_weight"`
}
type SellerRatingInfo struct {
	// 分数
	Score float64 `json:"score"`
	// 权重后分数
	ScoreWeight float64 `json:"score_weight"`
}
type ProdDsr struct {
	// 指标分数
	NodeScore float64 `json:"node_score"`
	// 指标值
	Score float64 `json:"score"`
	// 加权指标分数
	NodeScoreWeight float64 `json:"node_score_weight"`
}
