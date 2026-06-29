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

	// 检查 access_token 是否过期
	tokenExpired := false
	if expiresAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", expiresAt); err == nil && time.Now().After(t) {
			tokenExpired = true
		}
	}

	// 如果 access_token 过期，尝试用 refresh_token 续期
	if tokenExpired && refreshToken != "" {
		// 检查 refresh_token 是否也过期
		refreshExpired := false
		if refreshTokenExpiresAt != "" {
			if rt, err := time.Parse("2006-01-02 15:04:05", refreshTokenExpiresAt); err == nil && time.Now().After(rt) {
				refreshExpired = true
			}
		}

		if refreshExpired {
			g.Log().Infof(ctx, "[QianchuanPoller] access_token和refresh_token均已过期，请重新授权")
			return "", nil, fmt.Errorf("access_token和refresh_token均已过期，请重新完成巨量千川OAuth授权")
		}

		g.Log().Infof(ctx, "[QianchuanPoller] access_token已过期，尝试用refresh_token续期")
		renewResp, err := RefreshAccessToken(ctx, refreshToken)
		if err != nil {
			g.Log().Errorf(ctx, "[QianchuanPoller] refresh_token续期失败: %v，请重新授权", err)
			return "", nil, fmt.Errorf("token续期失败: %w，请重新完成巨量千川OAuth授权", err)
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
			advertiserIDs = qianchuanIDs
			g.Log().Infof(ctx, "[QianchuanPoller] token续期成功，新过期时间=%s，广告主数=%d(千川)", newExpiresAt, len(qianchuanIDs))
		}
	} else if tokenExpired {
		g.Log().Infof(ctx, "[QianchuanPoller] token已过期且无refresh_token，请重新授权")
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
	TotalSaved int      `json:"total_saved"`
	TotalPlans int      `json:"total_plans"`
	TotalFailed int     `json:"total_failed"`
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
	plans int
	saved int
	failed int
	err   string
}

// pullByDate 执行指定日期范围的数据拉取（并发拉取多个广告主）
// 每个 goroutine 内含 panic recovery + 单广告主超时，避免单点故障导致整体卡死
func (p *Poller) pullByDate(ctx context.Context, startDate, endDate string) (*PollerResult, error) {
	accessToken, advertiserIDs, err := p.getLatestToken(ctx)
	if err != nil {
		return nil, err
	}
	if len(advertiserIDs) == 0 {
		return nil, fmt.Errorf("千川广告主列表为空，请先调用 /api/qianchuan-poller/sync-advertisers 同步广告主")
	}

	g.Log().Infof(ctx, "[QianchuanPoller] pullByDate: 开始拉取 %s~%s, 广告主共%d个", startDate, endDate, len(advertiserIDs))

	client := NewClient(accessToken)
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
				// 单广告主超时，避免 API 卡住永久占用并发槽
				advCtx, cancel := context.WithTimeout(ctx, perAdvertiserTimeout)
				defer cancel()
				r := p.pullOneAdvertiser(advCtx, client, id, startDate, endDate)
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
		result.TotalSaved += r.saved
		result.TotalFailed += r.failed
	}

	g.Log().Infof(ctx, "[QianchuanPoller] pullByDate完成: 入库%d条, 失败%d条, 错误%d个", result.TotalSaved, result.TotalFailed, len(result.Errors))

	// 自动清理 40002 错误的无效广告主ID，避免后续重复拉取
	p.cleanInvalidAdvertisers(ctx, advertiserIDs, result.Errors)

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
func (p *Poller) pullOneAdvertiser(ctx context.Context, client *Client, advID int64, startDate, endDate string) advResult {
	pullResult, err := client.PullAllByDate(ctx, advID, startDate, endDate)
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

	if len(pullResult.Combined) == 0 {
		return advResult{plans: pullResult.TotalPlans}
	}

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
		args := make([]interface{}, 0, len(batch)*34)
		for _, item := range batch {
			placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args,
				item.AdvertiserID, item.StatDatetime,
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
			(advertiser_id, stat_datetime,
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
	return advResult{plans: pullResult.TotalPlans, saved: saved, failed: failed}
}
