package cpaypromotion_queryProductCpayPromotionAndShopStatus_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CpaypromotionQueryProductCpayPromotionAndShopStatusResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CpaypromotionQueryProductCpayPromotionAndShopStatusData `json:"data"`
}
type ProductDiscountAndPromotionActivityResponse struct {
	// 新模型优惠，如果已经报名，则不为空
	DiscountInfo *DiscountInfo `json:"discount_info"`
	// 新模型招商活动，如果需要重新报名，则需要依赖
	PromotionActivityInfo *PromotionActivityInfo `json:"promotion_activity_info"`
}
type BizBaseResponse struct {
	// 标识失败情况下，是否应该重试
	CanRetry bool `json:"can_retry"`
	// 响应code
	RetCode string `json:"ret_code"`
	// 响应msg
	RetMsg string `json:"ret_msg"`
	// 标识请求是否成功
	Success bool `json:"success"`
}
type CpaypromotionQueryProductCpayPromotionAndShopStatusData struct {
	// 商家是否准入，如果为false，则不允许使用报名商品管理
	ShopIsAccess bool `json:"shop_is_access"`
	// 店铺签约情况
	ShopStatusResponse *ShopStatusResponse `json:"shop_status_response"`
	// 商品管理报名商品限制规则
	ProductRegisterRuleResponse *ProductRegisterRuleResponse `json:"product_register_rule_response"`
	// 招商活动报名
	ProductDiscountAndPromotionActivityResponse *ProductDiscountAndPromotionActivityResponse `json:"product_discount_and_promotion_activity_response"`
	// 响应结果
	BizBaseResponse *BizBaseResponse `json:"biz_base_response"`
}
type ShopStatusResponse struct {
	// Deprecated 月付签约状态
	CpaySignStatus int32 `json:"cpay_sign_status"`
	// 营销签约状态
	PromotionSignStatus int32 `json:"promotion_sign_status"`
	// 待签约的协议列表
	ToSignContractList []ToSignContractListItem `json:"to_sign_contract_list"`
}
type ProductRegisterRuleResponse struct {
	// 商品限制规则
	Rule *Rule `json:"rule"`
}
type PeriodSharedProportionListItem struct {
	// 期次
	Period int32 `json:"period"`
	// 优惠后商家共担费率
	SharedProportion string `json:"shared_proportion"`
}
type PeriodSharedProportionConfig struct {
	// 已经报名的期次和费率
	PeriodSharedProportionList []PeriodSharedProportionListItem `json:"period_shared_proportion_list"`
}
type SupportPeriodItem struct {
	// 期数
	Period string `json:"period"`
	// 原始费率
	OriginRatio string `json:"origin_ratio"`
	// 实际费率
	ActualRatio string `json:"actual_ratio"`
}
type ToSignContractListItem struct {
	// 协议类型
	ContractType int32 `json:"contract_type"`
	// 合同名title
	ContractName string `json:"contract_name"`
	// 合同连接 url
	ContractUrl string `json:"contract_url"`
}
type Rule struct {
	// 不支持的商品类目
	UnSupportedCategoryIDs []int64 `json:"un_supported_category_i_ds"`
	// 商品最低价格，单位分
	MinPrice int64 `json:"min_price"`
}
type DiscountInfo struct {
	// 商家活动ID
	PromotionActivityId string `json:"promotion_activity_id"`
	// 活动模板ID
	DiscountId string `json:"discount_id"`
	// 期次与费率配置
	PeriodSharedProportionConfig *PeriodSharedProportionConfig `json:"period_shared_proportion_config"`
}
type PromotionActivityInfo struct {
	// 招商活动结束时间
	PromotionActivityEndTime int64 `json:"promotion_activity_end_time"`
	// 活动支持报名的期数优惠信息
	SupportPeriod []SupportPeriodItem `json:"support_period"`
	// 招商活动id
	ActivityId string `json:"activity_id"`
	// 招商活动开始时间
	PromotionActivityStartTime int64 `json:"promotion_activity_start_time"`
}
