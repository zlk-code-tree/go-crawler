package sku_syncStock_request

import (
	"doudian.com/open/sdk_golang/api/sku_syncStock/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SkuSyncStockRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SkuSyncStockParam 
}
func (c *SkuSyncStockRequest) GetUrlPath() string{
	return "/sku/syncStock"
}


func New() *SkuSyncStockRequest{
	request := &SkuSyncStockRequest{
		Param: &SkuSyncStockParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SkuSyncStockRequest) Execute(accessToken *doudian_sdk.AccessToken) (*sku_syncStock_response.SkuSyncStockResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &sku_syncStock_response.SkuSyncStockResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SkuSyncStockRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SkuSyncStockRequest) GetParams() *SkuSyncStockParam{
	return c.Param
}


type MultiTimeStocksItem struct {
	// 库存数
	StockNum *int64 `json:"stock_num"`
	// 时效描述
	TimeDesc *string `json:"time_desc"`
	// 时效
	TimeType *int64 `json:"time_type"`
}
type BatchStockItemsItem struct {
	// 批次库存类型：1为唯一码
	BatchType *int64 `json:"batch_type"`
	// 批次库存数量
	StockNum *int64 `json:"stock_num"`
	// 唯一码
	UniqueId *string `json:"unique_id"`
}
type SkuSyncStockParam struct {
	// sku编码
	Code *string `json:"code"`
	// 规格id；抖店系统生成，商品id下唯一。
	SkuId *int64 `json:"sku_id"`
	// 外部skuid，开发者在创建或更新商品时设置。
	OutSkuId *int64 `json:"out_sku_id"`
	// 商品ID，抖店系统生成，店铺下唯一；长度19位。
	ProductId *int64 `json:"product_id"`
	// 外部商品id；开发者在创建或更新商品时设置的。
	OutProductId *int64 `json:"out_product_id"`
	// 外部仓库id编码，开发者自定义；如传入传值表示更新区域库存，不传默认更新普通库存。可以使用【/warehouse/list】接口响应参数【out_warehouse_id】字段获取。超市小时达店铺类型，需传入store_id字段，store_id可以使用【/shop/batchCreateStore】创建成功后获取
	OutWarehouseId *string `json:"out_warehouse_id"`
	// 供应商ID
	SupplierId *string `json:"supplier_id"`
	// 库存更新方式；true-增量更新，同时idempotent_id字段必填；false-全量更新；默认为false
	Incremental *bool `json:"incremental"`
	// 幂等ID，当incremental=true时该参数必传
	IdempotentId *string `json:"idempotent_id"`
	// 库存值；可以设置为0，不能小于0，当incremental=true时stock_num支持加/减库存。支持传入“-”，例如：-5表示减5个库存；加库存例：原商品库存是10个，接口传入5个，执行成功商品库存是15个。减库存例：原商品库存是10个，接口stock_num传入-5，执行成功商品库存是5个。注意：减库存时【商品现库存-即将减少库存数不能<0】，否则会报错系统错误:更新库存数失败: 库存不足；
	StockNum int64 `json:"stock_num"`
	// 阶梯库存
	StepStockNum *int64 `json:"step_stock_num"`
	// 门店id①即时零售连锁版，通过传入主品id+门店id对该门店下的子品进行操作，即时零售单店版，无需使用②抖店服饰自提场景，通过传入主品id+门店id更新门店商品库存
	StoreId *int64 `json:"store_id"`
	// 时效库存
	MultiTimeStocks []MultiTimeStocksItem `json:"multi_time_stocks"`
	// 批次库存项
	BatchStockItems []BatchStockItemsItem `json:"batch_stock_items"`
}
