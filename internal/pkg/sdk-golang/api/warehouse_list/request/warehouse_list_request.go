package warehouse_list_request

import (
	"doudian.com/open/sdk_golang/api/warehouse_list/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type WarehouseListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *WarehouseListParam 
}
func (c *WarehouseListRequest) GetUrlPath() string{
	return "/warehouse/list"
}


func New() *WarehouseListRequest{
	request := &WarehouseListRequest{
		Param: &WarehouseListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *WarehouseListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*warehouse_list_response.WarehouseListResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &warehouse_list_response.WarehouseListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *WarehouseListRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *WarehouseListRequest) GetParams() *WarehouseListParam{
	return c.Param
}


type Addr struct {
	// 一级地址
	AddrId1 int64 `json:"addr_id1"`
	// 二级地址
	AddrId2 *int64 `json:"addr_id2"`
	// 三级地址
	AddrId3 *int64 `json:"addr_id3"`
	// 四级地址
	AddrId4 *int64 `json:"addr_id4"`
}
type WarehouseListParam struct {
	// 外部仓库ID
	OutWarehouseId *string `json:"out_warehouse_id"`
	// 仓库名称
	WarehouseName *string `json:"warehouse_name"`
	// 仓库覆盖地址
	Addr *Addr `json:"addr"`
	// 外部仓库ID列表
	OutWarehouseIds []string `json:"out_warehouse_ids"`
	// 排序方式，可选create_time、update_time
	OrderBy *string `json:"order_by"`
	// 顺序，可选desc、asc，与order_by同时生效
	Rank *string `json:"rank"`
	// 页码，从0开始，最大到100
	Page int64 `json:"page"`
	// 每页数量，最大100，超过100会限制到100
	Size int64 `json:"size"`
}
