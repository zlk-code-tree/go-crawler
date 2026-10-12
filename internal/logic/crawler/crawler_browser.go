package crawler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/service"
)

// startBrowser 启动浏览器
// startBrowser 启动浏览器 - headless: true表示无头模式，false表示有头模式（显示窗口）
func (s *sCrawler) startBrowser(ctx context.Context, taskId string, headless bool) (*rod.Browser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	l := launcher.New().
		Context(ctx).
		Headless(headless).
		Append("--start-maximized").
		Append("--disable-dev-shm-usage")

	url, err := l.Launch()
	if err != nil {
		if l.PID() != 0 {
			go l.Cleanup()
		}
		return nil, fmt.Errorf("启动浏览器失败：%w", err)
	}

	lifecycle := &browserLifecycle{launcher: l}
	browserCtx := context.WithValue(ctx, browserLifecycleKey{}, lifecycle)
	browser := rod.New().Context(browserCtx).ControlURL(url).SlowMotion(100 * time.Millisecond)
	if err := browser.Connect(); err != nil {
		l.Kill()
		go l.Cleanup()
		return nil, fmt.Errorf("连接浏览器失败：%w", err)
	}

	logInfo(ctx, taskId, fmt.Sprintf("🌐 浏览器启动成功：%s", url))
	return browser, nil
}

// checkIsLoggedIn 检测是否已登录
func (s *sCrawler) checkIsLoggedIn(ctx context.Context, page *rod.Page, taskId string) bool {
	selectors := []string{
		"//*[contains(text(), '经营概况')]",
		"//*[contains(text(), '抖音电商·罗盘')]",
		".el-container",
		".main-content",
	}

	for _, selector := range selectors {
		var err error
		if strings.HasPrefix(selector, "//") {
			_, err = page.Timeout(2 * time.Second).ElementX(selector)
		} else {
			_, err = page.Timeout(2 * time.Second).Element(selector)
		}
		if err == nil {
			logInfo(ctx, taskId, "✅ 检测到已登录状态")
			return true
		}
	}

	logInfo(ctx, taskId, "⚠️ 未检测到登录状态，需要登录")
	return false
}

// performLogin 执行登录
func (s *sCrawler) performLogin(ctx context.Context, page *rod.Page, taskId string, loginType, username, password string) bool {
	loginCtx, cancel := context.WithTimeout(ctx, 8*time.Minute)
	defer cancel()
	if loginCtx.Err() != nil {
		return false
	}
	page = page.Context(loginCtx)
	deadline, _ := loginCtx.Deadline()
	logInfo(ctx, taskId, fmt.Sprintf("⏳ 等待登录完成（超时时间：%d 分钟）", 8))

	if loginType == "password" && username != "" && password != "" {
		if err := s.fillPasswordLoginForm(page, username, password); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 自动填充账号密码失败，请在浏览器中手动输入：%v", err))
		} else {
			logInfo(ctx, taskId, "🔐 已在本地浏览器填充账号密码，如出现验证码请手动完成")
		}
	}

	// 等待用户手动完成登录（支持扫码或账号密码）
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-loginCtx.Done():
			if ctx.Err() == nil {
				logError(ctx, taskId, "⏰ 登录超时")
			}
			return false
		case <-ticker.C:
		}

		// 检测是否登录成功
		if s.checkIsLoggedIn(ctx, page, taskId) {
			return true
		}

		remaining := int(time.Until(deadline).Seconds())
		if remaining%30 == 0 {
			logInfo(ctx, taskId, fmt.Sprintf("⌛ 等待登录中...剩余时间：%d秒", remaining))
		}
	}
}

