package service

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

const (
	// FinanceSettingQianchuanPlanBlankOrderInferenceEnabled 控制“达人为空/0 的订单是否通过千川计划反推归属”。
	// 默认开启，保持历史计算口径不变。
	FinanceSettingQianchuanPlanBlankOrderInferenceEnabled = "finance.qianchuan_plan_blank_order_inference_enabled"

	financeSettingQianchuanPlanBlankOrderInferenceDescription = "财务佣金：达人为空/0的抖音订单是否通过千川计划反推账号归属"
)

// IsQianchuanPlanBlankOrderInferenceEnabled 返回千川计划反推空白单开关。
// 读取失败时返回 true，避免配置表未初始化时改变线上既有口径。
func IsQianchuanPlanBlankOrderInferenceEnabled(ctx context.Context) bool {
	enabled, err := GetQianchuanPlanBlankOrderInferenceSetting(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "[FinanceSetting] 读取千川空白单反推开关失败，使用默认开启: %v", err)
		return true
	}
	return enabled
}

// GetQianchuanPlanBlankOrderInferenceSetting 获取千川计划反推空白单开关。
func GetQianchuanPlanBlankOrderInferenceSetting(ctx context.Context) (bool, error) {
	row, err := g.DB().Model("app_setting").Ctx(ctx).
		Fields("setting_value").
		Where("setting_key", FinanceSettingQianchuanPlanBlankOrderInferenceEnabled).
		One()
	if err != nil {
		if isMissingAppSettingTableError(err) {
			return true, nil
		}
		return true, err
	}
	if row.IsEmpty() {
		return true, nil
	}

	value := strings.ToLower(strings.TrimSpace(row["setting_value"].String()))
	switch value {
	case "0", "false", "off", "disabled", "disable", "no", "n", "关闭":
		return false, nil
	default:
		return true, nil
	}
}

// SetQianchuanPlanBlankOrderInferenceSetting 设置千川计划反推空白单开关。
func SetQianchuanPlanBlankOrderInferenceSetting(ctx context.Context, enabled bool) error {
	if err := ensureAppSettingTable(ctx); err != nil {
		return err
	}

	value := "0"
	if enabled {
		value = "1"
	}
	_, err := g.DB().Exec(ctx, `
		INSERT INTO app_setting (setting_key, setting_value, description)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			setting_value = VALUES(setting_value),
			description = VALUES(description),
			updated_at = CURRENT_TIMESTAMP
	`, FinanceSettingQianchuanPlanBlankOrderInferenceEnabled, value, financeSettingQianchuanPlanBlankOrderInferenceDescription)
	if err != nil {
		return err
	}
	ClearEnrichedDouyinOrderCache()
	return nil
}

func ensureAppSettingTable(ctx context.Context) error {
	_, err := g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS app_setting (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		setting_key VARCHAR(120) NOT NULL DEFAULT '' COMMENT '配置键',
		setting_value VARCHAR(500) NOT NULL DEFAULT '' COMMENT '配置值',
		description VARCHAR(500) NOT NULL DEFAULT '' COMMENT '配置说明',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY uk_setting_key (setting_key)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用配置表'`)
	return err
}

func isMissingAppSettingTableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "app_setting") &&
		(strings.Contains(message, "doesn't exist") ||
			strings.Contains(message, "no such table") ||
			strings.Contains(message, "1146"))
}
