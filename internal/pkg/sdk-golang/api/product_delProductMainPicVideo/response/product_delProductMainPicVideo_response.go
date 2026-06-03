package product_delProductMainPicVideo_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductDelProductMainPicVideoResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductDelProductMainPicVideoData `json:"data"`
}
type ProductDelProductMainPicVideoData struct {
	// 删除的素材id
	MaterialId int64 `json:"material_id"`
}