func (s *sCrawler) fillPasswordLoginForm(page *rod.Page, username, password string) error {
	// 先尝试切换到账号/密码登录面板。找不到时继续在当前面板定位输入框。
	for _, xpath := range []string{
		"//*[contains(text(), '密码登录')]",
		"//*[contains(text(), '账号登录')]",
	} {
		if elem, err := page.Timeout(1500 * time.Millisecond).ElementX(xpath); err == nil {
			_ = elem.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(500 * time.Millisecond)
			break
		}
	}

	findInput := func(selectors []string) (*rod.Element, error) {
		var lastErr error
		for _, selector := range selectors {
			elem, err := page.Timeout(1500 * time.Millisecond).Element(selector)
			if err == nil {
				return elem, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}

	usernameInput, err := findInput([]string{
		"input[placeholder*='手机号']", "input[placeholder*='账号']",
		"input[type='tel']", "input[type='text']",
	})
	if err != nil || usernameInput == nil {
		return gerror.New("未找到账号输入框")
	}
	passwordInput, err := findInput([]string{"input[type='password']", "input[placeholder*='密码']"})
	if err != nil || passwordInput == nil {
		return gerror.New("未找到密码输入框")
	}

	_ = usernameInput.SelectAllText()
	if err := usernameInput.Input(username); err != nil {
		return gerror.Wrap(err, "输入账号失败")
	}
	_ = passwordInput.SelectAllText()
	if err := passwordInput.Input(password); err != nil {
		return gerror.Wrap(err, "输入密码失败")
	}

	if button, err := page.Timeout(2 * time.Second).ElementX("//button[contains(., '登录')]"); err == nil {
		if err := button.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return gerror.Wrap(err, "点击登录按钮失败")
		}
	}
	return nil
}

// injectCookies 将保存的 Cookie 注入浏览器页面
func (s *sCrawler) injectCookies(ctx context.Context, page *rod.Page, cookieManager *service.CookieManager, taskId string) bool {
	cookies, hasCookies, err := cookieManager.LoadCookies(ctx, taskId)
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 加载 Cookie 失败：%v", err))
		return false
	}
	if !hasCookies || len(cookies) == 0 {
		logInfo(ctx, taskId, "ℹ️ 没有可用的 Cookie")
		return false
	}

	// 转换为 CookieParam 格式用于注入
	var cookieParams []*proto.NetworkCookieParam
	for _, c := range cookies {
		param := &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
		}
		// 只设置有过期时间的 Cookie
		if c.Expires > 0 {
			param.Expires = c.Expires
		}
		cookieParams = append(cookieParams, param)
	}

	// 注入 Cookie 到浏览器
	err = proto.NetworkSetCookies{Cookies: cookieParams}.Call(page)
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注入 Cookie 失败：%v", err))
		return false
	}

	logInfo(ctx, taskId, fmt.Sprintf("🍪 成功注入 %d 个 Cookie", len(cookieParams)))

	// 刷新页面让 Cookie 生效
	if err := page.Reload(); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 刷新页面失败：%v", err))
		return false
	}
	page.WaitLoad()
	time.Sleep(3 * time.Second)

	return true
}

// setupQuotaBypass 通过 CDP Fetch 域劫持请求，绕过抖音罗盘每日查看次数限制。
// 使用 catch-all 模式匹配所有请求，在 handler 内通过 URL 判断是否需要劫持。
// 返回的 HijackRouter 需要调用者通过 defer router.Stop() 管理生命周期。
func (s *sCrawler) setupQuotaBypass(ctx context.Context, page *rod.Page, taskId string) *rod.HijackRouter {
	if !g.Cfg().MustGet(ctx, "crawler.video.quotaBypass", false).Bool() {
		return nil
	}

	router := page.HijackRequests()

	// 用最宽的模式匹配所有请求，handler 内部再做精确判断
	if err := router.Add("*", "", func(h *rod.Hijack) {
		url := h.Request.URL().String()

		// 非罗盘域名直接放行
		if !strings.Contains(url, "jinritemai.com") {
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		if strings.Contains(url, "/get_view_quota") {
			fmt.Printf("[QUOTA-BYPASS] 劫持 get_view_quota -> remaining_cnt=999\n")
			_ = h.Response.SetHeader("Content-Type", "application/json")
			_ = h.Response.SetBody(`{"data":{"remaining_cnt":999,"target_id_viewable":true},"msg":"","st":0}`)
			return
		}

		if strings.Contains(url, "/get_display_rule") {
			fmt.Printf("[QUOTA-BYPASS] 劫持 get_display_rule -> display_type=1\n")
			_ = h.Response.SetHeader("Content-Type", "application/json")
			_ = h.Response.SetBody(`{"data":{"display_type":1},"msg":"","st":0}`)
			return
		}

		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("注册配额绕过劫持失败：%v", err))
	}

	go router.Run()
	logInfo(ctx, taskId, "已启用配额绕过（Fetch劫持：get_view_quota->999次, get_display_rule->允许）")
	return router
}
