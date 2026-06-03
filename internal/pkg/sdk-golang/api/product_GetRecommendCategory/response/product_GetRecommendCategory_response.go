package product_GetRecommendCategory_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetRecommendCategoryResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetRecommendCategoryData `json:"data"`
}
type CategoryDetail struct {
	// 一级类目id
	FirstCid int64 `json:"first_cid"`
	// 一级类目名称
	FirstCname string `json:"first_cname"`
	// 二级类目id
	SecondCid int64 `json:"second_cid"`
	// 二级类目名称
	SecondCname string `json:"second_cname"`
	// 三级类目id
	ThirdCid int64 `json:"third_cid"`
	// 三级类目名称
	ThirdCname string `json:"third_cname"`
	// 四级类目id
	FourthCid int64 `json:"fourth_cid"`
	// 四级类目名称
	FourthCname string `json:"fourth_cname"`
}
type CategoryDetailsItem struct {
	// 类目详情
	CategoryDetail *CategoryDetail `json:"category_detail"`
	// 类目资质状态，0有资质；1资质过期；2无资质
	QualificationStatus int64 `json:"qualification_status"`
}
type ProductGetRecommendCategoryData struct {
	// 推荐类目结果
	CategoryDetails []CategoryDetailsItem `json:"categoryDetails"`
	// 类目预测记录id
	RecommendId string `json:"recommend_id"`
	// 深度思考异步任务ID，命中深度思考时返回，用于轮询深度思考结果
	AsyncTaskId string `json:"async_task_id"`
	// 深度思考任务状态，命中深度思考时返回。0-init 1-running 2-success 3-failed
	AsyncTaskStatus int64 `json:"async_task_status"`
	// 深度思考超时时间，单位：ms，命中深度思考时返回，用于轮询深度思考结果
	Timeout int64 `json:"timeout"`
	// 深度思考轮询间隔，单位：ms，命中深度思考时返回，用于轮询深度思考结果
	Interval int64 `json:"interval"`
}
