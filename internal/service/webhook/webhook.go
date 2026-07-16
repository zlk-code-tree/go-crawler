package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	financeService "go-crawler/internal/service"
	"go-crawler/internal/service/platform/oceanengine"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== 数据结构 ====================

// ProfitCardData 盈亏卡片数据
type ProfitCardData struct {
	ReportDate string         `json:"reportDate"`
	NodeName   string         `json:"nodeName"` // 节点名称，用于卡片标题
	Periods    []PeriodProfit `json:"periods"`
	UpdatedAt  string         `json:"updatedAt"`
}

// PeriodProfit 单个周期的盈亏数据
type PeriodProfit struct {
	Label            string  `json:"label"`            // "今日" / "近7日" / "近15日" / "近30日"
	Days             int     `json:"days"`             // 0(今日) / 7 / 15 / 30
	Revenue          float64 `json:"revenue"`          // 总流水
	Cost             float64 `json:"cost"`             // 货款成本
	AdCost           float64 `json:"adCost"`           // 广告消耗
	UnassignedAdCost float64 `json:"unassignedAdCost"` // 广告主总消耗中未归属到达人/节点的差额
	Commission       float64 `json:"commission"`       // 佣金
	RefundCost       float64 `json:"refundCost"`       // 发货后退款成本
	NetProfit        float64 `json:"netProfit"`        // 净利润
	OrderCount       int64   `json:"orderCount"`       // 有效订单数
}

// WebhookConfig webhook推送配置（对应 webhook_config 表）
type WebhookConfig struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"` // feishu/wecom
	WebhookUrl  string `json:"webhookUrl"`
	Secret      string `json:"secret"`
	NodeType    string `json:"nodeType"` // summary/company/department/trader
	NodeId      int64  `json:"nodeId"`
	NodeName    string `json:"nodeName"`
	PushMorning int    `json:"pushMorning"`
	PushEvening int    `json:"pushEvening"`
	PushPeriods string `json:"pushPeriods"` // "0,7,15,30"
	Enabled     int    `json:"enabled"`
}

// ==================== WebhookPushService ====================

// WebhookPushService webhook推送服务
type WebhookPushService struct{}

// NewWebhookPushService 创建推送服务
func NewWebhookPushService() *WebhookPushService {
	return &WebhookPushService{}
}

// SendDailyReport 发送每日盈亏报告
// 优先从 webhook_config 表读取配置，按节点分别推送；
// 若表无配置，降级为从 config.yaml 读取 webhook.feishu 推全公司汇总
// pushType: morning/evening，用于按配置的 push_morning/push_evening 过滤
func (s *WebhookPushService) SendDailyReport(ctx context.Context) error {
	return s.SendReport(ctx, "")
}

// SendReport 发送盈亏报告，pushType 为空表示两种都推
func (s *WebhookPushService) SendReport(ctx context.Context, pushType string) error {
	g.Log().Info(ctx, "[WebhookPush] 开始生成盈亏报告推送")

	configs, hasAnyConfig, err := s.loadConfigs(ctx)
	if err != nil {
		return fmt.Errorf("加载webhook配置失败: %w", err)
	}

	if err := s.refreshTodayQianchuan(ctx); err != nil {
		g.Log().Warningf(ctx, "[WebhookPush] 推送前补拉当天千川数据失败，继续推送现有数据: %v", err)
	}

	if !hasAnyConfig {
		// 降级：从 config.yaml 读取，推全公司汇总到飞书
		g.Log().Info(ctx, "[WebhookPush] webhook_config表无配置，降级使用config.yaml飞书配置推送汇总")
		return s.fallbackPush(ctx)
	}
	if len(configs) == 0 {
		g.Log().Info(ctx, "[WebhookPush] webhook_config表已有配置，但当前没有启用项，跳过推送")
		return nil
	}

	successCount := 0
	failCount := 0
	for _, cfg := range configs {
		// 按 pushType 过滤
		if pushType == "morning" && cfg.PushMorning != 1 {
			continue
		}
		if pushType == "evening" && cfg.PushEvening != 1 {
			continue
		}
		if cfg.Enabled != 1 {
			continue
		}

		if err := s.pushOneConfig(ctx, &cfg); err != nil {
			g.Log().Errorf(ctx, "[WebhookPush] 推送失败 name=%s node=%s/%d: %v", cfg.Name, cfg.NodeType, cfg.NodeId, err)
			failCount++
			continue
		}
		successCount++
		g.Log().Infof(ctx, "[WebhookPush] 推送成功 name=%s node=%s/%d platform=%s", cfg.Name, cfg.NodeType, cfg.NodeId, cfg.Platform)
	}

	g.Log().Infof(ctx, "[WebhookPush] 推送完成: 成功%d 失败%d", successCount, failCount)
	return nil
}

