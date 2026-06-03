package topup_accountTemplateList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupAccountTemplateListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *TopupAccountTemplateListData `json:"data"`
}
type TemplateListItem struct {
	// 模板id
	TemplateId int64 `json:"template_id"`
	// 模板名称
	Name string `json:"name"`
}
type TopupAccountTemplateListData struct {
	// 模板列表
	TemplateList []TemplateListItem `json:"template_list"`
}
