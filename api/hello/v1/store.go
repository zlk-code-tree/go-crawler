package v1

import (
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== 店铺管理 ====================

// StoreListReq 店铺列表请求
type StoreListReq struct {
	g.Meta   `path:"/store/list" method:"get" tags:"Store" summary:"获取店铺列表"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"20" dc:"每页数量"`
	Keyword  string `json:"keyword" in:"query" dc:"搜索关键词"`
	Platform string `json:"platform" in:"query" dc:"平台筛选"`
}

// StoreListRes 店铺列表响应
type StoreListRes struct {
	List  []model.Store `json:"list"`
	Total int64         `json:"total"`
}

// StoreDetailReq 店铺详情请求
type StoreDetailReq struct {
	g.Meta `path:"/store/detail" method:"get" tags:"Store" summary:"获取店铺详情"`
	Id     int `json:"id" in:"query" v:"required#店铺ID不能为空" dc:"店铺ID"`
}

// StoreDetailRes 店铺详情响应
type StoreDetailRes = model.Store

// StoreSaveReq 保存店铺请求
type StoreSaveReq struct {
	g.Meta          `path:"/store/save" method:"post" tags:"Store" summary:"保存店铺"`
	Id              int    `json:"id" dc:"店铺ID（更新时传）"`
	StoreName       string `json:"storeName" v:"required#店铺名称不能为空" dc:"店铺名称"`
	StoreId         string `json:"storeId" v:"required#店铺ID不能为空" dc:"店铺ID（外部平台店铺ID）"`
	Platform        string `json:"platform" v:"required|in:douyin,kuaishou,shipinhao#平台不能为空|平台非法" dc:"平台"`
	Status          int    `json:"status" d:"1" dc:"状态：0-禁用 1-启用"`
	Author          int    `json:"author" d:"0" dc:"授权状态：0-未授权 1-已授权"`
	WechatAppId     string `json:"wechatAppId" dc:"视频号店铺微信小店 AppID"`
	WechatAppSecret string `json:"wechatAppSecret" dc:"视频号店铺微信小店 AppSecret"`
}

// StoreSaveRes 保存店铺响应
type StoreSaveRes struct {
	Id int64 `json:"id"`
}

// StoreDeleteReq 删除店铺请求
type StoreDeleteReq struct {
	g.Meta `path:"/store/delete" method:"post" tags:"Store" summary:"删除店铺"`
	Id     int `json:"id" v:"required#店铺ID不能为空" dc:"店铺ID"`
}
