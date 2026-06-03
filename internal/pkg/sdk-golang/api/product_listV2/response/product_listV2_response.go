package product_listV2_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductListV2Response struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductListV2Data `json:"data"`
}
type OriginalCategoryDetail struct {
	// 一级类目id
	FirstCid int64 `json:"first_cid"`
	// 二级类目id
	SecondCid int64 `json:"second_cid"`
	// 三级类目id
	ThirdCid int64 `json:"third_cid"`
	// 四级类目id
	FourthCid int64 `json:"fourth_cid"`
	// 一级类目名
	FirstCname string `json:"first_cname"`
	// 二级类目名
	SecondCname string `json:"second_cname"`
	// 三级类目名
	ThirdCname string `json:"third_cname"`
	// 四级类目名
	FourthCname string `json:"fourth_cname"`
}
type FutureCategoryDetail struct {
	// 一级类目id
	FirstCid int64 `json:"first_cid"`
	// 二级类目id
	SecondCid int64 `json:"second_cid"`
	// 三级类目id
	ThirdCid int64 `json:"third_cid"`
	// 四级类目id
	FourthCid int64 `json:"fourth_cid"`
	// 一级类目名
	FirstCname string `json:"first_cname"`
	// 二级类目名
	SecondCname string `json:"second_cname"`
	// 三级类目名
	ThirdCname string `json:"third_cname"`
	// 四级类目名
	FourthCname string `json:"fourth_cname"`
}
type RectificationInfo struct {
	// 整改图片明细
	PicList []PicListItem `json:"pic_list"`
	// 整改sku明细
	SkuList []SkuListItem `json:"sku_list"`
}
type SpecPricesItem struct {
	// skuId
	Id int64 `json:"id"`
	// skuCode
	Code string `json:"code"`
	// SKU Barcode
	Barcodes []string `json:"barcodes"`
}
type ProductListV2Data struct {
	// 商品数据
	Data []DataItem `json:"data"`
	// 本次查询返回的商品总数
	Total int64 `json:"total"`
	// 当前页码
	Page int64 `json:"page"`
	// 页数（每页数量）
	Size int64 `json:"size"`
	// 使用游标时，每次查询会返回该值，作为下次查询的入参使用。
	CursorId string `json:"cursor_id"`
}
type RecommendCategory struct {
	// 叶子类目id
	LeafCid int64 `json:"leaf_cid"`
	// 类目是否允许发品
	Enable bool `json:"enable"`
	// 是否叶子类目
	IsLeaf bool `json:"is_leaf"`
	// 四级类目名称
	FourthCname string `json:"fourth_cname"`
	// 三级类目名称
	ThirdCname string `json:"third_cname"`
	// 二级类目名称
	SecondCname string `json:"second_cname"`
	// 一级类目名称
	FirstCname string `json:"first_cname"`
	// 四级类目id
	FourthCid int64 `json:"fourth_cid"`
	// 三级类目id
	ThirdCid int64 `json:"third_cid"`
	// 二级类目id
	SecondCid int64 `json:"second_cid"`
	// 一级类目id
	FirstCid int64 `json:"first_cid"`
}
type Pic34Suggestion struct {
	// 修改后图片，需要重新走素材中心转图
	AfterPics []string `json:"after_pics"`
}
type PicSuggestion struct {
	// 修改后图片，需要重新走素材中心转图
	AfterPics []string `json:"after_pics"`
}
type AuditRejectSuggestion struct {
	// 34主图建议
	Pic34Suggestion *Pic34Suggestion `json:"pic34_suggestion"`
	// 1:1主图建议
	PicSuggestion *PicSuggestion `json:"pic_suggestion"`
	// 标题建议
	NameSuggestion *NameSuggestion `json:"name_suggestion"`
	// 驳回原因
	RejectReason string `json:"reject_reason"`
}
type SkuListItem struct {
	// 规格图url
	Url string `json:"url"`
	// 规格名称
	Title string `json:"title"`
	// skuid
	SkuId int64 `json:"sku_id"`
}
type ShopCategory struct {
	// 店铺装修分类id
	LeafCategoryIds []int64 `json:"leaf_category_ids"`
}
type PropertyKey struct {
	// 属性id
	PropertyId int64 `json:"property_id"`
	// 属性名称
	PropertyName string `json:"property_name"`
}
type ProductPropertyItem struct {
	// 属性key
	PropertyKey *PropertyKey `json:"property_key"`
	// 属性值列表
	PropertyValue []PropertyValueItem `json:"property_value"`
}
type PicListItem struct {
	// 图片下标
	Index int64 `json:"index"`
	// 图片url
	Url string `json:"url"`
	// 图片类型
	PicType int32 `json:"pic_type"`
}
type RectifyInfoListItem struct {
	// 自动整改信息
	RectificationInfo *RectificationInfo `json:"rectification_info"`
	// 整改动作
	RectifyAction string `json:"rectify_action"`
	// 驳回原因
	RejectReason string `json:"reject_reason"`
}
type CategoryDetail struct {
	// 一级类目
	FirstCid int64 `json:"first_cid"`
	// 二级类目
	SecondCid int64 `json:"second_cid"`
	// 三级类目
	ThirdCid int64 `json:"third_cid"`
	// 四级类目
	FourthCid int64 `json:"fourth_cid"`
	// 一级类目名称
	FirstCname string `json:"first_cname"`
	// 二级类目名称
	SecondCname string `json:"second_cname"`
	// 三级类目名称
	ThirdCname string `json:"third_cname"`
	// 四级类目名称
	FourthCname string `json:"fourth_cname"`
}
type DataItem struct {
	// 门捷列夫类目详情
	OriginalCategoryDetail *OriginalCategoryDetail `json:"original_category_detail"`
	// 标准类目详情
	FutureCategoryDetail *FutureCategoryDetail `json:"future_category_detail"`
	// 类目错放整改状态: 1 待整改，2 超期未改
	CategoryMisplaceRectifyStatus int64 `json:"category_misplace_rectify_status"`
	// 推荐类目topN列表
	RecommendCategoryTopn []RecommendCategoryTopnItem `json:"recommend_category_topn"`
	// 类目错放整改时间，以秒记
	CategoryMisplaceAlertDeadline int64 `json:"category_misplace_alert_deadline"`
	// 推荐类目信息
	RecommendCategory *RecommendCategory `json:"recommend_category"`
	// 存在类目错放
	ExistCategoryMisplaced bool `json:"exist_category_misplaced"`
	// 商品类目属性信息
	ProductProperty []ProductPropertyItem `json:"product_property"`
	// 运费模版
	FreightId int64 `json:"freight_id"`
	// 是否二手数码(成色机)
	IsSecondHandDigital bool `json:"is_second_hand_digital"`
	// 是否需要核销
	NeedCheckOut bool `json:"need_check_out"`
	// 驳回修改建议
	AuditRejectSuggestion *AuditRejectSuggestion `json:"audit_reject_suggestion"`
	// 是否有驳回修改建议
	HaveAuditRejectSuggest bool `json:"have_audit_reject_suggest"`
	// 自动整改信息
	RectifyInfoList []RectifyInfoListItem `json:"rectify_info_list"`
	// sku信息
	SpecPrices []SpecPricesItem `json:"spec_prices"`
	// 标题后缀
	NameSuffix string `json:"name_suffix"`
	// 标题前缀
	NamePrefix string `json:"name_prefix"`
	// 标题长度限制，长度计算逻辑：汉字算两个字符，汉字以外的其他字符都算一个字符。
	TitleLimit *TitleLimit `json:"title_limit"`
	// 销量
	SellNum int64 `json:"sell_num"`
	// 不可搭配的原因，可搭配时为空
	CanNotCombineReason string `json:"can_not_combine_reason"`
	// 是否可搭配，只有入参传入can_combine为true才有值
	CanCombine bool `json:"can_combine"`
	// 店铺装修分类
	ShopCategory *ShopCategory `json:"shop_category"`
	// 商品ID，抖店系统生成，店铺下唯一
	ProductId int64 `json:"product_id"`
	// 商品在店铺中状态: 0-在线；1-下线；2-删除；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	Status int64 `json:"status"`
	// 商品审核状态: 1-未提交；2-待审核；3-审核通过；4-审核未通过；5-封禁；7-审核通过待上架；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	CheckStatus int64 `json:"check_status"`
	// 【已废弃】划线价，单位分
	MarketPrice int64 `json:"market_price"`
	// 【已废弃】售价，单位分
	DiscountPrice int64 `json:"discount_price"`
	// 商品图片url，返回商品主图的第一张图 (主要用来辅助页面展示)
	Img string `json:"img"`
	// 商品标题。 标题不规范会引起商品下架，影响您的正常销售，详见商品发布规范：https://school.jinritemai.com/doudian/web/article/101800?from=shop_article
	Name string `json:"name"`
	// 【已废弃】支持的支付方式：0货到付款 1在线支付 2两者都支持
	PayType int64 `json:"pay_type"`
	// 商品类型；0-普通；1-新客商品；3-虚拟；6-玉石闪购；7-云闪购 ；127-其他类型；
	ProductType int64 `json:"product_type"`
	// 商品规格，新增商品是全局唯一，注意：有部分历史存量商品可能存在规格复用
	SpecId int64 `json:"spec_id"`
	// 【已废弃】佣金比例
	CosRatio int64 `json:"cos_ratio"`
	// 商品创建时间，unix时间戳，单位：秒；
	CreateTime int64 `json:"create_time"`
	// 商品更新时间，unix时间戳，单位：秒；
	UpdateTime int64 `json:"update_time"`
	// 【即将废弃】外部商家编码，商家自定义字段。推荐使用，outer_product_id字段
	OutProductId int64 `json:"out_product_id"`
	// 【已废弃】该字段不返回信息。商品详情，最大支持50张图片，单张详情图宽高比不超2000*2000px，大小5M内，仅支持jpg/jpeg/png格式；注意：商品详情不规范，会引起商品下架，影响您的正常销售：https://school.jinritemai.com/doudian/web/article/101800?from=shop_article
	Description string `json:"description"`
	// 【已废弃】手机号
	Mobile string `json:"mobile"`
	// 额外信息，如资质
	Extra string `json:"extra"`
	// 【已废弃】商家推荐语
	RecommendRemark string `json:"recommend_remark"`
	// 类目详情；商品类目id可使用【/shop/getShopCategory】查询
	CategoryDetail *CategoryDetail `json:"category_detail"`
	// 外部商家编码，商家自定义字段，支持最多 255个字符
	OuterProductId string `json:"outer_product_id"`
	// 是否是组合商品，true-是，false-不是；
	IsPackageProduct bool `json:"is_package_product"`
	// 商品关联的组合主商品ID；当is_package_product=true，返回的是组套商品的product_id； 当is_package_product=false，返回当前商品的product_id；补充返回规则：参与的组套商品，下线会展示，目前组合商品支持下线的商品作为子品进行组套，删除会展示，但重新上架组套商品时，会被从列表中删除。
	PackageProductList []int64 `json:"package_product_list"`
	// 当is_package_product=true，返回的是组套商品的子品product_id； 当is_package_product=false，不返回；补充返回规则：参与的组套商品product_id，下线会展示，目前组合商品支持下线的商品作为子品进行组套，删除会展示，但重新上架组套商品时，会被从列表中删除。
	SubProductList []int64 `json:"sub_product_list"`
	// 小时达子品绑定的主品信息
	ChannelMainProduct *ChannelMainProduct `json:"channel_main_product"`
}
type RecommendCategoryTopnItem struct {
	// 叶子类目id
	LeafCid int64 `json:"leaf_cid"`
	// 类目是否允许发品
	Enable bool `json:"enable"`
}
type PropertyValueItem struct {
	// 属性值id
	ValueId int64 `json:"value_id"`
	// 属性值名称
	ValueName string `json:"value_name"`
}
type NameSuggestion struct {
	// 改后标题
	AfterName string `json:"after_name"`
}
type TitleLimit struct {
	// 商品标题最短长度
	MinLength int64 `json:"min_length"`
	// 商品标题最长长度
	MaxLength int64 `json:"max_length"`
}
type ChannelMainProduct struct {
	// 主品的商品id
	ProductId int64 `json:"product_id"`
	// 店铺id
	ShopId int64 `json:"shop_id"`
}
