package oceanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"go-crawler/internal/service"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// Poller 千川数据拉取器
type Poller struct {
	mu      sync.Mutex
	running bool
}

var (
	pollerOnce     sync.Once
	pollerInstance *Poller
	qianchuanTZ    = time.FixedZone("CST", 8*60*60)
)

func parseQianchuanTokenTime(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", value, qianchuanTZ)
}

// GetPoller 获取千川拉取器单例
func GetPoller() *Poller {
	pollerOnce.Do(func() {
		pollerInstance = &Poller{}
	})
	return pollerInstance
}

// GetLatestToken 获取当前可用的千川 access_token 和 advertiser_ids。
// 当 access_token 过期且 refresh_token 仍有效时，会自动尝试续期。
func GetLatestToken(ctx context.Context) (string, []int64, error) {
	return GetPoller().getLatestToken(ctx)
}

// GetLatestAccessToken 获取当前可用的千川 access_token。
func GetLatestAccessToken(ctx context.Context) (string, error) {
	accessToken, _, err := GetLatestToken(ctx)
	return accessToken, err
}

// Start 启动每日定时拉取
// 定时调度统一由 CronManager 管理（cmd.go 注册 builtin 任务 qianchuan_daily_pull），
// 这里仅通过 CronManager.ResumeTask 恢复被 Stop 暂停的任务，避免重复注册 gcron 导致冲突
func (p *Poller) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return
	}
	if service.GetCronManager().ResumeTask(ctx, "qianchuan_daily_pull") {
		p.running = true
		g.Log().Info(ctx, "[QianchuanPoller] 每日定时拉取已启动（04:37，由CronManager统一管理）")
	} else {
		g.Log().Warning(ctx, "[QianchuanPoller] 未找到 qianchuan_daily_pull 内置任务，请在 cmd.go 中注册")
	}
}

// Stop 停止定时拉取（委托给 CronManager）
func (p *Poller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	service.GetCronManager().Stop("qianchuan_daily_pull")
	g.Log().Info(context.Background(), "[QianchuanPoller] 每日定时拉取已停止")
}

// IsRunning 是否正在运行
func (p *Poller) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// getLatestToken 从数据库获取最新的 access_token 和 advertiser_ids
// 如果 access_token 已过期但 refresh_token 有效，会自动续期
func (p *Poller) getLatestToken(ctx context.Context) (string, []int64, error) {
	row, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil {
		return "", nil, fmt.Errorf("查询token失败: %w", err)
	}
	if row.IsEmpty() {
		return "", nil, fmt.Errorf("未找到有效的access_token，请先完成巨量千川OAuth授权")
	}

	accessToken := row["access_token"].String()
	expiresAt := row["expires_at"].String()
	refreshToken := row["refresh_token"].String()
	refreshTokenExpiresAt := row["refresh_token_expires_at"].String()
	tokenId := row["id"].Int64()

	// 默认从数据库读取广告主列表
	var advertiserIDs []int64
	if idsJSON := row["advertiser_ids"].String(); idsJSON != "" {
		json.Unmarshal([]byte(idsJSON), &advertiserIDs)
	}

	// 检查 access_token 是否过期（提前1小时触发续期，避免边界时刻过期）
	tokenExpired := false
	if expiresAt != "" {
		if t, err := parseQianchuanTokenTime(expiresAt); err == nil && time.Now().Add(time.Hour).After(t) {
			tokenExpired = true
		}
	}

	// 如果 access_token 过期，尝试用 refresh_token 续期
	if tokenExpired && refreshToken != "" {
		// 检查 refresh_token 是否也过期
		refreshExpired := false
		if refreshTokenExpiresAt != "" {
			if rt, err := parseQianchuanTokenTime(refreshTokenExpiresAt); err == nil && time.Now().After(rt) {
				refreshExpired = true
			}
		}

		if refreshExpired {
			g.Log().Infof(ctx, "[QianchuanPoller] access_token和refresh_token均已过期，请重新授权")
			// 发送系统通知，提醒管理员重新授权
			service.CreateNotification(ctx, "千川授权已过期",
				`{"reason":"access_token和refresh_token均已过期","action":"请重新完成巨量千川OAuth授权","authUrl":"/api/qianchuan/auth-url"}`,
				"system_alert", "qianchuan_token_expired", 0, service.NotifyLevelUrgent, 0)
			return "", nil, fmt.Errorf("access_token和refresh_token均已过期，请重新完成巨量千川OAuth授权")
		}

		g.Log().Infof(ctx, "[QianchuanPoller] access_token已过期，尝试用refresh_token续期")
		renewResp, err := RefreshAccessToken(ctx, refreshToken)
		if err != nil {
			g.Log().Errorf(ctx, "[QianchuanPoller] refresh_token续期失败: %v，请重新授权", err)
			service.CreateNotification(ctx, "千川授权续期失败",
				fmt.Sprintf(`{"reason":"refresh_token续期失败","detail":%q,"action":"请重新完成巨量千川OAuth授权","authUrl":"/api/qianchuan/auth-url"}`, err.Error()),
				"system_alert", "qianchuan_token_expired", 0, service.NotifyLevelUrgent, 0)
			return "", nil, fmt.Errorf("token续期失败: %w，请重新完成巨量千川OAuth授权", err)
		}
		if renewResp == nil || renewResp.Data == nil || renewResp.Data.AccessToken == nil || *renewResp.Data.AccessToken == "" {
			return "", nil, fmt.Errorf("token续期失败: 千川接口未返回新的access_token，请重新完成巨量千川OAuth授权")
		}

		// 续期成功，更新数据库
		if renewResp.Data != nil && renewResp.Data.AccessToken != nil {
			newAccessToken := *renewResp.Data.AccessToken
			newExpiresAt := ""
			if renewResp.Data.ExpiresIn != nil && *renewResp.Data.ExpiresIn > 0 {
				newExpiresAt = time.Now().Add(time.Duration(*renewResp.Data.ExpiresIn) * time.Second).Format("2006-01-02 15:04:05")
			}
			newRefreshToken := refreshToken
			newRefreshTokenExpiresAt := refreshTokenExpiresAt
			if renewResp.Data.RefreshToken != nil && *renewResp.Data.RefreshToken != "" {
				newRefreshToken = *renewResp.Data.RefreshToken
			}
			if renewResp.Data.RefreshTokenExpiresIn != nil && *renewResp.Data.RefreshTokenExpiresIn > 0 {
				newRefreshTokenExpiresAt = time.Now().Add(time.Duration(*renewResp.Data.RefreshTokenExpiresIn) * time.Second).Format("2006-01-02 15:04:05")
			}

			// 续期后用新 token 重新获取广告主列表，并解析为千川广告主ID
			platformIDs, advErr := getAdvertiserIDs(ctx, newAccessToken)
			if advErr != nil {
				g.Log().Warningf(ctx, "[QianchuanPoller] 续期后获取广告主列表失败: %v，保留现有广告主ID", advErr)
			}
			// 默认保留数据库现有 advertiser_ids，避免 API 返回空时清空
			qianchuanIDs := advertiserIDs
			if len(platformIDs) > 0 {
				if resolved, resolveErr := ResolveQianchuanAdvertiserIDs(ctx, newAccessToken, platformIDs); resolveErr == nil && len(resolved) > 0 {
					qianchuanIDs = resolved
					g.Log().Infof(ctx, "[QianchuanPoller] 千川广告主ID解析完成: platform=%d -> qianchuan=%d", len(platformIDs), len(resolved))
				} else {
					// 解析失败或返回空时保留数据库中已有的 advertiser_ids，不降级为原始平台ID
					g.Log().Warningf(ctx, "[QianchuanPoller] 千川广告主ID解析失败/返回空(err=%v, resolved=%d)，保留现有%d个广告主ID",
						resolveErr, len(resolved), len(advertiserIDs))
				}
			} else if advErr == nil {
				g.Log().Warningf(ctx, "[QianchuanPoller] 续期后广告主列表为空，保留现有%d个广告主ID", len(advertiserIDs))
			}
			idsJSON, _ := json.Marshal(qianchuanIDs)

			g.DB().Model("qianchuan_token").Ctx(ctx).Where("id", tokenId).Data(g.Map{
				"access_token":             newAccessToken,
				"refresh_token":            newRefreshToken,
				"expires_at":               newExpiresAt,
				"refresh_token_expires_at": newRefreshTokenExpiresAt,
				"advertiser_ids":           string(idsJSON),
			}).Update()

			accessToken = newAccessToken
			expiresAt = newExpiresAt
			advertiserIDs = qianchuanIDs
			g.Log().Infof(ctx, "[QianchuanPoller] token续期成功，新过期时间=%s，广告主数=%d(千川)", newExpiresAt, len(qianchuanIDs))
		}
	} else if tokenExpired {
		g.Log().Infof(ctx, "[QianchuanPoller] token已过期且无refresh_token，请重新授权")
		// 发送系统通知，提醒管理员重新授权
		service.CreateNotification(ctx, "千川授权已过期",
			`{"reason":"access_token已过期且无refresh_token","action":"请重新完成巨量千川OAuth授权","authUrl":"/api/qianchuan/auth-url"}`,
			"system_alert", "qianchuan_token_expired", 0, service.NotifyLevelUrgent, 0)
		return "", nil, fmt.Errorf("access_token已于%s过期且无refresh_token，请重新完成巨量千川OAuth授权", expiresAt[:10])
	}

	// 如果续期成功，advertiserIDs 已在续期逻辑中赋值；否则从数据库读取
	if advertiserIDs == nil {
		if idsJSON := row["advertiser_ids"].String(); idsJSON != "" {
			json.Unmarshal([]byte(idsJSON), &advertiserIDs)
		}
	}

	g.Log().Infof(ctx, "[QianchuanPoller] 使用token id=%d, advertiser_ids=%v, expires_at=%s",
		tokenId, advertiserIDs, expiresAt)

	return accessToken, advertiserIDs, nil
}

