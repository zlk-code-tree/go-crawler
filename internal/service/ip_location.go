package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/os/glog"
)

var ipLogger = glog.New()

// ipCacheEntry 缓存条目
type ipCacheEntry struct {
	location  string
	expiredAt time.Time
}

var (
	ipCache   = make(map[string]*ipCacheEntry)
	ipCacheMu sync.RWMutex
)

const ipCacheTTL = 24 * time.Hour

// ipApiResponse ip-api.com 返回结构
type ipApiResponse struct {
	Status  string `json:"status"`
	City    string `json:"city"`
	Region  string `json:"regionName"`
	Country string `json:"country"`
	Message string `json:"message"`
}

type IpLocationService struct{}

func NewIpLocationService() *IpLocationService {
	return &IpLocationService{}
}

// GetLocationByIP 根据IP获取地理位置
func (s *IpLocationService) GetLocationByIP(ip string) string {
	if ip == "" {
		return "未知"
	}

	// 清理 IP（去掉端口等）
	ip = cleanIP(ip)

	// 内网IP
	if isInternalIP(ip) {
		return "内网IP"
	}

	// 查缓存
	ipCacheMu.RLock()
	if entry, ok := ipCache[ip]; ok && time.Now().Before(entry.expiredAt) {
		ipCacheMu.RUnlock()
		return entry.location
	}
	ipCacheMu.RUnlock()

	// 调用外部 API 查询
	location := queryIPFromAPI(ip)

	// 写缓存
	ipCacheMu.Lock()
	ipCache[ip] = &ipCacheEntry{
		location:  location,
		expiredAt: time.Now().Add(ipCacheTTL),
	}
	ipCacheMu.Unlock()

	return location
}

// queryIPFromAPI 通过 HTTP API 查询 IP 地理位置
func queryIPFromAPI(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN&fields=status,message,city,regionName,country", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		ipLogger.Warningf(context.Background(), "IP地理位置请求构建失败: %v", err)
		return "未知"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ipLogger.Warningf(context.Background(), "IP地理位置查询失败 [%s]: %v", ip, err)
		return "未知"
	}
	defer resp.Body.Close()

	var result ipApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		ipLogger.Warningf(context.Background(), "IP地理位置解析失败 [%s]: %v", ip, err)
		return "未知"
	}

	if result.Status != "success" {
		ipLogger.Warningf(context.Background(), "IP地理位置查询返回失败 [%s]: %s", ip, result.Message)
		return "未知"
	}

	// 拼接：国家-省/区域-城市
	parts := make([]string, 0, 3)
	if result.Country != "" {
		parts = append(parts, result.Country)
	}
	if result.Region != "" && result.Region != result.Country {
		parts = append(parts, result.Region)
	}
	if result.City != "" && result.City != result.Region {
		parts = append(parts, result.City)
	}

	if len(parts) == 0 {
		return "未知"
	}
	return strings.Join(parts, " ")
}

// cleanIP 清理IP地址（去掉端口号等）
func cleanIP(ip string) string {
	// 处理 IPv6 格式 [::1]:port
	if strings.HasPrefix(ip, "[") {
		if idx := strings.LastIndex(ip, "]"); idx != -1 {
			ip = ip[1:idx]
		}
		return ip
	}
	// 处理 IPv4:port 格式
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

// isInternalIP 判断是否为内网IP
func isInternalIP(ip string) bool {
	if ip == "" || ip == "127.0.0.1" || ip == "localhost" || ip == "::1" {
		return true
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}

	// 回环地址
	if parsed.IsLoopback() {
		return true
	}

	// 私有地址段：10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	if parsed.IsPrivate() {
		return true
	}

	// 链路本地地址 169.254.x.x
	if parsed.IsLinkLocalUnicast() {
		return true
	}

	return false
}

// isInternalIPLegacy 旧版字符串判断（保留兼容）
func isInternalIPLegacy(ip string) bool {
	if ip == "" || ip == "127.0.0.1" || ip == "localhost" || ip == "::1" {
		return true
	}

	if strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "127.") {
		return true
	}

	// 172.16.0.0 - 172.31.255.255
	if strings.HasPrefix(ip, "172.") {
		parts := strings.Split(ip, ".")
		if len(parts) >= 2 {
			second, err := strconv.Atoi(parts[1])
			if err == nil && second >= 16 && second <= 31 {
				return true
			}
		}
	}

	return false
}
