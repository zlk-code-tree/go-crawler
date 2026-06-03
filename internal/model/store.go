package model

import "time"

// Store 店铺信息（外部平台店铺ID）
type Store struct {
	Id        int       `json:"id"`
	StoreName string    `json:"storeName"` // 店铺名称
	StoreId   string    `json:"storeId"`   // 店铺ID（外部平台店铺ID）
	Platform  string    `json:"platform"`  // 平台：douyin/kuaishou/shipinhao
	Status    int       `json:"status"`    // 状态：0-禁用 1-启用
	Author    int       `json:"author"`    // 授权状态：0-未授权 1-已授权
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// StoreSaveReq 保存店铺请求
type StoreSaveReq struct {
	Id        int    `json:"id"`
	StoreName string `json:"storeName" v:"required#店铺名称不能为空"`
	StoreId   string `json:"storeId" v:"required#店铺ID不能为空"`
	Platform  string `json:"platform" v:"required#平台不能为空"`
	Status    int    `json:"status" d:"1"`
	Author    int    `json:"author" d:"0"`
}