// PollerResult 拉取结果
type PollerResult struct {
	TotalSaved        int      `json:"total_saved"`
	TotalPlans        int      `json:"total_plans"`
	TotalOverallSaved int      `json:"total_overall_saved"`
	TotalPlanReports  int      `json:"total_plan_reports"`
	TotalPlanSaved    int      `json:"total_plan_saved"`
	TotalFailed       int      `json:"total_failed"`
	Errors            []string `json:"errors,omitempty"`
}

// ManualPull 手动拉取最近 N 天的数据
func (p *Poller) ManualPull(ctx context.Context, days int) (*PollerResult, error) {
	if days <= 0 {
		days = 1
	}
	if days > 30 {
		days = 30
	}
	now := time.Now()
	return p.pullByDate(ctx,
		now.AddDate(0, 0, -(days-1)).Format("2006-01-02"),
		now.Format("2006-01-02"))
}

// ManualPullByDate 手动拉取指定日期范围的数据
// startDate/endDate 格式: 2006-01-02
func (p *Poller) ManualPullByDate(ctx context.Context, startDate, endDate string) (*PollerResult, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	today := time.Now().Format("2006-01-02")
	todayDate, _ := time.Parse("2006-01-02", today)
	if start.After(todayDate) || end.After(todayDate) {
		return nil, fmt.Errorf("千川数据暂不支持拉取未来日期。当前服务器日期是 %s", today)
	}
	return p.pullByDate(ctx, startDate, endDate)
}

// ManualPullOverallByDate 仅拉取乘方数据。
// 乘方接口更容易触发限流，这里单独走一条轻量链路，只调用 all_promotion/get 并写入乘方表。
func (p *Poller) ManualPullOverallByDate(ctx context.Context, startDate, endDate string) (*PollerResult, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	today := time.Now().Format("2006-01-02")
	todayDate, _ := time.Parse("2006-01-02", today)
	if start.After(todayDate) || end.After(todayDate) {
		return nil, fmt.Errorf("千川数据暂不支持拉取未来日期。当前服务器日期是 %s", today)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("结束日期早于开始日期")
	}
	return p.pullOverallByDate(ctx, startDate, endDate)
}

// dailyPull 已废弃：定时调度统一由 CronManager 管理并记录日志/通知，
// cmd.go 直接调用 ManualPullByDate(ctx, yesterday, yesterday)
// 保留方法签名避免外部引用断裂，内部直接转发
func (p *Poller) dailyPull(ctx context.Context) error {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_, err := p.pullByDate(ctx, yesterday, yesterday)
	return err
}

// advResult 单个广告主拉取结果
type advResult struct {
	plans        int
	overallSaved int
	planReports  int
	planSaved    int
	saved        int
	failed       int
	err          string
}

type authorOrgInfo struct {
	AccountID        int
	CompanyID        int64
	DepartmentID     int64
	TraderID         int64
	AccountName      string
	PlatformNickname string
}

type authorMatchIndex struct {
	uidMap         map[int64][]authorOrgInfo
	showIDMap      map[string][]authorOrgInfo
	accountNameMap map[string][]authorOrgInfo
}

// pullByDate 执行指定日期范围的数据拉取（并发拉取多个广告主）
func (p *Poller) pullByDate(ctx context.Context, startDate, endDate string) (*PollerResult, error) {
	accessToken, advertiserIDs, err := p.getLatestToken(ctx)
	if err != nil {
		return nil, err
	}
	if len(advertiserIDs) == 0 {
		return nil, fmt.Errorf("千川广告主列表为空，请先调用 /api/qianchuan-poller/sync-advertisers 同步广告主")
	}

	g.Log().Infof(ctx, "[QianchuanPoller] pullByDate: 开始拉取 %s~%s, 广告主共%d个", startDate, endDate, len(advertiserIDs))

	// 使用指针传递 token，40102 过期时自动续期后所有 goroutine 共享新 token
	tokenPtr := &accessToken
	result := &PollerResult{}

	// 并发拉取（最多3个并发，避免触发API限流）
	const maxConcurrent = 3
	const perAdvertiserTimeout = 5 * time.Minute
	ch := make(chan advResult, len(advertiserIDs))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	done := false
	for i, advID := range advertiserIDs {
		if done {
			break
		}
		select {
		case <-ctx.Done():
			g.Log().Warningf(ctx, "[QianchuanPoller] 上下文已取消，剩余%d个广告主未派发", len(advertiserIDs)-i)
			done = true
		case sem <- struct{}{}:
			wg.Add(1)
			go func(id int64) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						g.Log().Errorf(ctx, "[QianchuanPoller] 广告主%d拉取panic: %v", id, r)
						ch <- advResult{err: fmt.Sprintf("广告主%d拉取panic: %v", id, r)}
					}
				}()
				advCtx, cancel := context.WithTimeout(ctx, perAdvertiserTimeout)
				defer cancel()
				r := p.pullOneAdvertiser(advCtx, tokenPtr, id, startDate, endDate)
				ch <- r
			}(advID)
		}
	}

	// 等待所有 goroutine 完成后关闭 channel，避免主循环死锁
	go func() {
		wg.Wait()
		close(ch)
	}()

	for r := range ch {
		if r.err != "" {
			result.Errors = append(result.Errors, r.err)
		}
		result.TotalPlans += r.plans
		result.TotalOverallSaved += r.overallSaved
		result.TotalPlanReports += r.planReports
		result.TotalPlanSaved += r.planSaved
		result.TotalSaved += r.saved
		result.TotalFailed += r.failed
	}

	g.Log().Infof(ctx, "[QianchuanPoller] pullByDate完成: 入库%d条(乘方%d条, 计划明细%d条), 失败%d条, 错误%d个",
		result.TotalSaved, result.TotalOverallSaved, result.TotalPlanSaved, result.TotalFailed, len(result.Errors))

	// 自动清理 40002 错误的无效广告主ID，避免后续重复拉取
	p.cleanInvalidAdvertisers(ctx, advertiserIDs, result.Errors)

	return result, nil
}

