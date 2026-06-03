package product_listChannelProduct_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductListChannelProductResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductListChannelProductData `json:"data"`
}
type MainImgMaterial struct {
	// 素材状态0：草稿
	MaterialStatus int64 `json:"material_status"`
	// 审核驳回原因
	RejectReason string `json:"reject_reason"`
	// 素材类型，主图为18
	MaterialType int64 `json:"material_type"`
}
type KolInfo struct {
	// 绑定达人数量
	BindKolNum int64 `json:"bind_kol_num"`
	// 绑定所有
	BindAll bool `json:"bind_all"`
}
type DataItem struct {
	// 标题前缀
	NamePrefix string `json:"name_prefix"`
	// 标题内容
	NameSubject string `json:"name_subject"`
	// 渠道品详情图信息
	DetailBriefMaterial *DetailBriefMaterial `json:"detail_brief_material"`
	// 渠道品主图信息
	MainImgMaterial *MainImgMaterial `json:"main_img_material"`
	// 商家编码与主品保持一致配置，true-保持一致，false-自定义
	SkuCodeFollowMain bool `json:"sku_code_follow_main"`
	// 渠道品标题
	Name string `json:"name"`
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型
	ChannelType int64 `json:"channel_type"`
	// 有效期开始时间
	StartTime string `json:"start_time"`
	// 有效期结束时间
	EndTime string `json:"end_time"`
	// 渠道品状态0：下架2：上架3：封禁
	Status int64 `json:"status"`
	// 绑定达人信息
	KolInfo *KolInfo `json:"kol_info"`
}
type ProductListChannelProductData struct {
	// 渠道品列表
	Data []DataItem `json:"data"`
	// 总数
	Total int64 `json:"total"`
	// 当前页面数
	Page int64 `json:"page"`
	// 每页数目大小
	Size int64 `json:"size"`
}
type DetailBriefMaterial struct {
	// 素材类型，详情图为28
	MaterialType int64 `json:"material_type"`
	// 审核驳回原因
	RejectReason string `json:"reject_reason"`
	// 素材状态0：草稿
	MaterialStatus int64 `json:"material_status"`
}
