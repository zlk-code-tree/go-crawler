package promise_SaveTemplate_response

import (
	"doudian.com/open/sdk_golang/core"
)

type PromiseSaveTemplateResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *PromiseSaveTemplateData `json:"data"`
}
type PromiseSaveTemplateData struct {
	// 模板ID
	TemplateId string `json:"template_id"`
}
