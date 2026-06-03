package platform

import (
	"context"
	"go-crawler/internal/model"
)

// PlatformAPI 所有平台必须实现的统一接口
type PlatformAPI interface {
	// Platform 返回平台标识（douyin/kuaishou/shipinhao）
	Platform() string

	// GetOrderList 获取订单列表
	GetOrderList(ctx context.Context, params model.PlatformOrderListParams) (*model.PlatformOrderListResult, error)

	// GetOrderDetail 获取订单详情
	GetOrderDetail(ctx context.Context, params model.PlatformOrderDetailParams) (*model.PlatformOrderItem, error)
}

// ============================================
// 平台工厂注册
// ============================================

var registry = make(map[string]PlatformAPI)

// Register 注册一个平台实现
func Register(api PlatformAPI) {
	registry[api.Platform()] = api
}

// GetPlatform 获取指定平台的实现
func GetPlatform(platform string) (PlatformAPI, bool) {
	api, ok := registry[platform]
	return api, ok
}