func (p *Poller) pullOverallByDate(ctx context.Context, startDate, endDate string) (*PollerResult, error) {
	accessToken, advertiserIDs, err := p.getLatestToken(ctx)
	if err != nil {
		return nil, err
	}
	if len(advertiserIDs) == 0 {
		return nil, fmt.Errorf("千川广告主列表为空，请先调用 /api/qianchuan-poller/sync-advertisers 同步广告主")
	}

	g.Log().Infof(ctx, "[QianchuanPoller] pullOverallByDate: 开始拉取乘方 %s~%s, 广告主共%d个", startDate, endDate, len(advertiserIDs))

	client := NewClient(accessToken)
	result := &PollerResult{}

	// 乘方接口较严格，单独降低并发，避免频控。
	const maxConcurrent = 1
	const perAdvertiserTimeout = 2 * time.Minute
	ch := make(chan advResult, len(advertiserIDs))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	done := false
	for i, advID := range advertiserIDs {
		if done {
			break
		}
		select {
		case <-ctx.Done():
			g.Log().Warningf(ctx, "[QianchuanPoller] 上下文已取消，剩余%d个广告主未派发", len(advertiserIDs)-i)
			done = true
		case sem <- struct{}{}:
			wg.Add(1)
			go func(id int64) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						g.Log().Errorf(ctx, "[QianchuanPoller] 广告主%d乘方拉取panic: %v", id, r)
						ch <- advResult{err: fmt.Sprintf("广告主%d乘方拉取panic: %v", id, r)}
					}
				}()
				advCtx, cancel := context.WithTimeout(ctx, perAdvertiserTimeout)
				defer cancel()
				ch <- p.pullOneOverallAdvertiser(advCtx, client, id, startDate, endDate)
			}(advID)
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for r := range ch {
		if r.err != "" {
			result.Errors = append(result.Errors, r.err)
		}
		result.TotalOverallSaved += r.overallSaved
		result.TotalSaved += r.saved
		result.TotalFailed += r.failed
	}

	g.Log().Infof(ctx, "[QianchuanPoller] pullOverallByDate完成: 入库%d条(乘方%d条), 失败%d条, 错误%d个",
		result.TotalSaved, result.TotalOverallSaved, result.TotalFailed, len(result.Errors))
	return result, nil
}

// cleanInvalidAdvertisers 从 qianchuan_token.advertiser_ids 中移除 40002 错误的无效广告主ID
func (p *Poller) cleanInvalidAdvertisers(ctx context.Context, allIDs []int64, errors []string) {
	invalidIDs := make(map[int64]bool)
	for _, e := range errors {
		// 从错误消息中提取广告主ID
		for _, id := range allIDs {
			if strings.Contains(e, fmt.Sprintf("广告主%d拉取失败", id)) && (strings.Contains(e, "40002") || strings.Contains(e, "doesn't exist")) {
				invalidIDs[id] = true
			}
		}
	}
	if len(invalidIDs) == 0 {
		return
	}

	// 从 advertiser_ids 中移除无效ID
	var validIDs []int64
	for _, id := range allIDs {
		if !invalidIDs[id] {
			validIDs = append(validIDs, id)
		}
	}

	idsJSON, _ := json.Marshal(validIDs)
	g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).Data(g.Map{
		"advertiser_ids": string(idsJSON),
	}).Update()

	g.Log().Warningf(ctx, "[QianchuanPoller] 自动清理%d个无效广告主ID，剩余%d个有效ID", len(invalidIDs), len(validIDs))
}

