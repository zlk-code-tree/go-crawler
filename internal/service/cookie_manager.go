package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/os/glog"
)

// CookieManager 管理浏览器 Cookie
type CookieManager struct {
	cookieFile string
}

// CookieData Cookie 数据结构
type CookieData struct {
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`
}

// NewCookieManager 创建 Cookie 管理器
func NewCookieManager(cookieDir string) *CookieManager {
	cookieFile := filepath.Join(cookieDir, "cookies.json")
	return &CookieManager{
		cookieFile: cookieFile,
	}
}

// SaveCookies 保存 Cookie 到文件
func (cm *CookieManager) SaveCookies(ctx context.Context, cookies []*proto.NetworkCookie, taskId string) error {
	if len(cookies) == 0 {
		glog.Warning(ctx, fmt.Sprintf("[%s] 没有可保存的 Cookie", taskId))
		return nil
	}

	// 转换为可序列化的格式
	var cookieData []CookieData
	for _, c := range cookies {
		cookieData = append(cookieData, CookieData{
			Domain:   c.Domain,
			Path:     c.Path,
			Name:     c.Name,
			Value:    c.Value,
			Expires:  float64(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			Session:  c.Session,
		})
	}

	// 保存到文件
	data, err := json.MarshalIndent(cookieData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 Cookie 失败：%w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(cm.cookieFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败：%w", err)
	}

	if err := os.WriteFile(cm.cookieFile, data, 0644); err != nil {
		return fmt.Errorf("写入 Cookie 文件失败：%w", err)
	}

	glog.Info(ctx, fmt.Sprintf("[%s] ✅ 保存 %d 个 Cookie 到：%s", taskId, len(cookieData), cm.cookieFile))
	return nil
}

// LoadCookies 从文件加载 Cookie
func (cm *CookieManager) LoadCookies(ctx context.Context, taskId string) ([]*proto.NetworkCookie, bool, error) {
	if _, err := os.Stat(cm.cookieFile); os.IsNotExist(err) {
		glog.Info(ctx, fmt.Sprintf("[%s] ⚠️ Cookie 文件不存在：%s", taskId, cm.cookieFile))
		return nil, false, nil
	}

	// 读取文件
	data, err := os.ReadFile(cm.cookieFile)
	if err != nil {
		return nil, false, fmt.Errorf("读取 Cookie 文件失败：%w", err)
	}

	var cookieData []CookieData
	if err := json.Unmarshal(data, &cookieData); err != nil {
		return nil, false, fmt.Errorf("解析 Cookie 文件失败：%w", err)
	}

	if len(cookieData) == 0 {
		glog.Info(ctx, fmt.Sprintf("[%s] ⚠️ Cookie 文件为空", taskId))
		return nil, false, nil
	}

	// 检查 Cookie 是否过期
	validCookies := cm.filterValidCookies(ctx, cookieData, taskId)
	if len(validCookies) == 0 {
		glog.Info(ctx, fmt.Sprintf("[%s] ⚠️ 所有 Cookie 已过期", taskId))
		// 删除过期文件
		os.Remove(cm.cookieFile)
		return nil, false, nil
	}

	// 转换为 proto.NetworkCookie 格式
	var cookies []*proto.NetworkCookie
	for _, c := range validCookies {
		cookie := &proto.NetworkCookie{
			Domain:   c.Domain,
			Path:     c.Path,
			Name:     c.Name,
			Value:    c.Value,
			Expires:  proto.TimeSinceEpoch(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			Session:  c.Session,
		}
		cookies = append(cookies, cookie)
	}

	glog.Info(ctx, fmt.Sprintf("[%s] ✅ 加载 %d 个有效 Cookie", taskId, len(validCookies)))
	return cookies, true, nil
}

// filterValidCookies 过滤掉过期的 Cookie
func (cm *CookieManager) filterValidCookies(ctx context.Context, cookies []CookieData, taskId string) []CookieData {
	now := time.Now().Unix()
	var valid []CookieData

	for _, c := range cookies {
		// Session Cookie（无过期时间）在浏览器关闭后失效，跳过
		if c.Session && c.Expires == 0 {
			glog.Debug(ctx, fmt.Sprintf("[%s] 🗑️ 跳过 Session Cookie（浏览器关闭后失效）: %s", taskId, c.Name))
			continue
		}

		// 如果没有过期时间但不是 Session Cookie，保留
		if c.Expires == 0 {
			valid = append(valid, c)
			continue
		}

		// 检查是否过期
		if int64(c.Expires) > now {
			valid = append(valid, c)
		} else {
			glog.Debug(ctx, fmt.Sprintf("[%s] 🗑️ 过滤过期 Cookie: %s", taskId, c.Name))
		}
	}

	return valid
}

// ClearCookies 清除保存的 Cookie
func (cm *CookieManager) ClearCookies(ctx context.Context, taskId string) error {
	if _, err := os.Stat(cm.cookieFile); err == nil {
		if err := os.Remove(cm.cookieFile); err != nil {
			return fmt.Errorf("删除 Cookie 文件失败：%w", err)
		}
		glog.Info(ctx, fmt.Sprintf("[%s] 🗑️ 已清除 Cookie 文件", taskId))
	}
	return nil
}

// HasCompassCookies 检查是否有抖音电商罗盘的 Cookie
func (cm *CookieManager) HasCompassCookies(ctx context.Context, taskId string) bool {
	if _, err := os.Stat(cm.cookieFile); os.IsNotExist(err) {
		return false
	}

	data, err := os.ReadFile(cm.cookieFile)
	if err != nil {
		return false
	}

	var cookies []CookieData
	if err := json.Unmarshal(data, &cookies); err != nil {
		return false
	}

	// 检查是否有罗盘相关的关键 Cookie
	keyCookies := []string{"passport_csrf_token", "sessionid", "ttwid"}
	hasKeyCookie := false

	for _, c := range cookies {
		if c.Domain != "" && (contains(c.Domain, "jinritemai") || contains(c.Domain, "bytedance")) {
			hasKeyCookie = true
			break
		}
		for _, key := range keyCookies {
			if contains(c.Name, key) {
				hasKeyCookie = true
				break
			}
		}
	}

	if hasKeyCookie {
		glog.Info(ctx, fmt.Sprintf("[%s] ✅ 检测到抖音电商罗盘 Cookie", taskId))
	} else {
		glog.Info(ctx, fmt.Sprintf("[%s] ⚠️ 未检测到有效的罗盘 Cookie", taskId))
	}

	return hasKeyCookie
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
