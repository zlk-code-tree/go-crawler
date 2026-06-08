package buyin_queryShopAllianceOrder_response

import (
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinQueryShopAllianceOrderResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinQueryShopAllianceOrderData `json:"data"`
}

type BuyinQueryShopAllianceOrderData struct {
	Code    string                     `json:"code"`
	CodeMsg string                     `json:"code_msg"`
	Cursor  string                     `json:"cursor"`
	Datas   []json.RawMessage          `json:"datas"` // 每条订单原始JSON
}
