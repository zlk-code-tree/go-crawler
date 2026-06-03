package topup_orderQueryCUC_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupOrderQueryCUCResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *TopupOrderQueryCUCData `json:"data"`
}
type TopupOrderQueryCUCData struct {
	// 查询流水号
	QueryId string `json:"queryId"`
	// 合作伙伴编号
	ParterNo string `json:"parterNo"`
	// 区号，话费不适用
	AreaCode string `json:"areaCode"`
	// 实际缴费金额
	Fee string `json:"fee"`
	// 其他额外信息
	Extra string `json:"extra"`
	// 电商订单Id
	PayTradeID string `json:"payTradeID"`
	// 充值手机号
	PhoneNo string `json:"phoneNo"`
	// 第一次订单完成时间 -- 充值成功
	FirCompleteTime string `json:"firCompleteTime"`
	// 第二次订单完成时间 -- 收货后退款
	SndCompleteTime string `json:"sndCompleteTime"`
	// 格式为:YYYYMMDDHH24MISS
	TimeStamp string `json:"timeStamp"`
}
