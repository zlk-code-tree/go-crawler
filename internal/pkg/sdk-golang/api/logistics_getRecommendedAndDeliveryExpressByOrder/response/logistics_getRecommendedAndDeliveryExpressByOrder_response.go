package logistics_getRecommendedAndDeliveryExpressByOrder_response

import (
	"doudian.com/open/sdk_golang/core"
)

type LogisticsGetRecommendedAndDeliveryExpressByOrderResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data []DataItem `json:"data"`
}
type DataItem struct {
	// 订单权益列表
	OrderPrivilegeInfoList []OrderPrivilegeInfoListItem `json:"order_privilege_info_list"`
	// 订单号
	OrderId string `json:"order_id"`
	// 推荐及可达物流商信息集合
	ExpressInfoList []ExpressInfoListItem `json:"express_info_list"`
}
type LogisticsGetRecommendedAndDeliveryExpressByOrderData struct {
	// 返回值
	Data []DataItem `json:"data"`
}
type OrderPrivilegeInfoListItem struct {
	// 订单打标原因
	Reason string `json:"reason"`
	// 订单标签描述
	Label string `json:"label"`
	// 订单标签code
	Code string `json:"code"`
}
type TailInfo struct {
	// 超长单占比超过百分之多少商家
	LevelPercent string `json:"level_percent"`
	// 超长单占比优化百分比
	OptimizedPercent string `json:"optimized_percent"`
	// 超长单占比
	TailRate string `json:"tail_rate"`
}
type CcrInfo struct {
	// 物流负向反馈率（狭义）超过百分之多少商家
	LevelPercent string `json:"level_percent"`
	// 物流负向反馈率（狭义）优化百分比
	OptimizedPercent string `json:"optimized_percent"`
	// 物流负向反馈率（狭义）
	CcrRate string `json:"ccr_rate"`
}
type CollectSignInfo struct {
	// 揽签时长,单位小时
	AvgCostHours string `json:"avg_cost_hours"`
	// 该线路商超过其他物流公司23.86%
	LevelPercent string `json:"level_percent"`
	// 优化百分比，时长下降6.1%
	OptimizedPercent string `json:"optimized_percent"`
}
type ExpressInfoListItem struct {
	// 超长单
	TailInfo *TailInfo `json:"tail_Info"`
	// 物流负向反馈率（狭义）
	CcrInfo *CcrInfo `json:"ccr_Info"`
	// 此线路上该物流商排名
	RankRoute int32 `json:"rank_route"`
	// 是否可用（含不可达与线路机构异常）
	UndeliverableReason string `json:"undeliverable_reason"`
	// 物流商编码
	Express string `json:"express"`
	// 是否可达
	IsDeliverable bool `json:"is_deliverable"`
	// 是否订购电子面单
	IsShopEBill bool `json:"is_shop_eBill"`
	// 是否推荐
	IsRecommended bool `json:"is_recommended"`
	// 时长维度的推荐原因
	CollectSignInfo *CollectSignInfo `json:"collect_sign_info"`
	// 推荐原因
	RecommendReason string `json:"recommend_reason"`
}
