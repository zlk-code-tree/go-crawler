package product_calDelayShipTimeForOpen_request

import (
	"doudian.com/open/sdk_golang/api/product_calDelayShipTimeForOpen/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductCalDelayShipTimeForOpenRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductCalDelayShipTimeForOpenParam 
}
func (c *ProductCalDelayShipTimeForOpenRequest) GetUrlPath() string{
	return "/product/calDelayShipTimeForOpen"
}


func New() *ProductCalDelayShipTimeForOpenRequest{
	request := &ProductCalDelayShipTimeForOpenRequest{
		Param: &ProductCalDelayShipTimeForOpenParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductCalDelayShipTimeForOpenRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_calDelayShipTimeForOpen_response.ProductCalDelayShipTimeForOpenResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_calDelayShipTimeForOpen_response.ProductCalDelayShipTimeForOpenResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductCalDelayShipTimeForOpenRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductCalDelayShipTimeForOpenRequest) GetParams() *ProductCalDelayShipTimeForOpenParam{
	return c.Param
}


type CpvPathItem struct {
	// cpid
	Cpid *int64 `json:"cpid"`
	// cpvid
	Cpvid *int64 `json:"cpvid"`
}
type ServiceQuotation struct {
	// 供应商id
	ProviderId *string `json:"provider_id"`
	// 所属模版id
	ServiceTemplateId *string `json:"service_template_id"`
	// 扩展信息
	Extra *string `json:"extra"`
	// sku粒度的服务商品信息服务报价id
	ServiceQuotationId *string `json:"service_quotation_id"`
}
type SellPropertiesItem_4_4 struct {
	// 规格值detailId，支持3级以上多规格，代替原specDetailId123
	ValueSpecDetailId *int64 `json:"value_spec_detail_id"`
	// property_name
	PropertyName *string `json:"property_name"`
	// 销售属性值路径
	CpvPath []CpvPathItem `json:"cpv_path"`
	// 备注
	Remark *string `json:"remark"`
	// property_id
	PropertyId *int64 `json:"property_id"`
	// 度量衡信息
	Measurement *Measurement `json:"measurement"`
	// 度量衡信息
	MeasureInfo *MeasureInfo_5_5 `json:"measure_info"`
	// value_name
	ValueName *string `json:"value_name"`
	// value_id
	ValueId *int64 `json:"value_id"`
}
type TaxExemptionSkuInfo struct {
	// 净含量
	Volume *int64 `json:"volume"`
	// 套装数量
	SuitNum *int64 `json:"suit_num"`
	// 是否套装（0 否 1 是）
	IsSuit *int32 `json:"is_suit"`
}
type SpecPricesV2Item struct {
	// 规格型号编码
	SpecModelCode []string `json:"spec_model_code"`
	// 绑定货品（如果传0 0 表示解绑）
	Cargo *Cargo `json:"cargo"`
	// 外部sku_id-i64
	OutSkuId *int64 `json:"out_sku_id"`
	// 销售属性（逐步替代specDetailId1、2、3，过渡期互相兼容，大于三级规格时只能用sellProperty）。
	SellProperties []SellPropertiesItem `json:"sell_properties"`
	// 供应商编码
	SupplierId *string `json:"supplier_id"`
	// skuId,支持open基于skuId修改sku信息
	Id *int64 `json:"id"`
	// 仓ID -> 库存数量
	StockNumMap *map[string]int64 `json:"stock_num_map"`
	// sku服务报价信息
	ServiceQuotation *ServiceQuotation `json:"service_quotation"`
	// 懂车帝商品规格值id
	SpecValueLibId *int64 `json:"spec_value_lib_id"`
	// 忽略价格更新
	IgnorePriceUpdate *bool `json:"ignore_price_update"`
	// 海关申报要素
	CustomsReportInfo *CustomsReportInfo `json:"customs_report_info"`
	// price
	Price *int64 `json:"price"`
	// SKU物流信息
	DeliveryInfos []DeliveryInfosItem `json:"delivery_infos"`
	// 二级规格值名称，已废弃，open对接不再使用
	SpecDetailName2 *string `json:"spec_detail_name2"`
	// sku分类信息
	SkuClassificationType *string `json:"sku_classification_type"`
	// 组套信息
	PackageSku []PackageSkuItem `json:"package_sku"`
	// 一级规格值名称，已废弃，open对接不再使用
	SpecDetailName1 *string `json:"spec_detail_name1"`
	// IMEI数量
	ImeiNum *int64 `json:"imei_num"`
	// 复制创建时，记录原商品skuId
	CopiedSkuId *int64 `json:"copied_sku_id"`
	// 外卖份量信息
	QuantityInfo *QuantityInfo `json:"quantity_info"`
	// 海南免税：Sku 信息
	TaxExemptionSkuInfo *TaxExemptionSkuInfo `json:"tax_exemption_sku_info"`
	// 外部sku_id-string
	OuterSkuId *string `json:"outer_sku_id"`
	// 供货价,单位：分，nil代表不修改，0代表清空
	SupplyPrice *int64 `json:"supply_price"`
	// 时效库存明细
	MultiTimeStocks []MultiTimeStocksItem `json:"multi_time_stocks"`
	// 条码
	Barcodes []string `json:"barcodes"`
	// SKU产品编码
	ProductCode *ProductCode `json:"product_code"`
	// 三级规格值名称，已废弃，open对接不再使用
	SpecDetailName3 *string `json:"spec_detail_name3"`
	// 黄金加工费，单位：分，nil代表不修改
	GoldProcessCharge *int64 `json:"gold_process_charge"`
	// sku状态是否有效，true上架 false下架
	SkuStatus *bool `json:"sku_status"`
	// SKU维度预售发货延迟时间
	PresellDelay *int64 `json:"presell_delay"`
	// 库存类型
	SkuType *int64 `json:"sku_type"`
	// 阶梯库存数
	StepStockNum *int64 `json:"step_stock_num"`
	// stock_num
	StockNum *int64 `json:"stock_num"`
	// code
	Code *string `json:"code"`
	// 批次库存
	BatchStocks []BatchStocksItem `json:"batch_stocks"`
}
type OpenLogisticsInfo struct {
	// 通关模式，1BBC 2BC 3邮关
	CustomsClearType *int64 `json:"customs_clear_type"`
	// 原产国id
	OriginCountryId *int64 `json:"origin_country_id"`
	// 货源国id
	SourceCountryId *int64 `json:"source_country_id"`
	// 商家仓id
	CrossWarehouseId *string `json:"cross_warehouse_id"`
	// 税金承担方，0商家承担，1用户承担
	TaxPayer *int64 `json:"tax_payer"`
	// 商品净重
	NetWeightQty *float64 `json:"net_weight_qty"`
	// 品牌所在地id
	BrandCountryId *int64 `json:"brand_country_id"`
}
type QuantityInfo struct {
	// unit_prefix
	UnitPrefix *string `json:"unit_prefix"`
	// unit_desc
	UnitDesc *string `json:"unit_desc"`
	// value
	Value *int64 `json:"value"`
	// unit_name
	UnitName []string `json:"unit_name"`
}
type BatchStockItemsItem struct {
	// 库存数
	StockNum *int64 `json:"stock_num"`
	// 唯一码
	UniqueId *string `json:"unique_id"`
	// 批次类型：0为时效 1为唯一码
	BatchType *int64 `json:"batch_type"`
}
type Measurement struct {
	// 单位id，与类目属性应该保持一致
	MeasureUnitId *int64 `json:"measure_unit_id"`
	// 单位名字（g或克等）
	MeasureUnit *string `json:"measure_unit"`
	// 输入值
	Value *string `json:"value"`
}
type DelayRule struct {
	// 商家配置的结束时间
	EndTime *int64 `json:"end_time"`
	// 商家配置的开始时间
	StartTime *int64 `json:"start_time"`
	// 配置类型
	ConfigType *int32 `json:"config_type"`
	// 配置值
	ConfigValue *int64 `json:"config_value"`
	// 规则生效范围
	RuleScope *int64 `json:"rule_scope"`
	// 是否开启
	Enable *bool `json:"enable"`
}
type CustomsReportInfo struct {
	// 法定第二计量单位
	SecondMeasureUnit *string `json:"second_measure_unit"`
	// 售卖单位
	Unit *string `json:"unit"`
	// 品牌
	ReportBrandName *string `json:"report_brand_name"`
	// 规格型号
	GModel *string `json:"g_model"`
	// 条形码
	BarCode *string `json:"bar_code"`
	// 法定第一计量数量
	FirstMeasureQty *float64 `json:"first_measure_qty"`
	// 法定第二计量数量
	SecondMeasureQty *float64 `json:"second_measure_qty"`
	// 用途
	Usage *string `json:"usage"`
	// 法定第一计量单位
	FirstMeasureUnit *string `json:"first_measure_unit"`
	// 海关代码
	HsCode *string `json:"hs_code"`
	// 品名
	ReportName *string `json:"report_name"`
}
type QualityListItem struct {
	// 资质附件
	QualityAttachments []QualityAttachmentsItem `json:"quality_attachments"`
	// 资质名
	QualityContentName *string `json:"quality_content_name"`
	// 资质种类名称
	QualityName *string `json:"quality_name"`
	// 资质id 复用
	QualityId *int64 `json:"quality_id"`
	// 资质种类key
	QualityKey *string `json:"quality_key"`
}
type ExtraInfo struct {
	// 商品主图，1—5张
	Pic []PicItem `json:"pic"`
	// 自售库存
	SelfSellStockNum *int64 `json:"self_sell_stock_num"`
	// 商品资质
	QualityList []QualityListItem `json:"quality_list"`
	// 国内货品信息
	BindCargoInfo *BindCargoInfo `json:"bind_cargo_info"`
	// 商品主图3:4，1—5张
	MainImageThreeToFour []MainImageThreeToFourItem `json:"main_image_three_to_four"`
}
type MultiTimeStocksItem struct {
	// 库存数
	StockNum *int64 `json:"stock_num"`
	// 时效描述
	TimeDesc *string `json:"time_desc"`
	// 时效类型
	TimeType *int64 `json:"time_type"`
}
type PicItem struct {
	// url
	Url *string `json:"url"`
	// process_info
	ProcessInfo *ProcessInfo `json:"process_info"`
	// 图片类型
	PicType *int64 `json:"pic_type"`
}
type BindCargoInfo struct {
	// 可选发发货模式类型
	DeliveryTypeOptions []string `json:"delivery_type_options"`
	// cargo_name
	CargoName *string `json:"cargo_name"`
	// 货品价格
	CostPrice *int64 `json:"cost_price"`
	// 是否禁止编辑库存
	DisableStock *bool `json:"disable_stock"`
	// 货品类型
	CargoType *int64 `json:"cargo_type"`
	// 货品图片
	ImgUrls []string `json:"img_urls"`
	// 货品标签
	Tags []string `json:"tags"`
	// 货品id
	CargoId *int64 `json:"cargo_id"`
	// 是否需要校验价格
	NeedValidatePrice *bool `json:"need_validate_price"`
	// 货品库存
	Stock *int64 `json:"stock"`
}
type ValuesItem struct {
	// 后缀
	Suffix *string `json:"suffix"`
	// 自定义度量衡扩展信息
	Extra *map[string]string `json:"extra"`
	// 值id
	ValueId *int64 `json:"value_id"`
	// 后缀id
	SuffixId *int64 `json:"suffix_id"`
	// 值内容
	Value *string `json:"value"`
	// 度量衡单位名称
	UnitName *string `json:"unit_name"`
	// 前缀
	Prefix *string `json:"prefix"`
	// 前缀id
	PrefixId *int64 `json:"prefix_id"`
	// 是否必填
	ValueRequired *bool `json:"value_required"`
	// 度量衡单位id
	UnitId *int64 `json:"unit_id"`
	// 模块id
	ModuleId *int64 `json:"module_id"`
}
type MeasureInfo struct {
	// 度量衡模板id
	TemplateId *int64 `json:"template_id"`
	// 度量衡内容
	Values []ValuesItem `json:"values"`
	// 自定义度量衡扩展信息
	Extra *map[string]string `json:"extra"`
}
type ProcessInfo struct {
	// origin_url
	OriginUrl *string `json:"origin_url"`
	// process_type
	ProcessType []string `json:"process_type"`
}
type QualityAttachmentsItem struct {
}
type MeasureInfo_5_5 struct {
	// 自定义度量衡扩展信息
	Extra *map[string]string `json:"extra"`
	// 度量衡内容
	Values []ValuesItem_6_6 `json:"values"`
	// 度量衡模板id
	TemplateId *int64 `json:"template_id"`
}
type PackageSkuItem struct {
	// sku_id
	SkuId *int64 `json:"sku_id"`
	// stock_type
	StockType *int32 `json:"stock_type"`
	// 叶子类目id
	CategoryLeafId *int64 `json:"category_leaf_id"`
	// 子品结算分摊比例
	SettlementRatio *int64 `json:"settlement_ratio"`
	// 扩展信息，受选项的控制
	ExtraInfo *ExtraInfo `json:"extra_info"`
	// combo_num
	ComboNum *int64 `json:"combo_num"`
	// stock_num
	StockNum *int64 `json:"stock_num"`
	// 销售属性
	SellProperties []SellPropertiesItem_4_4 `json:"sell_properties"`
	// sku实时价格
	Price *int64 `json:"price"`
	// short_name
	ShortName *string `json:"short_name"`
	// 图片url
	PicUrl *string `json:"pic_url"`
	// ori_price
	OriPrice *int64 `json:"ori_price"`
	// sub_sku_id
	SubSkuId *int64 `json:"sub_sku_id"`
	// sub_product_id
	SubProductId *int64 `json:"sub_product_id"`
}
type BatchStocksItem struct {
	// 外部仓id，仅区域库存模式填写
	OutWarehouseId *string `json:"out_warehouse_id"`
	// 批次库存项
	BatchStockItems []BatchStockItemsItem `json:"batch_stock_items"`
}
type SellPropertiesItem struct {
	// property_name
	PropertyName *string `json:"property_name"`
	// 度量衡信息
	Measurement *Measurement `json:"measurement"`
	// value_id
	ValueId *int64 `json:"value_id"`
	// 备注
	Remark *string `json:"remark"`
	// property_id
	PropertyId *int64 `json:"property_id"`
	// 度量衡信息
	MeasureInfo *MeasureInfo `json:"measure_info"`
	// 规格值detailId，支持3级以上多规格，代替原specDetailId123
	ValueSpecDetailId *int64 `json:"value_spec_detail_id"`
	// 销售属性值路径
	CpvPath []CpvPathItem `json:"cpv_path"`
	// value_name
	ValueName *string `json:"value_name"`
}
type MainImageThreeToFourItem struct {
	// url
	Url *string `json:"url"`
	// pic_type
	PicType *int64 `json:"pic_type"`
	// process_info
	ProcessInfo *ProcessInfo `json:"process_info"`
}
type ValuesItem_6_6 struct {
}
type Cargo struct {
	// cargo_id
	CargoId *string `json:"cargo_id"`
	// source_type
	SourceType *int64 `json:"source_type"`
}
type DeliveryInfosItem struct {
	// 信息计量单位
	InfoUnit *string `json:"info_unit"`
	// 信息值
	InfoValue *string `json:"info_value"`
	// 物流采集信息信息类型
	InfoType *string `json:"info_type"`
}
type ProductCode struct {
	// 二维码码值
	Code *string `json:"code"`
}
type ProductCalDelayShipTimeForOpenParam struct {
	// 类目id
	CategoryLeafId *int64 `json:"category_leaf_id"`
	// 预售结束时间
	PresellEndTime *string `json:"presell_end_time"`
	// 商品维度 延迟发货配置
	DelayRule *DelayRule `json:"delay_rule"`
	// 预售类型
	PresellType *int64 `json:"presell_type"`
	// "0": 普通商品-使用物流发货, "1": 虚拟商品-无需物流与电子交易凭证, "2": 虚拟商品-使用电子交易凭证,  "3": 虚拟商品-充值直连
	PickupMethod *string `json:"pickup_method"`
	// 是否大众拍卖商品
	IsMassAuctionProduct *bool `json:"is_mass_auction_product"`
	// 商品id
	ProductId *int64 `json:"product_id"`
	// 预售结束后几天发货
	PresellDelay *int64 `json:"presell_delay"`
	// 选择的时效 新阶梯
	SelectedMultiTimes []int64 `json:"selected_multi_times"`
	// 是否打开c2b定制商品开关
	IsC2bSwitchOn *bool `json:"is_c2b_switch_on"`
	// 预售类型 0 全款预售 1SKU预售 2时效SKU现货+预售  3时效SKU预售
	PresellConfigLevel *int64 `json:"presell_config_level"`
	// 承诺发货时间
	DeliveryDelayDay *int64 `json:"delivery_delay_day"`
	// 0 预售结束后发货  1支付完成后发货
	PresellDeliveryType *int64 `json:"presell_delivery_type"`
	// 新sku字段 时效SKU
	SpecPricesV2 []SpecPricesV2Item `json:"spec_prices_v2"`
	// 商品类型
	ProductType *int64 `json:"product_type"`
	// 跨境：物流信息
	OpenLogisticsInfo *OpenLogisticsInfo `json:"open_logistics_info"`
}
