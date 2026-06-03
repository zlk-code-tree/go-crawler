package product_GetRecommendCategory_request

import (
	"doudian.com/open/sdk_golang/api/product_GetRecommendCategory/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductGetRecommendCategoryRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductGetRecommendCategoryParam 
}
func (c *ProductGetRecommendCategoryRequest) GetUrlPath() string{
	return "/product/GetRecommendCategory"
}


func New() *ProductGetRecommendCategoryRequest{
	request := &ProductGetRecommendCategoryRequest{
		Param: &ProductGetRecommendCategoryParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductGetRecommendCategoryRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_GetRecommendCategory_response.ProductGetRecommendCategoryResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_GetRecommendCategory_response.ProductGetRecommendCategoryResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductGetRecommendCategoryRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductGetRecommendCategoryRequest) GetParams() *ProductGetRecommendCategoryParam{
	return c.Param
}


type PicItem struct {
	// 图片链接，必须是素材中心的url
	Url string `json:"url"`
}
type ProductFormatNewItem struct {
	// 属性id
	Value *int64 `json:"value"`
	// 属性名称
	Name *string `json:"name"`
}
type ProductGetRecommendCategoryParam struct {
	// category_infer: 基于标题、图片等推断商品类目；predict_by_title_and_img：基于标题、图片等推断商品类目（图片必传）；
	Scene string `json:"scene"`
	// 商品主图图片url，scene为predict_by_title_and_img时必传
	Pic []PicItem `json:"pic"`
	// 商品类目id，无需传递
	CategoryLeafId *int64 `json:"category_leaf_id"`
	// 商品标题
	Name *string `json:"name"`
	// 商品类目属性
	ProductFormatNew *map[int64][]ProductFormatNewItem `json:"product_format_new"`
	// 品牌id
	StandardBrandId *int64 `json:"standard_brand_id"`
	// 深度思考异步任务ID，命中深度思考时返回，用于轮询深度思考结果
	AsyncTaskId *string `json:"async_task_id"`
}
