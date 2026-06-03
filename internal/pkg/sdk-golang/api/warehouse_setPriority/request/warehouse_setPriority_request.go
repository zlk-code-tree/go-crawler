package warehouse_setPriority_request

import (
	"doudian.com/open/sdk_golang/api/warehouse_setPriority/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type WarehouseSetPriorityRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *WarehouseSetPriorityParam 
}
func (c *WarehouseSetPriorityRequest) GetUrlPath() string{
	return "/warehouse/setPriority"
}


func New() *WarehouseSetPriorityRequest{
	request := &WarehouseSetPriorityRequest{
		Param: &WarehouseSetPriorityParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *WarehouseSetPriorityRequest) Execute(accessToken *doudian_sdk.AccessToken) (*warehouse_setPriority_response.WarehouseSetPriorityResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &warehouse_setPriority_response.WarehouseSetPriorityResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *WarehouseSetPriorityRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *WarehouseSetPriorityRequest) GetParams() *WarehouseSetPriorityParam{
	return c.Param
}


type Addr struct {
	// 一级地址id
	AddrId1 int64 `json:"addr_id1"`
	// 二级地址id
	AddrId2 *int64 `json:"addr_id2"`
	// 三级地址id
	AddrId3 *int64 `json:"addr_id3"`
	// 四级地址id
	AddrId4 *int64 `json:"addr_id4"`
}
type WarehouseSetPriorityParam struct {
	// 配送地址
	Addr *Addr `json:"addr"`
	// 前面是out_warehouse_id，后面是仓优先级 仓优先级，0-5，0优先级最高，5最低
	Priorities map[string]int64 `json:"priorities"`
}
