package product_salesInherit_submitList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductSalesInheritSubmitListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductSalesInheritSubmitListData `json:"data"`
}
type SalesExtendRecordListItem struct {
	// 站外平台id（1:淘宝）
	PlatformId int64 `json:"platform_id"`
	// 提报时间，单位：秒；
	SubmitTime int64 `json:"submit_time"`
	// 提报销量
	SubmitSalesNum int64 `json:"submit_sales_num"`
	// 抖店商品id
	ProductId int64 `json:"product_id"`
	// 站外商品id
	OutsideProductId int64 `json:"outside_product_id"`
	// 审核状态（0:待审核；1:已通过；2:已驳回）
	AuditStatus int32 `json:"audit_status"`
	// 驳回原因
	AuditMsg string `json:"audit_msg"`
	// 审核结果时间包含审核失败和成功；单位：秒；
	AuditTime int64 `json:"audit_time"`
	// 实际审核通过后的继承销量，审核通过后可在抖店c端买家商详页面查看的销量；
	ExtendSalesNum int64 `json:"extend_sales_num"`
}
type ProductSalesInheritSubmitListData struct {
	// 销量继承记录列表
	SalesExtendRecordList []SalesExtendRecordListItem `json:"sales_extend_record_list"`
	// 总条数
	Total int64 `json:"total"`
}