// pullOneAdvertiser 拉取单个广告主的数据并批量入库
func (p *Poller) pullOneAdvertiser(ctx context.Context, tokenPtr *string, advID int64, startDate, endDate string) advResult {
	client := NewClient(*tokenPtr)
	pullResult, err := client.PullAllByDate(ctx, advID, startDate, endDate)
	if err != nil {
		// 如果是 token 过期（40102），强制续期后重试一次
		if strings.Contains(err.Error(), "40102") || strings.Contains(err.Error(), "access_token已过期") {
			g.Log().Warningf(ctx, "[QianchuanPoller] 广告主%d: token过期，尝试强制续期", advID)
			newToken, _, refreshErr := p.getLatestToken(ctx)
			if refreshErr == nil && newToken != "" && newToken != *tokenPtr {
				*tokenPtr = newToken
				g.Log().Infof(ctx, "[QianchuanPoller] token强制续期成功，重试广告主%d", advID)
				client = NewClient(newToken)
				pullResult, err = client.PullAllByDate(ctx, advID, startDate, endDate)
			} else {
				g.Log().Errorf(ctx, "[QianchuanPoller] token强制续期失败: %v", refreshErr)
			}
		}
	}
	if err != nil {
		errMsg := fmt.Sprintf("广告主%d拉取失败: %v", advID, err)
		g.Log().Error(ctx, "[QianchuanPoller] "+errMsg)
		if strings.Contains(err.Error(), "40002") || strings.Contains(err.Error(), "permission") {
			g.Log().Warningf(ctx, "[QianchuanPoller] 广告主%d无权限，跳过", advID)
		}
		return advResult{err: errMsg}
	}

	g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d: 计划%d条, 报表%d条", advID, pullResult.TotalPlans, pullResult.TotalCombined)

	// 计划有数据但合并报表为空 → 提示可能是权限问题
	if pullResult.TotalPlans > 0 && pullResult.TotalCombined == 0 {
		g.Log().Warningf(ctx, "[QianchuanPoller] 广告主%d有%d条计划但0条报表, 可能原因: 1)该日期范围无消耗 2)APP缺少千川报表权限(stat_cost等基础字段), 请在巨量引擎开放平台检查应用权限", advID, pullResult.TotalPlans)
	}

	overallSaved, overallFailed := p.saveOverallPromotionReports(ctx, client, pullResult.OverallReports)
	planSaved, planFailed := p.savePlanReports(ctx, pullResult.PlanReports)
	if len(pullResult.Combined) == 0 {
		return advResult{
			plans:        pullResult.TotalPlans,
			overallSaved: overallSaved,
			planReports:  pullResult.TotalPlanReports,
			planSaved:    planSaved,
			saved:        overallSaved + planSaved,
			failed:       overallFailed + planFailed,
		}
	}

	// 获取该广告主的千川账户名称；无权限时允许为空，避免用店铺映射名误当账户名。
	advName := p.lookupAdvertiserName(ctx, advID)

	// 批量入库：将 N 次 INSERT 合并为每批 50 条的批量 INSERT
	const batchSize = 50
	saved := 0
	failed := 0

	for batchStart := 0; batchStart < len(pullResult.Combined); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(pullResult.Combined) {
			batchEnd = len(pullResult.Combined)
		}
		batch := pullResult.Combined[batchStart:batchEnd]

		placeholders := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*35)
		for _, item := range batch {
			placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args,
				item.AdvertiserID, advName, item.StatDatetime,
				item.StatCost, item.ShowCnt, item.ClickCnt, item.Ctr, item.CpcPlatform, item.CpmPlatform,
				item.ConvertCnt, item.ConvertCost, item.ConvertRate, item.DeepConvertCnt, item.DeepConvertCost,
				item.PayOrderAmount, item.PayOrderCount, item.PayOrderCostPerOrder, item.PayOrderCouponAmount,
				item.CreateOrderAmount, item.CreateOrderCount, item.CreateOrderCouponAmount, item.CreateOrderRoi,
				item.PrepayOrderAmount, item.PrepayOrderCount, item.PrepayAndPayOrderRoi,
				item.AllOrderPayGmv7days, item.AllOrderPayCount7days, item.AllOrderPayRoi7days,
				item.AllOrderRefundAmount7d, item.AllOrderRefundCount7d,
				item.AllOrderRefundAmount14d, item.AllOrderRefundCount14d,
				item.DyFollow, item.TotalPlay,
			)
		}

		sql := fmt.Sprintf(`INSERT INTO qianchuan_advertiser_daily_report
			(advertiser_id, advertiser_name, stat_datetime,
			stat_cost, show_cnt, click_cnt, ctr, cpc_platform, cpm_platform,
			convert_cnt, convert_cost, convert_rate, deep_convert_cnt, deep_convert_cost,
			pay_order_amount, pay_order_count, pay_order_cost_per_order, pay_order_coupon_amount,
			create_order_amount, create_order_count, create_order_coupon_amount, create_order_roi,
			prepay_order_amount, prepay_order_count, prepay_and_pay_order_roi,
			all_order_pay_gmv_7days, all_order_pay_count_7days, all_order_pay_roi_7days,
			all_order_refund_amount_7d, all_order_refund_count_7d,
			all_order_refund_amount_14d, all_order_refund_count_14d,
			dy_follow, total_play)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			advertiser_name=VALUES(advertiser_name),
			stat_cost=VALUES(stat_cost), show_cnt=VALUES(show_cnt), click_cnt=VALUES(click_cnt),
			ctr=VALUES(ctr), cpc_platform=VALUES(cpc_platform), cpm_platform=VALUES(cpm_platform),
			convert_cnt=VALUES(convert_cnt), convert_cost=VALUES(convert_cost), convert_rate=VALUES(convert_rate),
			deep_convert_cnt=VALUES(deep_convert_cnt), deep_convert_cost=VALUES(deep_convert_cost),
			pay_order_amount=VALUES(pay_order_amount), pay_order_count=VALUES(pay_order_count),
			pay_order_cost_per_order=VALUES(pay_order_cost_per_order), pay_order_coupon_amount=VALUES(pay_order_coupon_amount),
			create_order_amount=VALUES(create_order_amount), create_order_count=VALUES(create_order_count),
			create_order_coupon_amount=VALUES(create_order_coupon_amount), create_order_roi=VALUES(create_order_roi),
			prepay_order_amount=VALUES(prepay_order_amount), prepay_order_count=VALUES(prepay_order_count),
			prepay_and_pay_order_roi=VALUES(prepay_and_pay_order_roi),
			all_order_pay_gmv_7days=VALUES(all_order_pay_gmv_7days), all_order_pay_count_7days=VALUES(all_order_pay_count_7days),
			all_order_pay_roi_7days=VALUES(all_order_pay_roi_7days),
			all_order_refund_amount_7d=VALUES(all_order_refund_amount_7d), all_order_refund_count_7d=VALUES(all_order_refund_count_7d),
			all_order_refund_amount_14d=VALUES(all_order_refund_amount_14d), all_order_refund_count_14d=VALUES(all_order_refund_count_14d),
			dy_follow=VALUES(dy_follow), total_play=VALUES(total_play)`,
			strings.Join(placeholders, ","))

		_, execErr := g.DB().Ctx(ctx).Exec(ctx, sql, args...)
		if execErr != nil {
			g.Log().Warningf(ctx, "[QianchuanPoller] 批量入库失败 advID=%d batch=%d-%d: %v", advID, batchStart, batchEnd, execErr)
			failed += batchEnd - batchStart
			continue
		}
		saved += batchEnd - batchStart
	}

	g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d: 入库%d条, 失败%d条", advID, saved, failed)

	// 拉取达人维度报表（按每个达人UID单独查询消耗）
	p.pullAuthorReports(ctx, client, advID, startDate, endDate)

	return advResult{
		plans:        pullResult.TotalPlans,
		overallSaved: overallSaved,
		planReports:  pullResult.TotalPlanReports,
		planSaved:    planSaved,
		saved:        saved + overallSaved + planSaved,
		failed:       failed + overallFailed + planFailed,
	}
}

func (p *Poller) pullOneOverallAdvertiser(ctx context.Context, client *Client, advID int64, startDate, endDate string) advResult {
	// 单独拉乘方时主动放缓一点，避免开发者频控把整批请求打断。
	defer time.Sleep(120 * time.Millisecond)

	reports, err := client.GetOverallPromotionReport(ctx, advID, startDate, endDate)
	if err != nil {
		errMsg := fmt.Sprintf("广告主%d乘方拉取失败: %v", advID, err)
		g.Log().Warning(ctx, "[QianchuanPoller] "+errMsg)
		return advResult{err: errMsg}
	}
	if len(reports) == 0 {
		return advResult{}
	}

	saved, failed := p.saveOverallPromotionReports(ctx, client, reports)
	g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d乘方日报: 拉取%d条, 入库%d条, 失败%d条", advID, len(reports), saved, failed)
	return advResult{
		overallSaved: saved,
		saved:        saved,
		failed:       failed,
	}
}

// pullAuthorReports 拉取达人维度报表。
// 优先按“千川广告主授权抖音号 -> 内部抖音账号(uid/抖音号)”直接匹配，
// 这样同一个广告主下挂多个达人时，也能把消耗正确归属到员工/团队。
func (p *Poller) pullAuthorReports(ctx context.Context, client *Client, advID int64, startDate, endDate string) {
	authorizedAwemes, err := client.GetAuthorizedAwemes(ctx, advID)
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanPoller] 查询广告主%d授权抖音号失败，回退旧映射逻辑: %v", advID, err)
		p.pullAuthorReportsByMapping(ctx, client, advID, startDate, endDate)
		return
	}
	if len(authorizedAwemes) == 0 {
		p.pullAuthorReportsByMapping(ctx, client, advID, startDate, endDate)
		return
	}

	matchIndex, err := p.loadAuthorMatchIndex(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanPoller] 构建达人归属索引失败，回退旧映射逻辑: %v", err)
		p.pullAuthorReportsByMapping(ctx, client, advID, startDate, endDate)
		return
	}

	matchedOrgMap := make(map[int64]authorOrgInfo)
	allReports := make([]AuthorReport, 0, len(authorizedAwemes))
	matchedCount := 0
	for _, aweme := range authorizedAwemes {
		if aweme.AwemeID == 0 {
			continue
		}
		org, matchType, ok := matchIndex.match(aweme)
		if !ok {
			g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d授权抖音号未匹配内部账号: awemeID=%d, showID=%s, name=%s",
				advID, aweme.AwemeID, aweme.AwemeShowID, aweme.AwemeName)
			continue
		}

		reports, queryErr := client.GetAuthorDimensionReports(ctx, advID, aweme.AwemeID, startDate, endDate)
		if queryErr != nil {
			g.Log().Warningf(ctx, "[QianchuanPoller] 广告主%d达人%d报表查询失败: %v", advID, aweme.AwemeID, queryErr)
			continue
		}
		if len(reports) == 0 {
			continue
		}

		for i := range reports {
			if reports[i].AwemeName == "" {
				reports[i].AwemeName = aweme.AwemeName
			}
			if reports[i].AwemeName == "" {
				reports[i].AwemeName = org.PlatformNickname
			}
			if reports[i].AwemeName == "" {
				reports[i].AwemeName = org.AccountName
			}
		}
		matchedOrgMap[aweme.AwemeID] = org
		allReports = append(allReports, reports...)
		matchedCount++

		g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d达人匹配成功: awemeID=%d match=%s accountID=%d traderID=%d",
			advID, aweme.AwemeID, matchType, org.AccountID, org.TraderID)
	}

	if len(allReports) == 0 {
		if matchedCount == 0 {
			p.pullAuthorReportsByMapping(ctx, client, advID, startDate, endDate)
		}
		return
	}

	saved, failed := p.saveAuthorReports(ctx, matchedOrgMap, allReports)
	g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d达人维度报表: 授权抖音号%d个, 成功匹配%d个, 入库%d条, 失败%d条",
		advID, len(authorizedAwemes), matchedCount, saved, failed)
}

