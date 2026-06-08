package alliance_getOrderList_request

import (
	"doudian.com/open/sdk_golang/api/alliance_getOrderList/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
	"fmt"
)

type AllianceGetOrderListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AllianceGetOrderListParam
}

func (c *AllianceGetOrderListRequest) GetUrlPath() string {
	return "/alliance/getOrderList"
}

func New() *AllianceGetOrderListRequest {
	request := &AllianceGetOrderListRequest{
		Param: &AllianceGetOrderListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request
}

func (c *AllianceGetOrderListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*alliance_getOrderList_response.AllianceGetOrderListResponse, error) {
	paramJson, _ := json.Marshal(c.Param)
	fmt.Printf("[AllianceAPI] 请求参数JSON: %s\n", string(paramJson))

	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[AllianceAPI] 返回JSON(前500字符): %s\n", truncateStr(responseJson, 500))

	response := &alliance_getOrderList_response.AllianceGetOrderListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max { return s }
	return s[:max] + "..."
}

func (c *AllianceGetOrderListRequest) GetParamObject() interface{} { return c.Param }
func (c *AllianceGetOrderListRequest) GetParams() *AllianceGetOrderListParam { return c.Param }

type AllianceGetOrderListParam struct {
	Size      int64   `json:"size"`                // 每页订单数目，1-20
	Cursor    string  `json:"cursor"`              // 下一页索引，第一页传"0"
	StartTime *string `json:"start_time,omitempty"` // 支付时间开始 yyyy-MM-dd HH:mm:ss
	EndTime   *string `json:"end_time,omitempty"`   // 支付时间结束
	OrderIds  []string `json:"order_ids,omitempty"` // 订单id列表
	TimeType  *string `json:"time_type,omitempty"`  // pay:支付时间 update:联盟侧更新时间
}