// refreshTodayQianchuan 推送前补拉当天千川数据，避免“今日”卡片沿用空数据。
func (s *WebhookPushService) refreshTodayQianchuan(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	pullCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	result, err := oceanengine.GetPoller().ManualPullByDate(pullCtx, today, today)
	if err != nil {
		return err
	}
	g.Log().Infof(ctx, "[WebhookPush] 推送前千川当天补拉完成 date=%s totalSaved=%d overall=%d plan=%d errors=%d",
		today, result.TotalSaved, result.TotalOverallSaved, result.TotalPlanSaved, len(result.Errors))
	if len(result.Errors) > 0 {
		return fmt.Errorf("千川当天补拉存在%d个错误: %s", len(result.Errors), strings.Join(result.Errors, "; "))
	}
	return nil
}

// loadConfigs 从 webhook_config 表加载所有启用的配置，并返回表内是否存在配置。
func (s *WebhookPushService) loadConfigs(ctx context.Context) ([]WebhookConfig, bool, error) {
	total, err := g.DB().Model("webhook_config").Ctx(ctx).Count()
	if err != nil {
		return nil, false, err
	}
	rows, err := g.DB().Model("webhook_config").Ctx(ctx).Where("enabled", 1).OrderAsc("id").All()
	if err != nil {
		return nil, total > 0, err
	}
	configs := make([]WebhookConfig, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, WebhookConfig{
			Id:          row["id"].Int64(),
			Name:        row["name"].String(),
			Platform:    row["platform"].String(),
			WebhookUrl:  row["webhook_url"].String(),
			Secret:      row["secret"].String(),
			NodeType:    row["node_type"].String(),
			NodeId:      row["node_id"].Int64(),
			NodeName:    row["node_name"].String(),
			PushMorning: row["push_morning"].Int(),
			PushEvening: row["push_evening"].Int(),
			PushPeriods: row["push_periods"].String(),
			Enabled:     row["enabled"].Int(),
		})
	}
	return configs, total > 0, nil
}

// pushOneConfig 按单个配置推送：构建卡片并发送到对应机器人
func (s *WebhookPushService) pushOneConfig(ctx context.Context, cfg *WebhookConfig) error {
	card, err := s.BuildProfitCardForNode(ctx, cfg)
	if err != nil {
		return fmt.Errorf("构建盈亏数据失败: %w", err)
	}

	switch cfg.Platform {
	case "feishu":
		return s.sendFeishu(ctx, cfg, card)
	case "wecom":
		return s.sendWecom(ctx, cfg, card)
	default:
		return fmt.Errorf("不支持的平台: %s", cfg.Platform)
	}
}

// sendFeishu 发送飞书卡片
func (s *WebhookPushService) sendFeishu(ctx context.Context, cfg *WebhookConfig, card *ProfitCardData) error {
	msg := buildFeishuCard(card)
	msg = signFeishuMessage(msg, cfg.Secret)
	return postJSON(ctx, cfg.WebhookUrl, msg)
}

// sendWecom 发送企业微信markdown消息
func (s *WebhookPushService) sendWecom(ctx context.Context, cfg *WebhookConfig, card *ProfitCardData) error {
	msg := buildWecomMarkdown(card)
	return postJSON(ctx, cfg.WebhookUrl, msg)
}

// SendCard 按配置发送指定卡片（供外部调用，如手动测试推送）
func (s *WebhookPushService) SendCard(ctx context.Context, cfg *WebhookConfig, card *ProfitCardData) error {
	switch cfg.Platform {
	case "feishu":
		return s.sendFeishu(ctx, cfg, card)
	case "wecom":
		return s.sendWecom(ctx, cfg, card)
	default:
		return fmt.Errorf("不支持的平台: %s", cfg.Platform)
	}
}

// fallbackPush 降级推送：webhook_config 表无配置时，用 config.yaml 的飞书配置推汇总
func (s *WebhookPushService) fallbackPush(ctx context.Context) error {
	if !g.Cfg().MustGet(ctx, "webhook.feishu.enabled").Bool() {
		g.Log().Info(ctx, "[WebhookPush] 飞书推送未启用，跳过")
		return nil
	}
	card, err := s.BuildProfitCard(ctx)
	if err != nil {
		return fmt.Errorf("构建盈亏数据失败: %w", err)
	}
	webhookURL := g.Cfg().MustGet(ctx, "webhook.feishu.webhook_url").String()
	if webhookURL == "" {
		return fmt.Errorf("飞书webhook地址未配置")
	}
	return postJSON(ctx, webhookURL, buildFeishuCard(card))
}

