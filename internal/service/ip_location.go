package service

import (
	"strings"
)

type IpLocationService struct{}

func NewIpLocationService() *IpLocationService {
	return &IpLocationService{}
}

// GetLocationByIP 根据IP获取地理位置
func (s *IpLocationService) GetLocationByIP(ip string) string {
	// 内网IP
	if s.isInternalIP(ip) {
		return "内网IP"
	}

	// TODO: 可以集成第三方IP地址库进行精确定位
	// 推荐方案：
	// 1. ip2region - 离线IP地址定位库（推荐）
	//    github.com/lionsoul2014/ip2region
	// 2. 在线API（需要网络请求）
	//    - 高德地图 IP定位API
	//    - 百度地图 IP定位API
	//    - ipapi.co

	return "未知"
}

// isInternalIP 判断是否为内网IP
func (s *IpLocationService) isInternalIP(ip string) bool {
	if ip == "" || ip == "127.0.0.1" || ip == "localhost" || ip == "::1" {
		return true
	}

	// 私有IP地址段
	// 10.0.0.0 - 10.255.255.255
	// 172.16.0.0 - 172.31.255.255
	// 192.168.0.0 - 192.168.255.255
	if strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "127.") {
		return true
	}

	// 172.16.0.0 - 172.31.255.255
	if strings.HasPrefix(ip, "172.") {
		parts := strings.Split(ip, ".")
		if len(parts) >= 2 {
			second := parts[1]
			// 检查第二段是否在16-31之间
			if second >= "16" && second <= "31" {
				return true
			}
		}
	}

	return false
}
