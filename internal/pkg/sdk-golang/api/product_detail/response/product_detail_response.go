package product_detail_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductDetailResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductDetailData `json:"data"`
}
type MultiTimeStocksItem struct {
	// 库存数
	StockNum int64 `json:"stock_num"`
	// 时效描述
	TimeDesc string `json:"time_desc"`
	// 时效
	TimeType int64 `json:"time_type"`
}
type CustomsReportInfo struct {
	// 海关代码
	HsCode string `json:"hs_code"`
	// 法定第一计量数量
	FirstMeasureQty float64 `json:"first_measure_qty"`
	// 法定第二计量数量
	SecondMeasureQty float64 `json:"second_measure_qty"`
	// 法定第一计量单位
	FirstMeasureUnit string `json:"first_measure_unit"`
	// 法定第二计量单位
	SecondMeasureUnit string `json:"second_measure_unit"`
	// 售卖单位
	Unit string `json:"unit"`
	// 品名
	ReportName string `json:"report_name"`
	// 品牌
	ReportBrandName string `json:"report_brand_name"`
	// 用途
	Usage string `json:"usage"`
	// 规格型号
	GModel string `json:"g_model"`
	// 条形码
	BarCode string `json:"bar_code"`
}
type TaxExemptionSkuInfo struct {
	// 是否套装（0 否 1 是）
	IsSuit int32 `json:"is_suit"`
	// 套装数量
	SuitNum int64 `json:"suit_num"`
	// 净含量
	Volume int64 `json:"volume"`
}
type DelayRule struct {
	// 生效维度0 无效 1 店铺（此时无法设置商品维度数据） 2 商品维度 99 平台默认（没有设置商品维度时的默认生效数据）
	RuleScope int64 `json:"rule_scope"`
	// 是否开启特殊日期延迟发货
	Enable bool `json:"enable"`
	// 1 时间点；2 时间范围
	ConfigType int32 `json:"config_type"`
	// 特殊日期延迟发货时间最晚发货时间，时间戳，单位秒
	ConfigValue int64 `json:"config_value"`
	// 特殊日期延迟发货时间下单开始时间，时间戳，单位秒
	StartTime int64 `json:"start_time"`
	// 特殊日期延迟发货时间下单结束时间，时间戳，单位秒
	EndTime int64 `json:"end_time"`
}
type CategoryPropertyPicsItem struct {
	// 属性图链接
	Urls []string `json:"urls"`
}
type QualityListItem struct {
	// 资质key
	QualityKey string `json:"quality_key"`
	// 资质名称
	QualityName string `json:"quality_name"`
	// 资质链接
	QualityAttachments []QualityAttachmentsItem `json:"quality_attachments"`
}
type PoiResource struct {
	// 卡券关联的门店库id
	PoiLibId int64 `json:"poi_lib_id"`
	// 领取须知
	Notification string `json:"notification"`
	// 客服电话
	ServiceNum string `json:"service_num"`
	// 有效期天数
	ValidDays int64 `json:"valid_days"`
	// 使用条件
	Condition string `json:"condition"`
	// 可使用次数
	TotalCanUseCount int32 `json:"total_can_use_count"`
	// 券码生成方式
	CodeType int64 `json:"code_type"`
	// 1 随时退+过期自动退，2 不支持退
	CouponReturnMethods []int64 `json:"coupon_return_methods"`
}
type CustomPropertyGroupsItem struct {
	// 属性详细信息
	PropertyDetail []PropertyDetailItem `json:"propertyDetail"`
	// 属性组名称
	Name string `json:"name"`
}
type SpecPricesItem struct {
	// SKU图URL
	SkuPictureUrl []string `json:"sku_picture_url"`
	// 规格型号编码
	SpecModelCode []string `json:"spec_model_code"`
	// 1 相对时间 ships in N days，2 固定日期 fixed date。注意仅在现货发货且开启sku时效时有值。
	ShipTimeType string `json:"ship_time_type"`
	// 发货时间，如果是相对时间，返回 N。如果是固定日期，返回格式为yyyy-MM-dd，例如 "2025-09-01"。注意仅在现货发货且开启sku时效时有值。
	ShipTime string `json:"ship_time"`
	// commit code
	CommitCode string `json:"commit_code"`
	// 条形码
	Barcodes []string `json:"barcodes"`
	// 时效库存
	MultiTimeStocks []MultiTimeStocksItem `json:"multi_time_stocks"`
	// sku状态 true上架 false下架
	SkuStatus bool `json:"sku_status"`
	// sku分类信息
	SkuClassificationType string `json:"sku_classification_type"`
	// sku对应的销售属性信息
	SellProperties []SellPropertiesItem `json:"sell_properties"`
	// 货品信息（国内）
	CargoInfo *CargoInfo `json:"cargo_info"`
	// Sku的黄金加工费，单位：分
	GoldProcessCharge int64 `json:"gold_process_charge"`
	// 规格id；抖店系统生成，商品id下唯一。
	SkuId int64 `json:"sku_id"`
	// 外部商家skui_id编码，商家自定义字段；推荐使用outer_sku_id字段
	OutSkuId int64 `json:"out_sku_id"`
	// 外部商家skui_id编码，商家自定义字段
	OuterSkuId string `json:"outer_sku_id"`
	// 规格id列表,多规格以”,“分隔；
	SpecDetailIds []int64 `json:"spec_detail_ids"`
	// 可售库存；当前现货可售库存；
	StockNum int64 `json:"stock_num"`
	// 商品价格；单位：分
	Price int64 `json:"price"`
	// 编码
	Code string `json:"code"`
	// 阶梯库存，规则详见名称解释：https://op.jinritemai.com/docs/guide-docs/202/170
	StepStockNum int64 `json:"step_stock_num"`
	// 活动库存，，规则详见名称解释：https://op.jinritemai.com/docs/guide-docs/202/170
	PromStockNum int64 `json:"prom_stock_num"`
	// 活动阶梯库存，，规则详见名称解释：https://op.jinritemai.com/docs/guide-docs/202/170
	PromStepStockNum int64 `json:"prom_step_stock_num"`
	// 【已废弃，无需使用】规格ID
	SpecDetailId1 int64 `json:"spec_detail_id1"`
	// 【已废弃，无需使用】规格ID
	SpecDetailId2 int64 `json:"spec_detail_id2"`
	// 【已废弃，无需使用】规格ID
	SpecDetailId3 int64 `json:"spec_detail_id3"`
	// sku类型；0-普通库存  1-区域库存  10-阶梯库存
	SkuType int64 `json:"sku_type"`
	// 供应商编码
	SupplierId string `json:"supplier_id"`
	// 活动现货库存
	PromotionStockNum int64 `json:"promotion_stock_num"`
	// 活动阶梯库存
	PromotionStepStockNum int64 `json:"promotion_step_stock_num"`
	// 海关申报要素（仅海淘商品返回信息）
	CustomsReportInfo *CustomsReportInfo `json:"customs_report_info"`
	// 现货订单锁定库存
	LockStockNum int64 `json:"lock_stock_num"`
	// 阶梯订单锁定库存
	LockStepStockNum int64 `json:"lock_step_stock_num"`
	// 仓ID->库存映射
	StockNumMap map[string]int64 `json:"stock_num_map"`
	// 海南免税sku信息（仅海淘商品返回信息）
	TaxExemptionSkuInfo *TaxExemptionSkuInfo `json:"tax_exemption_sku_info"`
	// 发货时间（全款预售模式时的发货时间/阶梯模式下阶梯发货时间），9999是当日发、1次日发、2 是48小时发、5、15、45天等。发货时间规则可使用【product/getProductUpdateRule】获取
	PresellDelay int64 `json:"presell_delay"`
	// SKU物流信息
	DeliveryInfos []DeliveryInfosItem `json:"delivery_infos"`
}
type ValuesItem struct {
	// 度量衡值
	Value string `json:"value"`
	// 度量衡单位名称
	UnitName string `json:"unit_name"`
	// 度量衡单位id
	UnitId int64 `json:"unit_id"`
	// 度量衡后缀
	Suffix string `json:"suffix"`
	// 度量衡前缀
	Prefix string `json:"prefix"`
	// 度量衡模块id
	ModuleId int64 `json:"module_id"`
}
type QualityInspectionInfo struct {
	// 1: 单库存模式，只允许售卖一个sku；2: 多库存模式，不限售卖次数
	Mode int32 `json:"mode"`
	// 质检证书编码
	CertificateCode string `json:"certificate_code"`
	// 机构编码
	Agency string `json:"agency"`
	// 是否勾选支持前置质检
	Supported bool `json:"supported"`
}
type ProductAuditInfo struct {
	// 素材审核状态，key=素材类型（0-主图视频、1-白底图、4-长图、17-测评视频），value=审核状态（0-初始化 1-审核中 2-审核通过 3-审核不通过）
	MaterialAuditStatus map[string]int64 `json:"material_audit_status"`
	// 素材审核驳回原因
	MaterialAuditReason []MaterialAuditReasonItem `json:"material_audit_reason"`
}
type DefaultProcessTime struct {
	// 处理时间，单位秒
	Value int64 `json:"value"`
}
type SpecPicsItem struct {
	// id
	SpecDetailId int64 `json:"spec_detail_id"`
	// 规格图片uri
	Pic string `json:"pic"`
}
type SellPropertiesItem struct {
	// 规格值id
	ValueSpecDetailId int64 `json:"value_spec_detail_id"`
	// 备注
	Remark string `json:"remark"`
	// 规格值名称
	ValueName string `json:"value_name"`
	// 销售属性值id，只有在规格由属性库下发时，这个才有值。 默认为0
	ValueId int64 `json:"value_id"`
	// 规格项名称
	PropertyName string `json:"property_name"`
	// 销售属性id，只有在规格由属性库下发时，这个字段才有值。 默认为0
	PerpertyId int64 `json:"perperty_id"`
}
type CargoInfo struct {
	// 货品类型，1-分销货品，2-云仓自流转货品，6-云仓非自流转货品
	SourceType int64 `json:"source_type"`
	// 货品id
	CargoId string `json:"cargo_id"`
}
type CpvPathItem struct {
	// 导航属性值id
	Cpvid int64 `json:"cpvid"`
	// 导航属性id
	Cpid int64 `json:"cpid"`
}
type LogisticsInfo struct {
	// 通关模式，1BBC 2BC 3邮关
	CustomsClearType int64 `json:"customs_clear_type"`
	// 原产国id
	OriginCountryId int64 `json:"origin_country_id"`
	// 货源国id
	SourceCountryId int64 `json:"source_country_id"`
	// 品牌所在地id
	BrandCountryId int64 `json:"brand_country_id"`
	// 税金承担方，0商家承担，1用户承担
	TaxPayer int64 `json:"tax_payer"`
	// 商品净重
	NetWeightQty float64 `json:"net_weight_qty"`
}
type MaterialAuditReasonItem struct {
	// 0-主图视频、1-白底图、4-长图、17-测评视频
	MaterialType string `json:"material_type"`
	// 素材类型对应的驳回原因
	Reasons []string `json:"reasons"`
}
type ShopCategory struct {
	// 店铺装修分类叶子类目id
	LeafCategoryIds string `json:"leaf_category_ids"`
}
type HomeInstallServiceInfo struct {
	// 门安装服务模版id
	HomeInstallServiceTemplateId int64 `json:"home_install_service_template_id"`
}
type AfterSaleServiceV2 struct {
	// 三包服务承诺配置信息
	ThreeGuarantees *ThreeGuarantees `json:"three_guarantees"`
	// 当前商品是否是大件商品 1-是
	IsLargeProduct int64 `json:"is_large_product"`
}
type DeliveryInfosItem struct {
	// 信息类型
	InfoType string `json:"info_type"`
	// 信息值
	InfoValue string `json:"info_value"`
	// 信息计量单位
	InfoUnit string `json:"info_unit"`
}
type MeasureInfo struct {
	// 度量衡模块
	Values []ValuesItem `json:"values"`
	// 度量衡模版id
	TemplateId int64 `json:"template_id"`
}
type ReferencePriceCertificate struct {
	// 凭证类型;1:"厂商建议零售价",2:"吊牌价",3:"定价",4:"官网零售售价"
	CertificateType string `json:"certificate_type"`
	// 凭证图片url
	CertificateUrls string `json:"certificate_urls"`
}
type ThreeGuarantees struct {
	// 售后天数，例如服务类型选择了寄修，并且该参数为180，表示售后180天内支持寄修服务
	Duration int64 `json:"duration"`
	// 服务类型 1-寄修 2-延保
	ServiceType int64 `json:"service_type"`
}
type CustomProperty struct {
	// 定制属性组信息
	CustomPropertyGroups []CustomPropertyGroupsItem `json:"customPropertyGroups"`
}
type OriginalCategoryDetail struct {
	// 一级类目id（临时字段）
	FirstCid int64 `json:"first_cid"`
	// 二级类目id（临时字段）
	SecondCid int64 `json:"second_cid"`
	// 三级类目id（临时字段）
	ThirdCid int64 `json:"third_cid"`
	// 四级类目id（临时字段）
	FourthCid int64 `json:"fourth_cid"`
	// 一级类目名（临时字段）
	FirstCname string `json:"first_cname"`
	// 二级类目名（临时字段）
	SecondCname string `json:"second_cname"`
	// 三级类目名（临时字段）
	ThirdCname string `json:"third_cname"`
	// 四级类目名（临时字段）
	FourthCname string `json:"fourth_cname"`
}
type FutureCategoryDetail struct {
	// 一级类目id（临时字段）
	FirstCid int64 `json:"first_cid"`
	// 二级类目id（临时字段）
	SecondCid int64 `json:"second_cid"`
	// 三级类目id（临时字段）
	ThirdCid int64 `json:"third_cid"`
	// 四级类目id（临时字段）
	FourthCid int64 `json:"fourth_cid"`
	// 一级类目名（临时字段）
	FirstCname string `json:"first_cname"`
	// 二级类目名（临时字段）
	SecondCname string `json:"second_cname"`
	// 三级类目名（临时字段）
	ThirdCname string `json:"third_cname"`
	// 四级类目名（临时字段）
	FourthCname string `json:"fourth_cname"`
}
type CategoryDetail struct {
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
type QualityAttachmentsItem struct {
	// 1-图片，2-pdf
	MediaType int64 `json:"media_type"`
	// 资质url
	Url string `json:"url"`
}
type UnitPriceInfo struct {
	// 1：投资金单位价格2：首饰金单位价格
	PriceRuleType int32 `json:"price_rule_type"`
}
type ValuesItem_3 struct {
	// 标准销售属性值id
	ValueId string `json:"value_id"`
	// 其他扩展信息：包括度量衡信息等
	Extra *Extra `json:"extra"`
	// 规格值id
	Id int64 `json:"id"`
	// 规格ID
	SpecId int64 `json:"spec_id"`
	// 销售属性值
	Name string `json:"name"`
	// 规格值父id
	Pid int64 `json:"pid"`
	// 是否是销售属性值
	IsLeaf int16 `json:"is_leaf"`
	// 在线状态，-2彻底删除、0在线、1下线、2删除
	Status int16 `json:"status"`
}
type Extra struct {
	// 度量衡结构化信息
	MeasureInfo *MeasureInfo `json:"measure_info"`
	// 导航路径信息
	CpvPath []CpvPathItem `json:"cpv_path"`
}
type SpecsItem struct {
	// 标准销售属性id
	PropertyId int64 `json:"property_id"`
	// 【无需使用】规格模版id
	Id int64 `json:"id"`
	// 【无需使用】规格id
	SpecId int64 `json:"spec_id"`
	// 销售属性
	Name string `json:"name"`
	// 【无需使用】规格父id
	Pid int64 `json:"pid"`
	// 是否是销售属性值
	IsLeaf int16 `json:"is_leaf"`
	// 销售属性值
	Values []ValuesItem_3 `json:"values"`
}
type PropertyDetailItem struct {
	// 属性扩展信息
	ExtraInfo *ExtraInfo `json:"extra_info"`
	// 商品属性名称
	Name string `json:"name"`
}
type ExtraInfo struct {
	// 处理时间，单位秒
	ProcessTime int64 `json:"process_time"`
}
type ProductDetailData struct {
	// 商品ID，抖店系统生成，店铺下唯一；长度19位。
	ProductId int64 `json:"product_id"`
	// 商品ID（字符串类型），抖店系统生成，店铺下唯一；长度19位。
	ProductIdStr string `json:"product_id_str"`
	// 【即将废弃】外部商家编码，商家自定义字段。推荐使用，outer_product_id字段
	OutProductId int64 `json:"out_product_id"`
	// 外部商家编码，商家自定义字段，支持最多 255个字符
	OuterProductId string `json:"outer_product_id"`
	// 【已废弃】open应用id
	OpenUserId int64 `json:"open_user_id"`
	// 商品标题，规则：至少输入8个字（16个字符）以上~输入30个字（60个字符）以内。；标题不规范会引起商品下架，影响您的正常销售，详见商品发布规范：https://school.jinritemai.com/doudian/web/article/101800?from=shop_article
	Name string `json:"name"`
	// 商品详情，最大支持50张图片，单张详情图宽高比不超2000*2000px，大小5M内，仅支持jpg/jpeg/png格式，返回HTML格式；注意：商品详情不规范，会引起商品下架，影响您的正常销售：https://school.jinritemai.com/doudian/web/article/101800?from=shop_article
	Description string `json:"description"`
	// 承诺发货时间，单位是天; 不传则默认为2天，当presell_type为0或2均只支持传入2；当presell_type为1时支持可选值为: 2、3、5、7、10、15；
	DeliveryMethod int32 `json:"delivery_method"`
	// 海南免税，海关限购分类编码，仅海淘商品有值返回。
	CdfCategory string `json:"cdf_category"`
	// 商品在店铺中状态: 0-在线；1-下线；2-删除；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	Status int64 `json:"status"`
	// 商品规格，新增商品是全局唯一，注意：有部分历史存量商品可能存在规格复用
	SpecId int64 `json:"spec_id"`
	// 商品审核状态: 1-未提交；2-待审核；3-审核通过；4-审核未通过；5-封禁；7-审核通过待上架；详见商品状态机：https://op.jinritemai.com/docs/question-docs/92/2070
	CheckStatus int64 `json:"check_status"`
	// 【已废弃】手机号
	Mobile string `json:"mobile"`
	// 【已废弃】品牌id，推荐使用standard_brand_id字段，通过【/brand/list】获取，无品牌id商品返回默认值：596120136；
	BrandId int64 `json:"brand_id"`
	// 是否是组合商品的子商品；true-是，false-不是；
	IsSubProduct bool `json:"is_sub_product"`
	// 草稿状态；0-无草稿,1-未提审,2-待审核,3-审核通过,4-审核未通过。详见：https://op.jinritemai.com/docs/question-docs/92/2070
	DraftStatus int64 `json:"draft_status"`
	// 类目详情；商品类目id可使用【/shop/getShopCategory】查询
	CategoryDetail *CategoryDetail `json:"category_detail"`
	// 【已废弃】支持的支付方式：0货到付款 1在线支付 2两者都支持
	PayType int64 `json:"pay_type"`
	// 【已废弃】商家推荐语
	RecommendRemark string `json:"recommend_remark"`
	// 额外信息，如资质
	Extra string `json:"extra"`
	// 【已废弃】无业务意义
	IsCreate int64 `json:"is_create"`
	// 商品创建时间，时间格式：yyyy-mm-dd hh:mm:ss
	CreateTime string `json:"create_time"`
	// 商品更新时间，时间格式：yyyy-mm-dd hh:mm:ss
	UpdateTime string `json:"update_time"`
	// 商品主图；最多支持5张图片；仅支持png，jpg，jpeg格式，宽高比例为1:1（至少600*600px），大小5M内
	Pic []string `json:"pic"`
	// 【即将废弃】推荐使用product_format_new；属性名称|属性值 之间用|分隔, 多组之间用^分开
	ProductFormat string `json:"product_format"`
	// 规格图片
	SpecPics []SpecPicsItem `json:"spec_pics"`
	// 商品sku详情
	SpecPrices []SpecPricesItem `json:"spec_prices"`
	// 规格信息，规格项和规格值
	Specs []SpecsItem `json:"specs"`
	// 头图，主图第一张
	Img string `json:"img"`
	// 预售类型，1-全款预售，0-非预售，2-阶梯库存
	PresellType int64 `json:"presell_type"`
	// 单用户下单限购件数
	MaximumPerOrder int64 `json:"maximum_per_order"`
	// 单用户累计限购件数
	LimitPerBuyer int64 `json:"limit_per_buyer"`
	// 用户每次下单至少购买的件数
	MinimumPerOrder int64 `json:"minimum_per_order"`
	// 资质信息
	QualityList []QualityListItem `json:"quality_list"`
	// 跨境物流信息（仅海淘商品返回）
	LogisticsInfo *LogisticsInfo `json:"logistics_info"`
	// 售后服务
	AfterSaleService string `json:"after_sale_service"`
	// 商品价格是否含税
	PriceHasTax int64 `json:"price_has_tax"`
	// 可预约发货天数
	AppointDeliveryDay int64 `json:"appoint_delivery_day"`
	// 类目属性
	ProductFormatNew string `json:"product_format_new"`
	// 品牌库brand id，原brand_id代表商标关系id
	StandardBrandId int64 `json:"standard_brand_id"`
	// 划线价 单位分
	MarketPrice int64 `json:"market_price"`
	// 售卖价 单位分
	DiscountPrice int64 `json:"discount_price"`
	// 汽车vin码
	CarVinCode string `json:"car_vin_code"`
	// 生活娱乐充值模式
	NeedRechargeMode bool `json:"need_recharge_mode"`
	// 多账号充值账号模板
	AccountTemplateId string `json:"account_template_id"`
	// 发货模式：presell_type = 0 现货；presell_type = 2 阶梯；presell_type = 1 && presell_config_level = 0 全款预售；presell_type = 1 && presell_config_level = 1 sku预售；presell_type = 1 && presell_config_level = 2 现货+预售；presell_type = 1 && presell_config_level = 3 新预售
	PresellConfigLevel int64 `json:"presell_config_level"`
	// 现货模式的发货天数；阶梯模式现货部分的发货天数，9999=当日发、1=次日发
	DeliveryDelayDay int64 `json:"delivery_delay_day"`
	// 阶梯模式阶梯部分的发货天数；商品全款预售模式，预售天数
	PresellDelay int64 `json:"presell_delay"`
	// 卡券信息
	PoiResource *PoiResource `json:"poi_resource"`
	// 特殊日期延迟发货规则
	DelayRule *DelayRule `json:"delay_rule"`
	// 3:4长图url(仅素材中心url有效)
	LongPicUrl string `json:"long_pic_url"`
	// 售卖方式;0:全渠道手售卖,1:仅指定直播间售卖
	SellChannel []int64 `json:"sell_channel"`
	// 运费模版ID
	FreightId int64 `json:"freight_id"`
	// 主图视频ID
	MaterialVideoId string `json:"material_video_id"`
	// 提取方式新字段，推荐使用。"0": 普通商品-使用物流发货, "1": 虚拟商品-无需物流与电子交易凭证, "2": 虚拟商品-使用电子交易凭证,  "3": 虚拟商品-充值直连
	PickupMethod string `json:"pickup_method"`
	// 尺码模板ID
	SizeInfoTemplateId int64 `json:"size_info_template_id"`
	// 白底图url(仅素材中心url有效)
	WhiteBackGroundPicUrl string `json:"white_back_ground_pic_url"`
	// 销售渠道类型，包括纯电商（onlineOnly）、专柜同款（sameAsOffline），云零售商家（https://fxg.jinritemai.com/ffa/merchant-growth/cloud-retail）可以设置
	SaleChannelType string `json:"sale_channel_type"`
	// 门店ID
	StoreId int64 `json:"store_id"`
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 限售模板ID
	SaleLimitId int64 `json:"sale_limit_id"`
	// 系统推荐的标题前缀
	NamePrefix string `json:"name_prefix"`
	// 商品类型，0-普通，3-虚拟，6玉石闪购，7云闪购
	ProductType int64 `json:"product_type"`
	// 预售发货方式配置 0-预售结束后xx天发货; 1-支付完成后xx天发货
	PresellDeliveryType int64 `json:"presell_delivery_type"`
	// 审核通过后上架售卖时间配置：0-立即上架售卖 1-放入仓库
	StartSaleType int64 `json:"start_sale_type"`
	// 库存扣减方式，1-拍下减库存 2-付款减库存
	ReduceType int64 `json:"reduce_type"`
	// 预售结束时间，格式2020-02-21 18:54:27，最多支持设置距离当前30天
	PresellEndTime string `json:"presell_end_time"`
	// 重量数值
	WeightValue float64 `json:"weight_value"`
	// 重量单位，0-kg, 1-g
	WeightUnit int64 `json:"weight_unit"`
	// 参考价凭证
	ReferencePriceCertificate *ReferencePriceCertificate `json:"reference_price_certificate"`
	// 参考价，单位分
	ReferencePrice int64 `json:"reference_price"`
	// 前置质检相关（特定二手商家、特定二手类目使用）
	QualityInspectionInfo *QualityInspectionInfo `json:"quality_inspection_info"`
	// 单位价格信息
	UnitPriceInfo *UnitPriceInfo `json:"unit_price_info"`
	// 商品主图3:4；最多支持5张图片；宽高比例为3:4
	MainImageThreeToFour string `json:"main_image_three_to_four"`
	// 是否支持c2b小程序（特定c2b定制商家使用）
	IsC2bSwitchOn bool `json:"is_c2b_switch_on"`
	// 支持c2b定制时的小程序id（特定c2b定制商家使用）
	MicroAppId string `json:"micro_app_id"`
	// 是否是自动充值商品
	IsAutoCharge bool `json:"is_auto_charge"`
	// spu_id 大于0代表有挂载spu
	SpuId int64 `json:"spu_id"`
	// 导购短标题，短标题可用于物流打单及商品搜索场景，若未填写，则系统将智能生成最优短标题在商详购买页面展示，说明详见：https://school.jinritemai.com/doudian/web/article/aHUW2MCvHqF3?from=shop_article
	ShortProductName string `json:"short_product_name"`
	// 售后服务新结构
	AfterSaleServiceV2 *AfterSaleServiceV2 `json:"after_sale_service_v2"`
	// 更新时间，和update_time一样，只是格式不同
	FormatUpdateTime string `json:"format_update_time"`
	// 商品销量
	SellNum int64 `json:"sell_num"`
	// 商品审核结果
	ProductAuditInfo *ProductAuditInfo `json:"product_audit_info"`
	// 商品主图3:4；最多支持5张图片；宽高比例为3:4
	MainPic34 []string `json:"main_pic_3_4"`
	// 店铺装修分类
	ShopCategory *ShopCategory `json:"shop_category"`
	// 标题后缀
	NameSuffix string `json:"name_suffix"`
	// 是否勾选使用品牌名
	UseBrandName bool `json:"use_brand_name"`
	// 新现货+预售商品标识，true-新现货+预售商品
	NewStepProduct bool `json:"new_step_product"`
	// 是否公益商品
	IsCharityProduct bool `json:"is_charity_product"`
	// 公益项目id
	CharityId string `json:"charity_id"`
	// 测评模块是否打开
	IsEvaluateOpened string `json:"is_evaluate_opened"`
	// 定制属性
	CustomProperty *CustomProperty `json:"custom_property"`
	// 商品默认处理时长
	DefaultProcessTime *DefaultProcessTime `json:"default_process_time"`
	// 是否二手数码(成色机)
	IsSecondHandDigital bool `json:"is_second_hand_digital"`
	// 发货时间子类型，该值仅在现货发货时效（presellType = 0）下有意义，1 sku维度发货时效，0或者缺省为商品维度发货时效
	DeliveryTimeSubType int64 `json:"delivery_time_sub_type"`
	// 上门安装服务信息
	HomeInstallServiceInfo *HomeInstallServiceInfo `json:"home_install_service_info"`
	// 门捷列夫类目详情，临时字段，类目数升级期间使用的字段，主要返回老的类目信息，方便开发者做映射使用，升级结束后字段会下线，下线时间另行通知
	OriginalCategoryDetail *OriginalCategoryDetail `json:"original_category_detail"`
	// 标准类目详情（新类目信息）方便开发者做映射使用，升级结束后字段会下线，下线时间另行通知
	FutureCategoryDetail *FutureCategoryDetail `json:"future_category_detail"`
	// 属性图，如水洗标等，key=属性id
	CategoryPropertyPics map[int64]CategoryPropertyPicsItem `json:"category_property_pics"`
}
