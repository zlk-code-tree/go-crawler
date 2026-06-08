package alliance_getOrderList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AllianceGetOrderListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *AllianceGetOrderListData `json:"data"`
}

type AllianceGetOrderListData struct {
	Code    string              `json:"code"`
	CodeMsg string              `json:"code_msg"`
	Cursor  string              `json:"cursor"`
	Orders  []AllianceOrderItem `json:"datas"`
}

// AllianceOrderItem 精选联盟订单
type AllianceOrderItem struct {
	OrderId            string `json:"order_id"`
	ProductId          string `json:"product_id"`
	AuthorAccount      string `json:"author_account"`
	CommissionRate     int64  `json:"commission_rate"`      // 500=5%
	EstimatedComission int64  `json:"estimated_comission"`  // 预估佣金(分)
	RealComission      int64  `json:"real_comission"`       // 实际佣金(分)
	TotalPayAmount     int64  `json:"total_pay_amount"`     // 支付金额(分)
	OrderStatus        string `json:"order_status"`
	ShopId             string `json:"shop_id"`
	AllianceBizType    string `json:"alliance_biz_type"`
	PhaseId            int64  `json:"phase_id"`
	ShortId            string `json:"short_id"`
}
