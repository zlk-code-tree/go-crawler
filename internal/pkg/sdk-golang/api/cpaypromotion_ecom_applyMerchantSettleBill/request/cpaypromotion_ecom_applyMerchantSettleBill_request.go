package cpaypromotion_ecom_applyMerchantSettleBill_request

import (
	"doudian.com/open/sdk_golang/api/cpaypromotion_ecom_applyMerchantSettleBill/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CpaypromotionEcomApplyMerchantSettleBillRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CpaypromotionEcomApplyMerchantSettleBillParam 
}
func (c *CpaypromotionEcomApplyMerchantSettleBillRequest) GetUrlPath() string{
	return "/cpaypromotion/ecom/applyMerchantSettleBill"
}


func New() *CpaypromotionEcomApplyMerchantSettleBillRequest{
	request := &CpaypromotionEcomApplyMerchantSettleBillRequest{
		Param: &CpaypromotionEcomApplyMerchantSettleBillParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CpaypromotionEcomApplyMerchantSettleBillRequest) Execute(accessToken *doudian_sdk.AccessToken) (*cpaypromotion_ecom_applyMerchantSettleBill_response.CpaypromotionEcomApplyMerchantSettleBillResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &cpaypromotion_ecom_applyMerchantSettleBill_response.CpaypromotionEcomApplyMerchantSettleBillResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CpaypromotionEcomApplyMerchantSettleBillRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CpaypromotionEcomApplyMerchantSettleBillRequest) GetParams() *CpaypromotionEcomApplyMerchantSettleBillParam{
	return c.Param
}


type CpaypromotionEcomApplyMerchantSettleBillParam struct {
	// 账单开始时间，格式：yyyyMMdd
	BillStartTime string `json:"bill_start_time"`
	// 账单结束时间，格式：yyyyMMdd
	BillEndTime string `json:"bill_end_time"`
}
