package buyin_allKolStrategyCreate_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinAllKolStrategyCreateResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinAllKolStrategyCreateData `json:"data"`
}
type BuyinAllKolStrategyCreateData struct {
	// 是否设置成功
	Success bool `json:"success"`
	// 策略ID
	StrategyId int64 `json:"strategy_id"`
}