// BuildProfitCard 构建全公司汇总盈亏数据（兼容旧调用）
func (s *WebhookPushService) BuildProfitCard(ctx context.Context) (*ProfitCardData, error) {
	return s.BuildProfitCardForNode(ctx, &WebhookConfig{
		NodeType:    "summary",
		NodeName:    "全公司汇总",
		PushPeriods: "0,7,15,30",
	})
}

// BuildProfitCardForNode 构建指定节点的多周期盈亏数据
func (s *WebhookPushService) BuildProfitCardForNode(ctx context.Context, cfg *WebhookConfig) (*ProfitCardData, error) {
	now := time.Now()
	today := now.Format("2006-01-02")

	nodeName := cfg.NodeName
	if nodeName == "" {
		nodeName = "全公司汇总"
	}

	card := &ProfitCardData{
		ReportDate: today,
		NodeName:   nodeName,
		UpdatedAt:  now.Format("2006-01-02 15:04:05"),
	}

	// 解析推送周期，默认 0,7,15,30
	periods := parsePeriods(cfg.PushPeriods)

	for _, days := range periods {
		label := periodLabel(days)
		profit, err := s.queryPeriodProfitForNode(ctx, days, cfg.NodeType, cfg.NodeId)
		if err != nil {
			g.Log().Warningf(ctx, "[WebhookPush] 查询%s(%d天)盈亏失败 node=%s/%d: %v", label, days, cfg.NodeType, cfg.NodeId, err)
			continue
		}
		profit.Label = label
		profit.Days = days
		card.Periods = append(card.Periods, *profit)
	}

	return card, nil
}

// parsePeriods 解析 "0,7,15,30" 为 []int
func parsePeriods(s string) []int {
	if s == "" {
		return []int{0, 7, 15, 30}
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return []int{0, 7, 15, 30}
	}
	return out
}

// periodLabel 周期转中文标签
func periodLabel(days int) string {
	switch days {
	case 0:
		return "今日"
	case 7:
		return "近7日"
	case 15:
		return "近15日"
	case 30:
		return "近30日"
	default:
		return fmt.Sprintf("近%d日", days)
	}
}

// queryPeriodProfitForNode 查询指定节点在指定天数的盈亏汇总
// nodeType: summary=全公司/company/department/trader
func (s *WebhookPushService) queryPeriodProfitForNode(ctx context.Context, days int, nodeType string, nodeId int64) (*PeriodProfit, error) {
	now := time.Now()
	var startDate, endDate string

	if days == 0 {
		startDate = now.Format("2006-01-02")
		endDate = startDate
	} else {
		startDate = now.AddDate(0, 0, -days).Format("2006-01-02")
		endDate = now.AddDate(0, 0, -1).Format("2006-01-02")
	}

	if nodeType != "summary" {
		req := &financeService.RealtimeProfitReq{
			StartDate: startDate,
			EndDate:   endDate,
		}
		switch nodeType {
		case "company":
			req.CompanyId = nodeId
		case "department":
			req.DepartmentId = nodeId
		case "trader":
			req.TraderId = nodeId
		default:
			return &PeriodProfit{}, nil
		}

		result, err := financeService.NewFinanceCommissionService().GetRealtimeProfit(ctx, req)
		if err != nil {
			return nil, err
		}
		return &PeriodProfit{
			Revenue:          result.PayAmount,
			Cost:             result.CostAmount,
			AdCost:           result.AdCost,
			UnassignedAdCost: s.queryUnassignedAdCost(ctx, startDate, endDate),
			Commission:       result.Commission,
			RefundCost:       result.RefundCost,
			NetProfit:        result.NetProfit,
			OrderCount:       result.OrderCount,
		}, nil
	}

	profit := &PeriodProfit{}
	startTime := startDate + " 00:00:00"
	endTime := endDate + " 23:59:59"

	validCond := "((o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51')))"
	refundCond := "o.after_sale_status = '12' AND o.send_time IS NOT NULL"

	platforms := []struct {
		table      string
		orderNoCol string
		isDouyin   bool
	}{
		{"order_info", "main_order_no", true},
		{"kuaishou_orders", "order_no", false},
		{"shipinhao_orders", "order_no", false},
	}

	for _, p := range platforms {
		sql := fmt.Sprintf(`
			SELECT
				COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
				COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
				COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, o.product_price) * o.product_num ELSE 0 END), 0) AS cost_amount,
					COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost
			FROM %s o
			LEFT JOIN goods g ON o.seller_code COLLATE utf8mb4_unicode_ci = g.goods_code COLLATE utf8mb4_unicode_ci
			WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
		`, validCond, p.orderNoCol, validCond, validCond, refundCond, p.table)

		params := []interface{}{startTime, endTime}

		r, err := g.DB().GetOne(ctx, sql, params...)
		if err != nil {
			g.Log().Warningf(ctx, "[WebhookPush] 查询%s失败: %v", p.table, err)
			continue
		}

		count := r["order_count"].Int64()
		if count == 0 {
			continue
		}

		profit.OrderCount += count
		payAmount := r["pay_amount"].Float64()
		profit.Revenue += payAmount
		profit.Cost += r["cost_amount"].Float64()
		profit.RefundCost += r["refund_cost"].Float64()
		if p.isDouyin {
			profit.Commission -= financeService.CalculateDouyinShopServiceFee(payAmount)
		}
	}

	// 留底3%和佣金
	reserveTotal := financeService.CalculateReserveAmount(profit.Revenue)
	shopServiceFee := -profit.Commission
	profit.Commission = profit.Revenue - reserveTotal - profit.Cost - shopServiceFee
	if profit.Commission < 0 {
		profit.Commission = 0
	}

	profit.AdCost = s.querySummaryAdCost(ctx, startDate, endDate)
	profit.UnassignedAdCost = s.queryUnassignedAdCost(ctx, startDate, endDate)

	// 净利润
	commissionDeduction := profit.Revenue * 0.005
	profit.NetProfit = financeService.CalculateOrderProfit(profit.Revenue, profit.Cost, reserveTotal, profit.RefundCost, shopServiceFee, commissionDeduction, 0, profit.AdCost)

	profit.Revenue = round2(profit.Revenue)
	profit.Cost = round2(profit.Cost)
	profit.Commission = round2(profit.Commission)
	profit.RefundCost = round2(profit.RefundCost)
	profit.AdCost = round2(profit.AdCost)
	profit.UnassignedAdCost = round2(profit.UnassignedAdCost)
	profit.NetProfit = round2(profit.NetProfit)

	return profit, nil
}

