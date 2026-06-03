package topup_CreateOrderPreCheck_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupCreateOrderPreCheckResponse struct {
	doudian_sdk.BaseDoudianOpSpiResponse
	Data *TopupCreateOrderPreCheckData `json:"data"`
}
func (c *TopupCreateOrderPreCheckResponse) GetData() interface{}{
	 return c.Data
}


type TopupCreateOrderPreCheckData struct {
}
