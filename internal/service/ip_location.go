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

const (
	ipCacheTTL      = 24 * time.Hour
	ipQueryTimeout  = 3 * time.Second
	ipMaxRetries    = 2
	ipRetryInterval = 500 * time.Millisecond
)

// ipApiResponse ip-api.com 返回结构
type ipApiResponse struct {
	Status  string `json:"status"`
	City    string `json:"city"`
	Region  string `json:"regionName"`
	Country string `json:"country"`
	Message string `json:"message"`
}

// ipInfoResponse ipinfo.io 返回结构
type ipInfoResponse struct {
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
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

	// 调用外部 API 查询（主 API 带重试，失败后尝试备选 API）
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

// queryIPFromAPI 查询 IP 地理位置：优先 ip-api.com（带重试），失败则降级 ipinfo.io
func queryIPFromAPI(ip string) string {
	// 主 API：ip-api.com（带重试）
	location := queryIPFromIPApi(ip)
	if location != "未知" {
		return location
	}

	// 备选 API：ipinfo.io
	ipLogger.Infof(context.Background(), "ip-api 查询失败 [%s]，降级到 ipinfo.io", ip)
	location = queryIPFromIPInfo(ip)
	if location != "未知" {
		return location
	}

	return "未知"
}

// queryIPFromIPApi 通过 ip-api.com 查询（带重试机制）
func queryIPFromIPApi(ip string) string {
	var lastErr error
	for attempt := 0; attempt <= ipMaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(ipRetryInterval)
			ipLogger.Debugf(context.Background(), "ip-api 重试第 %d 次 [%s]", attempt, ip)
		}

		location, err := doQueryIPApi(ip)
		if err == nil {
			return location
		}
		lastErr = err
	}

	ipLogger.Warningf(context.Background(), "ip-api 查询最终失败 [%s]（已重试 %d 次）: %v", ip, ipMaxRetries, lastErr)
	return "未知"
}

// doQueryIPApi 单次调用 ip-api.com
func doQueryIPApi(ip string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ipQueryTimeout)
	defer cancel()

	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN&fields=status,message,city,regionName,country", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "未知", fmt.Errorf("请求构建失败: %w", err)
	}

	client := &http.Client{Timeout: ipQueryTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "未知", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result ipApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "未知", fmt.Errorf("解析失败: %w", err)
	}

	if result.Status != "success" {
		return "未知", fmt.Errorf("API返回失败: %s", result.Message)
	}

	return joinLocationParts(result.Country, result.Region, result.City), nil
}

// queryIPFromIPInfo 通过 ipinfo.io 查询（备选 API）
func queryIPFromIPInfo(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), ipQueryTimeout)
	defer cancel()

	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		ipLogger.Warningf(context.Background(), "ipinfo 请求构建失败 [%s]: %v", ip, err)
		return "未知"
	}

	client := &http.Client{Timeout: ipQueryTimeout}
	resp, err := client.Do(req)
	if err != nil {
		ipLogger.Warningf(context.Background(), "ipinfo 查询失败 [%s]: %v", ip, err)
		return "未知"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ipLogger.Warningf(context.Background(), "ipinfo 返回非200 [%s]: %d", ip, resp.StatusCode)
		return "未知"
	}

	var result ipInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		ipLogger.Warningf(context.Background(), "ipinfo 解析失败 [%s]: %v", ip, err)
		return "未知"
	}

	return joinLocationParts(result.Country, result.Region, result.City)
}

// joinLocationParts 拼接地理位置字符串：国家 省/区域 城市
func joinLocationParts(country, region, city string) string {
	parts := make([]string, 0, 3)
	if country != "" {
		parts = append(parts, country)
	}
	if region != "" && region != country {
		parts = append(parts, region)
	}
	if city != "" && city != region {
		parts = append(parts, city)
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
