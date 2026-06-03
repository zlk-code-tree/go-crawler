package product_addV2_request

import (
	"doudian.com/open/sdk_golang/api/product_addV2/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductAddV2Request struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductAddV2Param 
}
func (c *ProductAddV2Request) GetUrlPath() string{
	return "/product/addV2"
}


func New() *ProductAddV2Request{
	request := &ProductAddV2Request{
		Param: &ProductAddV2Param{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductAddV2Request) Execute(accessToken *doudian_sdk.AccessToken) (*product_addV2_response.ProductAddV2Response, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_addV2_response.ProductAddV2Response{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductAddV2Request) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductAddV2Request) GetParams() *ProductAddV2Param{
	return c.Param
}


type PackageSkuItem struct {
	// 组套名称
	ShortName *string `json:"short_name"`
	// 组套子品skuid
	SubSkuId *int64 `json:"sub_sku_id"`
	// 组套子品id
	SubProductId *int64 `json:"sub_product_id"`
	// 组合数量
	ComboNum *int64 `json:"combo_num"`
}
type ValuesItem_4 struct {
	// 备注
	Remark *string `json:"remark"`
	// 规格值导航路径
	CpvPath []CpvPathItem `json:"cpv_path"`
	// 规格值标准属性值id
	ValueId *int64 `json:"value_id"`
	// 度量衡结构化信息
	MeasureInfo *MeasureInfo `json:"measure_info"`
	// 规格值名称
	ValueName string `json:"value_name"`
}
type QualityListItem struct {
	// 资质id
	QualityId *int64 `json:"quality_id"`
	// 资质名
	QualityContentName *string `json:"quality_content_name"`
	// 资质key
	QualityKey string `json:"quality_key"`
	// 资质名称
	QualityName string `json:"quality_name"`
	// 资质
	QualityAttachments []QualityAttachmentsItem `json:"quality_attachments"`
}
type UnitPriceInfo struct {
	// 如果传了unit_price_info字段，请请写死传0。内部不会消费，只是用于标识
	ProcessCharge int64 `json:"process_charge"`
	// 1表示投资金2表示首饰金
	PriceRuleType int32 `json:"price_rule_type"`
}
type ThreeGuarantees struct {
	// 售后天数
	Duration *int64 `json:"duration"`
	// 服务类型 1-寄修 2-延保
	ServiceType *int64 `json:"service_type"`
}
type AfterSaleServiceV2 struct {
	// 三包服务配置
	ThreeGuarantees *ThreeGuarantees `json:"three_guarantees"`
	// 当前商品是否是大件商品
	IsLargeProduct *int64 `json:"is_large_product"`
}
type SpecInfo struct {
	// 规格内容
	SpecValues []SpecValuesItem `json:"spec_values"`
}
type TaxExemptionSkuInfo struct {
	// 净含量
	Volume *int64 `json:"volume"`
	// 套装数量
	SuitNum *int64 `json:"suit_num"`
	// 是否套装（0 否 1 是）
	IsSuit *int32 `json:"is_suit"`
}
type BatchStockItemsItem struct {
	// 唯一码
	UniqueId *string `json:"unique_id"`
	// 批次库存类型: 1为唯一码
	BatchType *int64 `json:"batch_type"`
	// 批次库存值
	StockNum *int64 `json:"stock_num"`
}
type SellPropertiesItem struct {
	// 规格值名称
	ValueName *string `json:"value_name"`
	// 规格项名称
	PropertyName *string `json:"property_name"`
}
type SpecPricesV2Item struct {
	// 时效库存，一个sku可以配置0到多个时效库存
	MultiTimeStocks []MultiTimeStocksItem `json:"multi_time_stocks"`
	// 规格型号编码-国补场景下国补商品型号
	SpecModelCode []string `json:"spec_model_code"`
	// SKU运输信息
	DeliveryInfos []DeliveryInfosItem `json:"delivery_infos"`
	// sku分类信息，对应枚举值"main_sale_single_product" - 主售款（单品）"main_sale_multi_product" - "主售款（多件装）""main_sale_combination" - 主售款（组合品）"non_main_sale" - 非主售款"information_description" - 信息说明/防拍
	SkuClassificationType *string `json:"sku_classification_type"`
	// 外部skuid
	OuterSkuId *string `json:"outer_sku_id"`
	// SKU图URL：仅支持素材中心URL；要么全部sku填写，要么全部sku不填写；一个sku仅可设置一张图片，图片比例1:1，至少600x600，大小不超过5M，格式 JPG/JPEG/PNG
	SkuPictureUrl []string `json:"sku_picture_url"`
	// 国补自营货品相关
	GovSubsidyCargo *GovSubsidyCargo `json:"gov_subsidy_cargo"`
	// 价格
	Price *int64 `json:"price"`
	// 商家编码
	Code *string `json:"code"`
	// 海关申报要素
	CustomsReportInfo *CustomsReportInfo `json:"customs_report_info"`
	// 供应商编码
	SupplierId *string `json:"supplier_id"`
	// 供货价,单位：分，nil代表不修改，0代表清空
	SupplyPrice *int64 `json:"supply_price"`
	// 预售天数
	PresellDelay *int64 `json:"presell_delay"`
	// 海南免税：Sku 额外信息
	TaxExemptionSkuInfo *TaxExemptionSkuInfo `json:"tax_exemption_sku_info"`
	// 库存数量。跨境必须走区域库存，stock_num设置0。
	StockNum *int64 `json:"stock_num"`
	// 仓ID -> 库存数量。跨境业务大部分都是必传，格式为：Map<商家仓ID, 库存数量>
	StockNumMap *map[string]int64 `json:"stock_num_map"`
	// 阶梯库存数量
	StepStockNum *int64 `json:"step_stock_num"`
	// 条形码
	Barcodes []string `json:"barcodes"`
	// 黄金加工费，单位：分，nil代表不修改
	GoldProcessCharge *int64 `json:"gold_process_charge"`
	// 商品规格值id：懂车帝场景传递
	SpecValueLibId *int64 `json:"spec_value_lib_id"`
	// 库存类型 0-普通库存，1-区域库存，10-阶梯库存
	SkuType *int64 `json:"sku_type"`
	// 批次库存
	BatchStocks []BatchStocksItem `json:"batch_stocks"`
	// 外部skuid
	OutSkuId *int64 `json:"out_sku_id"`
	// 组套包信息
	PackageSku []PackageSkuItem `json:"package_sku"`
	// true：表示sku上架 false：表示sku下架
	SkuStatus *bool `json:"sku_status"`
	// 货品信息，如果传0 0 表示解绑）
	Cargo *Cargo `json:"cargo"`
	// 规格信息
	SellProperties []SellPropertiesItem `json:"sell_properties"`
}
type PropertyDetailItem struct {
	// 定制属性扩展信息
	ExtraInfo *ExtraInfo `json:"extra_info"`
	// 定制属性名称
	Name *string `json:"name"`
}
type CustomProperty struct {
	// 定制属性组信息，如：加工服务
	CustomPropertyGroups []CustomPropertyGroupsItem `json:"customPropertyGroups"`
}
type ValuesItem struct {
	// 模块id
	ModuleId *int64 `json:"module_id"`
	// 度量衡单位名称
	UnitName *string `json:"unit_name"`
	// 度量衡单位id
	UnitId *int64 `json:"unit_id"`
	// 度量衡值前缀
	Prefix *string `json:"prefix"`
	// 度量衡值
	Value *string `json:"value"`
	// 度量衡后缀
	Suffix *string `json:"suffix"`
}
type DefaultProcessTime struct {
	// 商品默认处理时长，单位秒
	Value *int64 `json:"value"`
}
type QualityInspectionInfo struct {
	// 机构编码，请通过/inspection/QueryBtasAgencyList接口获取
	Agency *string `json:"agency"`
	// 质检证书编码
	CertificateCode *string `json:"certificate_code"`
	// 是否支持前置质检
	Supported *bool `json:"supported"`
	// 1: 单库存模式，只允许售卖一个sku；2: 多库存模式，不限售卖次数
	Mode *int32 `json:"mode"`
}
type OpenLogisticsInfo struct {
	// 品牌所在地id
	BrandCountryId *int64 `json:"brand_country_id"`
	// 原产国id
	OriginCountryId *int64 `json:"origin_country_id"`
	// 税金承担方（一般都填0，走商家承担）：0商家承担，1用户承担
	TaxPayer *int64 `json:"tax_payer"`
	// 货源国id
	SourceCountryId *int64 `json:"source_country_id"`
	// 商品净重
	NetWeightQty *float64 `json:"net_weight_qty"`
	// 商家仓id
	CrossWarehouseId *string `json:"cross_warehouse_id"`
	// 通关模式，1：BBC 2：BC 3：CC邮关
	CustomsClearType *int64 `json:"customs_clear_type"`
}
type CustomPropertyGroupsItem struct {
	// 定制属性组名称
	Name *string `json:"name"`
	// 定制属性信息，如：宰杀，不宰杀
	PropertyDetail []PropertyDetailItem `json:"propertyDetail"`
}
type Cargo struct {
	// 货品id
	CargoId *string `json:"cargo_id"`
	// 货品类型，跨境的货品类型传3
	SourceType *int64 `json:"source_type"`
}
type PoiResource struct {
	// 领取须知
	Notification *string `json:"notification"`
	// 门店库id
	PoiLibId *int64 `json:"poi_lib_id"`
	// 卡券有效开始时间，秒单位时间戳
	ValidStart *int64 `json:"valid_start"`
	// 兑换链接
	Link *string `json:"link"`
	// 客服电话
	ServiceNum *string `json:"service_num"`
	// 券码总量，0/-1表示不限制，平台券时须\u003e0
	Count *int64 `json:"count"`
	// 可核销总次数
	TotalCanUseCount *int32 `json:"total_can_use_count"`
	// 有效时间段，从领取日开始计算，优先级高于ValidStart-ValidEnd
	ValidDays *int64 `json:"valid_days"`
	// 平台产生券码: 1 ; 合作方api实时传入的券码: 4
	CodeType *int64 `json:"code_type"`
	// 券码使用条件
	Condition *string `json:"condition"`
	// 数组当前只支持一个元素且只可为 1或2，其中 1 表示随时退+过期自动退，2 表示不支持退
	CouponReturnMethods []int64 `json:"coupon_return_methods"`
	// 0-不支持二次兑换，1-支持二次兑换
	CouponSecondExchange *int64 `json:"couponSecondExchange"`
	// 卡券有效截止时间，秒单位时间戳
	ValidEnd *int64 `json:"valid_end"`
}
type DelayRule struct {
	// 特殊日期延迟发货时间最晚发货时间，时间戳，单位秒：当config_type=1时，传时间戳，代表最晚x发货；当config_type=2时，传天数，代表延迟x天发货
	ConfigValue int64 `json:"config_value"`
	// 特殊日期延迟发货时间下单结束时间，时间戳，单位秒
	EndTime int64 `json:"end_time"`
	// 1 时间点；2 时间范围
	ConfigType int32 `json:"config_type"`
	// 是否开启特殊日期延迟发货
	Enable bool `json:"enable"`
	// 特殊日期延迟发货时间下单开始时间，时间戳，单位秒
	StartTime int64 `json:"start_time"`
}
type MultiTimeStocksItem struct {
	// 时效
	TimeType *int64 `json:"time_type"`
	// 时效描述
	TimeDesc *string `json:"time_desc"`
	// 时效库存数
	StockNum *int64 `json:"stock_num"`
}
type CustomsReportInfo struct {
	// 法定第一计量单位
	FirstMeasureUnit *string `json:"first_measure_unit"`
	// 法定第一计量数量
	FirstMeasureQty *float64 `json:"first_measure_qty"`
	// 售卖单位
	Unit *string `json:"unit"`
	// 品名
	ReportName *string `json:"report_name"`
	// 法定第二计量数量
	SecondMeasureQty *float64 `json:"second_measure_qty"`
	// 品牌
	ReportBrandName *string `json:"report_brand_name"`
	// 规格型号
	GModel *string `json:"g_model"`
	// 条形码
	BarCode *string `json:"bar_code"`
	// 法定第二计量单位
	SecondMeasureUnit *string `json:"second_measure_unit"`
	// 海关代码
	HsCode *string `json:"hs_code"`
	// 用途
	Usage *string `json:"usage"`
}
type CategoryPropertyPicsItem struct {
	// 属性图图片链接
	Urls []string `json:"urls"`
}
type QualityAttachmentsItem struct {
	// 凭证url
	Url string `json:"url"`
	// 1-图片
	MediaType int64 `json:"media_type"`
}
type RecruitInfo struct {
	// 招商类型
	RecruitType string `json:"recruit_type"`
	// 线索ID
	RecruitFollowId string `json:"recruit_follow_id"`
	// 发品源
	RecruitSource *string `json:"recruit_source"`
}
type ReferencePriceCertificate struct {
	// 图片url需要使用商品素材中心的url并且只能有一张
	CertificateUrls []string `json:"certificate_urls"`
	// 通过/product/getProductUpdateRule获取可选的参考价格类型
	CertificateType *int64 `json:"certificate_type"`
}
type MeasureInfo struct {
	// 模版id
	TemplateId *int64 `json:"template_id"`
	// 模块内容
	Values []ValuesItem `json:"values"`
}
type BatchStocksItem struct {
	// 批次库存项
	BatchStockItems []BatchStockItemsItem `json:"batch_stock_items"`
	// 外部仓id，区域库存模式填写
	OutWarehouseId *string `json:"out_warehouse_id"`
}
type ExtraInfo struct {
	// 属性处理时长，单位秒
	ProcessTime *int64 `json:"process_time"`
}
type ProductAddV2Param struct {
	// 推荐使用，外部商家编码，支持字符串。超市小时达场景不推荐使用。out_product_id 、outer_product_id  都传了的话，要求相等。
	OuterProductId *string `json:"outer_product_id"`
	// 0-普通，3-虚拟，6玉石闪购，7云闪购
	ProductType int64 `json:"product_type"`
	// 叶子类目ID通过/shop/getShopCategory接口获取
	CategoryLeafId int64 `json:"category_leaf_id"`
	// 商品名称，至少8个字符（4个汉字），最多60个字符(30个汉字)，不能含emoj表情
	Name string `json:"name"`
	// 商家推荐语，不能含emoj表情
	RecommendRemark *string `json:"recommend_remark"`
	// 商品轮播图，多张图片用 \"|\" 分开，第一张图为主图，最多5张，至少600x600，大小不超过5M。
	Pic string `json:"pic"`
	// 商品描述，目前只支持图片。多张图片用 \"|\" 分开。不能用其他网站的文本粘贴，这样会出现css样式不兼容。总图片数量不得超过50张。
	Description string `json:"description"`
	// 支付方式，0货到付款 1在线支付，2，货到付款+在线支付
	PayType *int64 `json:"pay_type"`
	// 海南免税生效。 0:离岛免税、1:邮寄、2:离岛自提+邮寄
	DeliveryMethod *int32 `json:"delivery_method"`
	// 海南免税：海关限购分类编码
	CdfCategory *string `json:"cdf_category"`
	// 1 减库存类型：1-拍下减库存 2-付款减库存
	ReduceType int64 `json:"reduce_type"`
	// 同店商品推荐：为空表示使用系统推荐；多个product_id使用“|”分开
	AssocIds *string `json:"assoc_ids"`
	// 运费模板id，传0表示包邮，通过/freightTemplate/list接口获取
	FreightId int64 `json:"freight_id"`
	// 重量
	Weight *float64 `json:"weight"`
	// 重量单位，0-kg, 1-g
	WeightUnit *int64 `json:"weight_unit"`
	// delivery_delay_day： 承诺发货时间，单位是天,不传则默认为2天。现货发货(presell_type=0)和阶梯发货模式(presell_type=2)时必填，支持传入9999 、1、 2 （分别表示当日发、次日发、48小时发），具体支持传入的参数范围/product/getProductUpdateRule
	DeliveryDelayDay *int64 `json:"delivery_delay_day"`
	// 发货模式，0-现货发货，1-预售发货，2-阶梯发货，默认0
	PresellType *int64 `json:"presell_type"`
	// 全款预售模式时的发货时间/阶梯模式下阶梯发货时间，具体支持传入的参数范围/product/getProductUpdateRule。
	PresellDelay *int64 `json:"presell_delay"`
	// 预售结束时间，格式2020-02-21 18:54:27，最多支持设置距离当前30天
	PresellEndTime *string `json:"presell_end_time"`
	// 【该字段将在2023年4月30日下线，请开发使用最新的after_sale_service字段传值；】是否支持7天无理由，0不支持，1支持，2支持（拆封后不支持）
	Supply7dayReturn *int64 `json:"supply_7day_return"`
	// 客服电话号码
	Mobile string `json:"mobile"`
	// false仅保存，true保存+提审
	Commit bool `json:"commit"`
	// 商家可见备注
	Remark *string `json:"remark"`
	// 外部product_id。超市小时达场景不推荐使用。out_product_id 、outer_product_id  都传了的话，要求相等。
	OutProductId *int64 `json:"out_product_id"`
	// 资质，可通过/product/qualificationConfig获取
	QualityList []QualityListItem `json:"quality_list"`
	// 如果不填，则规格名为各级规格名用 \"-\" 自动生成
	SpecName *string `json:"spec_name"`
	// 店铺通用规格，能被同类商品通用。多规格用^分隔，规格与规格值用|分隔，多个规格值用,分隔。如果创建或编辑现货+预售商品或新预售商品时，须包含发货时效，如：颜色|黑色,白色,黄色^尺码|S,M,L^发货时效|5天内发货,15天内发货
	Specs *string `json:"specs"`
	// sku详情，数量应该等于规格1规格2规格3，sku数量和规格组合数必须一致 sku不可售时，库存可设置为0。price单位为分。 delivery_infos为SKU物流信息，info_value为字符串类型（示例："12"），info_type填weight，info_unit支持mg,g,kg，超市小时达场景主品用普通库存，子品用区域库存（"sku_type": 1 // 区域库存，"stock_num_map": {"123": 99999 // 门店ID:库存数量}）; “gold_process_charge”为黄金加工费，只有计价金类目可填并且必填。sell_properties为sku规格信息，代替spec_detail_name1、spec_detail_name2、spec_detail_name3，支持填写超过三级规格，样式:[{"property_id":123,"property_name":"颜色","value_id":456,"value_name":"红色","remark":"偏深"},{"property_id":789,"property_name":"净含量","value_id":891,"value_name":"30ml","remark":null,"measurement":{"measure_unit":"ml","measure_unit_id":4,"value":"30"}}] 其中property_id规格项属性id,自定义时传0,property_name为规格项名称,value_id为规格值属性id自定义时传0,value_name为规格值名称,remark为规格值备注(选填),measurement为度量衡信息，当规格值为度量衡属性自定义值时传递。
	SpecPrices *string `json:"spec_prices"`
	// 如颜色-尺寸, 颜色就是主规格, 颜色如黑,白,黄,它们图片url以逗号分隔 注：\"pic\"、\"description\"、\"spec_pic\"这三个字段里的传入的图片数量总和，不得超过50张。
	SpecPic *string `json:"spec_pic"`
	// 每个用户每次下单限购件数
	MaximumPerOrder *int64 `json:"maximum_per_order"`
	// 每个用户累计限购件数
	LimitPerBuyer *int64 `json:"limit_per_buyer"`
	// 每个用户每次下单至少购买的件数
	MinimumPerOrder *int64 `json:"minimum_per_order"`
	// 属性，通过/product/getCatePropertyV2获取 格式：{"property_id":[{"value":value,"name":"property_name","diy_type":0}]} name的类型是string，value和diy_type的类型是number
	ProductFormatNew *string `json:"product_format_new"`
	// spu_id
	SpuId *int64 `json:"spu_id"`
	// 可预约发货天数
	AppointDeliveryDay *int64 `json:"appoint_delivery_day"`
	// third_url
	ThirdUrl *string `json:"third_url"`
	// extra
	Extra *string `json:"extra"`
	// src
	Src *string `json:"src"`
	// 品牌id，通过/brand/list获取，无品牌id则传596120136
	StandardBrandId *int64 `json:"standard_brand_id"`
	// 卡券类型需要传true
	NeedCheckOut *bool `json:"need_check_out"`
	// 卡券信息
	PoiResource *PoiResource `json:"poi_resource"`
	// 汽车vin码
	CarVinCode *string `json:"car_vin_code"`
	// 0：全款预售，1：sku预售，2：现货+预售 ，3：新预售
	PresellConfigLevel *int64 `json:"presell_config_level"`
	// 充值模式
	NeedRechargeMode *bool `json:"need_recharge_mode"`
	// 账号模板id
	AccountTemplateId *string `json:"account_template_id"`
	// 全款预售和sku预售时传递，其他不传： 0 预售结束后发货  1支付完成后发货
	PresellDeliveryType *int64 `json:"presell_delivery_type"`
	// 白底图url(仅素材中心url有效)，白底图比例要求1:1
	WhiteBackGroundPicUrl *string `json:"white_back_ground_pic_url"`
	// 3:4长图url(仅素材中心url有效)
	LongPicUrl *string `json:"long_pic_url"`
	// 推荐传入，售后服务参数,key:value格式。supply_day_return_selector(7天无理由选项)：N天-政策代号，N只支持7和15，政策代号枚举https://bytedance.feishu.cn/docs/doccnF946oh1c98e7mo9DlYZtig 。 supply_red_ass_return（红屁屁无忧）：0不支持，1支持。supply_allergy_return（过敏无忧,仅跨境可选）：0不支持，1支持。 damaged_order_return（坏损包退）：0不支持，1支持。 support_allergy_returnV2（过敏包退，境内可选）：0不支持，1支持
	AfterSaleService *map[string]string `json:"after_sale_service"`
	// 售卖方式;0:全渠道手售卖,1:仅指定直播间售卖
	SellChannel []int64 `json:"sell_channel"`
	// 审核通过后上架售卖时间配置:0-立即上架售卖 1-放入仓库
	StartSaleType *int64 `json:"start_sale_type"`
	// 特殊日期延迟发货规则
	DelayRule *DelayRule `json:"delay_rule"`
	// 主图视频ID，可以先通过https://op.jinritemai.com/docs/api-docs/69/1617接口上传视频，获取审核通过的视频素材ID进行传入 任务需要验证
	MaterialVideoId *string `json:"material_video_id"`
	// 提取方式新字段，推荐使用。"0": 普通商品-使用物流发货, "1": 虚拟商品-无需物流与电子交易凭证, "2": 虚拟商品-使用电子交易凭证,  "3": 虚拟商品-充值直连
	PickupMethod *string `json:"pickup_method"`
	// 尺码模板ID
	SizeInfoTemplateId *int64 `json:"size_info_template_id"`
	// 外部商品url
	SubstituteGoodsUrl *string `json:"substitute_goods_url"`
	// 销售渠道类型，可选onlineOnly（纯电商，仅在线上售卖）或sameAsOffline（专柜同款，线上线下都有售卖），云零售商家（https://fxg.jinritemai.com/ffa/merchant-growth/cloud-retail）可以设置
	SaleChannelType *string `json:"sale_channel_type"`
	// 招商信息
	RecruitInfo *RecruitInfo `json:"recruit_info"`
	// 门店ID
	StoreId *int64 `json:"store_id"`
	// 主商品ID
	MainProductId *int64 `json:"main_product_id"`
	// 限售模板ID
	SaleLimitId *int64 `json:"sale_limit_id"`
	// 通过/product/getRecommendName接口推荐的商品标题前缀（注意系统根据类目属性生成前缀字符串时，末尾有一个空格）
	NamePrefix *string `json:"name_prefix"`
	// 参考价，单位分，需大于商品价格并小于商品价格的10倍
	ReferencePrice *int64 `json:"reference_price"`
	// 参考价凭证信息
	ReferencePriceCertificate *ReferencePriceCertificate `json:"reference_price_certificate"`
	// 商品主图3:4，多张图片用 \"|\" 分开，最多5张，大小不超过5M。
	MainImageThreeToFour *string `json:"main_image_three_to_four"`
	// 商品价格规则信息
	UnitPriceInfo *UnitPriceInfo `json:"unit_price_info"`
	// 前置质检相关（特定二手商家、特定二手类目使用）
	QualityInspectionInfo *QualityInspectionInfo `json:"quality_inspection_info"`
	// 是否支持c2b小程序（特定c2b定制商家使用，请使用/product/getProductUpdateRule接口获取是否支持）
	IsC2bSwitchOn *bool `json:"is_c2b_switch_on"`
	// 支持c2b定制时的小程序id（特定c2b定制商家使用，请使用/product/getProductUpdateRule接口获取是否支持）
	MicroAppId *string `json:"micro_app_id"`
	// 是否是自动充值商品
	IsAutoCharge *bool `json:"is_auto_charge"`
	// 导购短标题，短标题可用于物流打单及商品搜索场景，若未填写，则系统将智能生成最优短标题在商详购买页面展示，说明详见：https://school.jinritemai.com/doudian/web/article/aHUW2MCvHqF3?from=shop_article
	ShortProductName *string `json:"short_product_name"`
	// 售后服务新结构（老的after_sale_service中的字段后续会慢慢迁移到这里）
	AfterSaleServiceV2 *AfterSaleServiceV2 `json:"after_sale_service_v2"`
	// 规格详细信息，代替specs和spec_name
	SpecInfo *SpecInfo `json:"spec_info"`
	// 新sku参数，代替spec_prices
	SpecPricesV2 []SpecPricesV2Item `json:"spec_prices_v2"`
	// true:主动设置库存类型（仅对区域库存生效，普通库存、阶梯库存不受此字段影响）
	WithSkuType *bool `json:"with_sku_type"`
	// 通过/product/getRecommendName接口推荐的商品标题后缀（注意系统根据类目属性生成后缀字符串时，前面可能有一个空格）
	NameSuffix *string `json:"name_suffix"`
	// 是否勾选使用品牌名, 注意勾选时需要接入/product/getRecommendName接口
	UseBrandName *bool `json:"use_brand_name"`
	// 跨境物流信息
	OpenLogisticsInfo *OpenLogisticsInfo `json:"open_logistics_info"`
	// 商品价格是否含税，跨境必需填1。  1：商品价格含税 2：商品价格不含税
	PriceHasTax *string `json:"price_has_tax"`
	// 商品是否跨境、教育等（跨境/海南商品填写2））
	BizKind *int64 `json:"biz_kind"`
	// 商品定制属性填写，（即时零售，外卖业务填写）
	CustomProperty *CustomProperty `json:"custom_property"`
	// 测评短视频开关
	IsEvaluateOpened *bool `json:"is_evaluate_opened"`
	// 商品默认处理时长
	DefaultProcessTime *DefaultProcessTime `json:"default_process_time"`
	// 管理单元ID
	CompanyId *int64 `json:"company_id"`
	// 是否二手数码(成色机)
	IsSecondHandDigital *bool `json:"is_second_hand_digital"`
	// 预测记录id，类目预测记录key=category_recommend_id
	RecommendIds *map[string]string `json:"recommend_ids"`
	// 虚拟商品设置是否订单展示核销去使用
	ShowCheckoutEntry *bool `json:"show_checkout_entry"`
	// 自提备货时长(单位天)；自提商品且不管理门店库存时必填
	PickupPrepareTime *int64 `json:"pickup_prepare_time"`
	// 上门安装模版，传递为空不处理，传递0代表C模式，传递>0代表A/B模式
	HomeInstallServiceTemplateId *int64 `json:"home_install_service_template_id"`
	// 属性图，key=属性id
	CategoryPropertyPics *map[int64]CategoryPropertyPicsItem `json:"category_property_pics"`
	// 是否平台自营商品
	IsPlatformSelfOperation *bool `json:"is_platform_self_operation"`
	// 自营品源品id
	OriginProductId *int64 `json:"origin_product_id"`
}
type CpvPathItem struct {
	// 属性值id
	Cpvid int64 `json:"cpvid"`
	// 属性id
	Cpid int64 `json:"cpid"`
}
type SpecValuesItem struct {
	// 规格项标准属性id
	PropertyId *int64 `json:"property_id"`
	// 颜色
	PropertyName string `json:"property_name"`
	// 规格值
	Values []ValuesItem_4 `json:"values"`
}
type DeliveryInfosItem struct {
	// 信息计量单位
	InfoUnit *string `json:"info_unit"`
	// 信息值
	InfoValue *string `json:"info_value"`
	// 信息类型
	InfoType *string `json:"info_type"`
}
type GovSubsidyCargo struct {
	// 国补自营货品ID
	CargoId *int64 `json:"cargo_id"`
}
