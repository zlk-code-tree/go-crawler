package buyin_queryShopAllianceProducts_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinQueryShopAllianceProductsResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinQueryShopAllianceProductsData `json:"data"`
}
type SampleConf struct {
	// 达人带货结算订单量≥所设订单量，返还达人买样全款；如果不支持达人买样返款此字段无意义
	BuyVolumeLimit int32 `json:"buy_volume_limit"`
	// 达人买样返款配置；1：支持达人买样返款，2-不支持达人买样返款
	BuyStyle int32 `json:"buy_style"`
}
type ProductListItem struct {
	// 达人买样返款设置信息
	SampleConf *SampleConf `json:"sample_conf"`
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 商品名称
	Title string `json:"title"`
	// 商品主图
	Cover string `json:"cover"`
	// 商品售价（单位为分）
	Price int64 `json:"price"`
	// 市场价 (单位为分)
	MarketPrice int64 `json:"market_price"`
	// 商品状态，2：商品在架；3：商家关闭推广；4：商品下架；6：平台治理关闭；99：商家删除
	Status int64 `json:"status"`
	// 普通佣金率，值乘以了100， 如50%返回50
	CosRatio float64 `json:"cos_ratio"`
	// 商品链接
	DetailUrl string `json:"detail_url"`
	// 销量（单位为分）
	Sales int64 `json:"sales"`
	// 佣金（单位为分）
	CosFee int64 `json:"cos_fee"`
	// 更新时间戳
	UpdateTime int64 `json:"update_time"`
	// 图片列表
	Imgs []string `json:"imgs"`
	// 审核状态。 0：未审核；2：审核未通过；3：审核通过
	CheckStatus int64 `json:"check_status"`
	// 创建时间戳
	CreateTime int64 `json:"create_time"`
	// 是否开启专属计划
	IsExclusive bool `json:"is_exclusive"`
}
type BuyinQueryShopAllianceProductsData struct {
	// 总数，最大10000
	Total int64 `json:"total"`
	// 商品列表
	ProductList []ProductListItem `json:"product_list"`
}
