package sku_syncStockBatch_request

import (
	"doudian.com/open/sdk_golang/api/sku_syncStockBatch/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SkuSyncStockBatchRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SkuSyncStockBatchParam 
}
func (c *SkuSyncStockBatchRequest) GetUrlPath() string{
	return "/sku/syncStockBatch"
}


func New() *SkuSyncStockBatchRequest{
	request := &SkuSyncStockBatchRequest{
		Param: &SkuSyncStockBatchParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SkuSyncStockBatchRequest) Execute(accessToken *doudian_sdk.AccessToken) (*sku_syncStockBatch_response.SkuSyncStockBatchResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &sku_syncStockBatch_response.SkuSyncStockBatchResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SkuSyncStockBatchRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SkuSyncStockBatchRequest) GetParams() *SkuSyncStockBatchParam{
	return c.Param
}


type SkuSyncStockBatchParam struct {
	// 商品ID，整型格式
	ProductId int64 `json:"product_id"`
	// 幂等ID，仅incremental=true时有用
	IdempotentId *string `json:"idempotent_id"`
	// true表示增量库存，false表示全量库存，默认为false
	Incremental bool `json:"incremental"`
	// 需要批量提交的skuid及对应的库存和仓；接口规则全部成功或全部失败，例：批量更新10个skuid库存，其中一个skuid信息不正确，这样整个请求都会失败，10个skuid都未更新成功。
	SkuSyncList []SkuSyncListItem `json:"sku_sync_list"`
	// 推荐使用，外部商家编码，支持字符串。
	OuterProductId *string `json:"outer_product_id"`
	// 即时零售连锁版，通过传入主品id+门店id对该门店下的子品进行操作，即时零售单店版，无需使用
	StoreId *int64 `json:"store_id"`
}
type BatchStockItemsItem struct {
	// 批次库存类型：1为唯一码
	BatchType *int64 `json:"batch_type"`
	// 批次库存数量
	StockNum *int64 `json:"stock_num"`
	// 唯一码
	UniqueId *string `json:"unique_id"`
}
type MultiTimeStocksItem struct {
	// 库存数
	StockNum *int64 `json:"stock_num"`
	// 时效描述
	TimeDesc *string `json:"time_desc"`
	// 时效
	TimeType *int64 `json:"time_type"`
}
type StockMapItem struct {
	// 批次库存项
	BatchStockItems []BatchStockItemsItem `json:"batch_stock_items"`
	// 仓id或到店自提门店id，场景①仓id，超市小时达子品传storeId。到店自提传入store_id（门店id）
	OutWarehouseId string `json:"out_warehouse_id"`
	// 库存数量
	StockNum int64 `json:"stock_num"`
}
type SkuSyncListItem struct {
	// 批次库存项
	BatchStockItems []BatchStockItemsItem `json:"batch_stock_items"`
	// 时效库存
	MultiTimeStocks []MultiTimeStocksItem `json:"multi_time_stocks"`
	// 外部sku编码，需要结合outer_product_id进行使用
	OuterSkuId *string `json:"outer_sku_id"`
	// skuid
	SkuId int64 `json:"sku_id"`
	// 库存类型：0普通库存，1区域库存,10阶梯库存。 超市小时达子品传区域库存
	SkuType int64 `json:"sku_type"`
	// 普通库存数量
	StockNum *int64 `json:"stock_num"`
	// 阶梯库存数量
	StepStockNum *int64 `json:"step_stock_num"`
	// 区域库存配置
	StockMap []StockMapItem `json:"stock_map"`
	// 供应商ID
	SupplierId *string `json:"supplier_id"`
}
