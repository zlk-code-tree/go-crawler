package address_listAftersaleStrategy_request

import (
	"doudian.com/open/sdk_golang/api/address_listAftersaleStrategy/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AddressListAftersaleStrategyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AddressListAftersaleStrategyParam 
}
func (c *AddressListAftersaleStrategyRequest) GetUrlPath() string{
	return "/address/listAftersaleStrategy"
}


func New() *AddressListAftersaleStrategyRequest{
	request := &AddressListAftersaleStrategyRequest{
		Param: &AddressListAftersaleStrategyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AddressListAftersaleStrategyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*address_listAftersaleStrategy_response.AddressListAftersaleStrategyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &address_listAftersaleStrategy_response.AddressListAftersaleStrategyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AddressListAftersaleStrategyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AddressListAftersaleStrategyRequest) GetParams() *AddressListAftersaleStrategyParam{
	return c.Param
}


type AddressListAftersaleStrategyParam struct {
	// 排序的字段;create_time-创建时间; update_time-更新时间;
	OrderByField *string `json:"order_by_field"`
	// 每页的条数, 每页最大不超过20个。当前单个商家只允许创建1000条策略，page * size > 1000会被拦截。
	Size *int64 `json:"size"`
	// 升序还是排序；asc-升序；desc-降序;
	OrderBy *string `json:"order_by"`
	// 页码， 从1开始,表示第1页;
	Page *int64 `json:"page"`
}
