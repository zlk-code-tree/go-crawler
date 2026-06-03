package product_isv_saveGoodsSupplyStatus_request

import (
	"doudian.com/open/sdk_golang/api/product_isv_saveGoodsSupplyStatus/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductIsvSaveGoodsSupplyStatusRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductIsvSaveGoodsSupplyStatusParam 
}
func (c *ProductIsvSaveGoodsSupplyStatusRequest) GetUrlPath() string{
	return "/product/isv/saveGoodsSupplyStatus"
}


func New() *ProductIsvSaveGoodsSupplyStatusRequest{
	request := &ProductIsvSaveGoodsSupplyStatusRequest{
		Param: &ProductIsvSaveGoodsSupplyStatusParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductIsvSaveGoodsSupplyStatusRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_isv_saveGoodsSupplyStatus_response.ProductIsvSaveGoodsSupplyStatusResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_isv_saveGoodsSupplyStatus_response.ProductIsvSaveGoodsSupplyStatusResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductIsvSaveGoodsSupplyStatusRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductIsvSaveGoodsSupplyStatusRequest) GetParams() *ProductIsvSaveGoodsSupplyStatusParam{
	return c.Param
}


type StatusItem struct {
	// 线索id
	ClueId int64 `json:"clue_id"`
	// 是否存在货源
	GoodsSupplyExists bool `json:"goods_supply_exists"`
}
type ProductIsvSaveGoodsSupplyStatusParam struct {
	// 列表数量不能超过200
	Status []StatusItem `json:"status"`
	// 平台
	Platform int32 `json:"platform"`
}
