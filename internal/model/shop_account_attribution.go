package model

import "time"

// ShopAccountAttribution 店铺借用账号归属规则
type ShopAccountAttribution struct {
	Id                 int       `json:"id"`
	ShopName           string    `json:"shopName"`           // 店铺名称
	SourceAccountUid   string    `json:"sourceAccountUid"`   // 借用的达人UID
	SourceAccountName  string    `json:"sourceAccountName"`  // 借用的达人名称
	TargetType         string    `json:"targetType"`         // 归属类型：operator=运营组, trader=投手
	TargetValue        string    `json:"targetValue"`        // 归属目标
	Remark             string    `json:"remark"`             // 备注
	Status             int       `json:"status"`             // 状态：0=禁用 1=启用
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// ShopAccountAttributionListReq 列表请求
type ShopAccountAttributionListReq struct {
	Page      int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize  int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	Keyword   string `json:"keyword"`   // 搜索关键词
	ShopName  string `json:"shopName"`  // 店铺筛选
}

// ShopAccountAttributionSaveReq 保存请求
type ShopAccountAttributionSaveReq struct {
	Id                int    `json:"id"`
	ShopName          string `json:"shopName" v:"required#店铺名称不能为空"`
	SourceAccountUid  string `json:"sourceAccountUid" v:"required#达人UID不能为空"`
	SourceAccountName string `json:"sourceAccountName"`
	TargetType        string `json:"targetType" v:"required#归属类型不能为空"`
	TargetValue       string `json:"targetValue" v:"required#归属目标不能为空"`
	Remark            string `json:"remark"`
}
