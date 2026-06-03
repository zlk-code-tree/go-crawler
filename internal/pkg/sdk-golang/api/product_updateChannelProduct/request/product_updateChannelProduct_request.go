package product_updateChannelProduct_request

import (
	"doudian.com/open/sdk_golang/api/product_updateChannelProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductUpdateChannelProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductUpdateChannelProductParam 
}
func (c *ProductUpdateChannelProductRequest) GetUrlPath() string{
	return "/product/updateChannelProduct"
}


func New() *ProductUpdateChannelProductRequest{
	request := &ProductUpdateChannelProductRequest{
		Param: &ProductUpdateChannelProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductUpdateChannelProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_updateChannelProduct_response.ProductUpdateChannelProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_updateChannelProduct_response.ProductUpdateChannelProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductUpdateChannelProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductUpdateChannelProductRequest) GetParams() *ProductUpdateChannelProductParam{
	return c.Param
}


type ProductUpdateChannelProductParam struct {
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型。不支持修改。3：商家自卖4：达人带货
	ChannelType int64 `json:"channel_type"`
	// 渠道品定制标题，必填
	Name *string `json:"name"`
	// 渠道品定制主图，最高支持5张。detail_brief_follow_main值为false时，必须设置商详图
	MainImgList []string `json:"main_img_list"`
	// 有效期开始时间
	StartTime *string `json:"start_time"`
	// 有效期结束时间
	EndTime *string `json:"end_time"`
	// 渠道品SKU价格是否跟随主品
	PriceFollowMain bool `json:"price_follow_main"`
	// 渠道品SKU库存是否与主商品共享库存
	ShareMainStock bool `json:"share_main_stock"`
	// 是否『达人专属』渠道品，不支持修改
	IsKolExclusive bool `json:"is_kol_exclusive"`
	// 渠道品限购信息
	ChannelBuyLimit *ChannelBuyLimit `json:"channel_buy_limit"`
	// 渠道SKU相关信息
	SkuChannel []SkuChannelItem `json:"sku_channel"`
	// 渠道品定制详情图，最高支持50张。可省略，默认使用主商品详情图列表
	DetailBriefImg []string `json:"detail_brief_img"`
	// 运费模板
	FreightTemplate *FreightTemplate `json:"freight_template"`
	// 商家编码与主品保持一致配置，true-商家编码与主品保持一致，false-商家编码自定义
	SkuCodeFollowMain *bool `json:"sku_code_follow_main"`
	// 商详与主品保持一致配置，true：保持一致；false：单独配置（此时必须传入商详图片，否则渠道品在c端将没有商详图）
	DetailBriefFollowMain *bool `json:"detail_brief_follow_main"`
	// 标题前缀
	NamePrefix *string `json:"name_prefix"`
	// 标题内容。注意：后续推荐使用name_prefix和name_subject设置标题，不建议使用name字段。并且name字段和name_prefix\name_subject2套传参互斥
	NameSubject *string `json:"name_subject"`
}
type ChannelBuyLimit struct {
	// 单用户累计购买上限数
	TotalBuyNum *int64 `json:"total_buy_num"`
	// 每次购买上限数
	MaxBuyNum *int64 `json:"max_buy_num"`
	// 每次至少购买数
	MinBuyNum *int64 `json:"min_buy_num"`
}
type SkuChannelItem struct {
	// 库存设置方式。0=设置绝对值，1=增量设置
	StockSetMode *int32 `json:"stock_set_mode"`
	// 渠道品定制SKU商家编码。可忽略，默认使用主商品SKU商家编码。
	Code *string `json:"code"`
	// 主商品SkuID
	SkuId int64 `json:"sku_id"`
	// 价格。设置价格跟随主品时，可不填
	Price *int64 `json:"price"`
	// 库存，勿大于主SKU的可售库存。设置库存与主商品共享时，可不填
	StockNum *int64 `json:"stock_num"`
}
type FreightTemplate struct {
	// 渠道品运费模板是否与主品保持一致
	FreightFollowMain *bool `json:"freight_follow_main"`
	// 渠道品定制运费模板ID，当设置了freight_follow_main=true时与主品运费模板保持一致；否则需要传渠道品定制运费模板。
	FreightId *int64 `json:"freight_id"`
}
