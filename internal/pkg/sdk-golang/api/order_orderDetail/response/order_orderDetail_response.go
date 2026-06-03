package order_orderDetail_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderOrderDetailResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderOrderDetailData `json:"data"`
}
type SkuCargoFutureStock struct {
	// 库存扣减信息
	SupplySkuSnapshots []SupplySkuSnapshotsItem `json:"supply_sku_snapshots"`
}
type PrescriptionInfoListItem struct {
	// 问诊单
	ConsultationDetail *ConsultationDetail `json:"consultation_detail"`
	// 处方单
	PrescriptionDetail *PrescriptionDetail `json:"prescription_detail"`
}
type PromotionDetail struct {
	// 店铺优惠信息
	ShopDiscountDetail *ShopDiscountDetail `json:"shop_discount_detail"`
	// 平台优惠信息
	PlatformDiscountDetail *PlatformDiscountDetail `json:"platform_discount_detail"`
	// 达人优惠信息
	KolDiscountDetail *KolDiscountDetail `json:"kol_discount_detail"`
}
type PromotionPayAmountDetailsItem struct {
	// 优惠类型，mixed_charge-混合出资；GovernmentReduce-政府补贴。
	PayPromotionType string `json:"pay_promotion_type"`
	// 承担方。
	ShareCost *ShareCost_5_5 `json:"share_cost"`
}
type OrderRecycleInfo struct {
	// 追缴信息，目前包含立减+先享抵扣信息;
	PressForPaymentAfterwards *PressForPaymentAfterwards `json:"press_for_payment_afterwards"`
	// 旧机回收信息
	RecycleActualAmountInfo *RecycleActualAmountInfo `json:"recycle_actual_amount_info"`
	// 换新额外补贴（已下线，使用新机单上的shop_recycle_subsidy_amount_info 替代）
	ShopSubsidyAmountInfo *ShopSubsidyAmountInfo `json:"shop_subsidy_amount_info"`
	// 回收方式; 1-上门回收  2-邮寄回收
	RecycleMode int64 `json:"recycle_mode"`
	// 预约拆机开始时间戳，单位：秒
	AppointRecycleStartTime int64 `json:"appoint_recycle_start_time"`
	// 预约拆机结束时间戳，单位：秒
	AppointRecycleEndTime int64 `json:"appoint_recycle_end_time"`
	// 回收商品描述
	PronductDesc string `json:"pronduct_desc"`
	// 旧机预估价-单位：分
	QuotedPrice int64 `json:"quoted_price"`
	// 回收旧机单单号
	RecycleOrderId string `json:"recycle_order_id"`
	// 回收状态；1,101,201-已接单；2,102-分配师傅；202-分配快递；3,103-师傅上门签到；203-待质检；204-质检中；104,205-质检完成；105,206-用户确认；4,106-服务商审核；5,107,207-回收完成；99,199,299,399-取消;
	RecycleStatus int64 `json:"recycle_status"`
	// 回收状态
	RecycleStatusDesc string `json:"recycle_status_desc"`
	// 旧机回收价发放状态(使用send_status代替)
	SubsidyReceiveStatus string `json:"subsidy_receive_status"`
	// 回收服务商code
	SupplierCode string `json:"supplier_code"`
	// 服务商名称
	SupplierName string `json:"supplier_name"`
}
type ReverseExpressInfo struct {
	// 物流公司中文
	LogisticsCompanyName string `json:"logistics_company_name"`
	// 物流公司
	LogisticsCompany string `json:"logistics_company"`
	// 物流单号
	LogisticsCode string `json:"logistics_code"`
}
type FullDiscountInfoItem struct {
	// 营销优惠扩展字段instant_activity_id是商家营销活动id
	ExtraMap map[string]string `json:"extra_map"`
	// 商家活动ID
	CampaignActivityId string `json:"campaign_activity_id"`
	// 活动ID
	CampaignId int64 `json:"campaign_id"`
	// 活动类型：100-店铺满减活动 110-平台满减活动 150-立减活动
	CampaignType int64 `json:"campaign_type"`
	// 成本分摊
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
	// 活动名称
	CampaignName string `json:"campaign_name"`
	// 活动优惠金额（单位：分）
	CampaignAmount int64 `json:"campaign_amount"`
	// 活动子类型：0 店铺活动，1 平台活动
	CampaignSubType int64 `json:"campaign_sub_type"`
}
type ContractInfo struct {
	// 合约ID
	ContractId string `json:"contract_id"`
	// 办理合约的手机号
	MobileNo string `json:"mobile_no"`
	Encrypt_mobile_no string `json:"encrypt_mobile_no"`
}
type LogisticsInfoItem struct {
	// 该轨迹的退回单号（仅中转物流的退回单号信息）
	ReverseExpressInfo *ReverseExpressInfo `json:"reverse_express_info"`
	// 包裹是否为合包
	TransitMergeType string `json:"transit_merge_type"`
	// 即时零售订单使用平台运力时的配送应付金额；单位：分
	SpTotalPrice int64 `json:"sp_total_price"`
	// 即时零售订单使用平台运力时的配送实付金额；单位：分
	SpPrice int64 `json:"sp_price"`
	// 即时零售订单使用平台运力时的配送优惠金额；单位：分
	SpDiscountPrice int64 `json:"sp_discount_price"`
	// 物流单号
	TrackingNo string `json:"tracking_no"`
	// 物流公司
	Company string `json:"company"`
	// 发货时间，时间戳，秒
	ShipTime int64 `json:"ship_time"`
	// 包裹id
	DeliveryId string `json:"delivery_id"`
	// 物流公司名称
	CompanyName string `json:"company_name"`
	// 商品信息
	ProductInfo []ProductInfoItem `json:"product_info"`
	// 物流增值服务列表
	AddedServices []AddedServicesItem `json:"added_services"`
	// 骑手取件码；触发场景①商家：抖超小时达商家通过平台运力发货并选择需要取件码时，该字段有值返回。应用场景①骑手到店后报流水号，门店员工根据流水号【示例值：流水号#1】找到对应包裹。②门店员工与骑手核对取件码，核对无误后交货给骑手；
	HourUpPickupCode string `json:"hour_up_pickup_code"`
	// 保价金额，单位：分（小时达业务使用）
	GuaranteeAmount int64 `json:"guarantee_amount"`
}
type CampaignInfoItem struct {
	// 活动额外信息；例如{    "instant_custom_info": "{\"instant_open_id\":\"101010\"}"}
	ExtraMap map[string]string `json:"extra_map"`
	// 商家活动ID
	CampaignActivityId string `json:"campaign_activity_id"`
	// 活动ID
	CampaignId int64 `json:"campaign_id"`
	// 活动类型：7-限时特卖 10-定金预售 11-定金预售尾款部分 13-达人专属价 14-限时限量限人 120-拼团
	CampaignType int64 `json:"campaign_type"`
	// 成本分摊
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
	// 活动名称
	CampaignName string `json:"campaign_name"`
	// 活动金额（单位：分）
	CampaignAmount int64 `json:"campaign_amount"`
	// 活动子类型：0-店铺活动，1-平台活动
	CampaignSubType int64 `json:"campaign_sub_type"`
}
type QualityCheckInfo struct {
	// 质检结果  WATI_CHECT:待质检, CHECK_PASS:质检通过, CHECK_FAIL:质检不通过
	CheckResultCode string `json:"check_result_code"`
	// 质检异常信息描述文案
	CheckFailMsg string `json:"check_fail_msg"`
	// 重新送检截止时间
	ResendCheckTime int64 `json:"resend_check_time"`
}
type ActualReceiveAmountInfo struct {
	// 商家收入金额（单位：分）
	ActualReceiveAmount int64 `json:"actual_receive_amount"`
	// 商家收入金额明细项
	ActualReceiveAmountDetails []ActualReceiveAmountDetailsItem `json:"actual_receive_amount_details"`
}
type Town struct {
	// 名称
	Name string `json:"name"`
	// 地区ID
	Id string `json:"id"`
}
type KolDiscountDetail struct {
	// 优惠总金额（单位：分）
	TotalAmount int64 `json:"total_amount"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 活动优惠金额（单位：分）
	FullDiscountAmount int64 `json:"full_discount_amount"`
	// 优惠券信息
	CouponInfo []CouponInfoItem_5_5 `json:"coupon_info"`
	// 优惠活动信息
	FullDiscountInfo []FullDiscountInfoItem_5_5 `json:"full_discount_info"`
	// 红包金额（单位：分）
	RedpackAmount int64 `json:"redpack_amount"`
	// 红包信息
	RedpackInfo []RedpackInfoItem `json:"redpack_info"`
}
type PicItem struct {
	// id
	Id int64 `json:"id"`
	// url
	Url string `json:"url"`
}
type ShopOrderDetail struct {
	// 订单额外信息扩展
	OrderExtendList []OrderExtendListItem `json:"order_extend_list"`
	// 商家收入金额信息
	ActualReceiveAmountInfo *ActualReceiveAmountInfo `json:"actual_receive_amount_info"`
	// 推荐快递清单
	RecommendLogisticsList []RecommendLogisticsListItem `json:"recommend_logistics_list"`
	// 订单操作记录
	OrderOperateRecordList []OrderOperateRecordListItem `json:"order_operate_record_list"`
	// 运费补差金额单位：分
	MakeUpPostAmount int64 `json:"make_up_post_amount"`
	// 订单时效信息
	PromiseDetail *PromiseDetail `json:"promise_detail"`
	// 业务身份-来源
	BmpSource string `json:"bmp_source"`
	// 业务身份-垂类
	BmpVerticalMarket string `json:"bmp_vertical_market"`
	// 业务身份-卖家身份
	BmpSellerType string `json:"bmp_seller_type"`
	// 催单次数
	UrgeDeliverTimes int64 `json:"urge_deliver_times"`
	// 周期购信息（有值表示为周期购订单，空表示不是周期购订单）
	ShopPeriodPurchaseInfo *ShopPeriodPurchaseInfo `json:"shop_period_purchase_info"`
	// 订单关联的订单号，如维修单主商品关联的维修单订单号
	RelationShopOrderIds []string `json:"relation_shop_order_ids"`
	// 额外优惠信息；字段文档：https://bytedance.larkoffice.com/docx/G7UodvZY0ojuF9x7fL7ceVF9nSe
	ExtraPromotionAmountDetail []ExtraPromotionAmountDetailItem `json:"extra_promotion_amount_detail"`
	// 换新额外补贴
	ShopRecycleSubsidyAmountInfo *ShopRecycleSubsidyAmountInfo `json:"shop_recycle_subsidy_amount_info"`
	// 订单的一些c端标签
	OrderTag map[string]string `json:"order_tag"`
	// 接单状态
	FulfillStatus string `json:"fulfill_status"`
	// 即时零售订单对应总店id
	OriginalShopId int64 `json:"original_shop_id"`
	// 抖音月付信息
	FreeInterest *FreeInterest `json:"free_interest"`
	// 小时达相关订单时效样式：RemindMerchantStockingTime：1701303000（秒级时间戳）相关枚举及含义：RemindMerchantStockingTime-提醒商家开始备货时间；LatestStockingCompletionTime --最晚备货完成时间；RemindMerchantDeliveryTime--提醒商家发配送时间；EtaDuration--eta时长（小时达）
	ReceiptTimeMap map[string]string `json:"receipt_time_map"`
	// 订单中转信息
	ConsolidateInfo *ConsolidateInfo `json:"consolidate_info"`
	// 达人抖音号，样品订单场景下才会返回的申样达人信息；
	AwemeId string `json:"aweme_id"`
	// 达人抖音头像，样品订单场景下才会返回的申样达人信息；
	UserIcon string `json:"user_icon"`
	// 达人抖音昵称，样品订单场景下才会返回的申样达人信息；
	UserNickName string `json:"user_nick_name"`
	// 用户选择的预约送达日期
	ReceiptDate string `json:"receipt_date"`
	// 建议商家最晚发货时间，秒
	RecommendEndShipTime int64 `json:"recommend_end_ship_time"`
	// 建议商家最早发货时间，秒
	RecommendStartShipTime int64 `json:"recommend_start_ship_time"`
	// 回收单信息
	OrderRecycleInfo *OrderRecycleInfo `json:"order_recycle_info"`
	// 跨境后置备案订单税费是否未计算完成 0-计算完成 1-未计算完成
	TaxAmountNotComeOut int64 `json:"tax_amount_not_come_out"`
	// 小时达订单的接单状态 0-未接单；1-已接单；2-超时取消，或商家取消
	AcceptOrderStatus int64 `json:"accept_order_status"`
	// 店铺id，抖店平台生成，平台下唯一；
	ShopId int64 `json:"shop_id"`
	// 商户名称
	ShopName string `json:"shop_name"`
	// 抖音小程序ID
	OpenId string `json:"open_id"`
	// 店铺父订单号，抖店平台生成，平台下唯一；
	OrderId string `json:"order_id"`
	// 订单层级，主订单是2级
	OrderLevel int64 `json:"order_level"`
	// 【业务来源】 1  -鲁班 2  -小店 3  -好好学习 4  -ev 5  -虚拟 6  -建站 7  -平台卡券 8  -玉石 9  -ez 10 -ep 11 -三方卡券 12 -服务市场 13 - EP 视频课 14 - EP 直播课 21 -跨境BBC 22 -跨境BC 23 -跨境CC|UPU 24 -手机充值 25 -拍卖保证金 26 -懂车帝抵扣券 27 -懂车帝返现券 28 -离岛免税 29 -海南会员购 30 -抽奖 31 -清北-企业代付 32 -抖+券 33 -联盟寄样 49 -刀剑 53 -通信卡 66 -加油包 76 -大闸蟹 99 -保险 101-三方卡券卡密模式 102-小店海外 108-上门取件收款
	Biz int64 `json:"biz"`
	// 业务来源描述
	BizDesc string `json:"biz_desc"`
	// 【订单类型】 0、普通订单 2、虚拟商品订单 4、电子券（poi核销） 5、三方核销  6、服务市场
	OrderType int64 `json:"order_type"`
	// 订单类型描述
	OrderTypeDesc string `json:"order_type_desc"`
	// 交易类型，不推荐使用，已废弃。请使用sku_order_list.trade_type即可。
	TradeType int64 `json:"trade_type"`
	// 交易类型描述,不推荐使用，已废弃。请使用sku_order_list. trade_type_desc下即可。
	TradeTypeDesc string `json:"trade_type_desc"`
	// 订单状态1 待确认/待支付（订单创建完毕）105 已支付 2 备货中 101 部分发货 3 已发货（全部发货）4 已取消5 已完成（已收货）
	OrderStatus int64 `json:"order_status"`
	// 订单状态描述
	OrderStatusDesc string `json:"order_status_desc"`
	// 主流程状态，1 待确认/待支付（订单创建完毕）103 部分支付105 已支付2 备货中101 部分发货3 已发货（全部发货）4 已取消5 已完成（已收货）21 发货前退款完结22 发货后退款完结39 收货后退款完结
	MainStatus int64 `json:"main_status"`
	// 主流程状态描述
	MainStatusDesc string `json:"main_status_desc"`
	// 支付时间，时间戳，秒
	PayTime int64 `json:"pay_time"`
	// 订单过期时间，时间戳，秒
	OrderExpireTime int64 `json:"order_expire_time"`
	// 订单完成时间，时间戳，秒
	FinishTime int64 `json:"finish_time"`
	// 下单时间，时间戳，秒
	CreateTime int64 `json:"create_time"`
	// 订单更新时间，时间戳，秒
	UpdateTime int64 `json:"update_time"`
	// 取消原因
	CancelReason string `json:"cancel_reason"`
	// 买家留言
	BuyerWords string `json:"buyer_words"`
	// 商家备注
	SellerWords string `json:"seller_words"`
	// 【下单端】 0、站外 1、火山 2、抖音 3、头条 4、西瓜 5、微信 6、值点app 7、头条lite 8、懂车帝 9、皮皮虾 11、抖音极速版 12、TikTok 13、musically 14、穿山甲 15、火山极速版 16、服务市场 26、番茄小说 27、UG教育营销电商平台 28、Jumanji 29、电商SDK
	BType int64 `json:"b_type"`
	// 下单端描述
	BTypeDesc string `json:"b_type_desc"`
	// 【下单场景】 0、未知 1、app内-原生 2、app内-小程序 3、H5 13、电商SDK-头条 35、电商SDK-头条lite
	SubBType int64 `json:"sub_b_type"`
	// 下单场景描述
	SubBTypeDesc string `json:"sub_b_type_desc"`
	// 具体某个小程序的ID
	AppId int64 `json:"app_id"`
	// 额；7=无需支付（0元单）；8=DOU分期（信用支付）；9=新卡支付；12=先用后付；16=收银台支付
	PayType int64 `json:"pay_type"`
	// 支付渠道的流水号
	ChannelPaymentNo string `json:"channel_payment_no"`
	// 订单金额（单位：分）
	OrderAmount int64 `json:"order_amount"`
	// 支付金额（单位：分）
	PayAmount int64 `json:"pay_amount"`
	// 快递费（单位：分）
	PostAmount int64 `json:"post_amount"`
	// 运费险金额（单位：分）
	PostInsuranceAmount int64 `json:"post_insurance_amount"`
	// 改价金额变化量（单位：分）
	ModifyAmount int64 `json:"modify_amount"`
	// 改价运费金额变化量（单位：分）
	ModifyPostAmount int64 `json:"modify_post_amount"`
	// 订单优惠总金额（单位：分） = 店铺优惠金额 + 平台优惠金额 + 达人优惠金额
	PromotionAmount int64 `json:"promotion_amount"`
	// 店铺优惠金额（单位：分），属于店铺的优惠活动、优惠券、红包的总优惠金额
	PromotionShopAmount int64 `json:"promotion_shop_amount"`
	// 平台优惠金额（单位：分），属于平台的优惠活动、优惠券、红包的总优惠金额
	PromotionPlatformAmount int64 `json:"promotion_platform_amount"`
	// 商家承担金额（单位：分），订单参与活动和优惠中商家承担部分的总金额
	ShopCostAmount int64 `json:"shop_cost_amount"`
	// 平台承担金额（单位：分），订单参与活动和优惠中平台+作者（达人）承担部分的总金额,包含作者（达人）承担金额：platform_cost_amount = only_platform_cost_amount + author_cost_amount
	PlatformCostAmount int64 `json:"platform_cost_amount"`
	// 达人优惠金额（单位：分），属于达人的优惠活动、优惠券、红包的总优惠金额
	PromotionTalentAmount int64 `json:"promotion_talent_amount"`
	// 支付优惠金额（单位：分），支付渠道上的优惠金额
	PromotionPayAmount int64 `json:"promotion_pay_amount"`
	// 收件人电话
	PostTel string `json:"post_tel"`
	Encrypt_post_tel string `json:"encrypt_post_tel"`
	// 收件人姓名
	PostReceiver string `json:"post_receiver"`
	Encrypt_post_receiver string `json:"encrypt_post_receiver"`
	// 收件人地址
	PostAddr *PostAddr `json:"post_addr"`
	// 预计发货时间，时间戳，秒
	ExpShipTime int64 `json:"exp_ship_time"`
	// 发货时间，时间戳，秒
	ShipTime int64 `json:"ship_time"`
	// 物流信息
	LogisticsInfo []LogisticsInfoItem `json:"logistics_info"`
	// 优惠信息，用户下单过程中享受到的优惠
	PromotionDetail *PromotionDetail `json:"promotion_detail"`
	// 订单详情信息，一个主订单下有可以最大返回40个子订单
	SkuOrderList []SkuOrderListItem `json:"sku_order_list"`
	// 插旗信息：0-灰 1-紫 2-青 3-绿 4-橙 5-红
	SellerRemarkStars int64 `json:"seller_remark_stars"`
	// 定金预售阶段单
	OrderPhaseList []OrderPhaseListItem `json:"order_phase_list"`
	// 开放平台地址id
	OpenAddressId string `json:"open_address_id"`
	// 用户唯一id，规则详见：https://op.jinritemai.com/docs/guide-docs/1368/4416
	DoudianOpenId string `json:"doudian_open_id"`
	// 商品序列号（IMEI码或SN码）
	SerialNumberList []string `json:"serial_number_list"`
	// 红包优惠金额（单位：分）
	PromotionRedpackAmount int64 `json:"promotion_redpack_amount"`
	// 平台红包优惠金额（单位：分），属于平台的红包的优惠金额
	PromotionRedpackPlatformAmount int64 `json:"promotion_redpack_platform_amount"`
	// 达人红包优惠金额（单位：分），属于达人的红包的优惠金额
	PromotionRedpackTalentAmount int64 `json:"promotion_redpack_talent_amount"`
	// 用户证件信息
	UserIdInfo *UserIdInfo `json:"user_id_info"`
	// 预约发货时间
	AppointmentShipTime int64 `json:"appointment_ship_time"`
	// 懂车帝购车信息
	DCarShopBizData *DCarShopBizData `json:"d_car_shop_biz_data"`
	// 店铺单标签
	ShopOrderTagUi []ShopOrderTagUiItem `json:"shop_order_tag_ui"`
	// 总优惠金额（单位：分），total_promotion_amount = promotion_amount + post_promotion_amount
	TotalPromotionAmount int64 `json:"total_promotion_amount"`
	// 运费原价（单位：分），post_origin_amount = post_amount + post_promotion_amount
	PostOriginAmount int64 `json:"post_origin_amount"`
	// 运费优惠金额（单位：分）
	PostPromotionAmount int64 `json:"post_promotion_amount"`
	// 用户特征标签；
	UserTagUi []UserTagUiItem `json:"user_tag_ui"`
	// 作者（达人）承担金额（单位：分），订单参与活动和优惠中作者（达人）承担部分的总金额
	AuthorCostAmount int64 `json:"author_cost_amount"`
	// 仅平台承担金额（单位：分），订单参与活动和优惠中平台承担部分的总金额
	OnlyPlatformCostAmount int64 `json:"only_platform_cost_amount"`
	// 履约时效信息(json串)
	PromiseInfo string `json:"promise_info"`
	// 收件人姓名（脱敏后）
	MaskPostReceiver string `json:"mask_post_receiver"`
	// 收件人电话（脱敏后），电商脱敏格式为1********35，即时零售脱敏格式为*******1035(后四位明文)
	MaskPostTel string `json:"mask_post_tel"`
	// 收件人地址（脱敏后）
	MaskPostAddr *MaskPostAddr `json:"mask_post_addr"`
	// 买家收货地址经纬度信息，高德坐标系；
	UserCoordinate *UserCoordinate `json:"user_coordinate"`
	// 预约配送单最早送达时间，Unix时间戳：秒；当early_arrival=false时使用，仅小时达业务返回；
	EarliestReceiptTime int64 `json:"earliest_receipt_time"`
	// 预约配送单最晚送达时间，Unix时间戳：秒；当early_arrival=false时使用，仅小时达业务返回；
	LatestReceiptTime int64 `json:"latest_receipt_time"`
	// 是否尽快送达，true-是（配合target_arrival_time字段使用），false-否（配合earliest_receipt_time和latest_receipt_time使用），仅小时达业务返回；
	EarlyArrival bool `json:"early_arrival"`
	// 尽快送达的时间点，Unix时间戳：秒；当early_arrival=true时使用，仅小时达业务返回；
	TargetArrivalTime int64 `json:"target_arrival_time"`
	// 打包费，单位：分；
	PackingAmount int64 `json:"packing_amount"`
	// 门店流水号，日内按下单时间从“1”递增生成，仅小时达业务返回；（注：商家自定义小票需加上日期信息，格式为“日期-流水号”，例如“18-1”，便于骑手取货）
	SupermarketOrderSerialNo string `json:"supermarket_order_serial_no"`
	// 税费(子单税费之和)
	TaxAmount int64 `json:"tax_amount"`
	// 地址标签列表
	AddressTagUi []AddressTagUiItem `json:"address_tag_ui"`
	// 脱敏下单人手机号（鲜花订单），电商脱敏格式为1********35，即时零售脱敏格式为*******1035(后四位明文)
	MaskPayTel string `json:"mask_pay_tel"`
	// 下单人手机号（鲜花订单）
	PayTel string `json:"pay_tel"`
	Encrypt_pay_tel string `json:"encrypt_pay_tel"`
	// 贺卡文字(鲜花订单)
	GreetWords string `json:"greet_words"`
}
type ShopOrderTagUiItem struct {
	// 标签 hover 信息
	HoverText string `json:"hover_text"`
	// 标签额外信息
	Extra string `json:"extra"`
	// 标签key
	Key string `json:"key"`
	// 标签名称
	Text string `json:"text"`
	// 帮助文档
	HelpDoc string `json:"help_doc"`
}
type RecommendLogisticsListItem_4_4 struct {
	// 该物流最晚揽收时间（秒级时间戳，精确到分钟）
	LogisticsCollectionEndTime int64 `json:"logistics_collection_end_time"`
	// 该物流承诺送达时间（秒级时间戳，精确到分钟）
	LogisticsLatestReceiptTime int64 `json:"logistics_latest_receipt_time"`
	// 该订单推荐使用的物流公司code
	LogisticsCompanyCode string `json:"logistics_company_code"`
	// 该订单推荐使用的物流公司名称
	LogisticsCompanyName string `json:"logistics_company_name"`
	// 快递产品类型type
	LogisticsCompanyProductType string `json:"logistics_company_product_type"`
	// 快递产品类型名称;
	LogisticsCompanyProductTypeName string `json:"logistics_company_product_type_name"`
	// 该推荐快递下，建议最晚发货时间时间
	LogisticsRecommendEndShipTime int64 `json:"logistics_recommend_end_ship_time"`
	// 该推荐快递下，建议最早发货时间时间
	LogisticsRecommendStartShipTime int64 `json:"logistics_recommend_start_ship_time"`
}
type InventoryListItem struct {
	// 外部仓id/编码
	OutWarehouseId string `json:"out_warehouse_id"`
	// 组
	Group string `json:"group"`
	// 仓库子类型
	WarehouseSubType int64 `json:"warehouse_sub_type"`
	// 仓类型：0-商家仓，1-跨境仓，2-云仓
	WarehouseType int64 `json:"warehouse_type"`
	// 库存变更数量
	Count int64 `json:"count"`
	// 仓名称
	WarehouseName string `json:"warehouse_name"`
	// 库存类型描述
	InventoryTypeDesc string `json:"inventory_type_desc"`
	// 库存类型，普通库存/区域库存
	InventoryType int64 `json:"inventory_type"`
	// 仓id
	WarehouseId string `json:"warehouse_id"`
}
type SkuOrderTagUiItem struct {
	// 标签key
	Key string `json:"key"`
	// 标签文案
	Text string `json:"text"`
	// 标签备注文案
	HoverText string `json:"hover_text"`
	// 标签类型，如颜色
	TagType string `json:"tag_type"`
	// 帮助文档
	HelpDoc string `json:"help_doc"`
	// 排序
	Sort int64 `json:"sort"`
	// 标签其他信息
	Extra string `json:"extra"`
}
type Detail struct {
	// 定制图片信息
	Pic []PicItem `json:"pic"`
	// 定制文案信息
	Text []TextItem `json:"text"`
	// 额外信息
	Extra string `json:"extra"`
}
type SkuCustomizationInfoItem struct {
	// 定制详情
	Detail *Detail `json:"detail"`
}
type AmountCompositionInfo struct {
	// 平台出资
	PlatformCost int64 `json:"platform_cost"`
	// 商家出资
	ShopCost int64 `json:"shop_cost"`
}
type ShopSubsidyAmountInfo struct {
	// 补贴渠道；1-返金额，2-平台红包
	SubsidyChannel int64 `json:"subsidy_channel"`
	// 补贴发送状态; 1-待发送 2-发送中 3-已发送
	SubsidyStatus int64 `json:"subsidy_status"`
	// 补贴金额，单位：分
	SubsidyaAmount int64 `json:"subsidya_amount"`
}
type ServiceInfo struct {
	// 完成时间
	ServiceFinishTime int64 `json:"service_finish_time"`
	// 上门签到时间
	SignInTime int64 `json:"sign_in_time"`
	// 上门安装预约时间
	AppointmentSliceTime *AppointmentSliceTime `json:"appointment_slice_time"`
	// 服务单状态描述
	ServiceStatusDesc string `json:"service_status_desc"`
	// 服务单状态
	ServiceStatus int64 `json:"service_status"`
}
type OrderOperateRecordListItem struct {
	// 订单操作时间
	OperateTime string `json:"operate_time"`
	// 操作人，买家，商家
	Operator string `json:"operator"`
	// 操作记录类型
	OpCode int64 `json:"op_code"`
	// 订单操作记录唯一ID
	OpId string `json:"op_id"`
}
type SkuSpecsItem struct {
	// 规格名称
	Name string `json:"name"`
	// 规格值
	Value string `json:"value"`
}
type BatchDetailItem struct {
	// 扣减库存数量
	StockNum int64 `json:"stock_num"`
	// 时效批次: X天后返货，只有库存类型为预售，该字段有意义。
	ArriveTime int64 `json:"arrive_time"`
	// 时效类型：0-绝对时效，1-相对时效，只有库存类型为预售该字段有意义，目前只有相对时效
	ArriveTimeType int64 `json:"arrive_time_type"`
	// 库存类型：现货/0、预售/1
	InventoryType int64 `json:"inventory_type"`
	// 库存批次信息
	BatchId int64 `json:"batch_id"`
}
type SupplySkuSnapshotsItem struct {
	// 库存扣减信息列表
	BatchDetail []BatchDetailItem `json:"batch_detail"`
	// 0-现货库存，1-预售库存
	InventoryType int64 `json:"inventory_type"`
}
type UserTagUiItem struct {
	// 标签key；【key: user_profile_buy_frequency = text: 服务优先】；【key: user_profile_buy_frequency = text: 高频购买 】；【key: user_profile_shop_customer_type = text: 店铺新客 】；【key:user_profile_shop_customer_type = text:店铺老客】；
	Key string `json:"key"`
	// 标签名称；【key: user_profile_buy_frequency = text: 服务优先】；【key: user_profile_buy_frequency = text: 高频购买 】；【key: user_profile_shop_customer_type = text: 店铺新客 】；【key:user_profile_shop_customer_type = text:店铺老客】；
	Text string `json:"text"`
}
type ActualReceiveAmountDetailsItem struct {
	// 明细项类型：1: 消费者实付金额2: 平台承担优惠金额3: 达人承担优惠金额4: 三方平台承担优惠金额5: 服务商承担优惠
	Type int32 `json:"type"`
	// 明细项金额（单位：分）
	Amount int64 `json:"amount"`
}
type ShareCost struct {
	// 达人承担；分
	AuthorCost int64 `json:"author_cost"`
	// 商家承担；分
	ShopCost int64 `json:"shop_cost"`
	// 平台承担；分
	PlatformCost int64 `json:"platform_cost"`
}
type RelationOrder struct {
	// 核销券码
	WriteOffNo string `json:"write_off_no"`
	// 关联店铺单订单id
	RelationOrderId string `json:"relation_order_id"`
}
type PlatformDiscountDetail_5_5 struct {
	// 官方立减金额（单位：分）
	OfficialDeductionAmount int64 `json:"official_deduction_amount"`
	// 购物补贴抵扣金额（单位：分）
	AllowanceAmount int64 `json:"allowance_amount"`
	// 余额抵扣金额（单位：分）
	UserBalanceAmount int64 `json:"user_balance_amount"`
	// 优惠总金额（单位：分）
	TotalAmount int64 `json:"total_amount"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 活动优惠金额（单位：分）
	FullDiscountAmount int64 `json:"full_discount_amount"`
	// 优惠券信息
	CouponInfo []CouponInfoItem_6_6 `json:"coupon_info"`
	// 优惠活动信息
	FullDiscountInfo []FullDiscountInfoItem_6_6 `json:"full_discount_info"`
	// 红包金额（单位：分）
	RedpackAmount int64 `json:"redpack_amount"`
	// 红包信息
	RedpackInfo []RedpackInfoItem `json:"redpack_info"`
	// 金币抵扣金额（单位：分）
	GoldCoinAmount int64 `json:"gold_coin_amount"`
}
type DCarShopBizData struct {
	// 选择的门店ID
	PoiId string `json:"poi_id"`
	// 选择的门店名称
	PoiName string `json:"poi_name"`
	// 选择的门店地址
	PoiAddr string `json:"poi_addr"`
	// 选择的门店电话
	PoiTel string `json:"poi_tel"`
	// 权益信息
	CouponRight []CouponRightItem `json:"coupon_right"`
	// 选择的门店所在省
	PoiPname string `json:"poi_pname"`
	// 选择的门店所在市
	PoiCityName string `json:"poi_city_name"`
	// 选择的门店所在区县
	PoiAdname string `json:"poi_adname"`
}
type UserCoordinate struct {
	// 买家收货地址经度信息，高德坐标系；
	UserCoordinateLongitude string `json:"user_coordinate_longitude"`
	// 买家收货地址纬度信息，高德坐标系；
	UserCoordinateLatitude string `json:"user_coordinate_latitude"`
}
type PromiseTimeDetail struct {
	// promise_type=2，承诺送达订单的最晚建议发货时间；
	PromiseRecommendEndShipTime int64 `json:"promise_recommend_end_ship_time"`
	// promise_type=1或者6有值，承诺发货时效订单/预约发货时效订单的最晚发货时间
	PromiseExpShipTime int64 `json:"promise_exp_ship_time"`
	// promise_type=2或者5，承诺送达时效订单/预约送达时效订单的最晚送达时间；
	PromiseLatestReceiptTime int64 `json:"promise_latest_receipt_time"`
	// 预约发货时间，promise_type=1或者6，消费者预约或者商家通过协商工具修改最晚发货时效后存在该值
	PromiseAppointmentShipTime int64 `json:"promise_appointment_ship_time"`
	// promise_type=2，承诺送达时效订单的最早建议发货时间；
	PromiseRecommendStartShipTime int64 `json:"promise_recommend_start_ship_time"`
}
type BatchAndTraceCodeListItem struct {
	// 溯源码code
	TraceCode string `json:"trace_code"`
	// 批次code
	BatchCode string `json:"batch_code"`
}
type SerialNoListItem struct {
	// SN/69/IMEI码类型。
	Key string `json:"key"`
	// SN/69/IMEI码的值。
	Value []string `json:"value"`
}
type SerialNoListItem_5 struct {
	// 2-绑定操作的序列号；3-发货回传的序列号；4-快递小哥拍照胡穿的物流单号。如果之前在商家后台绑定了SN/69/IMEI码，并且发货的时候重新回传了SN/69/IMEI码，则会覆盖绑定的SN/69/IMEI码，接口返回的绑定阶段的SN/69/IMEI码和发货阶段返回的值一致。
	Stage int64 `json:"stage"`
	// 每个阶段关联的SN/69/IMEI码信息。
	SerialNoList []SerialNoListItem `json:"serial_no_list"`
}
type SkuPeriodPurchaseInfo struct {
	// 当前期数
	PeriodNum int64 `json:"period_num"`
	// 类型（5预约送达，6预约发货）
	DeliverType int64 `json:"deliver_type"`
	// 预约时间（时间戳，单位秒）
	DeliverTime int64 `json:"deliver_time"`
}
type ShellOrderInfo struct {
	// 实际销售公司名称
	SupplierShopName string `json:"supplier_shop_name"`
	// 实际销售公司 id
	SupplierShopId string `json:"supplier_shop_id"`
}
type SkuMedicalTraceInfo struct {
	// 需要上传溯源码的件数
	ItemCount int64 `json:"item_count"`
	// 是否需要上传溯源码
	NeedTraceInfo bool `json:"need_trace_info"`
	// 是否强制校验需要上传溯源码必填
	StrongCheck bool `json:"strong_check"`
	// 溯源码信息
	TraceInfo *TraceInfo `json:"trace_info"`
}
type SerialNoInfo struct {
	// 绑定/发货操作回传/快递回传的SN/69/IMEI码信息。
	SerialNoList []SerialNoListItem_5 `json:"serial_no_list"`
	// 发货需要回传的SN/69/IMEI码信息；SN-SN码，IMEI-IMEI码，69 - 69码;
	ShippingNeedCheckCodes []string `json:"shipping_need_check_codes"`
	// 物流公司及商家编码，key是物流公司code,value是商家编码;
	AllLpShopCode map[string]string `json:"all_lp_shop_code"`
	// 所有的物流公司及微派code,key是物流公司code,value是微派code;
	AllLpTaskCode map[string]string `json:"all_lp_task_code"`
	// 可用的物流公司列表，国补订单发货时使用的物流公司code应该在该物流公司列表内。
	AvailableLp []string `json:"available_lp"`
	// 商品发布时绑定在该sku上的69码。
	SkuBind69Codes []string `json:"sku_bind_69_codes"`
	// true-需要，false-不需要。
	NeedPhoto bool `json:"need_photo"`
}
type LowPriceProduct struct {
	// 规格
	SkuSpecs []SkuSpecsItem `json:"sku_specs"`
	// 聚合商品id
	ProductId int64 `json:"product_id"`
	// 聚合商品skuid
	SkuId int64 `json:"sku_id"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 聚合商品图片
	Img string `json:"img"`
}
type SkuYunWarehouseInfoItem struct {
	// 预分配云仓信息
	PreAllocatedYunWarehouseInfo *PreAllocatedYunWarehouseInfo `json:"pre_allocated_yun_warehouse_info"`
	// 云仓实仓信息
	RealYunWarehouseInfo *RealYunWarehouseInfo `json:"real_yun_warehouse_info"`
	// 当前业务阶段 1-未分配 2-已分配未发货 2-已发货
	CurrentBusinessStage int64 `json:"current_business_stage"`
	// 该仓对应的skuId
	SkuId string `json:"sku_id"`
	// sku数量
	SkuCount int64 `json:"sku_count"`
}
type ExtraPromotionAmountDetailItem struct {
	// 活动类型
	InstantDiscountType int64 `json:"instant_discount_type"`
	// 平台活动id
	PromotionId string `json:"promotion_id"`
	// 商家活动信息
	ExtraMap map[string]string `json:"extra_map"`
	// 满20减元运费
	PromotionDesc string `json:"promotion_desc"`
	// 金额承担方
	ShareCost *ShareCost `json:"share_cost"`
	// freight_discount--运费优惠foodtakeout_shop_freight_discount--外卖运费优惠
	PromotionType string `json:"promotion_type"`
}
type AddedServicesItem struct {
	// 标签key
	Key string `json:"key"`
	// 物流服务名称
	Text string `json:"text"`
	// 抖音大学介绍地址
	HelpDoc string `json:"help_doc"`
	// 业务跳转地址
	Url string `json:"url"`
}
type CouponInfoItem struct {
	// 营销优惠扩展字段instant_activity_id是商家营销活动id
	ExtraMap map[string]string `json:"extra_map"`
	// 优惠券所属商家活动ID
	CouponActivityId string `json:"coupon_activity_id"`
	// 订单优惠券ID
	CouponId int64 `json:"coupon_id"`
	// 券类型；21-商家折扣券；22-商家直减券；23-商家满减券 ；41-单品折扣券；42-单品直减券；43 单品满减券；
	CouponType int64 `json:"coupon_type"`
	// 券批次ID
	CouponMetaId string `json:"coupon_meta_id"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 券名称
	CouponName string `json:"coupon_name"`
	// 成本分摊
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
}
type CouponInfoItem_5_5 struct {
	// 营销优惠扩展字段instant_activity_id是商家营销活动id
	ExtraMap map[string]string `json:"extra_map"`
	// 订单优惠券ID
	CouponId int64 `json:"coupon_id"`
	// 券类型;52-达人直减券;53-达人满减券;
	CouponType int64 `json:"coupon_type"`
	// 券批次ID
	CouponMetaId string `json:"coupon_meta_id"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 券名称
	CouponName string `json:"coupon_name"`
	// 成本分摊
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
}
type PlatformDiscountDetail struct {
	// 官方立减金额（单位：分）
	OfficialDeductionAmount int64 `json:"official_deduction_amount"`
	// 购物补贴抵扣金额（单位：分）
	AllowanceAmount int64 `json:"allowance_amount"`
	// 余额抵扣金额（单位：分）
	UserBalanceAmount int64 `json:"user_balance_amount"`
	// 优惠总金额（单位：分）
	TotalAmount int64 `json:"total_amount"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 活动优惠金额（单位：分）
	FullDiscountAmount int64 `json:"full_discount_amount"`
	// 优惠券信息
	CouponInfo []CouponInfoItem_5_5 `json:"coupon_info"`
	// 优惠活动信息
	FullDiscountInfo []FullDiscountInfoItem_5_5 `json:"full_discount_info"`
	// 红包金额（单位：分）
	RedpackAmount int64 `json:"redpack_amount"`
	// 红包信息
	RedpackInfo []RedpackInfoItem `json:"redpack_info"`
	// 金币抵扣金额（单位：分）
	GoldCoinAmount int64 `json:"gold_coin_amount"`
}
type BlindBoxInfo struct {
	// 是否已抽盒
	IsOpened bool `json:"is_opened"`
	// 抽盒结果图片url
	BlindBoxImg string `json:"blind_box_img"`
	// 抽盒结果主标题
	BlindBoxTitle string `json:"blind_box_title"`
	// 抽盒结果副标题
	BlindBoxSubTitle string `json:"blind_box_sub_title"`
	// 抽盒时间戳
	OpenTime int64 `json:"open_time"`
}
type FullDiscountInfoItem_6_6 struct {
	// 营销优惠扩展字段instant_activity_id是商家营销活动id
	ExtraMap map[string]string `json:"extra_map"`
	// 活动ID
	CampaignId int64 `json:"campaign_id"`
	// 活动类型：100-店铺满减活动 110-平台满减活动 150-立减活动
	CampaignType int64 `json:"campaign_type"`
	// 成本分摊
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
	// 活动名称
	CampaignName string `json:"campaign_name"`
	// 活动优惠金额（单位：分）
	CampaignAmount int64 `json:"campaign_amount"`
	// 活动子类型：0-店铺活动，1-平台活动
	CampaignSubType int64 `json:"campaign_sub_type"`
}
type StoreAddress struct {
	// 省
	Province *Province `json:"province"`
	// 市
	City *City `json:"city"`
	// 县/区
	Town *Town `json:"town"`
	// 街道
	Street *Street `json:"street"`
	// 详细信息
	Detail string `json:"detail"`
}
type FreeInterest struct {
	// 总交易金额
	TradeAmount int64 `json:"trade_amount"`
	// 抖音月付平台承担息费总金额，单位: 分
	TotalFreeInterestPlatformClearingAmount int64 `json:"total_free_interest_platform_clearing_amount"`
	// 抖音月付商家承担息费总金额，单位: 分
	TotalFreeInterestShopClearingAmount int64 `json:"total_free_interest_shop_clearing_amount"`
	// 分期期数
	Period int64 `json:"period"`
	// 抖音月付退款部分，平台承担免息费用，单位: 分
	RefundTotalFreeInterestPlatformClearingAmount int64 `json:"refund_total_free_interest_platform_clearing_amount"`
	// 抖音月付退款部分，商家承担免息费用，单位: 分
	RefundTotalFreeInterestShopClearingAmount int64 `json:"refund_total_free_interest_shop_clearing_amount"`
	// 抖音月付免息信息明细
	FreeInterestDetail []FreeInterestDetailItem `json:"free_interest_detail"`
}
type AmountDetailListItem struct {
	// 追缴金额，单位是“分”
	Amount int64 `json:"amount"`
	// 追缴类型，instant_discount-以旧换新下单立减，actual_deduction_amount-先享换新抵扣金额。
	Type string `json:"type"`
	// 目前有先享换新抵扣金额和以旧换新下单立减两种
	TypeDesc string `json:"type_desc"`
}
type FullDiscountInfoItem_5_5 struct {
	// 营销优惠扩展字段instant_activity_id是商家营销活动id
	ExtraMap map[string]string `json:"extra_map"`
	// 活动ID
	CampaignId int64 `json:"campaign_id"`
	// 活动类型；100-店铺满减活动；110-平台满减活动；150-新人首单立减活动；
	CampaignType int64 `json:"campaign_type"`
	// 成本分摊
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
	// 活动名称
	CampaignName string `json:"campaign_name"`
	// 活动优惠金额（单位：分）
	CampaignAmount int64 `json:"campaign_amount"`
	// 活动子类型：0-店铺活动，1-平台活动
	CampaignSubType int64 `json:"campaign_sub_type"`
}
type SkuInfo struct {
	// 发品对应的批次ID
	CommitCode string `json:"commit_code"`
}
type PatientInfo struct {
	// 1-过往病史;存在表示有 2-过敏史；存在表示有3-家族病史 ；存在表示有4-孕期/哺乳期 ；存在表示是5-肝功能；存在表示异常6-肾功能；存在表示异常
	MedicalHistory []int64 `json:"medical_history"`
	// 是否复诊
	IsFollowUpVisit bool `json:"is_follow_up_visit"`
	// 身份号；脱敏后
	IdCardNum string `json:"id_card_num"`
	// 年龄
	Age int32 `json:"age"`
	// 性别
	Gender string `json:"gender"`
	// 姓名
	Name string `json:"name"`
}
type KolDiscountDetail_5_5 struct {
	// 优惠总金额（单位：分）
	TotalAmount int64 `json:"total_amount"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 活动优惠金额（单位：分）
	FullDiscountAmount int64 `json:"full_discount_amount"`
	// 优惠券信息
	CouponInfo []CouponInfoItem_6_6 `json:"coupon_info"`
	// 优惠活动信息
	FullDiscountInfo []FullDiscountInfoItem_6_6 `json:"full_discount_info"`
	// 红包金额（单位：分）
	RedpackAmount int64 `json:"redpack_amount"`
	// 红包信息
	RedpackInfo []RedpackInfoItem `json:"redpack_info"`
}
type WriteoffInfoItem struct {
	// 核销券码
	WriteoffNo string `json:"writeoff_no"`
	// 核销有效开始时间
	WriteoffStartTime int64 `json:"writeoff_start_time"`
	// 核销过期时间
	WriteoffExpireTime int64 `json:"writeoff_expire_time"`
	// 核销状态
	WriteoffStatus int64 `json:"writeoff_status"`
	// 核销状态文案
	WriteoffStatusDesc string `json:"writeoff_status_desc"`
	// 核销店铺订单id
	VerifyOrderId string `json:"verify_order_id"`
	// 脱敏核销券码
	WriteoffNoMask string `json:"writeoff_no_mask"`
	// 已核销次数
	WrittenoffCount int64 `json:"writtenoff_count"`
	// 总核销次数
	WriteoffTotalCount int64 `json:"writeoff_total_count"`
}
type RealYunWarehouseInfo struct {
	// 云仓code
	YunWarehouseCode string `json:"yun_warehouse_code"`
	// 云仓名称
	YunWarehouseName string `json:"yun_warehouse_name"`
}
type City struct {
	// 名称
	Name string `json:"name"`
	// 地区ID
	Id string `json:"id"`
}
type FreeInterest_4_4 struct {
	// 交易金额，单位:  分
	TradeAmount int64 `json:"trade_amount"`
	// 抖音月付平台承担免息金额，单位：分
	TotalFreeInterestPlatformClearingAmount string `json:"total_free_interest_platform_clearing_amount"`
	// 抖音月付商家承担免息金额，单位：分
	TotalFreeInterestShopClearingAmount int64 `json:"total_free_interest_shop_clearing_amount"`
	// 免息期数
	Period int64 `json:"period"`
	// 商品单抖音月付退款部分平台承担免息金额，单位：分
	RefundTotalFreeInterestPlatformClearingAmount int64 `json:"refund_total_free_interest_platform_clearing_amount"`
	// 商品单抖音月付退款部分商家承担免息金额，单位：分
	RefundTotalFreeInterestShopClearingAmount int64 `json:"refund_total_free_interest_shop_clearing_amount"`
	// 商品单抖音月付免息信息明细
	FreeInterestDetail []FreeInterestDetailItem `json:"free_interest_detail"`
}
type SpecItem struct {
	// 规格名称
	Name string `json:"name"`
	// 规格值
	Value string `json:"value"`
}
type AddressTagUiItem struct {
	// 双地址标签 key
	Key string `json:"key"`
	// 双地址标签
	Text string `json:"text"`
	// 双地址标签hover提示
	HoverText string `json:"hover_text"`
}
type CouponRightItem struct {
	// 权益类型
	RightType int64 `json:"right_type"`
	// 权益名称
	RightName string `json:"right_name"`
	// 权益面值
	Quota int64 `json:"quota"`
}
type OrderOrderDetailData struct {
	// 订单信息
	ShopOrderDetail *ShopOrderDetail `json:"shop_order_detail"`
}
type RecycleActualAmountInfo struct {
	// 回收款发放状态1-已发放
	SendStatus int64 `json:"send_status"`
	// 回收款发放渠道1-抖音钱包
	SendChannel int64 `json:"send_channel"`
	// 实际回收价；单位：分
	ActualAmount int64 `json:"actual_amount"`
}
type CouponInfoItem_6_6 struct {
	// 营销优惠扩展字段instant_activity_id是商家营销活动id
	ExtraMap map[string]string `json:"extra_map"`
	// 订单优惠券ID
	CouponId int64 `json:"coupon_id"`
	// 券类型;52-达人直减券;53-达人满减券;
	CouponType int64 `json:"coupon_type"`
	// 券批次ID
	CouponMetaId string `json:"coupon_meta_id"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 券名称
	CouponName string `json:"coupon_name"`
	// 成本分摊
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
}
type OrderPhaseListItem struct {
	// 阶段单id
	PhaseOrderId string `json:"phase_order_id"`
	// 总共有几阶段
	TotalPhase int64 `json:"total_phase"`
	// 第几阶段
	CurrentPhase int64 `json:"current_phase"`
	// sku单价，单位：分
	SkuPrice int64 `json:"sku_price"`
	// 是否支付成功
	PaySuccess bool `json:"pay_success"`
	// 店铺子订单号，抖店平台生成，平台下唯一；注意：一笔订单下有一个子订单和父订单单号相同。
	SkuOrderId string `json:"sku_order_id"`
	// 活动id
	CampaignId string `json:"campaign_id"`
	// 阶段价格：定金单价、尾款单价，单位：分
	PhasePayablePrice int64 `json:"phase_payable_price"`
	// 支付类型:0-货到付款, 1-微信, 2-支付宝, 3-小程序, 4-银行卡, 5-余额, 7-无需支付, 8-放心花(信用支付), 9-新卡支付
	PhasePayType int64 `json:"phase_pay_type"`
	// 阶段单开始开启支付时间，时间戳
	PhaseOpenTime int64 `json:"phase_open_time"`
	// 阶段单支付成功时间，时间戳
	PhasePayTime int64 `json:"phase_pay_time"`
	// 阶段单关闭时间，时间戳
	PhaseCloseTime int64 `json:"phase_close_time"`
	// 渠道支付订单号
	ChannelPaymentNo string `json:"channel_payment_no"`
	// 阶段单总金额，单位分，为订单金额phase_sum_amount+运费phase_post_amount,为支付金额phase_pay_amount+优惠金额phase_promotion_amoun
	PhaseOrderAmount int64 `json:"phase_order_amount"`
	// 阶段单订单金额，单位分
	PhaseSumAmount int64 `json:"phase_sum_amount"`
	// 阶段单运费金额，单位分
	PhasePostAmount int64 `json:"phase_post_amount"`
	// 阶段单用户实际支付金额，单位分
	PhasePayAmount int64 `json:"phase_pay_amount"`
	// 阶段单总优惠金额，单位分
	PhasePromotionAmount int64 `json:"phase_promotion_amount"`
	// 阶段状态描述
	CurrentPhaseStatusDesc string `json:"current_phase_status_desc"`
}
type UserIdInfo struct {
	// 证件号
	IdCardNo string `json:"id_card_no"`
	Encrypt_id_card_no string `json:"encrypt_id_card_no"`
	// 证件姓名
	IdCardName string `json:"id_card_name"`
	Encrypt_id_card_name string `json:"encrypt_id_card_name"`
}
type ConsolidateInfo struct {
	// 中转类型，1-新疆中转
	ConsolidateType int64 `json:"consolidate_type"`
}
type ProductChannelInfo struct {
	// 3--商家自卖4且skuordertag中有kol_exclusive_product标签：专属达人带货；无标签的channeltype =4为达人带货
	ProductChannelType int64 `json:"product_channel_type"`
	// 渠道商品id
	ProductChannelId string `json:"product_channel_id"`
}
type LowPriceInfo struct {
	// 超值购商品信息
	LowPriceProduct *LowPriceProduct `json:"low_price_product"`
}
type AfterSaleInfo struct {
	// 售后状态；6-售后申请；27-拒绝售后申请；12-售后成功；7-售后退货中；11-售后已发货；29-售后退货拒绝；13-【换货返回：换货售后换货商家发货】，【补寄返回：补寄待用户收货】； 14-【换货返回：（换货）售后换货用户收货】，【补寄返回：（补寄）用户已收货】 ；28-售后失败；51-订单取消成功；53-逆向交易已完成；
	AfterSaleStatus int64 `json:"after_sale_status"`
	// 售后类型 ；0-退货退款;1-已发货仅退款;2-未发货仅退款;3-换货;4-系统取消;5-用户取消;6-价保;7-补寄;
	AfterSaleType int64 `json:"after_sale_type"`
	// 退款状态:1-待退款；3-退款成功； 4-退款失败；当买家发起售后后又主动取消售后，此时after_sale_status=28并且refund_status=1的状态不变，不会流转至4状态；
	RefundStatus int64 `json:"refund_status"`
}
type AccountInfoItem struct {
	// 买家账号名称
	AccountName string `json:"account_name"`
	// 买家账号类型;账号类型；枚举值：Mobile-手机号;Email-邮箱 ; IdCard-身份证;  Passport-护照;  BankCard-银行卡;  Number-纯数学;  NumberLetter-数字字母混合;
	AccountType string `json:"account_type"`
	// 账号值
	AccountId string `json:"account_id"`
	Encrypt_account_id string `json:"encrypt_account_id"`
}
type TextItem struct {
	// id
	Id int64 `json:"id"`
	// key
	Key string `json:"key"`
	// content
	Content string `json:"content"`
}
type PromiseDetail struct {
	// 时效类型，1-承诺发货时效；2-承诺送达时效；5-预约送达时效，6-预约发货时效
	PromiseType int64 `json:"promise_type"`
	// 订单时效信息
	PromiseTimeDetail *PromiseTimeDetail `json:"promise_time_detail"`
	// 推荐使用物流公司列表
	RecommendLogisticsList []RecommendLogisticsListItem_4_4 `json:"recommend_logistics_list"`
}
type ShopPeriodPurchaseInfo struct {
	// 类型（5预约送达，6预约发货）
	DeliverType int64 `json:"deliver_type"`
	// 总期数
	PeriodNum int64 `json:"period_num"`
}
type ShareDiscountCost struct {
	// 第三方平台承担明细
	ExtraMap map[string]string `json:"extra_map"`
	// 第三方平台承担金额，具体的承担明细详见extra_map
	ThirdPlatformCost int64 `json:"third_platform_cost"`
	// 作者承担金额，单位：分
	AuthorCost int64 `json:"author_cost"`
	// 商家承担金额，单位：分
	ShopCost int64 `json:"shop_cost"`
	// 平台承担金额，单位：分
	PlatformCost int64 `json:"platform_cost"`
}
type HomeInstallationServiceInfo struct {
	// 安装工人信息
	WorkerInfo *WorkerInfo `json:"worker_info"`
	// 服务商信息
	SupplierInfo *SupplierInfo `json:"supplier_info"`
	// 服务单信息
	ServiceInfo *ServiceInfo `json:"service_info"`
	// 安装服务是否有效 0-取消 1-生效
	LatestStatus int64 `json:"latest_status"`
}
type PromotionDetail_4_4 struct {
	// 店铺优惠信息
	ShopDiscountDetail *ShopDiscountDetail `json:"shop_discount_detail"`
	// 平台优惠信息
	PlatformDiscountDetail *PlatformDiscountDetail_5_5 `json:"platform_discount_detail"`
	// 达人优惠信息
	KolDiscountDetail *KolDiscountDetail_5_5 `json:"kol_discount_detail"`
}
type BundleSkuInfoItem struct {
	// 子商品图片
	PictureUrl string `json:"picture_url"`
	// 子商品在抖店的ID
	ProductId string `json:"product_id"`
	// 子商品名称
	ProductName string `json:"product_name"`
	// 子商品在抖店的SkuId
	SkuId string `json:"sku_id"`
	// 子商品数量
	ItemNum int64 `json:"item_num"`
	// 组套子商品外部编码
	Code string `json:"code"`
}
type PreAllocatedYunWarehouseInfo struct {
	// 云仓code
	YunWarehouseCode string `json:"yun_warehouse_code"`
	// 云仓名称
	YunWarehouseName string `json:"yun_warehouse_name"`
}
type SkuOrderListItem struct {
	// 商品单额外信息
	OrderExtendList []OrderExtendListItem `json:"order_extend_list"`
	// 借壳订单实际销售公司
	ShellOrderInfo *ShellOrderInfo `json:"shell_order_info"`
	// 仓库信息
	InventoryList []InventoryListItem `json:"inventory_list"`
	// 9900
	ActualReceiveAmountInfo *ActualReceiveAmountInfo `json:"actual_receive_amount_info"`
	// 拍卖信息
	AuctionInfo *AuctionInfo `json:"auction_info"`
	// 盲盒信息
	BlindBoxInfo *BlindBoxInfo `json:"blind_box_info"`
	// 溯源码信息
	SkuMedicalTraceInfo *SkuMedicalTraceInfo `json:"sku_medical_trace_info"`
	// 商品单对应的sku信息
	SkuInfo *SkuInfo `json:"sku_info"`
	// 后发赠品时效
	GivenExpShipTime int64 `json:"given_exp_ship_time"`
	// 后发赠品时效描述
	GivenExpShipTimeDesc string `json:"given_exp_ship_time_desc"`
	// 商品单上关联的政府补贴信息；比如一品多地等
	GovSubsidyDetail *GovSubsidyDetail `json:"gov_subsidy_detail"`
	// 商品上关联的SN/69/IMEI码信息
	SerialNoInfo *SerialNoInfo `json:"serial_no_info"`
	// 业务身份-来源
	BmpSource string `json:"bmp_source"`
	// 业务身份-垂类
	BmpVerticalMarket string `json:"bmp_vertical_market"`
	// 业务身份-卖家身份
	BmpSellerType string `json:"bmp_seller_type"`
	// 周期购信息
	SkuPeriodPurchaseInfo *SkuPeriodPurchaseInfo `json:"sku_period_purchase_info"`
	// 服务商品编码，如维修服务单上设置的服务商品编码
	SpProductId string `json:"sp_product_id"`
	// 与当前商品单有关联的商品单，如维修主商品单关联的维修单商品单号
	RelationSkuOrderIds []string `json:"relation_sku_order_ids"`
	// 库存扣减信息
	SkuCargoFutureStock *SkuCargoFutureStock `json:"sku_cargo_future_stock"`
	// 商品属性，适用于外卖或即时零售场景下商品存在属性的情况
	CustomProperties []CustomPropertiesItem `json:"custom_properties"`
	// 渠道商品信息
	ProductChannelInfo *ProductChannelInfo `json:"product_channel_info"`
	// 超值购信息
	LowPriceInfo *LowPriceInfo `json:"low_price_info"`
	// 打包费（单位：分）
	PackingChargeAmount int64 `json:"packing_charge_amount"`
	// 商品单的额外信息；"barcodes": "[\"6970848860361\"]",--商家设置商品单对应的69码
	ExtraInfo map[string]string `json:"extra_info"`
	// 处方单列表
	PrescriptionInfoList []PrescriptionInfoListItem `json:"prescription_info_list"`
	// 提货券商品抵扣金额 单位：分
	VoucherDeductionAmount int64 `json:"voucher_deduction_amount"`
	// 上门安装服务信息
	HomeInstallationServiceInfo *HomeInstallationServiceInfo `json:"home_installation_service_info"`
	// 商品单抖音月付免息信息
	FreeInterest *FreeInterest_4_4 `json:"free_interest"`
	// 支付优惠详情，与promotion_pay_amount对应
	PromotionPayAmountDetails []PromotionPayAmountDetailsItem `json:"promotion_pay_amount_details"`
	// 跨境后置备案订单税费是否未计算完成 0-计算完成 1-未计算完成
	TaxAmountNotComeOut int64 `json:"tax_amount_not_come_out"`
	// 商品原价（单位：分）
	GoodsPrice int64 `json:"goods_price"`
	// 车型
	SkuCarModelDesc string `json:"sku_car_model_desc"`
	// 质检状态;0-无需质检;1-可质检;2-必须质检
	QualityInspectionStatus int64 `json:"quality_inspection_status"`
	// 店铺子订单号，抖店平台生成，平台下唯一；注意：一笔订单下有一个子订单和父订单单号相同。
	OrderId string `json:"order_id"`
	// 店铺父订单号，抖店平台生成，平台下唯一；
	ParentOrderId string `json:"parent_order_id"`
	// 订单层级
	OrderLevel int64 `json:"order_level"`
	// 【业务来源】 1、鲁班 2、小店 3、好好学习 4、ev 5、虚拟 6、建站 7、核销 8、玉石 9、ez 10、ep 11、虚拟卡券 12、服务市场 13、EP视频课 14、EP直播课 21、跨境BBC 22、跨境BC 23、跨境CC,UPC 24、手机充值 25、拍卖保证金 26、懂车帝抵扣券 27、懂车帝返现券 28、离岛免税 29、海南会员购 30、抽奖 32、dou+券 76、大闸蟹 99、保险 102、小店海外
	Biz int64 `json:"biz"`
	// 业务来源描述
	BizDesc string `json:"biz_desc"`
	// 【订单类型】 0-普通订单 2-虚拟商品订单 4-电子券（poi核销） 5-三方核销
	OrderType int64 `json:"order_type"`
	// 订单类型描述
	OrderTypeDesc string `json:"order_type_desc"`
	// 交易类型：0-普通；1-拼团；2-定金预售；3-订金找货；4-拍卖；5-0元单；6-回收；7-寄卖；10-寄样；11-0元抽奖(超级福袋)；12-达人买样；13-普通定制；16-大众竞拍；18-小时达；102-定金预售的赠品单；103-收款；
	TradeType int64 `json:"trade_type"`
	// 交易类型描述
	TradeTypeDesc string `json:"trade_type_desc"`
	// 订单状态
	OrderStatus int64 `json:"order_status"`
	// 订单状态描述
	OrderStatusDesc string `json:"order_status_desc"`
	// 主流程状态
	MainStatus int64 `json:"main_status"`
	// 主流程状态描述
	MainStatusDesc string `json:"main_status_desc"`
	// 支付时间，时间戳，秒
	PayTime int64 `json:"pay_time"`
	// 订单过期时间，时间戳，秒
	OrderExpireTime int64 `json:"order_expire_time"`
	// 订单完成时间，时间戳，秒
	FinishTime int64 `json:"finish_time"`
	// 下单时间，时间戳，秒
	CreateTime int64 `json:"create_time"`
	// 订单更新时间，时间戳，秒
	UpdateTime int64 `json:"update_time"`
	// 取消原因
	CancelReason string `json:"cancel_reason"`
	// 【下单端】 0-站外 1-火山 2-抖音 3-头条 4-西瓜 5-微信 6-值点app 7、头条lite 8、懂车帝 9、皮皮虾 11、抖音极速版 12、TikTok 13、musically 14、穿山甲 15、火山极速版 16、服务市场 26、番茄小说 27、UG教育营销电商平台 28、Jumanji 29、电商SDK
	BType int64 `json:"b_type"`
	// 下单端描述
	BTypeDesc string `json:"b_type_desc"`
	// 【下单场景】 0-未知 1-app内-原生 2-app内-小程序 3-H5 13-电商SDK-头条 35-电商SDK-头条lite
	SubBType int64 `json:"sub_b_type"`
	// 下单场景描述
	SubBTypeDesc string `json:"sub_b_type_desc"`
	// 已废弃，不推荐使用。流量来源：1-鲁班广告 2-联盟 3-商城 4-自主经营 5-线索通支付表单 6-抖音门店 7-抖+ 8-穿山甲
	SendPay int64 `json:"send_pay"`
	// 已废弃，不推荐使用。流量来源描述
	SendPayDesc string `json:"send_pay_desc"`
	// 直播主播id（达人）;仅直播间和橱窗产生的订单会有值返回;
	AuthorId int64 `json:"author_id"`
	// 直播主播名称;仅直播间和橱窗产生的订单会有值返回
	AuthorName string `json:"author_name"`
	// 【下单来源】 0-其他 1-直播间（废弃）
	ThemeType string `json:"theme_type"`
	// 下单来源描述（废弃）
	ThemeTypeDesc string `json:"theme_type_desc"`
	// 具体某个小程序的ID
	AppId int64 `json:"app_id"`
	// 直播房间ID，字符串类型
	RoomId int64 `json:"room_id"`
	// 内容id。（该字段无实际意义，已废弃）
	ContentId string `json:"content_id"`
	// 视频id，有值则代表订单来自短视频
	VideoId string `json:"video_id"`
	// 已废弃，流量来源id
	OriginId string `json:"origin_id"`
	// 已废弃，广告id
	Cid int64 `json:"cid"`
	// 【C端流量来源】 0-unknown  2-精选联盟  8-小店自卖
	CBiz int64 `json:"c_biz"`
	// C端流量来源业务类型描述
	CBizDesc string `json:"c_biz_desc"`
	// 鲁班广告落地页ID
	PageId int64 `json:"page_id"`
	// 【支付类型】 0-货到付款 1-微信 2-支付宝 3-小程序 4-银行卡 5、余额 7、无需支付（0元单） 8、DOU分期（信用支付） 9、新卡支付
	PayType int64 `json:"pay_type"`
	// 支付渠道的流水号
	ChannelPaymentNo string `json:"channel_payment_no"`
	// 订单金额（单位：分）
	OrderAmount int64 `json:"order_amount"`
	// 支付金额（单位：分）
	PayAmount int64 `json:"pay_amount"`
	// 运费险金额（单位：分）
	PostInsuranceAmount int64 `json:"post_insurance_amount"`
	// 改价金额变化量（单位：分）
	ModifyAmount int64 `json:"modify_amount"`
	// 改价运费金额变化量（单位：分）
	ModifyPostAmount int64 `json:"modify_post_amount"`
	// 订单优惠总金额（单位：分） = 店铺优惠金额 + 平台优惠金额 + 达人优惠金额
	PromotionAmount int64 `json:"promotion_amount"`
	// 店铺优惠金额（单位：分），属于店铺的优惠活动、优惠券、红包的总优惠金额
	PromotionShopAmount int64 `json:"promotion_shop_amount"`
	// 平台优惠金额（单位：分），属于平台的优惠活动、优惠券、红包的总优惠金额
	PromotionPlatformAmount int64 `json:"promotion_platform_amount"`
	// 商家承担金额（单位：分），订单参与活动和优惠中商家承担部分的总金额
	ShopCostAmount int64 `json:"shop_cost_amount"`
	// 平台承担金额（单位：分），订单参与活动和优惠中平台+作者（达人）承担部分的总金额,包含作者（达人）承担金额：platform_cost_amount = only_platform_cost_amount + author_cost_amount
	PlatformCostAmount int64 `json:"platform_cost_amount"`
	// 达人优惠金额（单位：分），属于达人的优惠活动、优惠券、红包的总优惠金额
	PromotionTalentAmount int64 `json:"promotion_talent_amount"`
	// 支付优惠金额（单位：分），支付渠道上的优惠金额
	PromotionPayAmount int64 `json:"promotion_pay_amount"`
	// 商家后台商品编码
	Code string `json:"code"`
	// 收件人电话
	PostTel string `json:"post_tel"`
	Encrypt_post_tel string `json:"encrypt_post_tel"`
	// 收件人姓名
	PostReceiver string `json:"post_receiver"`
	Encrypt_post_receiver string `json:"encrypt_post_receiver"`
	// 收件人地址
	PostAddr *PostAddr `json:"post_addr"`
	// 预计发货时间
	ExpShipTime int64 `json:"exp_ship_time"`
	// 发货时间
	ShipTime int64 `json:"ship_time"`
	// 物流收货时间
	LogisticsReceiptTime int64 `json:"logistics_receipt_time"`
	// 用户确认收货时间
	ConfirmReceiptTime int64 `json:"confirm_receipt_time"`
	// 【商品类型】 0-实体 1-虚拟
	GoodsType int64 `json:"goods_type"`
	// 商品ID，字符串类型
	ProductId int64 `json:"product_id"`
	// 商品skuId
	SkuId int64 `json:"sku_id"`
	// 规格信息
	Spec []SpecItem `json:"spec"`
	// 一级类目
	FirstCid int64 `json:"first_cid"`
	// 二级类目
	SecondCid int64 `json:"second_cid"`
	// 三级类目
	ThirdCid int64 `json:"third_cid"`
	// 四级类目
	FourthCid int64 `json:"fourth_cid"`
	// 外部Skuid
	OutSkuId string `json:"out_sku_id"`
	// sku外部供应商编码
	SupplierId string `json:"supplier_id"`
	// 商品外部编码
	OutProductId string `json:"out_product_id"`
	// 仓id，废弃，使用inventory_list
	WarehouseIds []string `json:"warehouse_ids"`
	// 外部仓id，废弃，使用inventory_list
	OutWarehouseIds []string `json:"out_warehouse_ids"`
	// 库存类型，普通库存/区域库存，废弃，使用inventory_list
	InventoryType string `json:"inventory_type"`
	// 库存类型描述，废弃，使用inventory_list
	InventoryTypeDesc string `json:"inventory_type_desc"`
	// 【库存扣减方式】 1-下单减库存 2-支付减库存
	ReduceStockType int64 `json:"reduce_stock_type"`
	// 库存扣减方式名称
	ReduceStockTypeDesc string `json:"reduce_stock_type_desc"`
	// 商品现价（单位：分）
	OriginAmount int64 `json:"origin_amount"`
	// 是否包税
	HasTax bool `json:"has_tax"`
	// 订单商品数量
	ItemNum int64 `json:"item_num"`
	// 商品现价*件数
	SumAmount int64 `json:"sum_amount"`
	// 商品来源平台
	SourcePlatform string `json:"source_platform"`
	// 优惠信息
	PromotionDetail *PromotionDetail_4_4 `json:"promotion_detail"`
	// 活动信息，其中活动金额会体现到商品价格中（比如商品原价是100元，参加活动优惠20元，那商品现价就为80元）
	CampaignInfo []CampaignInfoItem `json:"campaign_info"`
	// 商品单标签
	SkuOrderTagUi []SkuOrderTagUiItem `json:"sku_order_tag_ui"`
	// 商品图片
	ProductPic string `json:"product_pic"`
	// 是否评价 :1-已评价，0-未评价，2 -追评
	IsComment int64 `json:"is_comment"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 仓库信息
	InventoryListV2 []InventoryListItem_4_4 `json:"-"`
	// 运费（单位：分）
	PostAmount int64 `json:"post_amount"`
	// 预售类型 ，0-现货类型，1-全款预售，2-阶梯发货
	PreSaleType int64 `json:"pre_sale_type"`
	// 售后信息
	AfterSaleInfo *AfterSaleInfo `json:"after_sale_info"`
	// 核销信息
	WriteoffInfo []WriteoffInfoItem `json:"writeoff_info"`
	// 关联订单
	RelationOrder *RelationOrder `json:"relation_order"`
	// 红包优惠金额（单位：分）
	PromotionRedpackAmount int64 `json:"promotion_redpack_amount"`
	// 平台红包优惠金额（单位：分），属于平台的红包的优惠金额
	PromotionRedpackPlatformAmount int64 `json:"promotion_redpack_platform_amount"`
	// 达人红包优惠金额（单位：分），属于达人的红包的优惠金额
	PromotionRedpackTalentAmount int64 `json:"promotion_redpack_talent_amount"`
	// 1-邮寄，2-自提
	ReceiveType int64 `json:"receive_type"`
	// 是否需要上传序列号，用于判断发货时是否需要上传商品序列号（IMEI码或SN码），true 表示是3C数码商品，需要上传序列号
	NeedSerialNumber bool `json:"need_serial_number"`
	// 广告来源，video-短视频  live-直播
	AdEnvType string `json:"ad_env_type"`
	// 商品ID，字符串类型
	ProductIdStr string `json:"product_id_str"`
	// 预约发货时间
	AppointmentShipTime int64 `json:"appointment_ship_time"`
	// 直播房间ID，字符串类型
	RoomIdStr string `json:"room_id_str"`
	// 绑定类型  MASTER-主品单  FREE-免费赠品
	GivenProductType string `json:"given_product_type"`
	// 废弃，使用master_sku_order_id_list；绑定的主品sku单单号 如果given_product_type=FREE 则master_sku_order_id为主品sku单单号
	MasterSkuOrderId string `json:"master_sku_order_id"`
	// 商品卡券基本信息
	CardVoucher *CardVoucher `json:"card_voucher"`
	// 组套商品子商品列表
	BundleSkuInfo []BundleSkuInfoItem `json:"bundle_sku_info"`
	// 当order_type=2，order_type_desc为虚拟订单时该参数会返回信息
	AccountList *AccountList `json:"account_list"`
	// 定制商品信息
	SkuCustomizationInfo []SkuCustomizationInfoItem `json:"sku_customization_info"`
	// 是否是补贴品订单
	IsActivity bool `json:"is_activity"`
	// 作者（达人）承担金额（单位：分），订单参与活动和优惠中作者（达人）承担部分的总金额
	AuthorCostAmount int64 `json:"author_cost_amount"`
	// 仅平台承担金额（单位：分），订单参与活动和优惠中平台承担部分的总金额
	OnlyPlatformCostAmount int64 `json:"only_platform_cost_amount"`
	// 履约时效信息(json串)
	PromiseInfo string `json:"promise_info"`
	// 门店信息
	StoreInfo *StoreInfo `json:"store_info"`
	// 收件人姓名（脱敏后）
	MaskPostReceiver string `json:"mask_post_receiver"`
	// 收件人电话（脱敏后），电商脱敏格式为1********35，即时零售脱敏格式为*******1035(后四位明文)
	MaskPostTel string `json:"mask_post_tel"`
	// 收件人地址（脱敏后）
	MaskPostAddr *MaskPostAddr `json:"mask_post_addr"`
	// 合约信息
	ContractInfo *ContractInfo `json:"contract_info"`
	// 绑定的主品sku单单号 如果given_product_type=FREE 则master_sku_order_id_list为主品sku单单号
	MasterSkuOrderIdList []string `json:"master_sku_order_id_list"`
	// 赠品活动信息
	GivenProductActivityInfo *GivenProductActivityInfo `json:"given_product_activity_info"`
	// 质检信息
	QualityCheckInfo *QualityCheckInfo `json:"quality_check_info"`
	// sku云仓择仓信息（云仓业务使用，非商品区域仓功能使用）
	SkuYunWarehouseInfo []SkuYunWarehouseInfoItem `json:"sku_yun_warehouse_info"`
	// 税费
	TaxAmount int64 `json:"tax_amount"`
}
type OrderExtendListItem struct {
	// 额外信息详情，需要自行反序列化
	Value string `json:"value"`
	// 额外信息唯一标识
	Key string `json:"key"`
}
type GivenProductActivityInfo struct {
	// NORMAL-普通的买赠，AMOUNT-满件赠，PRICE-满元赠
	GivenProductActivityType string `json:"given_product_activity_type"`
	// 满几件赠或满几元赠，满元赠场景单位是分
	GivenProductActivityValue string `json:"given_product_activity_value"`
}
type ShopRecycleSubsidyAmountInfo struct {
	// 换新额外补贴金额组成
	AmountCompositionInfo *AmountCompositionInfo `json:"amount_composition_info"`
	// 补贴渠道；1-返金额，2-平台红包
	SubsidyStatus int64 `json:"subsidy_status"`
	// 补贴渠道；1-返金额，2-平台红包
	SubsidyChannel int64 `json:"subsidy_channel"`
	// 换新额外补贴，单位：分
	SubsidyAmount int64 `json:"subsidy_amount"`
}
type FreeInterestDetailItem struct {
	// 抖音月付，平台承担免息金额，单位：分
	PlatformClearingAmount int64 `json:"platform_clearing_amount"`
	// 抖音月付，商家承担免息金额，单位：分
	ShopClearingAmount int64 `json:"shop_clearing_amount"`
	// 交易类型，1-支付，2-退款
	TradeType int64 `json:"trade_type"`
}
type ProductInfoItem struct {
	// 商品名称
	ProductName string `json:"product_name"`
	// 商品价格
	Price int64 `json:"price"`
	// 商家编码
	OuterSkuId string `json:"outer_sku_id"`
	// 商品skuId
	SkuId int64 `json:"sku_id"`
	// 规格信息
	SkuSpecs []SkuSpecsItem `json:"sku_specs"`
	// 发货商品数量
	ProductCount int64 `json:"product_count"`
	// 商品ID，字符串类型
	ProductId int64 `json:"product_id"`
	// 店铺子订单号，抖店平台生成，平台下唯一；注意：一笔订单下有一个子订单和父订单单号相同。
	SkuOrderId string `json:"sku_order_id"`
	// 商品ID，字符串类型
	ProductIdStr string `json:"product_id_str"`
}
type TraceInfo struct {
	// sku单id
	SkuOrderId string `json:"sku_order_id"`
	// 如果已经上传了溯源码，会在这里展示，item单维度会聚合到sku维度，举个例子，如果sku单有两个item，且已上传2个溯源码，则这里list里面有两个元素
	BatchAndTraceCodeList []BatchAndTraceCodeListItem `json:"batch_and_trace_code_list"`
}
type GovSubsidyDetail struct {
	// 实际销售公司
	SupplierShopName string `json:"supplier_shop_name"`
	// 实际销售公司ID
	SupplierShopId string `json:"supplier_shop_id"`
	// government_subsidy_multis：一品多地订单
	ProductTag string `json:"product_tag"`
}
type MaskPostAddr struct {
	// 省
	Province *Province `json:"province"`
	// 市
	City *City `json:"city"`
	// 县/区
	Town *Town `json:"town"`
	// 街道
	Street *Street `json:"street"`
	// 详细信息（脱敏后）
	Detail string `json:"detail"`
}
type AuctionInfo struct {
	// 拍卖订单类型
	AuctionOrderType string `json:"auction_order_type"`
	// 关联订单id列表
	RelatedOrderIds []string `json:"related_order_ids"`
	// 起拍价
	StartingPrice int64 `json:"starting_price"`
	// 拍卖开始时间
	AuctionBeginTime int64 `json:"auction_begin_time"`
	// 拍卖结束时间
	AuctionEndTime int64 `json:"auction_end_time"`
}
type InventoryListItem_4_4 struct {
	// 仓名称
	WarehouseName string `json:"warehouse_name"`
	// 仓id
	WarehouseId string `json:"warehouse_id"`
	// 外部仓id/编码
	OutWarehouseId string `json:"out_warehouse_id"`
	// 库存类型，普通库存/区域库存
	InventoryType int64 `json:"inventory_type"`
	// 库存类型描述
	InventoryTypeDesc string `json:"inventory_type_desc"`
	// 库存变更数量
	Count int64 `json:"count"`
	// 仓类型：0-商家仓，1-跨境仓，2-云仓
	WarehouseType int64 `json:"warehouse_type"`
}
type RecommendLogisticsListItem struct {
	// 该订单推荐使用的物流公司code
	LogisticsCompanyCode string `json:"logistics_company_code"`
	// 该订单推荐使用的物流公司名称
	LogisticsCompanyName string `json:"logistics_company_name"`
	// 快递产品类型type
	LogisticsProductType string `json:"logistics_product_type"`
	// 快递产品类型名称
	LogisticsProductTypeName string `json:"logistics_product_type_name"`
	// 该推荐快递下，建议最早发货时间（秒级时间戳，精确到分钟）
	LogisticsRecommendStartShipTime int64 `json:"logistics_recommend_start_ship_time"`
	// 该推荐快递下，建议最晚发货时间（秒级时间戳，精确到分钟）
	LogisticsRecommendEndShipTime int64 `json:"logistics_recommend_end_ship_time"`
	// 该物流最晚揽收时间（秒级时间戳，精确到分钟）
	LogisticsCollectionEndTime int64 `json:"logistics_collection_end_time"`
	// 该物流最晚发货时间（秒级时间戳，精确到分钟）
	LogisticsDeliveryEndTime int64 `json:"logistics_delivery_end_time"`
}
type RedpackInfoItem struct {
	// 营销优惠扩展字段instant_activity_id是商家营销活动id
	ExtraMap map[string]string `json:"extra_map"`
	// 分摊成本
	ShareDiscountCost *ShareDiscountCost `json:"share_discount_cost"`
	// 红包汇总id
	RedpackTransId string `json:"redpack_trans_id"`
	// 红包金额，单位：分
	RedpackAmount int64 `json:"redpack_amount"`
}
type PrescriptionDetail struct {
	// 处方图片
	ImgUrls []string `json:"img_urls"`
	// 处方单生成时间戳
	CreateTime int64 `json:"create_time"`
	// 处方单状态；1-收单 2-创建处方待审核 3-审核中 4-审核拒绝 5-审核同意 6-审核超时拒绝 20-过期取消
	Status string `json:"status"`
	// 问诊单 ID
	ConsultationId string `json:"consultation_id"`
	// 处方单 ID
	PrescriptionId string `json:"prescription_id"`
}
type StoreInfo struct {
	// 门店标签; delivery_service_pkg枚举值1=平配服务包 2=自配服务包
	StoreTagMap map[string]string `json:"store_tag_map"`
	// 门店id
	StoreId string `json:"store_id"`
	// 门店名称
	StoreName string `json:"store_name"`
	// 门店电话
	StoreTel string `json:"store_tel"`
	// 门店地址
	StoreAddress *StoreAddress `json:"store_address"`
	// 扩展字段
	Extra string `json:"extra"`
}
type Street struct {
	// 名称
	Name string `json:"name"`
	// 地区ID
	Id string `json:"id"`
}
type WorkerInfo struct {
	// 手机号
	WorkerTel string `json:"worker_tel"`
	// 姓名
	WorkerName string `json:"worker_name"`
}
type AppointmentSliceTime struct {
	// 更新时间
	UpdateTime int64 `json:"update_time"`
	// 结束时间
	EndTime int64 `json:"end_time"`
	// 起始时间
	StartTime int64 `json:"start_time"`
}
type PressForPaymentAfterwards struct {
	// 追缴明细
	AmountDetailList []AmountDetailListItem `json:"amount_detail_list"`
	// 追缴状态描述，目前包含追缴中和追缴完成两个状态。
	PayStatusDesc string `json:"pay_status_desc"`
	// 总的追缴金额，单位是“分”
	Amount int64 `json:"amount"`
	// 追缴状态，只有追缴中和追缴完成会下发该信息。1-追缴中，2-追缴完成。
	PayStatus int64 `json:"pay_status"`
}
type CardVoucher struct {
	// 自领取之日起有效天数(如果存在优先使用)
	ValidDays int64 `json:"valid_days"`
	// 卡券开始时间
	ValidStart int64 `json:"valid_start"`
	// 卡券截止时间
	ValidEnd int64 `json:"valid_end"`
}
type CustomPropertiesItem struct {
	// 属性的Name
	Name string `json:"name"`
	// 属性的Code
	Code string `json:"code"`
}
type SupplierInfo struct {
	// 辅材 URL
	AuxiliaryMaterialUrl string `json:"auxiliary_material_url"`
	// 收费说明
	SupplierChargeDesc string `json:"supplier_charge_desc"`
	// 服务商 ShopId
	SupplierShopId string `json:"supplier_shop_id"`
	// 电话
	SupplierTel string `json:"supplier_tel"`
	// 名称
	SupplierName string `json:"supplier_name"`
}
type ShareCost_5_5 struct {
	// 指定店铺返回分期数、分期利率分期数：fxh_periods："3"  分3期"partner_rate":"0.018" 分期利率
	ExtraMap map[string]string `json:"extra_map"`
	// 商家贴息金额（单位：分）
	ShopInterestCost int64 `json:"shop_interest_cost"`
	// 商家承担金额(单位：分)
	ShopCost int64 `json:"shop_cost"`
	// 平台承担金额(单位：分)
	PlatformCost int64 `json:"platform_cost"`
}
type Province struct {
	// 名称
	Name string `json:"name"`
	// 地区ID
	Id string `json:"id"`
}
type ShopDiscountDetail struct {
	// 红包明细
	RedpackInfo []RedpackInfoItem `json:"redpack_info"`
	// 红包金额，单位：分
	RedpackAmount int64 `json:"redpack_amount"`
	// 优惠总金额（单位：分）
	TotalAmount int64 `json:"total_amount"`
	// 券优惠金额（单位：分）
	CouponAmount int64 `json:"coupon_amount"`
	// 活动优惠金额（单位：分）
	FullDiscountAmount int64 `json:"full_discount_amount"`
	// 优惠券信息
	CouponInfo []CouponInfoItem `json:"coupon_info"`
	// 优惠活动信息
	FullDiscountInfo []FullDiscountInfoItem `json:"full_discount_info"`
}
type PostAddr struct {
	// 省/直辖市
	Province *Province `json:"province"`
	// 市
	City *City `json:"city"`
	// 区县
	Town *Town `json:"town"`
	// 街道
	Street *Street `json:"street"`
	// 详细地址
	Detail string `json:"detail"`
	Encrypt_detail string `json:"encrypt_detail"`
}
type ConsultationDetail struct {
	// 患者信息
	PatientInfo *PatientInfo `json:"patient_info"`
	// 问诊单 ID
	ConsultationId string `json:"consultation_id"`
}
type AccountList struct {
	// 买家账号信息
	AccountInfo []AccountInfoItem `json:"account_info"`
}