func (s *WebhookPushService) querySummaryAdCost(ctx context.Context, startDate, endDate string) float64 {
	row, err := g.DB().Model("qianchuan_advertiser_daily_report").
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COALESCE(SUM(stat_cost), 0) AS cost").
		One()
	if err == nil && row["cost"].Float64() > 0 {
		return row["cost"].Float64()
	}

	row, err = g.DB().Model("qianchuan_author_daily_report").
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COALESCE(SUM(stat_cost), 0) AS cost").
		One()
	if err != nil {
		return 0
	}
	return row["cost"].Float64()
}

func (s *WebhookPushService) queryUnassignedAdCost(ctx context.Context, startDate, endDate string) float64 {
	advertiserRow, err := g.DB().Model("qianchuan_advertiser_daily_report").
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COALESCE(SUM(stat_cost), 0) AS cost").
		One()
	if err != nil {
		g.Log().Warningf(ctx, "[WebhookPush] 查询广告主消耗失败: %v", err)
		return 0
	}

	authorRow, err := g.DB().Model("qianchuan_author_daily_report").
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COALESCE(SUM(stat_cost), 0) AS cost").
		One()
	if err != nil {
		g.Log().Warningf(ctx, "[WebhookPush] 查询达人归属消耗失败: %v", err)
		return 0
	}

	diff := advertiserRow["cost"].Float64() - authorRow["cost"].Float64()
	if diff <= 0 {
		return 0
	}
	return round2(diff)
}

// postJSON 发送JSON请求到webhook
func postJSON(ctx context.Context, webhookURL string, body interface{}) error {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	client := g.Client()
	client.SetHeader("Content-Type", "application/json")
	resp, err := client.Post(ctx, webhookURL, jsonBytes)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Close()

	respBody := resp.ReadAllString()
	if resp.StatusCode != 200 {
		return fmt.Errorf("webhook返回错误: status=%d, body=%s", resp.StatusCode, respBody)
	}

	var result struct {
		Code   int    `json:"errcode"`
		Msg    string `json:"errmsg"`
		FSCode int    `json:"code"`
		FSMsg  string `json:"msg"`
	}
	if err := json.Unmarshal([]byte(respBody), &result); err == nil {
		if result.Code != 0 || result.FSCode != 0 {
			msg := result.Msg
			if msg == "" {
				msg = result.FSMsg
			}
			return fmt.Errorf("平台返回错误: %s", msg)
		}
	}

	return nil
}

func signFeishuMessage(msg map[string]interface{}, secret string) map[string]interface{} {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return msg
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	stringToSign := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	msg["timestamp"] = timestamp
	msg["sign"] = sign
	return msg
}

// formatMoney 格式化金额
func formatMoney(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("¥%.2f", v)
	}
	return fmt.Sprintf("-¥%.2f", -v)
}

// round2 保留2位小数
func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
