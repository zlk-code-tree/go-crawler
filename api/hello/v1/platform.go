package v1

import (
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== 第三方平台对接 ====================

// PlatformOrderListReq 获取平台订单列表
type PlatformOrderListReq struct {
	g.Meta      `path:"/platform/order-list" method:"get" tags:"Platform" summary:"获取平台订单列表"`
	Platform    string `json:"platform" in:"query" v:"required|in:douyin,kuaishou,shipinhao" dc:"平台标识"`
	Page        int    `json:"page" in:"query" d:"0" dc:"页码（从0开始）"`
	PageSize    int    `json:"pageSize" in:"query" d:"20" dc:"每页数量（最大100）"`
	StartTime   string `json:"startTime" in:"query" dc:"开始时间 2006-01-02 15:04:05"`
	EndTime     string `json:"endTime" in:"query" dc:"结束时间 2006-01-02 15:04:05"`
	OrderStatus string `json:"orderStatus" in:"query" dc:"订单状态"`
	AccountId   int64  `json:"accountId" in:"query" dc:"账号ID"`
	StoreId     int    `json:"storeId" in:"query" dc:"店铺ID（store表主键）"`
}

// PlatformOrderListRes 平台订单列表响应
type PlatformOrderListRes = model.PlatformOrderListResult

// PlatformOrderDetailReq 获取平台订单详情
type PlatformOrderDetailReq struct {
	g.Meta   `path:"/platform/order-detail" method:"get" tags:"Platform" summary:"获取平台订单详情"`
	Platform string `json:"platform" in:"query" v:"required|in:douyin,kuaishou,shipinhao" dc:"平台标识"`
	OrderNo  string `json:"orderNo" in:"query" v:"required" dc:"订单号"`
	AccountId int64  `json:"accountId" in:"query" dc:"账号ID"`
	StoreId   int    `json:"storeId" in:"query" dc:"店铺ID（store表主键）"`
}

// PlatformOrderDetailRes 平台订单详情响应
type PlatformOrderDetailRes = model.PlatformOrderItem
