package product_addChannelProduct_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductAddChannelProductResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductAddChannelProductData `json:"data"`
}
type Channel struct {
	// 渠道类型（3：商家自卖，4：达人带货）
	ChannelType int64 `json:"channel_type"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
}
type ProductAddChannelProductData struct {
	// 店铺ID
	ShopId int64 `json:"shop_id"`
	// 主商品ID
	ProductId int64 `json:"product_id"`
	// 渠道信息
	Channel *Channel `json:"channel"`
	// 是否『达人专属』渠道品
	IsKolExclusive bool `json:"is_kol_exclusive"`
}
