package open_getAuthInfo_request

import (
	"doudian.com/open/sdk_golang/api/open_getAuthInfo/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OpenGetAuthInfoRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OpenGetAuthInfoParam 
}
func (c *OpenGetAuthInfoRequest) GetUrlPath() string{
	return "/open/getAuthInfo"
}


func New() *OpenGetAuthInfoRequest{
	request := &OpenGetAuthInfoRequest{
		Param: &OpenGetAuthInfoParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OpenGetAuthInfoRequest) Execute(accessToken *doudian_sdk.AccessToken) (*open_getAuthInfo_response.OpenGetAuthInfoResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &open_getAuthInfo_response.OpenGetAuthInfoResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OpenGetAuthInfoRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OpenGetAuthInfoRequest) GetParams() *OpenGetAuthInfoParam{
	return c.Param
}


type OpenGetAuthInfoParam struct {
	// 授权主体ID，这里为店铺ID
	AuthId string `json:"auth_id"`
	// 授权主体类型，默认为店铺授权，不需要填写。其他类型枚举值如下：YunCang -云仓；WuLiuShang -物流商；WLGongYingShang -物流供应商；MiniApp -小程序 MCN-联盟MCN机构 DouKe-联盟抖客 Colonel-联盟团长
	AuthSubjectType string `json:"auth_subject_type"`
}
