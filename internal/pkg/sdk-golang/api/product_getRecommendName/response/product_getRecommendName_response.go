package product_getRecommendName_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetRecommendNameResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetRecommendNameData `json:"data"`
}
type RecommendNameItem struct {
	// 推荐场景：1. product_name_prefix 前缀场景，需要将返回的recommend_value拼接在name标题自动的最前面2. product_name_suffix 后缀场景，需要将返回的recommend_value拼接在name标题自动的最后面
	RecommendScene string `json:"recommend_scene"`
	// 推荐结果，推荐结果需要根据recommend_scene判断前缀或者后缀，拼接在name字段的最前面或者最后面
	RecommendValue string `json:"recommend_value"`
}
type ProductGetRecommendNameData struct {
	// 商品标题推荐结果，当返回值为空时，使用【/product/getProductUpdateRule】接口查询类目id查看recommend_name_rule.satisfy_prefix是否=true
	RecommendName []RecommendNameItem `json:"recommend_name"`
}
