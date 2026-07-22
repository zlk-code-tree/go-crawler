package model

import "time"

// Store 店铺信息（外部平台店铺ID）
type Store struct {
	Id                    int        `json:"id"`
	StoreName             string     `json:"storeName"` // 店铺名称
	StoreId               string     `json:"storeId"`   // 店铺ID（外部平台店铺ID）
	Platform              string     `json:"platform"`  // 平台：douyin/kuaishou/shipinhao
	Status                int        `json:"status"`    // 状态：0-禁用 1-启用
	Author                int        `json:"author"`    // 授权状态：0-未授权 1-已授权
	BlankOrderEnabled     int        `json:"blankOrderEnabled"`
	BlankOrderOwnerName   string     `json:"blankOrderOwnerName"`
	LastPollTime          *time.Time `json:"lastPollTime,omitempty"`
	WechatAppId           string     `json:"wechatAppId,omitempty"`
	HasWechatAppSecret    int        `json:"hasWechatAppSecret,omitempty"`
	MaskedWechatAppSecret string     `json:"maskedWechatAppSecret,omitempty"`
	WechatTokenExpiresAt  *time.Time `json:"wechatTokenExpiresAt,omitempty"`
	WechatLastPullTime    *time.Time `json:"wechatLastPullTime,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

// StoreSaveReq 保存店铺请求
type StoreSaveReq struct {
	Id                  int    `json:"id"`
	StoreName           string `json:"storeName" v:"required#店铺名称不能为空"`
	StoreId             string `json:"storeId" v:"required#店铺ID不能为空"`
	Platform            string `json:"platform" v:"required|in:douyin,kuaishou,shipinhao#平台不能为空|平台非法"`
	Status              int    `json:"status" d:"1"`
	Author              int    `json:"author" d:"0"`
	BlankOrderEnabled   int    `json:"blankOrderEnabled" d:"1"`
	BlankOrderOwnerName string `json:"blankOrderOwnerName"`
	WechatAppId         string `json:"wechatAppId"`
	WechatAppSecret     string `json:"wechatAppSecret"`
}
