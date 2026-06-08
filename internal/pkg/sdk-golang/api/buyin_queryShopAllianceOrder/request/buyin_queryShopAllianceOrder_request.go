package buyin_queryShopAllianceOrder_request

import (
	"doudian.com/open/sdk_golang/api/buyin_queryShopAllianceOrder/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinQueryShopAllianceOrderRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinQueryShopAllianceOrderParam
}

func (c *BuyinQueryShopAllianceOrderRequest) GetUrlPath() string {
	return "/buyin/queryShopAllianceOrder"
}

func New() *BuyinQueryShopAllianceOrderRequest {
	request := &BuyinQueryShopAllianceOrderRequest{
		Param: &BuyinQueryShopAllianceOrderParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request
}

func (c *BuyinQueryShopAllianceOrderRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_queryShopAllianceOrder_response.BuyinQueryShopAllianceOrderResponse, error) {
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_queryShopAllianceOrder_response.BuyinQueryShopAllianceOrderResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil
}

func (c *BuyinQueryShopAllianceOrderRequest) GetParamObject() interface{} { return c.Param }
func (c *BuyinQueryShopAllianceOrderRequest) GetParams() *BuyinQueryShopAllianceOrderParam { return c.Param }

// BuyinQueryShopAllianceOrderParam 请求参数（全部透传，无默认值）
type BuyinQueryShopAllianceOrderParam struct {
	Size      *int64  `json:"size,omitempty"`
	Cursor    *string `json:"cursor,omitempty"`
	StartTime *string `json:"start_time,omitempty"`
	EndTime   *string `json:"end_time,omitempty"`
	OrderIds  *string `json:"order_ids,omitempty"`
	TimeType  *string `json:"time_type,omitempty"`
}
