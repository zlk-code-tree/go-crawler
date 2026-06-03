package product_listV2_request

import (
	"doudian.com/open/sdk_golang/api/product_listV2/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductListV2Request struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductListV2Param 
}
func (c *ProductListV2Request) GetUrlPath() string{
	return "/product/listV2"
}


func New() *ProductListV2Request{
	request := &ProductListV2Request{
		Param: &ProductListV2Param{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductListV2Request) Execute(accessToken *doudian_sdk.AccessToken) (*product_listV2_response.ProductListV2Response, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_listV2_response.ProductListV2Response{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductListV2Request) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductListV2Request) GetParams() *ProductListV2Param{
	return c.Param
}


type LookupOption struct {
	// 是否需要获取标题前后缀
	NeedNameAffix *bool `json:"need_name_affix"`
	// 是否需要获取商品标题长度限制规则
	NeedTitleLimit *bool `json:"need_title_limit"`
}
type QueryOptions struct {
	// 只查询类目错放的商品
	ExistCategoryMisplaced *bool `json:"exist_category_misplaced"`
	// 只查询有驳回建议的商品
	ExistAuditRejectSuggest *bool `json:"exist_audit_reject_suggest"`
	// 需要返回审核驳回建议信息
	NeedAuditRejectSuggest *bool `json:"need_audit_reject_suggest"`
}
type ProductListV2Param struct {
	// 商品在店铺中状态: 0-在线；1-下线；2-删除；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	Status *int64 `json:"status"`
	// 商品审核状态: 1-未提交；2-待审核；3-审核通过；4-审核未通过；5-封禁；7-审核通过待上架；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	CheckStatus *int64 `json:"check_status"`
	// 商品类型；0-普通；1-新客商品；3-虚拟；6-玉石闪购；7-云闪购 ；127-其他类型；
	ProductType *int64 `json:"product_type"`
	// 商品创建开始时间，unix时间戳，单位：秒；
	StartTime *int64 `json:"start_time"`
	// 商品创建结束时间，unix时间戳，单位：秒；
	EndTime *int64 `json:"end_time"`
	// 商品更新开始时间，unix时间戳，单位：秒；注意：查询范围是update_start_time和update_end_time之间的数据，不包含入参时间。
	UpdateStartTime *int64 `json:"update_start_time"`
	// 商品更新结束时间，unix时间戳，单位：秒；注意：查询范围是update_start_time和update_end_time之间的数据，不包含入参时间。
	UpdateEndTime *int64 `json:"update_end_time"`
	// 页码，从1开始，最大为100；page* size最大不能超过1万条
	Page int64 `json:"page"`
	// 页数，填写范围1~100；page* size最大不能超过1万条
	Size int64 `json:"size"`
	// 小时达商家使用的门店id，即时零售单店版，无需使用
	StoreId *int64 `json:"store_id"`
	// 商品标题，支持模糊匹配
	Name *string `json:"name"`
	// 商品id，最大支持传入100个；
	ProductId []string `json:"product_id"`
	// 是否使用游标，true 使用，false 不使用；使用游标能遍历全量商品，不会受到只能查询到10000个数据的影响。特别说明：使用游标查询时，page字段失效，只会查询当前游标后面size条数据；根据获取查询数据为空来判断本次遍历结束；查询过程中数据发生变更，可能重复、遗漏数据；需要保证每次查询的筛选条件一致。
	UseCursor *bool `json:"use_cursor"`
	// 游标id，结合use_cursor使用（use_cursor=true）;首次查询时不需要传值，每次查询结果会返回cursor_id，下一次查询时复制给该字段。当查询返回数据为空时，结束遍历。
	CursorId *string `json:"cursor_id"`
	// 能否参加搭配
	CanCombineProduct *bool `json:"can_combine_product"`
	// 查询option
	LookupOption *LookupOption `json:"lookup_option"`
	// 是否需要自动整改信息
	NeedRectificationInfo *bool `json:"need_rectification_info"`
	// 查询定制参数
	QueryOptions *QueryOptions `json:"query_options"`
	// 是否需要核销
	NeedCheckOut *bool `json:"need_check_out"`
	// sku商家编码，最大支持传入50个
	SkuCodes []string `json:"sku_codes"`
}
