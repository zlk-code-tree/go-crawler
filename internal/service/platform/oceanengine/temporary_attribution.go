package oceanengine

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/frame/g"
)

type qianchuanTemporaryAttributionRule struct {
	AdvertiserID       int64
	SourceTraderID     int64
	TargetTraderID     int64
	TargetCompanyID    int64
	TargetDepartmentID int64
	EffectiveStartDate string
	EffectiveEndDate   string
}

type qianchuanTemporaryAttributionRules map[int64][]qianchuanTemporaryAttributionRule

func loadQianchuanTemporaryAttributionRules(ctx context.Context, advertiserIDs []int64) qianchuanTemporaryAttributionRules {
	result := make(qianchuanTemporaryAttributionRules)
	if len(advertiserIDs) == 0 {
		return result
	}

	rows, err := g.DB().Model("qianchuan_advertiser_attribution_rule r").Ctx(ctx).
		InnerJoin("trader target", "target.id = r.target_trader_id AND target.status = 1").
		WhereIn("r.advertiser_id", advertiserIDs).
		Where("r.status", 1).
		Fields(`r.advertiser_id, r.source_trader_id, r.target_trader_id,
			target.company_id AS target_company_id, target.group_id AS target_department_id,
			r.effective_start_date, r.effective_end_date`).
		OrderDesc("r.effective_start_date, r.id").
		All()
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanPoller] 查询千川临时归属规则失败: %v", err)
		return result
	}

	for _, row := range rows {
		advertiserID := row["advertiser_id"].Int64()
		result[advertiserID] = append(result[advertiserID], qianchuanTemporaryAttributionRule{
			AdvertiserID:       advertiserID,
			SourceTraderID:     row["source_trader_id"].Int64(),
			TargetTraderID:     row["target_trader_id"].Int64(),
			TargetCompanyID:    row["target_company_id"].Int64(),
			TargetDepartmentID: row["target_department_id"].Int64(),
			EffectiveStartDate: normalizeStatDate(row["effective_start_date"].String()),
			EffectiveEndDate:   normalizeStatDate(row["effective_end_date"].String()),
		})
	}
	return result
}

func (rules qianchuanTemporaryAttributionRules) resolve(advertiserID int64, statDate string, mapping qianchuanOrgMapping) (qianchuanOrgMapping, bool) {
	statDate = normalizeStatDate(statDate)
	if statDate == "" {
		return mapping, false
	}
	for _, rule := range rules[advertiserID] {
		if statDate < rule.EffectiveStartDate {
			continue
		}
		if rule.EffectiveEndDate != "" && statDate > rule.EffectiveEndDate {
			continue
		}
		mapping.TraderID = rule.TargetTraderID
		mapping.CompanyID = rule.TargetCompanyID
		mapping.DepartmentID = rule.TargetDepartmentID
		mapping.OrganizationID = rule.TargetCompanyID
		return mapping, true
	}
	return mapping, false
}

func (rules qianchuanTemporaryAttributionRules) resolveFallback(advertiserID int64, statDate string, mapping qianchuanOrgMapping, hasConcreteAccount bool) (qianchuanOrgMapping, bool) {
	if hasConcreteAccount {
		return mapping, false
	}
	return rules.resolve(advertiserID, statDate, mapping)
}

// SyncQianchuanTemporaryAttribution reapplies every stored rule to existing daily reports.
// Rules for one advertiser must retain the same source owner and must not overlap.
func SyncQianchuanTemporaryAttribution(ctx context.Context) error {
	if err := validateQianchuanTemporaryAttributionRules(ctx); err != nil {
		return err
	}
	tables := []string{
		"qianchuan_ad_report",
		"qianchuan_author_daily_report",
		"qianchuan_overall_promotion_daily_report",
	}
	for _, table := range tables {
		if err := syncQianchuanTemporaryAttributionTable(ctx, table); err != nil {
			return err
		}
	}
	return nil
}

func validateQianchuanTemporaryAttributionRules(ctx context.Context) error {
	row, err := g.DB().GetOne(ctx, `SELECT COUNT(1) AS conflict_count
		FROM qianchuan_advertiser_attribution_rule left_rule
		JOIN qianchuan_advertiser_attribution_rule right_rule
			ON right_rule.advertiser_id = left_rule.advertiser_id
			AND right_rule.id > left_rule.id
			AND right_rule.status = 1
			AND left_rule.status = 1
			AND right_rule.effective_start_date <= COALESCE(left_rule.effective_end_date, '9999-12-31')
			AND left_rule.effective_start_date <= COALESCE(right_rule.effective_end_date, '9999-12-31')`)
	if err != nil {
		return err
	}
	if row["conflict_count"].Int64() > 0 {
		return fmt.Errorf("千川临时归属规则存在重叠生效区间")
	}

	row, err = g.DB().GetOne(ctx, `SELECT COUNT(1) AS conflict_count FROM (
		SELECT advertiser_id
		FROM qianchuan_advertiser_attribution_rule
		GROUP BY advertiser_id
		HAVING COUNT(DISTINCT source_trader_id) > 1
	) inconsistent_owner`)
	if err != nil {
		return err
	}
	if row["conflict_count"].Int64() > 0 {
		return fmt.Errorf("同一千川账户存在多个原负责人")
	}
	return nil
}

func syncQianchuanTemporaryAttributionTable(ctx context.Context, table string) error {
	allowed := map[string]bool{
		"qianchuan_ad_report":                      true,
		"qianchuan_author_daily_report":            true,
		"qianchuan_overall_promotion_daily_report": true,
	}
	if !allowed[table] {
		return nil
	}

	// Restore everything after the first rule, then rebuild all active intervals.
	restoreSQL := `UPDATE ` + table + ` report
		JOIN (
			SELECT advertiser_id, MIN(effective_start_date) AS restore_start_date,
				MIN(source_trader_id) AS source_trader_id
			FROM qianchuan_advertiser_attribution_rule
			GROUP BY advertiser_id
		) rule_owner
			ON rule_owner.advertiser_id = report.advertiser_id
			AND LEFT(report.stat_datetime, 10) >= rule_owner.restore_start_date
		JOIN trader source ON source.id = rule_owner.source_trader_id
		SET report.trader_id = source.id,
			report.company_id = source.company_id,
			report.department_id = source.group_id`
	if table == "qianchuan_ad_report" {
		restoreSQL += ", report.organization_id = source.company_id"
	}
	if _, err := g.DB().Ctx(ctx).Exec(ctx, restoreSQL); err != nil {
		return err
	}

	applySQL := `UPDATE ` + table + ` report
		JOIN qianchuan_advertiser_attribution_rule rule
			ON rule.advertiser_id = report.advertiser_id
			AND rule.status = 1
			AND LEFT(report.stat_datetime, 10) >= rule.effective_start_date
			AND (rule.effective_end_date IS NULL OR LEFT(report.stat_datetime, 10) <= rule.effective_end_date)
		JOIN trader target ON target.id = rule.target_trader_id AND target.status = 1
		SET report.trader_id = target.id,
			report.company_id = target.company_id,
			report.department_id = target.group_id`
	if table == "qianchuan_ad_report" {
		applySQL += ", report.organization_id = target.company_id"
	}
	_, err := g.DB().Ctx(ctx).Exec(ctx, applySQL)
	return err
}
