package buyin_promotionStrategyList_request

import (
	"doudian.com/open/sdk_golang/api/buyin_promotionStrategyList/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinPromotionStrategyListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinPromotionStrategyListParam 
}
func (c *BuyinPromotionStrategyListRequest) GetUrlPath() string{
	return "/buyin/promotionStrategyList"
}


func New() *BuyinPromotionStrategyListRequest{
	request := &BuyinPromotionStrategyListRequest{
		Param: &BuyinPromotionStrategyListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinPromotionStrategyListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_promotionStrategyList_response.BuyinPromotionStrategyListResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_promotionStrategyList_response.BuyinPromotionStrategyListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinPromotionStrategyListRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinPromotionStrategyListRequest) GetParams() *BuyinPromotionStrategyListParam{
	return c.Param
}


type CosRangeFilter struct {
	// 0：区间；1：大于上限；2：小于下限；3：不限
	Type int64 `json:"type"`
	// 20%传20
	Low int64 `json:"low"`
	// 20%传20
	Up int64 `json:"up"`
}
type BuyinPromotionStrategyListParam struct {
	// 商品ID
	ProductId *string `json:"product_id"`
	// 根据buyin_id搜索
	KolBuyinIds []int64 `json:"kol_buyin_ids"`
	// 状态 0：全部 1：生效中 2：已结束 3：已失效
	Status int64 `json:"status"`
	// 生效达人范围 1：全部达人 2：指定达人 3：条件圈选
	ValidKolFilter *int64 `json:"valid_kol_filter"`
	// 佣金规则 1：固定佣金 2：阶梯佣金
	CosType *int32 `json:"cos_type"`
	// 结束时间 1：1天内，2：3天内，3：7天内
	EndTimeFilter *int64 `json:"end_time_filter"`
	// 佣金率范围
	CosRangeFilter *CosRangeFilter `json:"cos_range_filter"`
	// 策略id
	StrategyId []string `json:"strategy_id"`
	// 分页
	Page int64 `json:"page"`
	// 分页大小
	PageSize int64 `json:"page_size"`
}
