package order_queryOrderLogistics_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderQueryOrderLogisticsResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderQueryOrderLogisticsData `json:"data"`
}
type OrderQueryOrderLogisticsData struct {
	// 订单包裹数据
	OrderPackageInfo *OrderPackageInfo `json:"order_package_info"`
}
type ActualSubSkuItem struct {
	// 商品ID
	SkuId int64 `json:"sku_id"`
	// sku图片
	SkuImg string `json:"sku_img"`
	// sku件数
	Num int64 `json:"num"`
	// 是否是赠品，true赠品
	IsFree bool `json:"is_free"`
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 商品名称
	ProductName string `json:"product_name"`
}
type SubSkuDescItem struct {
	// 商品单ID
	ProductId string `json:"product_id"`
	// 标准数量
	UnitNum int64 `json:"unit_num"`
	// 子品
	SkuId string `json:"sku_id"`
}
type ProductInfoItem struct {
	// sku件数
	Num int64 `json:"num"`
	// 子品信息
	ActualSubSku []ActualSubSkuItem `json:"actual_sub_sku"`
	// 商品单ID
	SkuOrderId string `json:"sku_order_id"`
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 商品ID
	SkuId int64 `json:"sku_id"`
	// sku图片
	SkuImg string `json:"sku_img"`
	// 是否是赠品，true赠品false主品
	IsFree bool `json:"is_free"`
	// 子品规格信息
	SubSkuDesc []SubSkuDescItem `json:"sub_sku_desc"`
}
type ExpressInfo struct {
	// 快递单号
	ExpressNo string `json:"express_no"`
	// 快递公司中文名称：顺丰快递
	CompanyName string `json:"company_name"`
	// 快递公司：shunfeng
	ExpressCompany string `json:"express_company"`
	// 快递公司电话
	Telephone string `json:"telephone"`
	// 快递公司图标
	Icon string `json:"icon"`
	// 快递公司官网地址
	Url string `json:"url"`
}
type ReverseExpressInfo struct {
	// 快递信息
	ExpressInfo *ExpressInfo `json:"express_info"`
	// 包裹ID
	DeliverId string `json:"deliver_id"`
	// 物流单号
	LogisticsNo string `json:"logistics_no"`
	// 快递单号
	TrackNo string `json:"track_no"`
}
type CompanyInfo struct {
	// 快递单号
	ExpressNo string `json:"express_no"`
	// 快递公司中文名称：顺丰快递
	CompanyName string `json:"company_name"`
	// 快递公司：shunfeng
	ExpressCompany string `json:"express_company"`
	// 快递公司电话
	Telephone string `json:"telephone"`
	// 快递公司图标
	Icon string `json:"icon"`
	// 快递公司官网地址
	Url string `json:"url"`
}
type PreDeliverInfo struct {
	// 快递信息
	ExpressInfo *ExpressInfo `json:"express_info"`
	// 包裹ID
	DeliverId string `json:"deliver_id"`
	// 物流单号
	LogisticsNo string `json:"logistics_no"`
	// 快递单号
	TrackNo string `json:"track_no"`
}
type PackageInfoItem struct {
	// 商品信息
	ProductInfo []ProductInfoItem `json:"product_info"`
	// 快递轨迹信息
	DeliveryInfo []DeliveryInfoItem `json:"delivery_info"`
	// 快递状态
	TrackState string `json:"track_state"`
	// 是否补发包裹:正向补发包裹
	IsComplementDeliver bool `json:"is_complement_deliver"`
	// 发货类型 1手动，2csv,3电子面单，4erp，5线下发货
	SendType int64 `json:"send_type"`
	// 发货时间
	ShipTime int64 `json:"ship_time"`
}
type TrackInfoItem struct {
	// 轨迹文案
	Context string `json:"context"`
	// 状态变更时间
	TimeStamp int64 `json:"time_stamp"`
	// 站点信息
	Site string `json:"site"`
	// 物流状态
	State string `json:"state"`
	// 物流状态描述
	StateDesc string `json:"state_desc"`
}
type DeliveryInfoItem struct {
	// deliverId
	DeliverId string `json:"deliver_id"`
	// 仓包、发货包裹、补发包裹等等
	DeliverType int64 `json:"deliver_type"`
	// 发货至, 1用户,2BIC,3商户, 默认1
	SendTo int64 `json:"send_to"`
	// 新疆中转B段合单类型
	TransitMergeType string `json:"transit_merge_type"`
	// 快递轨迹信息
	TrackInfo []TrackInfoItem `json:"track_info"`
	// 当前段拒收退回的包裹信息
	ReverseExpressInfo *ReverseExpressInfo `json:"reverse_express_info"`
	// 快递单号
	TrackNo string `json:"track_no"`
	// 发货来源 1用户,2BIC,3商户, 默认3
	SendFrom int64 `json:"send_from"`
	// 快递公司信息
	CompanyInfo *CompanyInfo `json:"company_info"`
	// 上一段的包裹信息
	PreDeliverInfo *PreDeliverInfo `json:"pre_deliver_info"`
}
type OrderPackageInfo struct {
	// 店铺单号
	ShopOrderId string `json:"shop_order_id"`
	// 订单状态
	OrderStatus int64 `json:"order_status"`
	// 包裹数据
	PackageInfo []PackageInfoItem `json:"package_info"`
	// 店铺Id
	ShopId int64 `json:"shop_id"`
}
