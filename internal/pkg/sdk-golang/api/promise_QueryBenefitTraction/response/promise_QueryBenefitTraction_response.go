package promise_QueryBenefitTraction_response

import (
	"doudian.com/open/sdk_golang/core"
)

type PromiseQueryBenefitTractionResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *PromiseQueryBenefitTractionData `json:"data"`
}
type ProductBenefitMetricsListItem struct {
	// 商品 ID
	ProductId int64 `json:"product_id"`
	// 商品利益点+数据
	ProductBenefitMetrics *ProductBenefitMetrics `json:"product_benefit_metrics"`
}
type TractionInfoByFulfillmentModeMapItem struct {
	// 商品维度查询 or needAggregationBenefitByShop=false 时无此数据
	ShopBenefitMetrics *ShopBenefitMetrics `json:"shop_benefit_metrics"`
	// 商品维度利益点
	ProductBenefitMetricsList []ProductBenefitMetricsListItem `json:"product_benefit_metrics_list"`
}
type PromiseQueryBenefitTractionData struct {
	// 商家 48h 发货占比
	MerchantShipRate48h float64 `json:"merchant_ship_rate48h"`
	// key为发货模式。enum FulfillmentMode {    IN_STOCK_PRE_SALE     = 1,  // (新)现货+预售    FULL_PAYMENT_PRE_SALE = 2  // 全款预售    ALL=3 // [(新)现货+预售]+[全款预售]}
	TractionInfoByFulfillmentModeMap map[int32]TractionInfoByFulfillmentModeMapItem `json:"traction_info_by_fulfillment_mode_map"`
}
type ShopBenefitMetrics struct {
	// 预计结算 GMV 相对提升比例
	ExpectedSettlementGmvLift float64 `json:"expected_settlement_gmv_lift"`
	// 近 14 天预售订单量
	SalesVolumeLast14Days int64 `json:"sales_volume_last14_days"`
	// 是否有定量利益点数据
	HasAccurateBenefit bool `json:"has_accurate_benefit"`
	// 预计结算率提升（如 0.05 = 提升 5pp）
	ExpectedSettlementRateLift float64 `json:"expected_settlement_rate_lift"`
	// 预计曝光 PV 提升比例
	ExpectedExposurePvLift float64 `json:"expected_exposure_pv_lift"`
	// 预计发货前退款率降低比例
	ExpectedPreShipmentRefundRateDecrease float64 `json:"expected_pre_shipment_refund_rate_decrease"`
}
type SkuRecommendationsMapItem struct {
	// SKU ID
	SkuId int64 `json:"sku_id"`
	// 库存建议增量（件数，正数=需补货,0代表无建议）
	RecommendedStockIncrement int64 `json:"recommended_stock_increment"`
}
type ProductBenefitMetrics struct {
	// 是否有定量利益点数据
	HasAccurateBenefit bool `json:"has_accurate_benefit"`
	// 预计曝光 PV 提升比例
	ExpectedExposurePvLift float64 `json:"expected_exposure_pv_lift"`
	// 预计发货前退款率降低比例
	ExpectedPreShipmentRefundRateDecrease float64 `json:"expected_pre_shipment_refund_rate_decrease"`
	// 预计结算 GMV 相对提升比例
	ExpectedSettlementGmvLift float64 `json:"expected_settlement_gmv_lift"`
	// 商品维度库存建议增量（件）
	RecommendedStockIncrement int64 `json:"recommended_stock_increment"`
	// SKU 维度推荐
	SkuRecommendationsMap map[int64]SkuRecommendationsMapItem `json:"sku_recommendations_map"`
	// 商品 ID
	ProductId int64 `json:"product_id"`
	// 预计结算率提升（如 0.05 = 提升 5pp）
	ExpectedSettlementRateLift float64 `json:"expected_settlement_rate_lift"`
	// 近 14 天预售订单量
	SalesVolumeLast14Days int64 `json:"sales_volume_last14_days"`
}
