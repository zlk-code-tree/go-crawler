package model

// ============================================
// 第三方平台对接 - 通用模型
// ============================================

// PlatformOrderListParams 获取订单列表参数（通用）
type PlatformOrderListParams struct {
	Platform      string `json:"platform" v:"required|in:douyin,kuaishou,shipinhao"`
	Page          int    `json:"page" d:"0"`
	PageSize      int    `json:"pageSize" d:"20"`
	StartTime     string `json:"startTime" dc:"开始时间 2006-01-02 15:04:05"`
	EndTime       string `json:"endTime" dc:"结束时间 2006-01-02 15:04:05"`
	AccountId     int64  `json:"accountId" dc:"账号ID"`
	StoreId       int    `json:"storeId" dc:"店铺ID（store表主键）"`
	OrderStatus   string `json:"orderStatus" dc:"订单状态"`
	UseUpdateTime bool   `json:"useUpdateTime" dc:"是否按订单更新时间查询"`
}

// PlatformOrderDetailParams 获取订单详情参数（通用）
type PlatformOrderDetailParams struct {
	Platform  string `json:"platform" v:"required|in:douyin,kuaishou,shipinhao"`
	OrderNo   string `json:"orderNo" v:"required"`
	AccountId int64  `json:"accountId"`
	StoreId   int    `json:"storeId" dc:"店铺ID（store表主键）"`
}

// PlatformOrderItem 统一订单结构（各平台映射到此结构）
type PlatformOrderItem struct {
	// ===== 店铺级信息（同一店铺订单号下每个SKU重复） =====
	Platform        string `json:"platform"`
	OrderNo         string `json:"orderNo"`         // 店铺订单号
	ShopId          int64  `json:"shopId"`          // 店铺ID
	ShopName        string `json:"shopName"`        // 店铺名称
	OrderStatus     int64  `json:"orderStatus"`     // 订单状态码
	OrderStatusDesc string `json:"orderStatusDesc"` // 订单状态描述
	MainStatus      int64  `json:"mainStatus"`      // 主流程状态
	MainStatusDesc  string `json:"mainStatusDesc"`  // 主流程状态描述
	PayTime         string `json:"payTime"`         // 支付时间
	CreateTime      string `json:"createTime"`      // 下单时间
	FinishTime      string `json:"finishTime"`      // 完成时间
	UpdateTime      string `json:"updateTime"`      // 更新时间
	CancelReason    string `json:"cancelReason"`    // 取消原因

	// ===== 金额信息（店铺级，单位：元） =====
	PayAmount               float64 `json:"payAmount"`               // 支付金额
	OrderAmount             float64 `json:"orderAmount"`             // 订单原价
	PostAmount              float64 `json:"postAmount"`              // 运费
	PostInsuranceAmount     float64 `json:"postInsuranceAmount"`     // 运费险
	PromotionAmount         float64 `json:"promotionAmount"`         // 优惠总金额
	PromotionShopAmount     float64 `json:"promotionShopAmount"`     // 店铺优惠
	PromotionPlatformAmount float64 `json:"promotionPlatformAmount"` // 平台优惠
	PromotionTalentAmount   float64 `json:"promotionTalentAmount"`   // 达人优惠
	PromotionPayAmount      float64 `json:"promotionPayAmount"`      // 支付优惠
	ShopCostAmount          float64 `json:"shopCostAmount"`          // 商家承担金额
	PlatformCostAmount      float64 `json:"platformCostAmount"`      // 平台承担金额
	AuthorCostAmount        float64 `json:"authorCostAmount"`        // 达人承担金额
	ModifyAmount            float64 `json:"modifyAmount"`            // 改价金额
	ModifyPostAmount        float64 `json:"modifyPostAmount"`        // 改价运费

	// ===== 收货信息（店铺级） =====
	PostReceiver string `json:"postReceiver"` // 收件人姓名（脱敏）
	PostTel      string `json:"postTel"`      // 收件人电话（脱敏）
	PostAddr     string `json:"postAddr"`     // 收件人完整地址
	PostProvince string `json:"postProvince"` // 省
	PostCity     string `json:"postCity"`     // 市
	PostTown     string `json:"postTown"`     // 区
	PostDetail   string `json:"postDetail"`   // 详细地址

	// ===== 物流信息（店铺级） =====
	TrackingNo  string `json:"trackingNo"`  // 物流单号（首个）
	CompanyName string `json:"companyName"` // 物流公司（首个）
	ShipTime    string `json:"shipTime"`    // 发货时间

	// ===== 买家/商家信息（店铺级） =====
	BuyerWords  string `json:"buyerWords"`  // 买家留言
	SellerWords string `json:"sellerWords"` // 商家备注
	PayType     int64  `json:"payType"`     // 支付方式
	BType       int64  `json:"bType"`       // 下单端
	BTypeDesc   string `json:"bTypeDesc"`   // 下单端描述
	OpenId      string `json:"openId"`      // 买家open_id

	// ===== SKU级信息 =====
	SubOrderNo   string  `json:"subOrderNo"`   // 商品单号（SKU订单号）
	ProductId    int64   `json:"productId"`    // 商品ID
	ProductIdStr string  `json:"productIdStr"` // 商品ID（字符串）
	SkuId        int64   `json:"skuId"`        // SKU ID
	Code         string  `json:"code"`         // 商家后台商品编码
	OutSkuId     string  `json:"outSkuId"`     // 外部SKU编码
	OutProductId string  `json:"outProductId"` // 外部商品编码
	ProductName  string  `json:"productName"`  // 商品名称
	ProductPic   string  `json:"productPic"`   // 商品图片
	ProductNum   int     `json:"productNum"`   // 购买数量
	ProductPrice float64 `json:"productPrice"` // 商品单价（原价）
	SumAmount    float64 `json:"sumAmount"`    // 商品总价（单价*数量）
	Spec         string  `json:"spec"`         // 规格信息（JSON）
	GoodsType    int64   `json:"goodsType"`    // 商品类型：0实体 1虚拟

	// ===== 售后信息（SKU级） =====
	AfterSaleStatus int64 `json:"afterSaleStatus"` // 售后状态码
	AfterSaleType   int64 `json:"afterSaleType"`   // 售后类型

	// ===== 达人信息（SKU级） =====
	AuthorId   int64  `json:"authorId"`   // 达人ID
	AuthorName string `json:"authorName"` // 达人名称

	// ===== 来源信息（SKU级） =====
	Biz       int64  `json:"biz"`       // 业务来源
	BizDesc   string `json:"bizDesc"`   // 业务来源描述
	CBiz      int64  `json:"cBiz"`      // C端流量来源
	CBizDesc  string `json:"cBizDesc"`  // C端流量来源描述
	AdEnvType string `json:"adEnvType"` // 广告来源 video/live
}

