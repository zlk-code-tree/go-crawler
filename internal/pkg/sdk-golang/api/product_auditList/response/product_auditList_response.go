package product_auditList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductAuditListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductAuditListData `json:"data"`
}
type AuditReasonDetailsItem struct {
	// 详细的驳回明细原因
	RejectReasonList []RejectReasonListItem `json:"reject_reason_list"`
	// 驳回模块名称
	Title string `json:"title"`
	// 驳回模块类型
	Type string `json:"type"`
}
type RecordsItem struct {
	// 驳回原因列表
	AuditReasonDetails []AuditReasonDetailsItem `json:"audit_reason_details"`
	// 商品id
	ProductId int64 `json:"product_id"`
	// 商品名称
	Title string `json:"title"`
	// 商品主图
	ImgUrl string `json:"img_url"`
	// 0-审核中 1-审核通过 2-审核拒绝
	PublishStatus int64 `json:"publish_status"`
	// 提审时间，单位为秒
	PublishTime int64 `json:"publish_time"`
	// 审核通过/拒绝时间，单位为秒
	AuditTime int64 `json:"audit_time"`
	// 审核未通过理由
	AuditReason map[string][]string `json:"audit_reason"`
}
type ProductAuditListData struct {
	// 审核记录列表
	Records []RecordsItem `json:"records"`
	// 总数量
	Total int64 `json:"total"`
}
type SkuInfosItem struct {
	// skuid
	SkuId int64 `json:"sku_id"`
	// 规格名
	SkuTitle string `json:"sku_title"`
}
type RejectImgListItem struct {
	// 图片url
	Url string `json:"url"`
}
type RejectReasonListItem struct {
	// 明细sku
	SkuInfos []SkuInfosItem `json:"sku_infos"`
	// 驳回图片
	RejectImgList []RejectImgListItem `json:"reject_img_list"`
	// 驳回原因
	ReasonText string `json:"reason_text"`
}
