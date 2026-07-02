package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

// SendToFeishu 发送飞书卡片消息
func SendToFeishu(ctx context.Context, card *ProfitCardData) error {
	webhookURL := g.Cfg().MustGet(ctx, "webhook.feishu.webhook_url").String()
	if webhookURL == "" {
		return fmt.Errorf("飞书webhook地址未配置")
	}

	msg := buildFeishuCard(card)
	return postJSON(ctx, webhookURL, msg)
}

// buildFeishuCard 构建飞书消息卡片
func buildFeishuCard(card *ProfitCardData) map[string]interface{} {
	elements := []map[string]interface{}{
		{
			"tag": "hr",
		},
	}
	for _, p := range card.Periods {
		elements = append(elements,
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": fmt.Sprintf("**%s**\n流水 %s  成本 %s  广告 %s  佣金 %s  净利 **%s**", p.Label, formatMoney(p.Revenue), formatMoney(p.Cost), formatMoney(p.AdCost), formatMoney(p.Commission), formatMoney(p.NetProfit)),
				},
			},
		)
	}
	elements = append(elements,
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "note",
			"elements": []map[string]interface{}{
				{
					"tag":     "plain_text",
					"content": fmt.Sprintf("数据更新时间: %s | 由系统自动推送", card.UpdatedAt),
				},
			},
		},
	)

	// 计算今日净利润用于标题
	todayProfit := "—"
	if len(card.Periods) > 0 {
		todayProfit = formatMoney(card.Periods[0].NetProfit)
	}

	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": fmt.Sprintf("📊 %s 盈亏日报 - %s  今日净利润: %s", card.NodeName, card.ReportDate, todayProfit),
				},
				"template": "blue",
			},
			"elements": elements,
		},
	}
}

// buildWecomMarkdown 构建企业微信markdown消息
func buildWecomMarkdown(card *ProfitCardData) map[string]interface{} {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 📊 %s 盈亏日报 - %s\n\n", card.NodeName, card.ReportDate))
	sb.WriteString(fmt.Sprintf("> 数据更新时间: %s\n\n", card.UpdatedAt))

	for _, p := range card.Periods {
		sb.WriteString(fmt.Sprintf("**%s**  流水 %s  成本 %s  广告 %s  佣金 %s  净利 **%s**\n",
			p.Label, formatMoney(p.Revenue), formatMoney(p.Cost), formatMoney(p.AdCost), formatMoney(p.Commission), formatMoney(p.NetProfit)))
	}

	sb.WriteString("\n---\n> 由系统自动推送")

	return map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": sb.String(),
		},
	}
}
