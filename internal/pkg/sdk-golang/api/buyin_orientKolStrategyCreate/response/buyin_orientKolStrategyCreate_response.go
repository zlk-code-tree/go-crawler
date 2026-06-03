package buyin_orientKolStrategyCreate_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinOrientKolStrategyCreateResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinOrientKolStrategyCreateData `json:"data"`
}
type FailedItem struct {
	// 达人百应ID
	BuyinId int64 `json:"buyin_id"`
	// 失败原因
	Reason string `json:"reason"`
}
type BuyinOrientKolStrategyCreateData struct {
	// 策略是否设置成功;
	Success bool `json:"success"`
	// 策略ID
	StrategyId int64 `json:"strategy_id"`
	// 失败的达人维度结果信息
	Failed []FailedItem `json:"failed"`
}
