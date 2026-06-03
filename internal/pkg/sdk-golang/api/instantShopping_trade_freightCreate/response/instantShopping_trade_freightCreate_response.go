package instantShopping_trade_freightCreate_response

import (
	"doudian.com/open/sdk_golang/core"
)

type InstantShoppingTradeFreightCreateResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *InstantShoppingTradeFreightCreateData `json:"data"`
}
type InstantShoppingTradeFreightCreateData struct {
	// 模板id
	TemplateId int64 `json:"template_id"`
}
