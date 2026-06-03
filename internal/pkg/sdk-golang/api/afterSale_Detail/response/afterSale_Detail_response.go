package afterSale_Detail_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AfterSaleDetailResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *AfterSaleDetailData `json:"data"`
}
type DeductionPriceDetailItem struct {
	// 金额明细
	PriceText string `json:"price_text"`
	// 金额
	Amount int64 `json:"amount"`
}
type AllEvidenceItem struct {
	// 证据类型  1:图片 2:视频 3:视频 4:文本 5:pdf
	Type int32 `json:"type"`
	// 资源地址
	Urls []string `json:"urls"`
	// 证据描述
	Desc string `json:"desc"`
}
type PropertyDetailsItem struct {
	// 属性名称
	Name string `json:"name"`
}
type InsuranceTagsItem struct {
	// 服务标签名称
	TagDetail string `json:"tag_detail"`
	// 服务标签英文代号
	TagDetailEn string `json:"tag_detail_en"`
	// 服务跳转地址
	TagLinkUrl string `json:"tag_link_url"`
}
type Town struct {
	// 县ID
	Id string `json:"id"`
	// 县名字
	Name string `json:"name"`
}
type ReturnAddress struct {
	// 省
	Province *Province `json:"province"`
	// 市
	City *City `json:"city"`
	// 县/区
	Town *Town `json:"town"`
	// 街道
	Street *Street `json:"street"`
	// 收件地址标志物
	Landmark string `json:"landmark"`
	// 详细地址
	Detail string `json:"detail"`
}
type Resend struct {
	// 物流单号
	TrackingNo string `json:"tracking_no"`
	// 物流公司名称
	CompanyName string `json:"company_name"`
	// 物流公司编码
	CompanyCode string `json:"company_code"`
	// 卖家填写补寄物流时间
	LogisticsTime int64 `json:"logistics_time"`
}
type CheckPrice struct {
	// 总价
	ActualAmount *ActualAmount `json:"actual_amount"`
	// 原价
	OriginAmount *OriginAmount `json:"origin_amount"`
	// 减数明细
	DeductionPriceDetail []DeductionPriceDetailItem `json:"deduction_price_detail"`
}
type SkuOrderInfosItem struct {
	// 称重品信息
	DeliveryInfos []DeliveryInfosItem `json:"delivery_infos"`
	// 仅即时零售使用, 判断是否是小时达, 次日达
	OrderPkgType string `json:"order_pkg_type"`
	// 商品属性，适用于外卖或即时零售场景下商品存在属性的情况
	CustomPropertyList []CustomPropertyListItem `json:"custom_property_list"`
	// sku单ID
	SkuOrderId int64 `json:"sku_order_id"`
	// 订单状态
	OrderStatus int64 `json:"order_status"`
	// 买家实付金额（分）
	PayAmount int64 `json:"pay_amount"`
	// 买家购买运费（分）
	PostAmount int64 `json:"post_amount"`
	// 订单件数
	ItemQuantity int64 `json:"item_quantity"`
	// 订单创建时间
	CreateTime int64 `json:"create_time"`
	// 购买税费（分）
	TaxAmount int64 `json:"tax_amount"`
	// 是否为跨境业务
	IsOverseaOrder int64 `json:"is_oversea_order"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 商品图片
	ProductImage string `json:"product_image"`
	// 商品标签
	Tags []TagsItem `json:"tags"`
	// 商品规格信息
	SkuSpec []SkuSpecItem `json:"sku_spec"`
	// 商家sku自定义编码
	ShopSkuCode string `json:"shop_sku_code"`
	// skuID
	SkuId int64 `json:"sku_id"`
	// sku商品总原价（不含优惠）
	ItemSumAmount int64 `json:"item_sum_amount"`
	// 商品实际支付金额
	SkuPayAmount int64 `json:"sku_pay_amount"`
	// 优惠总金额
	PromotionAmount int64 `json:"promotion_amount"`
	// 支付方式
	PayType int64 `json:"pay_type"`
	// 保险及其状态
	InsuranceTags []InsuranceTagsItem `json:"insurance_tags"`
	// 商品单对应的售后数量
	AfterSaleItemCount int64 `json:"after_sale_item_count"`
	// 赠品信息
	GivenSkuDetails []GivenSkuDetailsItem `json:"given_sku_details"`
}
type OrderInfo struct {
	// 核销状态文案
	OrderVoucherCodeStName string `json:"order_voucher_code_st_name"`
	// 核销状态码 1: "未核销", 2: "已核销", 3: "已过期",
	OrderVoucherCodeSt int64 `json:"order_voucher_code_st"`
	// 店铺单ID
	ShopOrderId int64 `json:"shop_order_id"`
	// sku单信息
	SkuOrderInfos []SkuOrderInfosItem `json:"sku_order_infos"`
}
type City struct {
	// 市ID
	Id string `json:"id"`
	// 市名字
	Name string `json:"name"`
}
type PostAddress struct {
	// 省
	Province *Province `json:"province"`
	// 市
	City *City `json:"city"`
	// 县
	Town *Town `json:"town"`
	// 地址详情，此字段已加密，使用前需要解密
	Detail string `json:"detail"`
	Encrypt_detail string `json:"encrypt_detail"`
	// 收件地址标志物
	Landmark string `json:"landmark"`
	// 街道
	Street *Street `json:"street"`
}
type ContactInfo struct {
	// 联系地址
	Addr *Addr `json:"addr"`
	// 姓名
	Name string `json:"name"`
	// 电话
	Tel string `json:"tel"`
	Encrypt_tel string `json:"encrypt_tel"`
}
type AfterSaleInfo struct {
	// 政府渠道补贴金额
	GovernmentChannelSubsidyAmount int64 `json:"government_channel_subsidy_amount"`
	// 达人回收运费补贴
	KolPostDiscountReturnAmount string `json:"kol_post_discount_return_amount"`
	// 商家回收运费补贴
	ShopPostDiscountReturnAmount int64 `json:"shop_post_discount_return_amount"`
	// 售后单ID
	AfterSaleId int64 `json:"after_sale_id"`
	// 售后状态：6-售后申请；7-售后退货中；8-【补寄\维修返回：售后待商家发货】；11-售后已发货；12-售后成功；13-【换货\补寄\维修返回：售后商家已发货，待用户收货】； 14-【换货\补寄\维修返回：售后用户已收货】 ；27-拒绝售后申请；28-售后失败；29-售后退货拒绝；51-订单取消成功；53-逆向交易已完成；
	AfterSaleStatus int64 `json:"after_sale_status"`
	// 售后状态文案
	AfterSaleStatusDesc string `json:"after_sale_status_desc"`
	// 退款方式
	RefundType int64 `json:"refund_type"`
	// 退款方式文案
	RefundTypeText string `json:"refund_type_text"`
	// 退款状态;1-待退款;2-退款中;3-退款成功;4退款失败;5追缴成功;
	RefundStatus int64 `json:"refund_status"`
	// 售后总金额（含运费）
	RefundTotalAmount int64 `json:"refund_total_amount"`
	// 售后运费
	RefundPostAmount int64 `json:"refund_post_amount"`
	// 退款补贴总金额（此字段只有退款成功后才会记录实际支付补贴回收的金额）
	RefundPromotionAmount int64 `json:"refund_promotion_amount"`
	// 售后单申请时间
	ApplyTime int64 `json:"apply_time"`
	// 售后单更新时间
	UpdateTime int64 `json:"update_time"`
	// 申请原因
	Reason string `json:"reason"`
	// 原因码；通过【afterSale/rejectReasonCodeList】接口获取
	ReasonCode int64 `json:"reason_code"`
	// 售后类型： 0-售后退货退款；1-售后仅退款；2-发货前退款；3-换货；4-系统取消；5-用户取消；6-价保；7-补寄；8-维修
	AfterSaleType int64 `json:"after_sale_type"`
	// 售后单类型文案
	AfterSaleTypeText string `json:"after_sale_type_text"`
	// 申请描述
	ReasonRemark string `json:"reason_remark"`
	// 买家申请退款图片凭证；仅支持图片，最大返回8张图片。
	Evidence []string `json:"evidence"`
	// 用户申请售后件数
	AfterSaleApplyCount int64 `json:"after_sale_apply_count"`
	// 用户需退回件数, 数值为用户申请售后件数 - 商家未发货件数
	NeedReturnCount int64 `json:"need_return_count"`
	// 平台需要回收的金额（分）
	DeductionAmount int64 `json:"deduction_amount"`
	// 作废的券ID
	DisableCouponId string `json:"disable_coupon_id"`
	// 平台优惠退回金额
	PlatformDiscountReturnAmount int64 `json:"platform_discount_return_amount"`
	// [字段无效]平台优惠退回状态，枚举：0:待退补贴；1:退补贴成功；2:退补贴失败
	PlatformDiscountReturnStatus int64 `json:"platform_discount_return_status"`
	// 达人优惠退回金额
	KolDiscountReturnAmount int64 `json:"kol_discount_return_amount"`
	// 达人优惠退回状态，枚举：0:待退补贴；1:退补贴成功；2:退补贴失败
	KolDiscountReturnStatus int64 `json:"kol_discount_return_status"`
	// 换货、补寄时的收货人名字（只有换货、补寄时，这个字段才会有值），此字段已加密，使用前需要解密
	PostReceiver string `json:"post_receiver"`
	Encrypt_post_receiver string `json:"encrypt_post_receiver"`
	// 换货、补寄时的收货人的联系电话（只有换货、补寄时，这个字段才会有值），此字段已加密，使用前需要解密
	PostTelSec string `json:"post_tel_sec"`
	Encrypt_post_tel_sec string `json:"encrypt_post_tel_sec"`
	// 换货、补寄时的收货四级地址（只有换货、补寄时，这个字段才会有值）
	PostAddress *PostAddress `json:"post_address"`
	// 物流异常风控编码
	RiskDecsisonCode int64 `json:"risk_decsison_code"`
	// 物流异常风控理由
	RiskDecsisonReason string `json:"risk_decsison_reason"`
	// 物流异常风控描述
	RiskDecsisonDescription string `json:"risk_decsison_description"`
	// 退货地址
	ReturnAddress *ReturnAddress `json:"return_address"`
	// 实际退款金额;单位：分（此字段只有退款成功后才会记录实际减去支付补贴回收的金额）
	RealRefundAmount int64 `json:"real_refund_amount"`
	// 买家是否收到货，0-表示未收到货；1-表示收到货
	GotPkg int64 `json:"got_pkg"`
	// 逾期时间
	StatusDeadline int64 `json:"status_deadline"`
	// 退货地址ID
	ReturnAddressId int64 `json:"return_address_id"`
	// 换货SKU信息
	ExchangeSkuInfo *ExchangeSkuInfo `json:"exchange_sku_info"`
	// 平台回收运费补贴
	PostDiscountReturnAmount int64 `json:"post_discount_return_amount"`
	// 运费优惠退回状态，枚举：0:待退补贴；1:退补贴成功；2:退补贴失败
	PostDiscountReturnStatus int64 `json:"post_discount_return_status"`
	// 部分退状态，0为全额退款，1为部分退
	PartType int64 `json:"part_type"`
	// 用户申请售后选择的二级原因标签
	ReasonSecondLabels []ReasonSecondLabelsItem `json:"reason_second_labels"`
	// 卡券商品申请退款的张数
	RefundVoucherNum int64 `json:"refund_voucher_num"`
	// 多次券商品申请退款的次数，对于单次券，此字段值与refund_voucher_num相同
	RefundVoucherTimes int64 `json:"refund_voucher_times"`
	// 退金币金额
	GoldCoinReturnAmount int64 `json:"gold_coin_return_amount"`
	// 退款失败文案
	RefundFailReason string `json:"refund_fail_reason"`
	// 售后标签
	AftersaleTags []AftersaleTagsItem `json:"aftersale_tags"`
	// 门店ID
	StoreId string `json:"store_id"`
	// 门店名称
	StoreName string `json:"store_name"`
	// 售后订单类型，枚举为-1(历史订单),1(商品单),2(店铺单)
	AfterSaleOrderType int64 `json:"after_sale_order_type"`
	// 售后打包费退款金额，单位：分，商家退给用户打包费后，该字段则有值；仅小时达店铺使用；
	RefundPackingChargeAmount int64 `json:"refund_packing_charge_amount"`
	// 商家优惠退回金额（包含供应商优惠退回金额）
	ShopDiscountReturnAmount int64 `json:"shop_discount_return_amount"`
	// 售后子类型：8001-以换代修
	AfterSaleSubType int64 `json:"after_sale_sub_type"`
	// 商家是否同意拒签后退款（0-默认值；1-同意拒签后退款；2-不同意拒签后退款；257-拒签上标；258-拒签下标）
	AgreeRefuseSign int64 `json:"agree_refuse_sign"`
	// 售后申请角色：1-买家；2-商家；3-客服；4-系统
	ApplyRole int64 `json:"apply_role"`
	// 退款成功时间，即平台最后原路退回金额到用户账户的成功时间，Unix时间戳：秒
	RefundTime int64 `json:"refund_time"`
	// 售后完结时间，完结时间是平台根据商品的类型，售后状态等综合判断生成，当售后单有完结时间返回时售后单不可再做做任何操作；未完结售后单的该字段值为0；Unix时间戳：秒
	AftersaleStatusToFinalTime int64 `json:"aftersale_status_to_final_time"`
	// 用户申请时的退货方式：0-未选择，1-用户上传物流，2-上门取件，4-用户到店退货，5-商家自有运力上门取货，6-平台运力上门取货，7-商家线下取货，10-到家场景下商家同意退货后默认为10，确定具体取货方式后流转到4、5、6。（具体以实际用户选择为准）
	ReturnMethod int64 `json:"return_method"`
	// 自动审核方式：1-发货前极速退；2-小助手自动同意退款；3-发货后极速退；4-闪电退货；5-跨境零秒退；6-云仓拦截自动退；7-小助手自动同意退货；8-小助手自动同意拒签后退款；9-商家代客填写卡片发起售后；10-治理未发货自动同意退款；11-治理已发货自动同意退款；12-商家快递拦截成功自动退款；13-质检商品免审核；14-协商方案自动同意退款；15-平台卡券自动同意退款；16-三方卡券自动同意退款；17-治理一审自动同意退货退款
	AutoAuditBits []int64 `json:"auto_audit_bits"`
	// 用户退货物流偏好
	ReturnBookInfo *ReturnBookInfo `json:"return_book_info"`
}
type StandardPrice struct {
	// 总价
	ActualAmount *ActualAmount `json:"actual_amount"`
	// 原价
	OriginAmount *OriginAmount `json:"origin_amount"`
	// 减数明细
	DeductionPriceDetail []DeductionPriceDetailItem `json:"deduction_price_detail"`
}
type RefundDetail struct {
	// 总价
	ActualAmount *ActualAmount `json:"actual_amount"`
	// 原价
	OriginAmount *OriginAmount `json:"origin_amount"`
	// 减数明细
	DeductionPriceDetail []DeductionPriceDetailItem `json:"deduction_price_detail"`
}
type PriceProtectionDetail struct {
	// 价保文案标题
	Title string `json:"title"`
	// 价保计算公式
	PriceProtectionFormulas string `json:"price_protection_formulas"`
	// 基准价
	StandardPrice *StandardPrice `json:"standard_price"`
	// 核准价
	CheckPrice *CheckPrice `json:"check_price"`
	// 退款明细
	RefundDetail *RefundDetail `json:"refund_detail"`
	// 钱款承担方
	RefundBearerList []RefundBearerListItem `json:"refund_bearer_list"`
	// 平台价保补贴商家金额进度状态，1表示成功
	PlatformToMerchantRefundStatus int64 `json:"platform_to_merchant_refund_status"`
	// 平台价保回收金额
	PlatformRecycleAmount int64 `json:"platform_recycle_amount"`
}
type OrderItem struct {
	// 1. 是合包发货  0：不是合包发货
	MergeDelivery int64 `json:"merge_delivery"`
	// 1. 是中转服务  0：不是中转服务
	IsTransit int64 `json:"is_transit"`
	// 1. 是偏远服务  0：不是偏远服务
	IsRemoteService int64 `json:"is_remote_service"`
	// 物流单号
	TrackingNo string `json:"tracking_no"`
	// 物流公司名称
	CompanyName string `json:"company_name"`
	// 物流公司编码
	CompanyCode string `json:"company_code"`
	// 物流状态到达时间
	LogisticsTime int64 `json:"logistics_time"`
	// 正向物流状态
	LogisticsState int64 `json:"logistics_state"`
	// 增值服务标签
	ValueAddedServices []ValueAddedServicesItem `json:"value_added_services"`
}
type ReturnBookInfo struct {
	// 上门取件精确日历：截止时间戳，单位秒
	BookTimeEndTs int64 `json:"book_time_end_ts"`
	// 上门取件精确日历：开始时间戳，单位秒
	BookTimeBeginTs int64 `json:"book_time_begin_ts"`
	// 寄回类型，  1-自行寄回， 2-上门取件，4-到店退货，5-商家自有运力上门取货，3-快递柜，8-驿站， 10-商家上门取货 【商家取货的统称：自有运力、平台运力、线下自取】
	ReturnType int32 `json:"return_type"`
	// 上门取件预约开始时间段
	BookTimeBegin string `json:"book_time_begin"`
	// 上门取件预约结束时间段
	BookTimeEnd string `json:"book_time_end"`
	// 上门取件地址信息
	ContactInfo *ContactInfo `json:"contact_info"`
	// 地址id
	SenderAddressId string `json:"sender_address_id"`
	// 驿站id
	StationId string `json:"station_id"`
}
type ArbitrateEvidenceTmpl struct {
	// 仲裁图片示例
	Images []string `json:"images"`
	// 仲裁描述
	Describe string `json:"describe"`
	// 示例截止时间
	DeadLine int64 `json:"dead_line"`
}
type AfterSaleServiceTag struct {
	// 售后服务标签
	AfterSaleServiceTag []AfterSaleServiceTagItem `json:"after_sale_service_tag"`
}
type ValueAddedServicesItem struct {
	// 标签编码
	Code string `json:"code"`
	// 标签名称
	Name string `json:"name"`
}
type SkuSpecItem struct {
	// 规格名称
	Name string `json:"name"`
	// 规格值
	Value string `json:"value"`
}
type AfterSaleDetailData struct {
	// 售后单及相关信息
	ProcessInfo *ProcessInfo `json:"process_info"`
	// 售后单关联订单信息
	OrderInfo *OrderInfo `json:"order_info"`
}
type Province struct {
	// 省ID
	Id string `json:"id"`
	// 省名字
	Name string `json:"name"`
}
type ExchangeSkuInfo struct {
	// 商品skuid
	SkuId string `json:"sku_id"`
	// 商品编码
	Code string `json:"code"`
	// 替换数量
	Num int64 `json:"num"`
	// 商家编号
	OutSkuId string `json:"out_sku_id"`
	// 区域库存仓ID
	OutWarehouseId string `json:"out_warehouse_id"`
	// sku外部供应商编码供应商ID
	SupplierId string `json:"supplier_id"`
	// 商品图片url
	Url string `json:"url"`
	// 商品名称
	Name string `json:"name"`
	// 换货商品的价格，单位分
	Price string `json:"price"`
	// sku规格信息
	SpecDesc string `json:"spec_desc"`
}
type AftersaleTagsItem struct {
	// 标签名称
	TagDetail string `json:"tag_detail"`
	// 标签关键字
	TagDetailEn string `json:"tag_detail_en"`
	// 标签悬浮文案的占位符定义
	TagDetailText string `json:"tag_detail_text"`
	// 标签跳转链接
	TagLinkUrl string `json:"tag_link_url"`
	// 标签悬浮文案的占位符定义
	Placeholder map[string]PlaceholderItem `json:"placeholder"`
}
type ArbitrateInfo struct {
	// 仲裁单id
	ArbitrateId string `json:"arbitrate_id"`
	// 仲裁状态
	ArbitrateStatus int64 `json:"arbitrate_status"`
	// 是否需要上传凭证
	IsRequiredEvidence bool `json:"is_required_evidence"`
	// 仲裁截止时间
	ArbitrateStatusDeadline string `json:"arbitrate_status_deadline"`
	// 仲裁原因
	ArbitrateOpinion string `json:"arbitrate_opinion"`
	// 仲裁结果
	ArbitrateConclusion int64 `json:"arbitrate_conclusion"`
	// 仲裁单创建时间
	ArbitrateCreateTime int64 `json:"arbitrate_create_time"`
	// 仲裁单更新时间
	ArbitrateUpdateTime int64 `json:"arbitrate_update_time"`
	// 凭证示例
	ArbitrateEvidenceTmpl *ArbitrateEvidenceTmpl `json:"arbitrate_evidence_tmpl"`
	// 仲裁证据
	ArbitrateEvidence *ArbitrateEvidence `json:"arbitrate_evidence"`
	// 仲裁责任方
	ArbitrateBlame int64 `json:"arbitrate_blame"`
}
type LogisticsInfo struct {
	// 买家退货物流信息
	Return *Return `json:"return"`
	// 卖家换货物流信息
	Exchange *Exchange `json:"exchange"`
	// 卖家发货正向物流信息
	Order []OrderItem `json:"order"`
	// 补寄物流
	Resend *Resend `json:"resend"`
}
type ActualAmount struct {
	// 金额明细
	PriceText string `json:"price_text"`
	// 金额
	Amount int64 `json:"amount"`
}
type DescKvsItem struct {
	// 行头
	Key string `json:"key"`
	// 内容
	Value string `json:"value"`
	// 是否高亮展示
	Highlight bool `json:"highlight"`
	// 额外提示
	Notice string `json:"notice"`
}
type ProcessInfo struct {
	// 售后单信息
	AfterSaleInfo *AfterSaleInfo `json:"after_sale_info"`
	// 仲裁信息
	ArbitrateInfo *ArbitrateInfo `json:"arbitrate_info"`
	// 售后标签
	AfterSaleServiceTag *AfterSaleServiceTag `json:"after_sale_service_tag"`
	// 物流信息
	LogisticsInfo *LogisticsInfo `json:"logistics_info"`
	// 售后备注
	AfterSaleShopRemarks []AfterSaleShopRemarksItem `json:"after_sale_shop_remarks"`
	// 价保详情
	PriceProtectionDetail *PriceProtectionDetail `json:"price_protection_detail"`
	// 协商记录
	RecordLogsList []RecordLogsListItem `json:"record_logs_list"`
}
type ReasonSecondLabelsItem struct {
	// 二级原因标签编码
	Code int64 `json:"code"`
	// 二级原因标签名称
	Name string `json:"name"`
}
type AfterSaleServiceTagItem struct {
	// 服务标签名称
	TagDetail string `json:"tag_detail"`
	// 服务标签英文代号
	TagDetailEn string `json:"tag_detail_en"`
	// 服务跳转地址
	TagLinkUrl string `json:"tag_link_url"`
}
type CustomPropertyListItem struct {
	// 属性详情列表
	PropertyDetails []PropertyDetailsItem `json:"property_details"`
	// 属性组名称
	GroupName string `json:"group_name"`
}
type GivenSkuDetailsItem struct {
	// 赠品个数
	ItemOrderNum int64 `json:"item_order_num"`
	// 赠品商家编码
	ShopSkuCode string `json:"shop_sku_code"`
	// 赠品商品ID
	ProductId string `json:"product_id"`
	// 赠品商品订单ID
	SkuOrderId string `json:"sku_order_id"`
	// 赠品名称
	ProductName string `json:"product_name"`
}
type Street struct {
	// 街道ID
	Id string `json:"id"`
	// 街道名字
	Name string `json:"name"`
}
type RefundBearerListItem struct {
	// 文案
	PriceText string `json:"price_text"`
	// 金额
	Amount int64 `json:"amount"`
}
type TagsItem struct {
	// 服务标签名称
	TagDetail string `json:"tag_detail"`
	// 服务标签英文代号
	TagDetailEn string `json:"tag_detail_en"`
	// 服务跳转地址
	TagLinkUrl string `json:"tag_link_url"`
}
type AfterSaleShopRemarksItem struct {
	// 备注关联的订单ID
	OrderId int64 `json:"order_id"`
	// 备注关联的售后ID
	AfterSaleId int64 `json:"after_sale_id"`
	// 备注内容
	Remark string `json:"remark"`
	// 创建时间
	CreateTime int64 `json:"create_time"`
	// 操作人
	Operator string `json:"operator"`
}
type Addr struct {
	// 省
	Province *Province `json:"province"`
	// 市
	City *City `json:"city"`
	// 区
	Town *Town `json:"town"`
	// 详细地址,此字段已加密，使用前需要解密
	Detail string `json:"detail"`
	Encrypt_detail string `json:"encrypt_detail"`
	// 街道
	Street *Street `json:"street"`
	// 地址ID
	AddressId string `json:"address_id"`
}
type ArbitrateEvidence struct {
	// 仲裁图片
	Images []string `json:"images"`
	// 仲裁图片描述
	Describe string `json:"describe"`
}
type Return struct {
	// 物流单号
	TrackingNo string `json:"tracking_no"`
	// 物流公司名称
	CompanyName string `json:"company_name"`
	// 物流公司编码
	CompanyCode string `json:"company_code"`
	// 买家填写退货物流时间
	LogisticsTime int64 `json:"logistics_time"`
}
type Exchange struct {
	// 物流单号
	TrackingNo string `json:"tracking_no"`
	// 物流公司名称
	CompanyName string `json:"company_name"`
	// 物流公司编码
	CompanyCode string `json:"company_code"`
	// 卖家填写换货物流时间
	LogisticsTime int64 `json:"logistics_time"`
}
type RecordLogsListItem struct {
	// 操作人
	Operator string `json:"operator"`
	// 操作时间
	Time string `json:"time"`
	// 操作内容
	Text string `json:"text"`
	// 图片列表
	Images []string `json:"images"`
	// 额外信息
	DescKvs []DescKvsItem `json:"desc_kvs"`
	// 动作
	Action string `json:"action"`
	// 角色；1-买家，2-商家，3-平台客服，4-系统
	Role int64 `json:"role"`
	// 所有类型凭证
	AllEvidence []AllEvidenceItem `json:"all_evidence"`
}
type DeliveryInfosItem struct {
	// g
	InfoUnit string `json:"info_unit"`
	// 220
	InfoValue string `json:"info_value"`
	// weight
	InfoType string `json:"info_type"`
}
type PlaceholderItem struct {
	// 占位符文案
	Text string `json:"text"`
	// 占位符跳转链接
	Url string `json:"url"`
}
type OriginAmount struct {
	// 金额明细
	PriceText string `json:"price_text"`
	// 金额
	Amount int64 `json:"amount"`
}