func (p *Poller) pullAuthorReportsByMapping(ctx context.Context, client *Client, advID int64, startDate, endDate string) {
	rows, err := g.DB().Model("account_qianchuan_mapping aqm").
		LeftJoin("account a", "aqm.account_id = a.id").
		LeftJoin("trader_account_relation tar", "aqm.account_id = tar.account_id").
		LeftJoin("trader t", "tar.trader_id = t.id").
		Where("aqm.qianchuan_advertiser_id", advID).
		Fields("aqm.account_id, a.uid, a.account_name, a.platform_nickname, aqm.company_id, a.department_id, MIN(tar.trader_id) AS trader_id, MIN(t.group_id) AS trader_group_id").
		Group("aqm.account_id, a.uid, a.account_name, a.platform_nickname, aqm.company_id, a.department_id").
		All()
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanPoller] 查询广告主%d关联账号失败: %v", advID, err)
		return
	}
	if len(rows) == 0 {
		return
	}

	orgMap := make(map[int64]authorOrgInfo)
	var allReports []AuthorReport
	for _, row := range rows {
		uidStr := row["uid"].String()
		if uidStr == "" {
			continue
		}
		awemeID, parseErr := strconv.ParseInt(uidStr, 10, 64)
		if parseErr != nil || awemeID == 0 {
			continue
		}

		reports, queryErr := client.GetAuthorDimensionReports(ctx, advID, awemeID, startDate, endDate)
		if queryErr != nil {
			g.Log().Warningf(ctx, "[QianchuanPoller] 广告主%d达人%d报表查询失败: %v", advID, awemeID, queryErr)
			continue
		}
		if len(reports) == 0 {
			continue
		}

		orgMap[awemeID] = authorOrgInfo{
			AccountID:        row["account_id"].Int(),
			CompanyID:        row["company_id"].Int64(),
			DepartmentID:     fallbackDepartmentID(row["department_id"].Int64(), row["trader_group_id"].Int64()),
			TraderID:         row["trader_id"].Int64(),
			AccountName:      row["account_name"].String(),
			PlatformNickname: row["platform_nickname"].String(),
		}
		for i := range reports {
			if reports[i].AwemeName == "" {
				reports[i].AwemeName = row["platform_nickname"].String()
			}
			if reports[i].AwemeName == "" {
				reports[i].AwemeName = row["account_name"].String()
			}
		}
		allReports = append(allReports, reports...)
	}

	if len(allReports) == 0 {
		return
	}

	saved, failed := p.saveAuthorReports(ctx, orgMap, allReports)
	g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d达人维度报表(旧映射回退): 查询%d个达人, 入库%d条, 失败%d条", advID, len(rows), saved, failed)
}

func (p *Poller) loadAuthorMatchIndex(ctx context.Context) (*authorMatchIndex, error) {
	rows, err := g.DB().Model("account a").
		LeftJoin("trader_account_relation tar", "a.id = tar.account_id").
		LeftJoin("trader t", "tar.trader_id = t.id").
		Fields("a.id AS account_id, a.uid, a.account_name, a.platform_nickname, a.company_id, a.department_id, MIN(tar.trader_id) AS trader_id, MIN(t.group_id) AS trader_group_id").
		Where("a.platform", "douyin").
		Group("a.id, a.uid, a.account_name, a.platform_nickname, a.company_id, a.department_id").
		All()
	if err != nil {
		return nil, err
	}

	index := &authorMatchIndex{
		uidMap:         make(map[int64][]authorOrgInfo),
		showIDMap:      make(map[string][]authorOrgInfo),
		accountNameMap: make(map[string][]authorOrgInfo),
	}
	for _, row := range rows {
		info := authorOrgInfo{
			AccountID:        row["account_id"].Int(),
			CompanyID:        row["company_id"].Int64(),
			DepartmentID:     fallbackDepartmentID(row["department_id"].Int64(), row["trader_group_id"].Int64()),
			TraderID:         row["trader_id"].Int64(),
			AccountName:      row["account_name"].String(),
			PlatformNickname: row["platform_nickname"].String(),
		}
		if awemeID, err := strconv.ParseInt(row["uid"].String(), 10, 64); err == nil && awemeID > 0 {
			index.uidMap[awemeID] = append(index.uidMap[awemeID], info)
		}
		if showID := normalizeAuthorMatchValue(row["platform_nickname"].String()); showID != "" {
			index.showIDMap[showID] = append(index.showIDMap[showID], info)
		}
		if accountName := normalizeAuthorMatchValue(row["account_name"].String()); accountName != "" {
			index.accountNameMap[accountName] = append(index.accountNameMap[accountName], info)
		}
	}
	return index, nil
}

func normalizeAuthorMatchValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func fallbackDepartmentID(accountDepartmentID, traderGroupID int64) int64 {
	if accountDepartmentID > 0 {
		return accountDepartmentID
	}
	return traderGroupID
}

func (m *authorMatchIndex) match(aweme AuthorizedAweme) (authorOrgInfo, string, bool) {
	if m == nil {
		return authorOrgInfo{}, "", false
	}
	if aweme.AwemeID > 0 {
		if candidates := m.uidMap[aweme.AwemeID]; len(candidates) == 1 {
			return candidates[0], "uid", true
		}
	}
	if showID := normalizeAuthorMatchValue(aweme.AwemeShowID); showID != "" {
		if candidates := m.showIDMap[showID]; len(candidates) == 1 {
			return candidates[0], "show_id", true
		}
	}
	if name := normalizeAuthorMatchValue(aweme.AwemeName); name != "" {
		if candidates := m.accountNameMap[name]; len(candidates) == 1 {
			return candidates[0], "account_name", true
		}
	}
	return authorOrgInfo{}, "", false
}

func (m *authorMatchIndex) matchOverallProductAweme(item OverallProductAwemeReportItem) (authorOrgInfo, string, bool) {
	if m == nil {
		return authorOrgInfo{}, "", false
	}
	if anchorID, err := strconv.ParseInt(strings.TrimSpace(item.AnchorID), 10, 64); err == nil && anchorID > 0 {
		if candidates := m.uidMap[anchorID]; len(candidates) == 1 {
			return candidates[0], "uid", true
		}
	}
	if showID := normalizeAuthorMatchValue(item.AnchorShowID); showID != "" {
		if candidates := m.showIDMap[showID]; len(candidates) == 1 {
			return candidates[0], "show_id", true
		}
	}
	if name := normalizeAuthorMatchValue(item.AnchorName); name != "" {
		if candidates := m.accountNameMap[name]; len(candidates) == 1 {
			return candidates[0], "account_name", true
		}
	}
	return authorOrgInfo{}, "", false
}

func overallReportOrgCacheKey(advertiserID int64, statDate string) string {
	return fmt.Sprintf("%d_%s", advertiserID, normalizeStatDate(statDate))
}

func isActiveOverallProductAweme(item OverallProductAwemeReportItem) bool {
	return item.StatCostForOverallRoi2 > 0 ||
		item.StatCostForRoi2 > 0 ||
		item.ShopEstimatedComissionCost > 0 ||
		item.TotalOrderSettleAmountForRoi21h > 0 ||
		item.TotalOrderRealSettleAmountForRoi21h > 0 ||
		item.TotalOrderSettleCountForRoi21h > 0
}

