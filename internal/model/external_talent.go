package model

import "time"

// ExternalTalent 外部达人台账
type ExternalTalent struct {
	Id                    int       `json:"id"`
	TeamName              string    `json:"teamName"`
	PlatformType          string    `json:"platformType"`
	ShopName              string    `json:"shopName"`
	PlatformShopLabel     string    `json:"platformShopLabel"`
	Uid                   string    `json:"uid"`
	AccountName           string    `json:"accountName"`
	ShowId                string    `json:"showId"`
	LinkCode              string    `json:"linkCode"`
	TalentName            string    `json:"talentName"`
	TalentType            string    `json:"talentType"`
	ProductId             string    `json:"productId"`
	GoodsCode             string    `json:"goodsCode"`
	CustomerPrice         float64   `json:"customerPrice"`
	OnlineCommissionRate  float64   `json:"onlineCommissionRate"`
	OfflineCommissionRate float64   `json:"offlineCommissionRate"`
	RetainRate            float64   `json:"retainRate"`
	Remark                string    `json:"remark"`
	Status                int       `json:"status"`
	SourceFile            string    `json:"sourceFile"`
	SourceSheet           string    `json:"sourceSheet"`
	SourceRowNo           int       `json:"sourceRowNo"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

// ExternalTalentListReq 外部达人列表请求
type ExternalTalentListReq struct {
	Page         int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize     int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	Keyword      string `json:"keyword"`
	TeamName     string `json:"teamName"`
	PlatformType string `json:"platformType"`
	Status       int    `json:"status"`
}

// ExternalTalentSaveReq 外部达人保存请求
type ExternalTalentSaveReq struct {
	Id                    int     `json:"id"`
	TeamName              string  `json:"teamName" v:"required#团队名称不能为空"`
	PlatformType          string  `json:"platformType" v:"required#平台类型不能为空"`
	ShopName              string  `json:"shopName" v:"required#店铺名称不能为空"`
	PlatformShopLabel     string  `json:"platformShopLabel"`
	Uid                   string  `json:"uid"`
	AccountName           string  `json:"accountName"`
	ShowId                string  `json:"showId"`
	LinkCode              string  `json:"linkCode"`
	TalentName            string  `json:"talentName"`
	ProductId             string  `json:"productId"`
	GoodsCode             string  `json:"goodsCode"`
	CustomerPrice         float64 `json:"customerPrice"`
	OnlineCommissionRate  float64 `json:"onlineCommissionRate"`
	OfflineCommissionRate float64 `json:"offlineCommissionRate"`
	RetainRate            float64 `json:"retainRate"`
	Remark                string  `json:"remark"`
	Status                int     `json:"status"`
	SourceFile            string  `json:"sourceFile"`
	SourceSheet           string  `json:"sourceSheet"`
	SourceRowNo           int     `json:"sourceRowNo"`
}

// ExternalTalentStats 外部账号统计
type ExternalTalentStats struct {
	TotalCount       int64 `json:"totalCount"`
	DouyinCount      int64 `json:"douyinCount"`
	KuaishouCount    int64 `json:"kuaishouCount"`
	ShipinhaoCount   int64 `json:"shipinhaoCount"`
	XiaohongshuCount int64 `json:"xiaohongshuCount"`
	EnabledCount     int64 `json:"enabledCount"`
}
