package sku_syncStockBatchMultiProducts_request

import (
	"doudian.com/open/sdk_golang/api/sku_syncStockBatchMultiProducts/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type SkuSyncStockBatchMultiProductsRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *SkuSyncStockBatchMultiProductsParam 
}
func (c *SkuSyncStockBatchMultiProductsRequest) GetUrlPath() string{
	return "/sku/syncStockBatchMultiProducts"
}


func New() *SkuSyncStockBatchMultiProductsRequest{
	request := &SkuSyncStockBatchMultiProductsRequest{
		Param: &SkuSyncStockBatchMultiProductsParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *SkuSyncStockBatchMultiProductsRequest) Execute(accessToken *doudian_sdk.AccessToken) (*sku_syncStockBatchMultiProducts_response.SkuSyncStockBatchMultiProductsResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &sku_syncStockBatchMultiProducts_response.SkuSyncStockBatchMultiProductsResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *SkuSyncStockBatchMultiProductsRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *SkuSyncStockBatchMultiProductsRequest) GetParams() *SkuSyncStockBatchMultiProductsParam{
	return c.Param
}


type ItemsItem struct {
	// 相对于 store_id 是平台侧的门店 id，store_code 是外部门店编码，使用上，选择其中一个，优先使用 store_id
	StoreCode *string `json:"store_code"`
	// sku 商家编码，开发者在创建或更新商品时设置的，商家自行要保证一个商品下的所有 sku 编码唯一，否则通过 sku_code+商品 id 无法准确定位唯一的sku，会导致更新库存至非预期内的sku 上
	SkuCode *string `json:"sku_code"`
	// 外部商品id；开发者在创建或更新商品时设置的，是out_product_id的字符串形式
	OuterProductId *string `json:"outer_product_id"`
	// 即时零售小时达店铺更新配送门店商品库存，抖店自提店铺用于设置商品自提门店库存
	StoreId *int64 `json:"store_id"`
	// 外部skuid，开发者在创建或更新商品时设置。
	OutSkuId *int64 `json:"out_sku_id"`
	// 外部商品id；开发者在创建或更新商品时设置的。
	OutProductId *int64 `json:"out_product_id"`
	// 商品ID
	ProductId *int64 `json:"product_id"`
	// 商品规格ID；一次请求最大支持50个sku更新
	SkuId *int64 `json:"sku_id"`
	// 外部仓库id编码，开发者自定义；如传入传值表示更新区域库存，不传默认更新普通库存。可以使用【/warehouse/list】接口响应参数【out_warehouse_id】字段获取。超市小时达店铺类型，需传入store_id字段，store_id可以使用【/shop/batchCreateStore】创建成功后获取
	OutWarehouseId *string `json:"out_warehouse_id"`
	// 现货库存数
	StockNum *int64 `json:"stock_num"`
	// 阶梯库存数
	StepStockNum *int64 `json:"step_stock_num"`
	// 批量请求中item的唯一标识
	UniqId *string `json:"uniq_id"`
}
type SkuSyncStockBatchMultiProductsParam struct {
	// 请求列表
	Items []ItemsItem `json:"items"`
	// 幂等ID，平台会校验校验成功的idempotent_id值，请保证24小时内在店铺下唯一性，建议使用年月日时分秒+随机数生成。重复的idempotent_id会触发“Token已被使用”报错
	IdempotentId string `json:"idempotent_id"`
	// 库存更新方式，必填字段；true-增量更新，同时stock_num和step_stock_num字段可以为正数或负数；false-全量更新，同时stock_num和step_stock_num字段只能为正数数
	Incremental bool `json:"incremental"`
	// 请求来源，开发者自定义；
	Source *string `json:"source"`
	// 是否允许部分成功失败，默认接口级别成功失败
	AllowPartialSuccess *bool `json:"allow_partial_success"`
	// 是否允许负库存
	InventoryLoss *bool `json:"inventory_loss"`
}