func (p *Poller) resolveOverallReportOrgMappings(
	ctx context.Context,
	client *Client,
	reports []OverallPromotionDailyReport,
	fallback map[int64]qianchuanOrgMapping,
) map[string]qianchuanOrgMapping {
	resolved := make(map[string]qianchuanOrgMapping, len(reports))
	if len(reports) == 0 || client == nil {
		return resolved
	}

	matchIndex, err := p.loadAuthorMatchIndex(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanPoller] 构建乘方抖音号归属索引失败，回退广告主映射: %v", err)
		for _, report := range reports {
			key := overallReportOrgCacheKey(report.AdvertiserID, report.StatDatetime)
			if mapping, ok := fallback[report.AdvertiserID]; ok {
				resolved[key] = companyOnlyOverallMapping(mapping)
			}
		}
		return resolved
	}

	for _, report := range reports {
		key := overallReportOrgCacheKey(report.AdvertiserID, report.StatDatetime)
		if _, exists := resolved[key]; exists {
			continue
		}

		mapping, status := p.resolveOverallReportOrgMapping(ctx, client, report, matchIndex)
		switch status {
		case "matched":
			resolved[key] = mapping
		case "conflict":
			// 同一广告主同一天命中多个组织时，不再回退广告主级绑定，避免误归属。
			resolved[key] = qianchuanOrgMapping{}
		default:
			if fallbackMapping, ok := fallback[report.AdvertiserID]; ok {
				resolved[key] = companyOnlyOverallMapping(fallbackMapping)
			}
		}
	}

	return resolved
}

func (p *Poller) resolveOverallReportOrgMapping(
	ctx context.Context,
	client *Client,
	report OverallPromotionDailyReport,
	matchIndex *authorMatchIndex,
) (qianchuanOrgMapping, string) {
	reportDate := normalizeStatDate(report.StatDatetime)
	page := 1
	pageSize := 200
	totalPages := 1

	uniqueCompanies := make(map[int64]struct{})
	matchedCount := 0

	for page <= totalPages {
		result, err := client.GetOverallProductAwemeReport(ctx, report.AdvertiserID, reportDate, reportDate, page, pageSize, "")
		if err != nil {
			g.Log().Warningf(ctx, "[QianchuanPoller] 查询广告主%d乘方抖音号明细失败(%s,page=%d): %v", report.AdvertiserID, reportDate, page, err)
			return qianchuanOrgMapping{}, "none"
		}
		if result == nil {
			break
		}
		if result.PageSize > 0 && result.Total > result.PageSize {
			totalPages = int((result.Total + result.PageSize - 1) / result.PageSize)
		}

		for _, item := range result.List {
			if !isActiveOverallProductAweme(item) {
				continue
			}
			org, _, ok := matchIndex.matchOverallProductAweme(item)
			if !ok {
				continue
			}
			matchedCount++
			if org.CompanyID > 0 {
				uniqueCompanies[org.CompanyID] = struct{}{}
			}
		}
		page++
	}

	if matchedCount == 0 || len(uniqueCompanies) == 0 {
		return qianchuanOrgMapping{}, "none"
	}
	if len(uniqueCompanies) == 1 {
		var companyID int64
		for id := range uniqueCompanies {
			companyID = id
		}
		return qianchuanOrgMapping{
			AccountID:      0,
			CompanyID:      companyID,
			DepartmentID:   0,
			TraderID:       0,
			OrganizationID: companyID,
		}, "matched"
	}

	g.Log().Warningf(ctx, "[QianchuanPoller] 广告主%d在%s命中多个公司，跳过乘方汇总归属: companies=%d",
		report.AdvertiserID, reportDate, len(uniqueCompanies))
	return qianchuanOrgMapping{}, "conflict"
}

// saveAuthorReports 批量保存达人维度报表
func (p *Poller) saveAuthorReports(ctx context.Context, orgMap map[int64]authorOrgInfo, reports []AuthorReport) (int, int) {
	if len(reports) == 0 {
		return 0, 0
	}

	const batchSize = 50
	saved := 0
	failed := 0
	for batchStart := 0; batchStart < len(reports); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(reports) {
			batchEnd = len(reports)
		}
		batch := reports[batchStart:batchEnd]

		placeholders := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*11)
		for _, item := range batch {
			org := orgMap[item.AwemeID]
			statDate := item.StatDatetime
			if len(statDate) >= 10 {
				statDate = statDate[:10]
			}
			placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args,
				item.AdvertiserID, org.AccountID, org.CompanyID, org.DepartmentID, org.TraderID,
				item.AwemeID, item.AwemeName, item.StatCost, item.PayOrderGmv, item.PayOrderCount,
				statDate,
			)
		}

		sql := fmt.Sprintf(`INSERT INTO qianchuan_author_daily_report
			(advertiser_id, account_id, company_id, department_id, trader_id,
			aweme_id, aweme_name, stat_cost, pay_order_gmv, pay_order_count, stat_datetime)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			account_id=VALUES(account_id), company_id=VALUES(company_id),
			department_id=VALUES(department_id), trader_id=VALUES(trader_id),
			aweme_name=VALUES(aweme_name), stat_cost=VALUES(stat_cost),
			pay_order_gmv=VALUES(pay_order_gmv), pay_order_count=VALUES(pay_order_count)`,
			strings.Join(placeholders, ","))

		_, execErr := g.DB().Ctx(ctx).Exec(ctx, sql, args...)
		if execErr != nil {
			failed += batchEnd - batchStart
			continue
		}
		saved += batchEnd - batchStart
	}
	return saved, failed
}

type qianchuanOrgMapping struct {
	AccountID      int
	CompanyID      int64
	DepartmentID   int64
	TraderID       int64
	OrganizationID int64
}

func companyOnlyOverallMapping(mapping qianchuanOrgMapping) qianchuanOrgMapping {
	if mapping.CompanyID <= 0 {
		return qianchuanOrgMapping{}
	}
	return qianchuanOrgMapping{
		CompanyID:      mapping.CompanyID,
		OrganizationID: mapping.CompanyID,
	}
}

func (p *Poller) loadOrgMappings(ctx context.Context, advertiserIDs []int64) map[int64]qianchuanOrgMapping {
	result := make(map[int64]qianchuanOrgMapping)
	if len(advertiserIDs) == 0 {
		return result
	}

	rows, err := g.DB().Model("account_qianchuan_mapping aqm").
		LeftJoin("account a", "aqm.account_id = a.id").
		LeftJoin("trader_account_relation tar", "aqm.account_id = tar.account_id").
		LeftJoin("trader t", "tar.trader_id = t.id").
		WhereIn("aqm.qianchuan_advertiser_id", advertiserIDs).
		Fields("aqm.qianchuan_advertiser_id, aqm.account_id, aqm.company_id, a.department_id, MIN(tar.trader_id) AS trader_id, MIN(t.group_id) AS trader_group_id").
		Group("aqm.qianchuan_advertiser_id, aqm.account_id, aqm.company_id, a.department_id").
		All()
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanPoller] 查询千川组织映射失败: %v", err)
		return result
	}
	for _, row := range rows {
		companyID := row["company_id"].Int64()
		result[row["qianchuan_advertiser_id"].Int64()] = qianchuanOrgMapping{
			AccountID:      row["account_id"].Int(),
			CompanyID:      companyID,
			DepartmentID:   fallbackDepartmentID(row["department_id"].Int64(), row["trader_group_id"].Int64()),
			TraderID:       row["trader_id"].Int64(),
			OrganizationID: companyID,
		}
	}
	return result
}

