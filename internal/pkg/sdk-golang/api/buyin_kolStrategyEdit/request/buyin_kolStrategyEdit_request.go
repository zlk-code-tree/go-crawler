package buyin_kolStrategyEdit_request

import (
	"doudian.com/open/sdk_golang/api/buyin_kolStrategyEdit/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinKolStrategyEditRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinKolStrategyEditParam 
}
func (c *BuyinKolStrategyEditRequest) GetUrlPath() string{
	return "/buyin/kolStrategyEdit"
}


func New() *BuyinKolStrategyEditRequest{
	request := &BuyinKolStrategyEditRequest{
		Param: &BuyinKolStrategyEditParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinKolStrategyEditRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_kolStrategyEdit_response.BuyinKolStrategyEditResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_kolStrategyEdit_response.BuyinKolStrategyEditResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinKolStrategyEditRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinKolStrategyEditRequest) GetParams() *BuyinKolStrategyEditParam{
	return c.Param
}


type BuyinKolStrategyEditParam struct {
	// 1:修改 2关闭 3 策略恢复 4:删除
	Status int64 `json:"status"`
	// 策略ID
	StrategyId string `json:"strategy_id"`
	// 需要新增的达人百应账号
	AddBuyinIds []int64 `json:"add_buyin_ids"`
	// 需要删除的达人百应账号
	DelBuyinIds []int64 `json:"del_buyin_ids"`
	// 结束时间 2020-05-19 00:00:00
	EndTime *string `json:"end_time"`
	// 是否长期有效，和end_time互斥；全部达人佣金策略不允许指定该字段
	LongTerm *bool `json:"long_term"`
	// 佣金率
	CosRatio *int64 `json:"cos_ratio"`
}
