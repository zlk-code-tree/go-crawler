package order_queryOrderPackageInfo_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderQueryOrderPackageInfoResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderQueryOrderPackageInfoData `json:"data"`
}
type SubSkuDescItem struct {
	// 子商品id
	ProductId string `json:"product_id"`
	// 标准数量
	UnitNum int64 `json:"unit_num"`
	// 子商品在抖店的SkuId
	SkuId string `json:"sku_id"`
}
type CompanyInfo struct {
	// 快递公司名称
	CompanyName string `json:"company_name"`
	// 快递公司编码
	ExpressCompany string `json:"express_company"`
	// 快递公司电话
	Telephone string `json:"telephone"`
	// 快递公司图标
	Icon string `json:"icon"`
	// 快递公司官网
	Url string `json:"url"`
	// 快递单号
	ExpressNo string `json:"express_no"`
}
type DeliveryInfoItem struct {
	// 包裹单id，抖店平台生成
	DeliverId string `json:"deliver_id"`
	// 包裹类型（仓包、发货包裹、补发包裹等等）
	DeliverType int64 `json:"deliver_type"`
	// 快递运单号
	TrackNo string `json:"track_no"`
	// 创建时间unix时间戳
	CreateTime int64 `json:"create_time"`
	// 快递物流公司信息
	CompanyInfo *CompanyInfo `json:"company_info"`
	// 轨迹节点信息
	TrackInfo []TrackInfoItem `json:"track_info"`
}
type PackageInfoItem struct {
	// 当前轨迹状态code；枚举值详见：https://op.jinritemai.com/docs/question-docs/94/1642
	TrackState string `json:"track_state"`
	// 是否补发包裹:正向补发包裹
	IsComplementDeliver bool `json:"is_complement_deliver"`
	// 发货类型 1手动，2csv,3电子面单，4erp，5线下发货
	SendType int64 `json:"send_type"`
	// 商品信息
	ProductInfo []ProductInfoItem `json:"product_info"`
	// 物流信息
	DeliveryInfo []DeliveryInfoItem `json:"delivery_info"`
}
type OrderQueryOrderPackageInfoData struct {
	// 订单包裹信息
	OrderPackageInfo *OrderPackageInfo `json:"order_package_info"`
}
type ActualSubSkuItem struct {
	// 子商品id
	ProductId int64 `json:"product_id"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 子商品skuid
	SkuId int64 `json:"sku_id"`
	// 子商品图片
	SkuImg string `json:"sku_img"`
	// 子商品数量
	Num int64 `json:"num"`
	// 是否是赠品
	IsFree bool `json:"is_free"`
}
type ProductInfoItem struct {
	// 商品绑定的店铺子订单号，抖店平台生成，平台下唯一；注意：一笔订单下有一个子订单和父订单单号相同。
	SkuOrderId string `json:"sku_order_id"`
	// 商品id
	ProductId int64 `json:"product_id"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 商品skuid
	SkuId int64 `json:"sku_id"`
	// 商品图片
	SkuImg string `json:"sku_img"`
	// 商品数量
	Num int64 `json:"num"`
	// 是否是赠品
	IsFree bool `json:"is_free"`
	// 子品规格信息
	SubSkuDesc []SubSkuDescItem `json:"sub_sku_desc"`
	// 实际子品信息
	ActualSubSku []ActualSubSkuItem `json:"actual_sub_sku"`
}
type TrackInfoItem struct {
	// 轨迹文案
	Context string `json:"context"`
	// unix时间戳
	TimeStamp int64 `json:"time_stamp"`
	// 站点名称
	Site string `json:"site"`
	// 轨迹状态code；枚举值详见：https://op.jinritemai.com/docs/question-docs/94/1642
	State string `json:"state"`
	// 轨迹状态详情
	StateDesc string `json:"state_desc"`
}
type WaitSendProductInfo struct {
	// 商品信息
	ProductInfo []ProductInfoItem `json:"product_info"`
	// 物流信息
	DeliveryInfo []DeliveryInfoItem `json:"delivery_info"`
}
type OrderPackageInfo struct {
	// 店铺id，抖店平台生成，平台下唯一
	ShopId int64 `json:"shop_id"`
	// 订单id，抖店平台生成，平台下唯一
	ShopOrderId string `json:"shop_order_id"`
	// 订单状态1 待确认/待支付（订单创建完毕）105 已支付 2 备货中 101 部分发货 3 已发货（全部发货）4 已取消5 已完成（已收货）
	OrderStatus int64 `json:"order_status"`
	// 是否是一单多包裹
	IsMultiPackage bool `json:"is_multi_package"`
	// 已发货订单包裹信息
	PackageInfo []PackageInfoItem `json:"package_info"`
	// 待发货商品信息
	WaitSendProductInfo *WaitSendProductInfo `json:"wait_send_product_info"`
}