func (p *Poller) saveOverallPromotionReports(ctx context.Context, client *Client, reports []OverallPromotionDailyReport) (int, int) {
	if len(reports) == 0 {
		return 0, 0
	}

	advSet := make(map[int64]bool)
	var advIDs []int64
	for _, report := range reports {
		if !advSet[report.AdvertiserID] {
			advSet[report.AdvertiserID] = true
			advIDs = append(advIDs, report.AdvertiserID)
		}
	}
	orgMappings := p.loadOrgMappings(ctx, advIDs)
	detailDerivedMappings := p.resolveOverallReportOrgMappings(ctx, client, reports, orgMappings)

	const batchSize = 100
	saved := 0
	failed := 0
	for batchStart := 0; batchStart < len(reports); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(reports) {
			batchEnd = len(reports)
		}
		batch := reports[batchStart:batchEnd]

		placeholders := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*38)
		for _, item := range batch {
			mapping := orgMappings[item.AdvertiserID]
			if detailMapping, ok := detailDerivedMappings[overallReportOrgCacheKey(item.AdvertiserID, item.StatDatetime)]; ok {
				mapping = detailMapping
			}
			mapping = companyOnlyOverallMapping(mapping)
			payloadBytes, _ := json.Marshal(item)
			placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args,
				mapping.AccountID, mapping.CompanyID, mapping.DepartmentID, mapping.TraderID,
				item.AdvertiserID, normalizeStatDate(item.StatDatetime), item.AdlabScene, item.DataPeriod, item.MarketingGoal, item.OrderPlatform,
				item.ShopEstimatedComissionCost, item.StatCostForOverallRoi2, item.StatCostForRoi2,
				item.TotalCostPerPayOrderForRoi2, item.TotalCostPerPayOrderSettleForOverallRoi21h, item.TotalCostPerPayOrderSettleForRoi21h,
				item.TotalEcomPlatformSubsidyAmountForRoi2, item.TotalOrderRealSettleAmountForRoi21h, item.TotalOrderSettleAmountForRoi21h,
				item.TotalOrderSettleAmountRateForRoi21h, item.TotalOrderSettleCountForRoi21h, item.TotalOrderSettleCountRateForRoi21h,
				item.TotalPayOrderCountForRoi2, item.TotalPayOrderCouponAmountForRoi2, item.TotalPayOrderGmvForRoi2, item.TotalPayOrderGmvIncludeCouponForRoi2,
				item.TotalPrepayAndPayOrderRoi2, item.TotalPrepayAndPaySettleOverallRoi21h, item.TotalPrepayAndPaySettleRoi21h,
				item.TotalPrepayOrderCountForRoi2, item.TotalPrepayOrderGmvForRoi2, item.TotalRefundOrderCountForRoi21h,
				item.TotalRefundOrderGmvForRoi21hAll, item.TotalRefundOrderGmvForRoi21hRate,
				item.TotalUnfinishedEstimateOrderGmvForRoi2, item.NoRefundEcomCouponAmountForRoi2, item.NoRefundEcomPlatformSubsidyAmountForRoi2,
				string(payloadBytes),
			)
		}
		if len(placeholders) == 0 {
			continue
		}

		sql := fmt.Sprintf(`INSERT INTO qianchuan_overall_promotion_daily_report
			(account_id, company_id, department_id, trader_id,
			advertiser_id, stat_datetime, adlab_scene, data_period, marketing_goal, order_platform,
			shop_estimated_comission_cost, stat_cost_for_overall_roi2, stat_cost_for_roi2,
			total_cost_per_pay_order_for_roi2, total_cost_per_pay_order_settle_for_overall_roi2_1h, total_cost_per_pay_order_settle_for_roi2_1h,
			total_ecom_platform_subsidy_amount_for_roi2, total_order_real_settle_amount_for_roi2_1h, total_order_settle_amount_for_roi2_1h,
			total_order_settle_amount_rate_for_roi2_1h, total_order_settle_count_for_roi2_1h, total_order_settle_count_rate_for_roi2_1h,
			total_pay_order_count_for_roi2, total_pay_order_coupon_amount_for_roi2, total_pay_order_gmv_for_roi2, total_pay_order_gmv_include_coupon_for_roi2,
			total_prepay_and_pay_order_roi2, total_prepay_and_pay_settle_overall_roi2_1h, total_prepay_and_pay_settle_roi2_1h,
			total_prepay_order_count_for_roi2, total_prepay_order_gmv_for_roi2, total_refund_order_count_for_roi2_1h,
			total_refund_order_gmv_for_roi2_1h_all, total_refund_order_gmv_for_roi2_1h_rate,
			total_unfinished_estimate_order_gmv_for_roi2, no_refund_ecom_coupon_amount_for_roi2, no_refund_ecom_platform_subsidy_amount_for_roi2,
			raw_payload)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			account_id=VALUES(account_id), company_id=VALUES(company_id), department_id=VALUES(department_id), trader_id=VALUES(trader_id),
			shop_estimated_comission_cost=VALUES(shop_estimated_comission_cost),
			stat_cost_for_overall_roi2=VALUES(stat_cost_for_overall_roi2), stat_cost_for_roi2=VALUES(stat_cost_for_roi2),
			total_cost_per_pay_order_for_roi2=VALUES(total_cost_per_pay_order_for_roi2),
			total_cost_per_pay_order_settle_for_overall_roi2_1h=VALUES(total_cost_per_pay_order_settle_for_overall_roi2_1h),
			total_cost_per_pay_order_settle_for_roi2_1h=VALUES(total_cost_per_pay_order_settle_for_roi2_1h),
			total_ecom_platform_subsidy_amount_for_roi2=VALUES(total_ecom_platform_subsidy_amount_for_roi2),
			total_order_real_settle_amount_for_roi2_1h=VALUES(total_order_real_settle_amount_for_roi2_1h),
			total_order_settle_amount_for_roi2_1h=VALUES(total_order_settle_amount_for_roi2_1h),
			total_order_settle_amount_rate_for_roi2_1h=VALUES(total_order_settle_amount_rate_for_roi2_1h),
			total_order_settle_count_for_roi2_1h=VALUES(total_order_settle_count_for_roi2_1h),
			total_order_settle_count_rate_for_roi2_1h=VALUES(total_order_settle_count_rate_for_roi2_1h),
			total_pay_order_count_for_roi2=VALUES(total_pay_order_count_for_roi2),
			total_pay_order_coupon_amount_for_roi2=VALUES(total_pay_order_coupon_amount_for_roi2),
			total_pay_order_gmv_for_roi2=VALUES(total_pay_order_gmv_for_roi2),
			total_pay_order_gmv_include_coupon_for_roi2=VALUES(total_pay_order_gmv_include_coupon_for_roi2),
			total_prepay_and_pay_order_roi2=VALUES(total_prepay_and_pay_order_roi2),
			total_prepay_and_pay_settle_overall_roi2_1h=VALUES(total_prepay_and_pay_settle_overall_roi2_1h),
			total_prepay_and_pay_settle_roi2_1h=VALUES(total_prepay_and_pay_settle_roi2_1h),
			total_prepay_order_count_for_roi2=VALUES(total_prepay_order_count_for_roi2),
			total_prepay_order_gmv_for_roi2=VALUES(total_prepay_order_gmv_for_roi2),
			total_refund_order_count_for_roi2_1h=VALUES(total_refund_order_count_for_roi2_1h),
			total_refund_order_gmv_for_roi2_1h_all=VALUES(total_refund_order_gmv_for_roi2_1h_all),
			total_refund_order_gmv_for_roi2_1h_rate=VALUES(total_refund_order_gmv_for_roi2_1h_rate),
			total_unfinished_estimate_order_gmv_for_roi2=VALUES(total_unfinished_estimate_order_gmv_for_roi2),
			no_refund_ecom_coupon_amount_for_roi2=VALUES(no_refund_ecom_coupon_amount_for_roi2),
			no_refund_ecom_platform_subsidy_amount_for_roi2=VALUES(no_refund_ecom_platform_subsidy_amount_for_roi2),
			raw_payload=VALUES(raw_payload)`,
			strings.Join(placeholders, ","))

		if _, err := g.DB().Ctx(ctx).Exec(ctx, sql, args...); err != nil {
			g.Log().Warningf(ctx, "[QianchuanPoller] 乘方报表入库失败 batch=%d-%d: %v", batchStart, batchEnd, err)
			failed += len(placeholders)
			continue
		}
		saved += len(placeholders)
	}

	g.Log().Infof(ctx, "[QianchuanPoller] 乘方日报入库%d条, 失败%d条", saved, failed)
	return saved, failed
}

