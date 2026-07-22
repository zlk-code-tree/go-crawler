package model

import "time"

// QianchuanAccountOwner 千川账户归属管理
type QianchuanAccountOwner struct {
	Id             int64      `json:"id"`
	AdvertiserId   int64      `json:"advertiserId"`
	AdvertiserName string     `json:"advertiserName"`
	OwnerName      string     `json:"ownerName"`
	TraderId       int64      `json:"traderId"`
	TraderName     string     `json:"traderName"`
	CompanyId      int64      `json:"companyId"`
	CompanyName    string     `json:"companyName"`
	DepartmentId   int64      `json:"departmentId"`
	DepartmentName string     `json:"departmentName"`
	Remark         string     `json:"remark"`
	Status         int        `json:"status"`
	SourceFile     string     `json:"sourceFile"`
	SourceSheet    string     `json:"sourceSheet"`
	SourceRowNo    int        `json:"sourceRowNo"`
	CreatedAt      *time.Time `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
}

// QianchuanAccountOwnerListReq 列表请求
type QianchuanAccountOwnerListReq struct {
	Page         int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize     int    `json:"pageSize" v:"required|min:1|max:200#每页数量不能为空|每页数量最小为1|每页数量最大为200"`
	Keyword      string `json:"keyword"`
	OwnerName    string `json:"ownerName"`
	AdvertiserId int64  `json:"advertiserId"`
	Status       int    `json:"status"`
}

// QianchuanAccountOwnerSaveReq 保存请求
type QianchuanAccountOwnerSaveReq struct {
	Id             int64  `json:"id"`
	AdvertiserId   int64  `json:"advertiserId" v:"required#千川户号不能为空"`
	AdvertiserName string `json:"advertiserName" v:"required#千川户名称不能为空"`
	OwnerName      string `json:"ownerName" v:"required#达人/归属不能为空"`
	TraderId       int64  `json:"traderId"`
	Remark         string `json:"remark"`
	Status         int    `json:"status" d:"1"`
}
