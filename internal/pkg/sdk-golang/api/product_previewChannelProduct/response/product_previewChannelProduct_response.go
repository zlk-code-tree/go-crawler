package product_previewChannelProduct_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductPreviewChannelProductResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductPreviewChannelProductData `json:"data"`
}
type FreightTemplate struct {
	// 渠道品运费模板是否与主品保持一致
	FreightFollowMain bool `json:"freight_follow_main"`
	// 渠道品定制运费模板ID，当设置了freight_follow_main=true时与主品运费模板保持一致；否则需要传值。
	FreightId int64 `json:"freight_id"`
}
type ProductPreviewChannelProductData struct {
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型
	ChannelType int64 `json:"channel_type"`
	// 渠道品标题，包含标题前缀+标题内容
	Name string `json:"name"`
	// 有效期开始时间
	StartTime string `json:"start_time"`
	// 有效期结束时间
	EndTime string `json:"end_time"`
	// 价格是否跟随主品
	PriceFollowMain bool `json:"price_follow_main"`
	// 库存是否与主品共享
	ShareMainStock bool `json:"share_main_stock"`
	// 是否『达人专属』渠道品
	IsKolExclusive bool `json:"is_kol_exclusive"`
	// 渠道品状态0：下架2：上架3：封禁
	Status int64 `json:"status"`
	// 限购信息
	ChannelBuyLimit *ChannelBuyLimit `json:"channel_buy_limit"`
	// SKU信息
	SkuChannel []SkuChannelItem `json:"sku_channel"`
	// 渠道品主图信息
	MainImgMaterial *MainImgMaterial `json:"main_img_material"`
	// 渠道品详情图信息
	DetailBriefMaterial *DetailBriefMaterial `json:"detail_brief_material"`
	// 运费模板配置
	FreightTemplate *FreightTemplate `json:"freight_template"`
	// 商家编码与主品保持一致配置，true-与主品保持一致，false-自定义
	SkuCodeFollowMain bool `json:"sku_code_follow_main"`
	// 渠道品审核状态，1：未提交审核；2：审核中；3：审核通过；4：审核拒绝
	CheckStatus int64 `json:"check_status"`
	// 标题前缀
	NamePrefix string `json:"name_prefix"`
	// 标题内容
	NameSubject string `json:"name_subject"`
}
type ChannelBuyLimit struct {
	// 每用户总限购数
	TotalBuyNum int64 `json:"total_buy_num"`
	// 单次购买上限
	MaxBuyNum int64 `json:"max_buy_num"`
	// 单次限购下限
	MinBuyNum int64 `json:"min_buy_num"`
}
type SkuChannel struct {
	// 渠道品定制SKU商家编码。可忽略，默认使用主商品SKU商家编码。
	Code string `json:"code"`
	// 主品SKU ID
	SkuId int64 `json:"sku_id"`
	// 渠道品SKU价格
	Price int64 `json:"price"`
	// 渠道品SKU库存
	StockNum int64 `json:"stock_num"`
}
type SkuMain struct {
	// 主品SKU价格
	Price int64 `json:"price"`
	// 主品SKU库存
	Stock int64 `json:"stock"`
}
type SkuChannelItem struct {
	// 主品SKU ID
	SkuId int64 `json:"sku_id"`
	// SKU名称
	Name string `json:"name"`
	// 渠道品SKU
	SkuChannel *SkuChannel `json:"sku_channel"`
	// 主品SKU
	SkuMain *SkuMain `json:"sku_main"`
	// SKU状态0：渠道品有该SKU1：渠道品无该SKU
	Status int64 `json:"status"`
}
type MainImgMaterial struct {
	// 主图列表
	ImgList []string `json:"img_list"`
	// 素材类型，主图为18
	MaterialType int64 `json:"material_type"`
	// 已经废弃字段，请不要再使用，图片不单独审核，只需要看渠道品审核字段check_status
	MaterialStatus int64 `json:"material_status"`
	// 审核驳回原因
	RejectReason string `json:"reject_reason"`
}
type DetailBriefMaterial struct {
	// 审核驳回原因
	RejectReason string `json:"reject_reason"`
	// 已经废弃字段，请不要再使用，描述不单独审核，只需要看渠道品审核字段check_status
	MaterialStatus int64 `json:"material_status"`
	// 素材类型，详情图为28
	MaterialType int64 `json:"material_type"`
	// 详情图列表
	ImgList []string `json:"img_list"`
}
