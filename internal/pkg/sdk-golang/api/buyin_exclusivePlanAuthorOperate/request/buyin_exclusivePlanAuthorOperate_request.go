package buyin_exclusivePlanAuthorOperate_request

import (
	"doudian.com/open/sdk_golang/api/buyin_exclusivePlanAuthorOperate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinExclusivePlanAuthorOperateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinExclusivePlanAuthorOperateParam 
}
func (c *BuyinExclusivePlanAuthorOperateRequest) GetUrlPath() string{
	return "/buyin/exclusivePlanAuthorOperate"
}


func New() *BuyinExclusivePlanAuthorOperateRequest{
	request := &BuyinExclusivePlanAuthorOperateRequest{
		Param: &BuyinExclusivePlanAuthorOperateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinExclusivePlanAuthorOperateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_exclusivePlanAuthorOperate_response.BuyinExclusivePlanAuthorOperateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_exclusivePlanAuthorOperate_response.BuyinExclusivePlanAuthorOperateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinExclusivePlanAuthorOperateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinExclusivePlanAuthorOperateRequest) GetParams() *BuyinExclusivePlanAuthorOperateParam{
	return c.Param
}


type BuyinExclusivePlanAuthorOperateParam struct {
	// 操作类型 0: 添加达人; 1: 开启达人； 2：关闭达人； 3: 删除达人
	OpType int32 `json:"op_type"`
	// 达人百应ID
	AuthorBuyinId int64 `json:"author_buyin_id"`
}
