package oceanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
)

// Poller 千川数据拉取器
type Poller struct {
	mu      sync.Mutex
	running bool
}

var (
	pollerOnce     sync.Once
	pollerInstance *Poller
)

// GetPoller 获取千川拉取器单例
func GetPoller() *Poller {
	pollerOnce.Do(func() {
		pollerInstance = &Poller{}
	})
	return pollerInstance
}

// Start 启动每日定时拉取（每天凌晨4:37）
func (p *Poller) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return
	}
	p.running = true

	gcron.AddSingleton(ctx, "37 4 * * *", func(ctx context.Context) {
		p.dailyPull(ctx)
	}, "qianchuan_daily_pull")

	g.Log().Info(ctx, "[QianchuanPoller] 每日定时拉取已启动（04:37）")
}

// Stop 停止定时拉取
func (p *Poller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	gcron.Remove("qianchuan_daily_pull")
	g.Log().Info(context.Background(), "[QianchuanPoller] 每日定时拉取已停止")
}

// IsRunning 是否正在运行
func (p *Poller) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// getLatestToken 从数据库获取最新的 access_token 和 advertiser_ids
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
	tokenId := row["id"].Int64()

	// 检查是否过期
	if expiresAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", expiresAt); err == nil && time.Now().After(t) {
			g.Log().Infof(ctx, "[QianchuanPoller] 最新token id=%d 已过期(expires_at=%s)，请重新授权", tokenId, expiresAt)
			return "", nil, fmt.Errorf("access_token已于%s过期，请重新完成巨量千川OAuth授权", expiresAt[:10])
		}
	}

	var advertiserIDs []int64
	if idsJSON := row["advertiser_ids"].String(); idsJSON != "" {
		json.Unmarshal([]byte(idsJSON), &advertiserIDs)
	}

	g.Log().Infof(ctx, "[QianchuanPoller] 使用token id=%d, advertiser_ids=%v, expires_at=%s",
		tokenId, advertiserIDs, expiresAt)
	return accessToken, advertiserIDs, nil
}

// PollerResult 拉取结果
type PollerResult struct {
	TotalSaved int      `json:"total_saved"`
	TotalPlans int      `json:"total_plans"`
	Errors     []string `json:"errors,omitempty"`
}

// ManualPull 手动拉取最近 N 天的数据
func (p *Poller) ManualPull(ctx context.Context, days int) (*PollerResult, error) {
	now := time.Now()
	return p.pullByDate(ctx,
		now.AddDate(0, 0, -days).Format("2006-01-02"),
		now.Format("2006-01-02"))
}

// ManualPullByDate 手动拉取指定日期范围的数据
// startDate/endDate 格式: 2006-01-02
func (p *Poller) ManualPullByDate(ctx context.Context, startDate, endDate string) (*PollerResult, error) {
	return p.pullByDate(ctx, startDate, endDate)
}

// dailyPull 每日自动拉取（拉取昨天的数据）
func (p *Poller) dailyPull(ctx context.Context) {
	g.Log().Info(ctx, "[QianchuanPoller] 开始每日自动拉取")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	logId, _ := service.TaskLogStart(ctx, "千川每日自动拉取", "qianchuan_daily_pull", "", fmt.Sprintf(`{"date":"%s"}`, yesterday))

	result, err := p.pullByDate(ctx, yesterday, yesterday)
	durationMs := int(time.Since(time.Now().AddDate(0, 0, -1)).Milliseconds()) // approximate

	if err != nil {
		g.Log().Errorf(ctx, "[QianchuanPoller] 每日自动拉取失败: %v", err)
		if logId > 0 {
			service.TaskLogFailed(ctx, logId, err.Error(), 0)
		}
		return
	}

	respBody, _ := json.Marshal(g.Map{
		"totalSaved": result.TotalSaved,
		"totalPlans": result.TotalPlans,
		"errors":     result.Errors,
	})
	if logId > 0 {
		service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
		level := service.NotifyLevelNormal
		if len(result.Errors) > 0 {
			level = service.NotifyLevelImportant
		}
		service.CreateNotification(ctx, "千川每日数据拉取完成", string(respBody), "task_result", "qianchuan_daily_pull", logId, level, 0)
	}
	g.Log().Infof(ctx, "[QianchuanPoller] 每日自动拉取完成: 入库%d条", result.TotalSaved)
}

// pullByDate 执行指定日期范围的数据拉取
func (p *Poller) pullByDate(ctx context.Context, startDate, endDate string) (*PollerResult, error) {
	accessToken, advertiserIDs, err := p.getLatestToken(ctx)
	if err != nil {
		return nil, err
	}

	client := NewClient(accessToken)
	result := &PollerResult{}

	for _, advID := range advertiserIDs {
		pullResult, err := client.PullAllByDate(ctx, advID, startDate, endDate)
		if err != nil {
			errMsg := fmt.Sprintf("广告主%d拉取失败: %v", advID, err)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Error(ctx, "[QianchuanPoller] "+errMsg)
			// 如果是权限错误，记录但继续处理其他广告主
			if strings.Contains(err.Error(), "40002") || strings.Contains(err.Error(), "permission") {
				g.Log().Warningf(ctx, "[QianchuanPoller] 广告主%d无权限，跳过", advID)
			}
			continue
		}
		result.TotalPlans += pullResult.TotalPlans

		saved := 0
		for _, item := range pullResult.Combined {
			data := g.Map{
				"advertiser_id":   item.AdvertiserID,
				"advertiser_name": item.AdvertiserName,
				"ad_id":           item.AdID,
				"ad_name":         item.AdName,
				"stat_cost":       item.StatCost,
				"stat_datetime":   item.StatDatetime,
				"updated_at":      time.Now(),
			}

			count, _ := g.DB().Model("qianchuan_ad_report").Ctx(ctx).
				Where("ad_id", item.AdID).
				Where("stat_datetime", item.StatDatetime).
				Count()

			if count > 0 {
				g.DB().Model("qianchuan_ad_report").Ctx(ctx).
					Where("ad_id", item.AdID).
					Where("stat_datetime", item.StatDatetime).
					Data(data).Update()
			} else {
				g.DB().Model("qianchuan_ad_report").Ctx(ctx).Data(data).Insert()
			}
			saved++
		}
		result.TotalSaved += saved

		g.Log().Infof(ctx, "[QianchuanPoller] 广告主%d: 计划%d条, 入库报表%d条",
			advID, pullResult.TotalPlans, saved)
	}

	return result, nil
}