func (p *Poller) savePlanReports(ctx context.Context, reports []AdPlanDailyReport) (int, int) {
	if len(reports) == 0 {
		return 0, 0
	}

	advSet := make(map[int64]bool)
	var advIDs []int64
	for _, report := range reports {
		if !advSet[report.AdvertiserID] {
			advSet[report.AdvertiserID] = true
			advIDs = append(advIDs, report.AdvertiserID)
		}
	}
	orgMappings := p.loadOrgMappings(ctx, advIDs)

	const batchSize = 100
	saved := 0
	failed := 0
	for batchStart := 0; batchStart < len(reports); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(reports) {
			batchEnd = len(reports)
		}
		batch := reports[batchStart:batchEnd]

		placeholders := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*23)
		for _, item := range batch {
			if item.AdID == 0 {
				failed++
				continue
			}
			mapping := orgMappings[item.AdvertiserID]
			awemeID, _ := strconv.ParseInt(item.AwemeID, 10, 64)
			placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args,
				mapping.OrganizationID, mapping.AccountID, mapping.CompanyID, mapping.DepartmentID, mapping.TraderID,
				item.AdvertiserID, "", item.AdID, item.AdName, item.ProductID, item.ProductName,
				awemeID, item.AwemeName, item.AwemeShowID, item.StatCost, item.StatDatetime,
				item.ShowCnt, item.ClickCnt, item.Ctr, item.ConvertCnt, item.ConvertCost,
				item.PayOrderCount, item.CreateOrderCount,
			)
		}
		if len(placeholders) == 0 {
			continue
		}

		sql := fmt.Sprintf(`INSERT INTO qianchuan_ad_report
			(organization_id, account_id, company_id, department_id, trader_id,
			advertiser_id, advertiser_name, ad_id, ad_name, product_id, product_name,
			aweme_id, aweme_name, aweme_show_id, stat_cost, stat_datetime,
			show_cnt, click_cnt, ctr, convert_cnt, convert_cost, pay_order_count, create_order_count)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			organization_id=VALUES(organization_id), account_id=VALUES(account_id),
			company_id=VALUES(company_id), department_id=VALUES(department_id), trader_id=VALUES(trader_id),
			advertiser_name=VALUES(advertiser_name), ad_name=VALUES(ad_name),
			product_id=VALUES(product_id), product_name=VALUES(product_name),
			aweme_id=VALUES(aweme_id), aweme_name=VALUES(aweme_name), aweme_show_id=VALUES(aweme_show_id),
			stat_cost=VALUES(stat_cost), show_cnt=VALUES(show_cnt), click_cnt=VALUES(click_cnt),
			ctr=VALUES(ctr), convert_cnt=VALUES(convert_cnt), convert_cost=VALUES(convert_cost),
			pay_order_count=VALUES(pay_order_count), create_order_count=VALUES(create_order_count)`,
			strings.Join(placeholders, ","))

		if _, err := g.DB().Ctx(ctx).Exec(ctx, sql, args...); err != nil {
			g.Log().Warningf(ctx, "[QianchuanPoller] 计划明细入库失败 batch=%d-%d: %v", batchStart, batchEnd, err)
			failed += len(placeholders)
			continue
		}
		saved += len(placeholders)
	}
	g.Log().Infof(ctx, "[QianchuanPoller] 计划明细入库%d条, 失败%d条", saved, failed)
	return saved, failed
}

// SavePlanReports 保存千川计划维度日结报表，供 OAuth 回调和定时拉取复用同一套入库逻辑。
func SavePlanReports(ctx context.Context, reports []AdPlanDailyReport) (int, int) {
	return GetPoller().savePlanReports(ctx, reports)
}

// advertiserNameCache 广告主名称缓存（advertiser_id → name）
var advertiserNameCache sync.Map

// lookupAdvertiserName 查找千川账户名称（优先实时接口，其次DB回退）
func (p *Poller) lookupAdvertiserName(ctx context.Context, advID int64) string {
	// 1. 内存缓存
	if v, ok := advertiserNameCache.Load(advID); ok {
		return v.(string)
	}

	// 2. 调用 advertiser/info API 获取当前广告主ID的真实名称。
	// 旧表里可能存在按店铺/映射写入的冗余名称，不能优先用于区分多个千川账户。
	name := fetchAdvertiserNameByAPI(ctx, advID)
	if name != "" {
		advertiserNameCache.Store(advID, name)
		return name
	}

	// 3. 数据库回退（qianchuan_advertiser_daily_report）
	row, err := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
		Where("advertiser_id", advID).
		Where("advertiser_name != ''").
		Fields("advertiser_name").
		OrderDesc("stat_datetime").
		Limit(1).
		One()
	if err == nil && !row.IsEmpty() {
		name := row["advertiser_name"].String()
		advertiserNameCache.Store(advID, name)
		return name
	}

	// 4. 兼容历史计划表
	row, err = g.DB().Model("qianchuan_ad_report").Ctx(ctx).
		Where("advertiser_id", advID).
		Where("advertiser_name != ''").
		Fields("advertiser_name").
		Limit(1).
		One()
	if err == nil && !row.IsEmpty() {
		name := row["advertiser_name"].String()
		advertiserNameCache.Store(advID, name)
		return name
	}
	return ""
}

// fetchAdvertiserNameByAPI 通过千川 advertiser/type/get API 获取广告主名称
func fetchAdvertiserNameByAPI(ctx context.Context, advID int64) string {
	row, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil || row.IsEmpty() {
		return ""
	}
	token := row["access_token"].String()

	// 千川专用 API: /open_api/v1.0/qianchuan/advertiser/type/get/
	url := fmt.Sprintf("https://api.oceanengine.com/open_api/v1.0/qianchuan/advertiser/type/get/?advertiser_ids=%s&access_token=%s",
		url.QueryEscape(fmt.Sprintf("[%d]", advID)), token)

	client := g.Client()
	resp, err := client.Get(ctx, url)
	if err != nil {
		g.Log().Warningf(ctx, "[lookupName] advID=%d: API请求失败: %v", advID, err)
		return ""
	}
	defer resp.Close()
	body := resp.ReadAllString()

	var result struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Data    struct {
			List []struct {
				AdvertiserId   int64  `json:"advertiser_id"`
				AdvertiserName string `json:"advertiser_name"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		g.Log().Warningf(ctx, "[lookupName] advID=%d: 解析失败 body=%s", advID, body[:min(len(body), 200)])
		return ""
	}
	if result.Code != 0 {
		g.Log().Warningf(ctx, "[lookupName] advID=%d: API返回错误 code=%d msg=%s", advID, result.Code, result.Message)
		return ""
	}
	for _, item := range result.Data.List {
		if item.AdvertiserId == advID && item.AdvertiserName != "" {
			g.Log().Infof(ctx, "[lookupName] advID=%d → %s", advID, item.AdvertiserName)
			return item.AdvertiserName
		}
	}
	g.Log().Infof(ctx, "[lookupName] advID=%d: API返回空列表 (code=0, msg=%s)", advID, result.Message)
	return ""
}
