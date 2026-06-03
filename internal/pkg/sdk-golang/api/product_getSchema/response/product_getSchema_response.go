package product_getSchema_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetSchemaResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetSchemaData `json:"data"`
}
type ProductData struct {
	// 商品在店铺中状态: 0-在线；1-下线；2-删除；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	Status int64 `json:"status"`
	// 商品审核状态: 1-未提交；2-待审核；3-审核通过；4-审核未通过；5-封禁；7-审核通过待上架；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	CheckStatus int64 `json:"check_status"`
	// 草稿状态；0-无草稿,1-未提审,2-待审核,3-审核通过,4-审核未通过。详见：https://op.jinritemai.com/docs/question-docs/92/2070
	DraftStatus int64 `json:"draft_status"`
}
type ProductGetSchemaData struct {
	// schem数据、json格式
	Model string `json:"model"`
	// 商品数据
	ProductData *ProductData `json:"product_data"`
}
