package topup_accountTemplateList_request

import (
	"doudian.com/open/sdk_golang/api/topup_accountTemplateList/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type TopupAccountTemplateListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *TopupAccountTemplateListParam 
}
func (c *TopupAccountTemplateListRequest) GetUrlPath() string{
	return "/topup/accountTemplateList"
}


func New() *TopupAccountTemplateListRequest{
	request := &TopupAccountTemplateListRequest{
		Param: &TopupAccountTemplateListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *TopupAccountTemplateListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*topup_accountTemplateList_response.TopupAccountTemplateListResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &topup_accountTemplateList_response.TopupAccountTemplateListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *TopupAccountTemplateListRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *TopupAccountTemplateListRequest) GetParams() *TopupAccountTemplateListParam{
	return c.Param
}


type TopupAccountTemplateListParam struct {
	// 叶子类目id
	CategoryId int64 `json:"category_id"`
}
