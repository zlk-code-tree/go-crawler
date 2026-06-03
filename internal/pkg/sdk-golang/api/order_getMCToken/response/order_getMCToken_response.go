package order_getMCToken_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderGetMCTokenResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderGetMCTokenData `json:"data"`
}
type OrderGetMCTokenData struct {
	// 前端组件token
	Token string `json:"token"`
	// token过期时间，时间戳，秒级
	ExpireTime int64 `json:"expire_time"`
}
