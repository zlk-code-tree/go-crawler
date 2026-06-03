package material_searchMaterial_request

import (
	"doudian.com/open/sdk_golang/api/material_searchMaterial/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type MaterialSearchMaterialRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *MaterialSearchMaterialParam 
}
func (c *MaterialSearchMaterialRequest) GetUrlPath() string{
	return "/material/searchMaterial"
}


func New() *MaterialSearchMaterialRequest{
	request := &MaterialSearchMaterialRequest{
		Param: &MaterialSearchMaterialParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *MaterialSearchMaterialRequest) Execute(accessToken *doudian_sdk.AccessToken) (*material_searchMaterial_response.MaterialSearchMaterialResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &material_searchMaterial_response.MaterialSearchMaterialResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *MaterialSearchMaterialRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *MaterialSearchMaterialRequest) GetParams() *MaterialSearchMaterialParam{
	return c.Param
}


type MaterialSearchMaterialParam struct {
	// 素材id
	MaterialId *string `json:"material_id"`
	// 文件名称，支持模糊匹配
	MaterialName *string `json:"material_name"`
	// 素材类型，空-不限 photo-图片 video-视频
	MaterialType []string `json:"material_type"`
	// 素材状态，0-待下载 1-有效 4-回收站中
	OperateStatus []int32 `json:"operate_status"`
	// 审核状态，1-待审核 2-审核中 3-通过 4-拒绝
	AuditStatus []int32 `json:"audit_status"`
	// 搜索创建开始时间，格式：yyyy-MM-dd HH:mm:ss
	CreateTimeStart *string `json:"create_time_start"`
	// 搜索创建结束时间，格式：yyyy-MM-dd HH:mm:ss
	CreateTimeEnd *string `json:"create_time_end"`
	// 文件夹id，"0"--素材中心 "-1"--回收站
	FolderId *string `json:"folder_id"`
	// 素材id列表
	MaterialIdList []string `json:"material_id_list"`
	// 第几页，1，2，……，默认值：1
	PageNum *int32 `json:"page_num"`
	// 页大小，1，2，……，100，默认值：50
	PageSize *int32 `json:"page_size"`
	// 排序方式，0-按照创建时间倒序 1-按照创建时间升序 6-按照素材大小降序 7-按照素材大小升序
	OrderType *int32 `json:"order_type"`
}
