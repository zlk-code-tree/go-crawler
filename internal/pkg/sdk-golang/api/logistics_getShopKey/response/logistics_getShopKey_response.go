package logistics_getShopKey_response

import (
	"doudian.com/open/sdk_golang/core"
)

type LogisticsGetShopKeyResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *LogisticsGetShopKeyData `json:"data"`
}
type LogisticsGetShopKeyData struct {
	// 1
	Key string `json:"key"`
}
