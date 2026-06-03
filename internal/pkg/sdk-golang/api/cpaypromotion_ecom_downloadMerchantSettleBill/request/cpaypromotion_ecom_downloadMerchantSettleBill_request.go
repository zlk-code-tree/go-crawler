package cpaypromotion_ecom_downloadMerchantSettleBill_request

import (
	"doudian.com/open/sdk_golang/api/cpaypromotion_ecom_downloadMerchantSettleBill/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CpaypromotionEcomDownloadMerchantSettleBillRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CpaypromotionEcomDownloadMerchantSettleBillParam 
}
func (c *CpaypromotionEcomDownloadMerchantSettleBillRequest) GetUrlPath() string{
	return "/cpaypromotion/ecom/downloadMerchantSettleBill"
}


func New() *CpaypromotionEcomDownloadMerchantSettleBillRequest{
	request := &CpaypromotionEcomDownloadMerchantSettleBillRequest{
		Param: &CpaypromotionEcomDownloadMerchantSettleBillParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CpaypromotionEcomDownloadMerchantSettleBillRequest) Execute(accessToken *doudian_sdk.AccessToken) (*cpaypromotion_ecom_downloadMerchantSettleBill_response.CpaypromotionEcomDownloadMerchantSettleBillResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &cpaypromotion_ecom_downloadMerchantSettleBill_response.CpaypromotionEcomDownloadMerchantSettleBillResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CpaypromotionEcomDownloadMerchantSettleBillRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CpaypromotionEcomDownloadMerchantSettleBillRequest) GetParams() *CpaypromotionEcomDownloadMerchantSettleBillParam{
	return c.Param
}


type CpaypromotionEcomDownloadMerchantSettleBillParam struct {
	// 下载id，通过调用/cpaypromotion/ecom/applyMerchantSettleBill接口获取
	DownloadId string `json:"download_id"`
}
