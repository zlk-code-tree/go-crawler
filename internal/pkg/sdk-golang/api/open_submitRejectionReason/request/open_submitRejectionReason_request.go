package open_submitRejectionReason_request

import (
	"doudian.com/open/sdk_golang/api/open_submitRejectionReason/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OpenSubmitRejectionReasonRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OpenSubmitRejectionReasonParam 
}
func (c *OpenSubmitRejectionReasonRequest) GetUrlPath() string{
	return "/open/submitRejectionReason"
}


func New() *OpenSubmitRejectionReasonRequest{
	request := &OpenSubmitRejectionReasonRequest{
		Param: &OpenSubmitRejectionReasonParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OpenSubmitRejectionReasonRequest) Execute(accessToken *doudian_sdk.AccessToken) (*open_submitRejectionReason_response.OpenSubmitRejectionReasonResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &open_submitRejectionReason_response.OpenSubmitRejectionReasonResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OpenSubmitRejectionReasonRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OpenSubmitRejectionReasonRequest) GetParams() *OpenSubmitRejectionReasonParam{
	return c.Param
}


type OpenSubmitRejectionReasonParam struct {
	// rejection_reason_code=2商品来源名称；rejection_reason_code=4标品类目名称
	ProductCategoryName *string `json:"product_category_name"`
	// 预测记录id
	RecommendId string `json:"recommend_id"`
	// rejection_reason_code=100时，填写有效 拒绝原因描述
	Reason *string `json:"reason"`
	// 拒绝原因 1-商家不认可TOP1推荐类目、自行手动修改 2-TOP1类目非货源原始类目 3-商家无TOP1类目资质 4-商品为标品类目，无法采纳⑤和ISV自身预测类目不一致 100-其他
	RejectionReasonCode int32 `json:"rejection_reason_code"`
}
