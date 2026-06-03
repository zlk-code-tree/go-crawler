package open_materialToken_request

import (
	"doudian.com/open/sdk_golang/api/open_materialToken/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OpenMaterialTokenRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OpenMaterialTokenParam 
}
func (c *OpenMaterialTokenRequest) GetUrlPath() string{
	return "/open/materialToken"
}


func New() *OpenMaterialTokenRequest{
	request := &OpenMaterialTokenRequest{
		Param: &OpenMaterialTokenParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OpenMaterialTokenRequest) Execute(accessToken *doudian_sdk.AccessToken) (*open_materialToken_response.OpenMaterialTokenResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &open_materialToken_response.OpenMaterialTokenResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OpenMaterialTokenRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OpenMaterialTokenRequest) GetParams() *OpenMaterialTokenParam{
	return c.Param
}


type OpenMaterialTokenParam struct {
	// 需要上传的素材数量
	UploadNum int64 `json:"upload_num"`
	// 文件后缀名称，必须以"."开头
	FileExtension *string `json:"file_extension"`
	// 业务类型，  1: 素材中心 。 2: ImageX
	BizType *int32 `json:"biz_type"`
	// 当 biz_type 为 2，则可以自定义路径名
	StoreKeysArrayJson *string `json:"store_keys_array_json"`
}
