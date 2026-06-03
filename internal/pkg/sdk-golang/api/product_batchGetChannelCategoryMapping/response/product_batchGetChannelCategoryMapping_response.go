package product_batchGetChannelCategoryMapping_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductBatchGetChannelCategoryMappingResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductBatchGetChannelCategoryMappingData `json:"data"`
}
type CategoryMappingsItem struct {
	// 状态：1有效；2无效
	Valid int64 `json:"valid"`
	// 映射类型:0门捷列夫类目到标准类目；1标准类目到门捷列夫类目
	MappingType int32 `json:"mapping_type"`
	// 映射关系类型：0一对一；1一对多；2多对一；3多对多
	RelationType int32 `json:"relation_type"`
	// 映射类目id
	MappingCategoryId int64 `json:"mapping_category_id"`
}
type RespItemsItem struct {
	// 映射列表
	CategoryMappings []CategoryMappingsItem `json:"category_mappings"`
	// 入参类目ID
	SourceCategoryId int64 `json:"source_category_id"`
}
type ProductBatchGetChannelCategoryMappingData struct {
	// 映射关系返回结果列表
	RespItems []RespItemsItem `json:"resp_items"`
}
