package warehouse_setAddrBatch_request

import (
	"doudian.com/open/sdk_golang/api/warehouse_setAddrBatch/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type WarehouseSetAddrBatchRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *WarehouseSetAddrBatchParam 
}
func (c *WarehouseSetAddrBatchRequest) GetUrlPath() string{
	return "/warehouse/setAddrBatch"
}


func New() *WarehouseSetAddrBatchRequest{
	request := &WarehouseSetAddrBatchRequest{
		Param: &WarehouseSetAddrBatchParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *WarehouseSetAddrBatchRequest) Execute(accessToken *doudian_sdk.AccessToken) (*warehouse_setAddrBatch_response.WarehouseSetAddrBatchResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &warehouse_setAddrBatch_response.WarehouseSetAddrBatchResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *WarehouseSetAddrBatchRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *WarehouseSetAddrBatchRequest) GetParams() *WarehouseSetAddrBatchParam{
	return c.Param
}


type AddrListItem struct {
	// 一级地址id
	AddrId1 int64 `json:"addr_id1"`
	// 二级地址id
	AddrId2 *int64 `json:"addr_id2"`
	// 三级地址id
	AddrId3 *int64 `json:"addr_id3"`
	// 四级地址id
	AddrId4 *int64 `json:"addr_id4"`
}
type WarehouseSetAddrBatchParam struct {
	// 外部仓库ID
	OutWarehouseId string `json:"out_warehouse_id"`
	// 仓库配送地址列表
	AddrList []AddrListItem `json:"addr_list"`
}
