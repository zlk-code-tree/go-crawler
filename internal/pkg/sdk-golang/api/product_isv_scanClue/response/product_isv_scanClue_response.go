package product_isv_scanClue_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductIsvScanClueResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data []DataItem `json:"data"`
}
type ProductIsvScanClueData struct {
	// 线索列表
	Data []DataItem `json:"data"`
}
type DataItem struct {
	// 线索ID
	ClueId int64 `json:"clue_id"`
	// 线索名
	ClueName string `json:"clue_name"`
	// 线索图片URL
	PicUrl string `json:"pic_url"`
	// 一级类目ID
	FirstCid int64 `json:"first_cid"`
	// 一级类目名
	FirstName string `json:"first_name"`
	// 二级类目ID
	SecondCid int64 `json:"second_cid"`
	// 二级类目名
	SecondName string `json:"second_name"`
	// 品牌ID
	BrandId int64 `json:"brand_id"`
	// 品牌名-中文
	BrandNameCn string `json:"brand_name_cn"`
	// 品牌名-英文
	BrandNameEn string `json:"brand_name_en"`
	// 建议售价最小值, 单位 分
	PriceMin int64 `json:"price_min"`
	// 建议售价最大值, 单位 分
	PriceMax int64 `json:"price_max"`
	// 搜索热度-根据搜索人数进行指数化处理，热度越高代表搜索人数越多
	SearchHeat int64 `json:"search_heat"`
	// 需供比-根据近30天的用户搜索量 / 商品数量进行指数化处理，需供比越大代表当前商机潜力越大（供不应求）
	DemandAndSupplyRate float64 `json:"demand_and_supply_rate"`
	// 三级类目ID
	ThirdCid int64 `json:"third_cid"`
	// 三级类目名
	ThirdName string `json:"third_name"`
	// 四级类目ID
	FourthCid int64 `json:"fourth_cid"`
	// 四级类目名
	FourthName string `json:"fourth_name"`
	// 线索名的分词列表-底层是搜索审核打分的分词规则
	QueryNameCutList []string `json:"query_name_cut_list"`
}
