package warehouse_createV2_request

import (
	"doudian.com/open/sdk_golang/api/warehouse_createV2/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type WarehouseCreateV2Request struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *WarehouseCreateV2Param 
}
func (c *WarehouseCreateV2Request) GetUrlPath() string{
	return "/warehouse/createV2"
}


func New() *WarehouseCreateV2Request{
	request := &WarehouseCreateV2Request{
		Param: &WarehouseCreateV2Param{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *WarehouseCreateV2Request) Execute(accessToken *doudian_sdk.AccessToken) (*warehouse_createV2_response.WarehouseCreateV2Response, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &warehouse_createV2_response.WarehouseCreateV2Response{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *WarehouseCreateV2Request) GetParamObject() interface{}{
	return c.Param
}


func (c *WarehouseCreateV2Request) GetParams() *WarehouseCreateV2Param{
	return c.Param
}


type WarehouseLocation struct {
	// 地址1级编码
	AddressId1 int64 `json:"address_id1"`
	// 地址2级编码
	AddressId2 int64 `json:"address_id2"`
	// 地址3级编码
	AddressId3 int64 `json:"address_id3"`
	// 地址4级编码
	AddressId4 *int64 `json:"address_id4"`
}
type Warehouse struct {
	// 仓的地址编码
	WarehouseLocation *WarehouseLocation `json:"warehouse_location"`
	// 仓的地址详情
	AddressDetail string `json:"address_detail"`
}
type WarehouseCreateV2Param struct {
	// 仓相关信息
	Warehouse *Warehouse `json:"warehouse"`
	// 外部仓ID
	OutWarehouseId string `json:"out_warehouse_id"`
	// 仓名称
	Name string `json:"name"`
	// 仓介绍
	Intro string `json:"intro"`
}
