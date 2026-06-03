package product_batchGetProductLocks_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductBatchGetProductLocksResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductBatchGetProductLocksData `json:"data"`
}
type SkuLockInfosItem struct {
	// sku id
	SkuId int64 `json:"sku_id"`
	// sku锁。1000: 单sku规格锁1001: 单sku价格锁1002: 单sku规格图锁1003: 单sku库存类型锁1004: 单sku库存增加锁1005: 单sku库存减少锁1006: 单sku库存区域仓锁
	SkuLocks []int64 `json:"sku_locks"`
}
type ProductLockInfosItem struct {
	// 商品锁4:  商品规格锁5:  运费模版锁7:  sku价格锁（商品下所有sku价格不能修改）8:  物流信息锁9:  发货方式和时间锁10: 发货方式锁11: 发货时间锁12: 库存类型锁14: 商品标题锁15: 商品描述锁16: 商品白底图锁17: 商品下架操作锁18: 商品库存增加锁（商品下所有sku都不能增加库存）19: 商品库存减少锁（商品下所有sku都不能减少库存）20: 类目锁21: 预售发货模式锁22: 商品主图锁23: 商品编辑操作锁24: 库存区域仓锁25: 商品删除操作锁26: 商品上架操作锁27: 特殊时间延迟发货锁28: 7天无理由权益锁29: 过敏包退权益锁
	ProductLocks []int64 `json:"product_locks"`
	// sku锁信息
	SkuLockInfos []SkuLockInfosItem `json:"sku_lock_infos"`
	// 商品ID
	ProductId int64 `json:"product_id"`
}
type ProductBatchGetProductLocksData struct {
	// 商品锁信息
	ProductLockInfos []ProductLockInfosItem `json:"product_lock_infos"`
}