// PlatformOrderListResult 订单列表结果（通用）
type PlatformOrderListResult struct {
	Total  int                 `json:"total"`
	Page   int                 `json:"page"`
	Size   int                 `json:"size"`
	Orders []PlatformOrderItem `json:"orders"`
}

// ==================== 精选联盟订单 ====================

// PlatformAllianceOrderListParams 精选联盟订单列表参数
type PlatformAllianceOrderListParams struct {
	Platform  string `json:"platform"`
	Size      int    `json:"size" d:"20"` // 每页数量，最大20
	Cursor    string `json:"cursor"`      // 游标，首页传"0"
	StartTime string `json:"startTime"`   // 支付时间开始 yyyy-MM-dd HH:mm:ss
	EndTime   string `json:"endTime"`     // 支付时间结束
	StoreId   int    `json:"storeId"`
	OrderIds  string `json:"orderIds"` // 订单号筛选
	TimeType  string `json:"timeType"` // pay:支付时间 update:联盟更新时间
}

// PlatformAllianceOrderListResult 精选联盟订单列表结果
type PlatformAllianceOrderListResult struct {
	Cursor string                      `json:"cursor"` // 下一页游标，空表示没有更多
	Orders []PlatformAllianceOrderItem `json:"orders"`
}

// PlatformAllianceOrderItem 精选联盟订单
type PlatformAllianceOrderItem struct {
	Platform           string `json:"platform"`
	OrderId            string `json:"orderId"`
	ProductId          string `json:"productId"`
	AuthorAccount      string `json:"authorAccount"`
	CommissionRate     int64  `json:"commissionRate"`     // 500=5%
	EstimatedComission int64  `json:"estimatedComission"` // 预估佣金(分)
	RealComission      int64  `json:"realComission"`      // 实际佣金(分)
	TotalPayAmount     int64  `json:"totalPayAmount"`     // 支付金额(分)
	OrderStatus        string `json:"orderStatus"`
	ShopId             string `json:"shopId"`
	AllianceBizType    string `json:"allianceBizType"`
	PhaseId            int64  `json:"phaseId"`
	ShortId            string `json:"shortId"`
}

