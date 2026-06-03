package openCloud_v1_imageVersion_create_request

import (
	"doudian.com/open/sdk_golang/api/openCloud_v1_imageVersion_create/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OpenCloudV1ImageVersionCreateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OpenCloudV1ImageVersionCreateParam 
}
func (c *OpenCloudV1ImageVersionCreateRequest) GetUrlPath() string{
	return "/openCloud/v1/imageVersion/create"
}


func New() *OpenCloudV1ImageVersionCreateRequest{
	request := &OpenCloudV1ImageVersionCreateRequest{
		Param: &OpenCloudV1ImageVersionCreateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OpenCloudV1ImageVersionCreateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*openCloud_v1_imageVersion_create_response.OpenCloudV1ImageVersionCreateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &openCloud_v1_imageVersion_create_response.OpenCloudV1ImageVersionCreateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OpenCloudV1ImageVersionCreateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OpenCloudV1ImageVersionCreateRequest) GetParams() *OpenCloudV1ImageVersionCreateParam{
	return c.Param
}


type OpenCloudV1ImageVersionCreateParam struct {
	// 容器服务id
	CsId string `json:"cs_id"`
	// 版本名称
	VersionName string `json:"version_name"`
	// 备注
	Remark string `json:"remark"`
	// 程序包上传素材中心后返回的uri
	FileUri string `json:"file_uri"`
	// 文件名称
	FileName string `json:"file_name"`
	// 容器名称
	CsName string `json:"cs_name"`
	// 如果是强管控、跨境、微应用等租户，需要传火山账号id
	VolcAccountId *int64 `json:"volc_account_id"`
}
