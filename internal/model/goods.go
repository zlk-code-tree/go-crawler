package model

// GoodsSaveReq 保存商品请求
type GoodsSaveReq struct {
	Id         int     `json:"id" dc:"商品ID，编辑时必填"`
	GoodsName  string  `json:"goods_name" v:"required" dc:"商品名称"`
	GoodsCode  string  `json:"goods_code" v:"required" dc:"商品编码"`
	BasePrice  float64 `json:"base_price" d:"0.00" dc:"商品底价"`
	Status     int     `json:"status" d:"1" dc:"状态：1正常 2下架 3删除"`
}