// ==================== 抖店售后/退款订单 ====================

// PlatformRefundOrderListParams 售后/退款订单列表参数
type PlatformRefundOrderListParams struct {
	Platform      string  `json:"platform"`
	Page          int     `json:"page" d:"0"`
	PageSize      int     `json:"pageSize" d:"100"`
	StartTime     string  `json:"startTime" dc:"开始时间 2006-01-02 15:04:05"`
	EndTime       string  `json:"endTime" dc:"结束时间 2006-01-02 15:04:05"`
	StoreId       int     `json:"storeId" dc:"店铺ID（store表主键）"`
	OrderNo       string  `json:"orderNo" dc:"父订单号"`
	AftersaleId   string  `json:"aftersaleId" dc:"售后单号"`
	AftersaleType *int64  `json:"aftersaleType" dc:"售后类型"`
	StatusList    []int64 `json:"statusList" dc:"售后状态列表"`
	UseUpdateTime bool    `json:"useUpdateTime" dc:"是否按售后更新时间查询"`
}

// PlatformRefundOrderListResult 售后/退款订单列表结果
type PlatformRefundOrderListResult struct {
	Total   int                       `json:"total"`
	Page    int                       `json:"page"`
	Size    int                       `json:"size"`
	HasMore bool                      `json:"hasMore"`
	Orders  []PlatformRefundOrderItem `json:"orders"`
}

// PlatformRefundOrderItem 售后/退款订单
type PlatformRefundOrderItem struct {
	Platform                   string  `json:"platform"`
	ShopId                     int64   `json:"shopId"`
	ShopName                   string  `json:"shopName"`
	ShopOrderId                string  `json:"shopOrderId"`
	SkuOrderId                 string  `json:"skuOrderId"`
	AftersaleId                string  `json:"aftersaleId"`
	AftersaleOrderType         int64   `json:"aftersaleOrderType"`
	AftersaleType              int64   `json:"aftersaleType"`
	AftersaleTypeText          string  `json:"aftersaleTypeText"`
	AftersaleStatus            int64   `json:"aftersaleStatus"`
	AftersaleStatusText        string  `json:"aftersaleStatusText"`
	AftersaleRefundType        int64   `json:"aftersaleRefundType"`
	AftersaleRefundTypeText    string  `json:"aftersaleRefundTypeText"`
	RefundType                 int64   `json:"refundType"`
	RefundStatus               int64   `json:"refundStatus"`
	ReasonText                 string  `json:"reasonText"`
	RefundAmount               float64 `json:"refundAmount"`
	RefundPostAmount           float64 `json:"refundPostAmount"`
	RefundTaxAmount            float64 `json:"refundTaxAmount"`
	ReturnPromotionAmount      float64 `json:"returnPromotionAmount"`
	AftersaleNum               int64   `json:"aftersaleNum"`
	PartType                   int64   `json:"partType"`
	ApplyTime                  string  `json:"applyTime"`
	UpdateTime                 string  `json:"updateTime"`
	CreateTime                 string  `json:"createTime"`
	StatusDeadline             string  `json:"statusDeadline"`
	AftersaleStatusToFinalTime string  `json:"aftersaleStatusToFinalTime"`
	ProductId                  string  `json:"productId"`
	ProductName                string  `json:"productName"`
	ProductImage               string  `json:"productImage"`
	ProductSpec                string  `json:"productSpec"`
	ShopSkuCode                string  `json:"shopSkuCode"`
	ProductNum                 int64   `json:"productNum"`
	ProductPrice               float64 `json:"productPrice"`
	PayAmount                  float64 `json:"payAmount"`
	PostAmount                 float64 `json:"postAmount"`
	OrderStatus                int64   `json:"orderStatus"`
	OrderCreateTime            string  `json:"orderCreateTime"`
	ReturnLogisticsCode        string  `json:"returnLogisticsCode"`
	ReturnLogisticsCompanyName string  `json:"returnLogisticsCompanyName"`
	OrderFlag                  int64   `json:"orderFlag"`
	RiskDecisionCode           int64   `json:"riskDecisionCode"`
	RiskDecisionReason         string  `json:"riskDecisionReason"`
	RiskDecisionDescription    string  `json:"riskDecisionDescription"`
	ArbitrateStatus            int64   `json:"arbitrateStatus"`
	Remark                     string  `json:"remark"`
	RawJSON                    string  `json:"rawJson"`
}
