package buyin_promotionStrategyList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinPromotionStrategyListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinPromotionStrategyListData `json:"data"`
}
type ValidKolInfo struct {
	// 生效达人范围 1：全部达人 2：指定达人 3：条件圈选 4：专属指定达人
	ValidKolRange int64 `json:"valid_kol_range"`
	// 指定达人列表
	KolInfos []KolInfosItem `json:"kol_infos"`
	// 指定达人数量
	KolNum int64 `json:"kol_num"`
}
type CosRatio struct {
	// 佣金率 如果是阶梯佣金 存第一阶梯的佣金率
	CosRatio int64 `json:"cos_ratio"`
	// 阶梯佣金率门槛 非阶梯为0
	LadderBar int64 `json:"ladder_bar"`
	// 阶梯佣金率 非阶梯为0
	LadderCosRatio int64 `json:"ladder_cos_ratio"`
}
type FeatureCosRatio struct {
	// 佣金率 如果是阶梯佣金 存第一阶梯的佣金率
	CosRatio int64 `json:"cos_ratio"`
	// 阶梯佣金率门槛 非阶梯为0
	LadderBar int64 `json:"ladder_bar"`
	// 阶梯佣金率 非阶梯为0
	LadderCosRatio int64 `json:"ladder_cos_ratio"`
}
type CosInfo struct {
	// 佣金规则 1：固定佣金 2：阶梯佣金
	CosType int32 `json:"cos_type"`
	// 生效佣金率
	CosRatio *CosRatio `json:"cos_ratio"`
	// 商品建议佣金率
	CosRatioSug string `json:"cos_ratio_sug"`
	// 同叶子类目同品牌等级商品个数
	SimilarPrdCount string `json:"similar_prd_count"`
	// 修改佣金率未来佣金率
	FeatureCosRatio *FeatureCosRatio `json:"feature_cos_ratio"`
	// 修改佣金率未来佣金率生效时间
	CosValidTime string `json:"cos_valid_time"`
	// 更高优佣金率
	HigerPriorityCosRatio float64 `json:"higer_priority_cos_ratio"`
	// 商品佣金率及所处竞争力百分位，key为佣金率
	CosRatioLevel map[string]string `json:"cos_ratio_level"`
}
type DataItem struct {
	// 策略id
	StrategyId string `json:"strategy_id"`
	// 状态 1：生效中 2：已结束 3：已失效
	Status int64 `json:"status"`
	// 结束时间提示 1：1天内，2：3天内，3：7天内
	EndTimeTip int64 `json:"end_time_tip"`
	// 生效达人信息
	ValidKolInfo *ValidKolInfo `json:"valid_kol_info"`
	// 佣金信息
	CosInfo *CosInfo `json:"cos_info"`
	// 开始时间 2020-05-19 00:00:00
	StartTime string `json:"start_time"`
	// 结束时间 2020-05-19 00:00:00
	EndTime string `json:"end_time"`
}
type BuyinPromotionStrategyListData struct {
	// 策略列表
	Data []DataItem `json:"data"`
	// 总数
	Total int64 `json:"total"`
}
type KolInfosItem struct {
	// 百应ID
	BuyinId int64 `json:"buyin_id"`
	// 达人头像
	AvatarUrl string `json:"avatar_url"`
	// 达人昵称
	AccountName string `json:"account_name"`
	// 添加时间
	AddTime string `json:"add_time"`
}
