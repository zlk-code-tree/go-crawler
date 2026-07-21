package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

type QianchuanAttributionRuleService struct{}

func NewQianchuanAttributionRuleService() *QianchuanAttributionRuleService {
	return &QianchuanAttributionRuleService{}
}

func (s *QianchuanAttributionRuleService) List(ctx context.Context, req *model.QianchuanAttributionRuleListReq) ([]g.Map, int64, error) {
	query := g.DB().Model("qianchuan_advertiser_attribution_rule r").Ctx(ctx).
		LeftJoin("trader source", "source.id = r.source_trader_id").
		LeftJoin("trader target", "target.id = r.target_trader_id")

	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(`(
			CAST(r.advertiser_id AS CHAR) LIKE ?
			OR r.advertiser_name LIKE ?
			OR r.source_account_names LIKE ?
			OR source.name LIKE ?
			OR target.name LIKE ?
			OR r.reason LIKE ?
		)`, like, like, like, like, like, like)
	}
	if req.AdvertiserId > 0 {
		query = query.Where("r.advertiser_id", req.AdvertiserId)
	}
	if req.Status == 0 || req.Status == 1 {
		query = query.Where("r.status", req.Status)
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	rows, err := query.Fields(`
		r.id,
		r.advertiser_id,
		r.advertiser_name,
		r.source_account_names,
		r.source_trader_id,
		COALESCE(source.name, '') AS source_trader_name,
		r.target_trader_id,
		COALESCE(target.name, '') AS target_trader_name,
		r.effective_start_date,
		r.effective_end_date,
		r.reason,
		r.status,
		r.created_at,
		r.updated_at
	`).Page(req.Page, req.PageSize).OrderDesc("r.id").All()
	if err != nil {
		return nil, 0, err
	}

	items := make([]g.Map, 0, len(rows))
	for _, row := range rows {
		endDate := row["effective_end_date"].String()
		items = append(items, g.Map{
			"id":                 row["id"],
			"advertiserId":       row["advertiser_id"],
			"advertiserName":     row["advertiser_name"],
			"sourceAccountNames": row["source_account_names"],
			"sourceTraderId":     row["source_trader_id"],
			"sourceTraderName":   row["source_trader_name"],
			"targetTraderId":     row["target_trader_id"],
			"targetTraderName":   row["target_trader_name"],
			"effectiveStartDate": row["effective_start_date"],
			"effectiveEndDate":   endDate,
			"isPermanent":        endDate == "",
			"reason":             row["reason"],
			"status":             row["status"],
			"createdAt":          row["created_at"],
			"updatedAt":          row["updated_at"],
		})
	}
	return items, int64(total), nil
}

func (s *QianchuanAttributionRuleService) GetById(ctx context.Context, id int64) (g.Map, error) {
	row, err := g.DB().Model("qianchuan_advertiser_attribution_rule").Ctx(ctx).Where("id", id).One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, fmt.Errorf("规则不存在")
	}
	return row.Map(), nil
}

func (s *QianchuanAttributionRuleService) Save(ctx context.Context, req *model.QianchuanAttributionRuleSaveReq) (int64, error) {
	req.AdvertiserName = strings.TrimSpace(req.AdvertiserName)
	req.SourceAccountNames = normalizeQianchuanSourceAccountNames(req.SourceAccountNames)
	req.EffectiveStartDate = strings.TrimSpace(req.EffectiveStartDate)
	req.EffectiveEndDate = strings.TrimSpace(req.EffectiveEndDate)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.AdvertiserId <= 0 {
		return 0, fmt.Errorf("千川户不能为空")
	}
	if req.SourceAccountNames == "" {
		return 0, fmt.Errorf("命中达人不能为空")
	}
	if req.SourceTraderId <= 0 || req.TargetTraderId <= 0 {
		return 0, fmt.Errorf("原归属组和目标归属组不能为空")
	}
	if req.SourceTraderId == req.TargetTraderId {
		return 0, fmt.Errorf("原归属组和目标归属组不能相同")
	}
	if req.EffectiveStartDate == "" {
		return 0, fmt.Errorf("生效开始日期不能为空")
	}
	if req.IsPermanent {
		req.EffectiveEndDate = ""
	}
	if req.EffectiveEndDate != "" && req.EffectiveEndDate < req.EffectiveStartDate {
		return 0, fmt.Errorf("生效结束日期不能早于开始日期")
	}
	if req.AdvertiserName == "" {
		req.AdvertiserName = s.lookupAdvertiserName(ctx, req.AdvertiserId)
	}
	if req.Reason == "" {
		req.Reason = "千川消耗归属调整"
	}
	status := req.Status
	if status != 0 {
		status = 1
	}

	data := g.Map{
		"advertiser_id":        req.AdvertiserId,
		"advertiser_name":      req.AdvertiserName,
		"source_account_names": req.SourceAccountNames,
		"source_trader_id":     req.SourceTraderId,
		"target_trader_id":     req.TargetTraderId,
		"effective_start_date": req.EffectiveStartDate,
		"effective_end_date":   nil,
		"reason":               req.Reason,
		"status":               status,
	}
	if req.EffectiveEndDate != "" {
		data["effective_end_date"] = req.EffectiveEndDate
	}

	if req.Id > 0 {
		result, err := g.DB().Model("qianchuan_advertiser_attribution_rule").Ctx(ctx).
			Where("id", req.Id).
			Data(data).
			Update()
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	}

	result, err := g.DB().Model("qianchuan_advertiser_attribution_rule").Ctx(ctx).Data(data).Insert()
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *QianchuanAttributionRuleService) Delete(ctx context.Context, id int64) (int64, error) {
	result, err := g.DB().Model("qianchuan_advertiser_attribution_rule").Ctx(ctx).Where("id", id).Delete()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *QianchuanAttributionRuleService) Toggle(ctx context.Context, id int64, status int) (int64, error) {
	if status != 0 {
		status = 1
	}
	result, err := g.DB().Model("qianchuan_advertiser_attribution_rule").Ctx(ctx).
		Where("id", id).
		Data(g.Map{"status": status}).
		Update()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *QianchuanAttributionRuleService) GetAdvertisers(ctx context.Context, keyword string) ([]g.Map, error) {
	keyword = strings.TrimSpace(keyword)
	query := g.DB().Model("qianchuan_ad_report").Ctx(ctx).
		Fields("advertiser_id, MAX(NULLIF(advertiser_name, '')) AS advertiser_name").
		Where("advertiser_id > 0").
		Group("advertiser_id")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(CAST(advertiser_id AS CHAR) LIKE ? OR advertiser_name LIKE ?)", like, like)
	}
	rows, err := query.OrderDesc("MAX(stat_datetime)").Limit(50).All()
	if err != nil {
		return nil, err
	}
	items := make([]g.Map, 0, len(rows))
	for _, row := range rows {
		items = append(items, g.Map{
			"advertiserId":   row["advertiser_id"],
			"advertiserName": row["advertiser_name"],
		})
	}
	return items, nil
}

func (s *QianchuanAttributionRuleService) GetTraders(ctx context.Context) ([]g.Map, error) {
	rows, err := g.DB().Model("trader").Ctx(ctx).
		Fields("id, name, company_id, group_id").
		Where("status", 1).
		Order("name ASC, id ASC").
		All()
	if err != nil {
		return nil, err
	}
	items := make([]g.Map, 0, len(rows))
	for _, row := range rows {
		items = append(items, g.Map{
			"id":           row["id"],
			"name":         row["name"],
			"companyId":    row["company_id"],
			"departmentId": row["group_id"],
		})
	}
	return items, nil
}

func (s *QianchuanAttributionRuleService) SearchAccounts(ctx context.Context, keyword string) ([]g.Map, error) {
	return NewShopAccountAttributionService().SearchAccounts(ctx, keyword)
}

func (s *QianchuanAttributionRuleService) lookupAdvertiserName(ctx context.Context, advertiserID int64) string {
	row, err := g.DB().GetOne(ctx, `
		SELECT advertiser_name
		FROM (
			SELECT advertiser_name, stat_datetime FROM qianchuan_ad_report
			WHERE advertiser_id = ? AND advertiser_name <> ''
			UNION ALL
			SELECT qianchuan_advertiser_name AS advertiser_name, updated_at AS stat_datetime
			FROM account_qianchuan_mapping
			WHERE qianchuan_advertiser_id = ? AND qianchuan_advertiser_name <> ''
		) names
		ORDER BY stat_datetime DESC
		LIMIT 1
	`, advertiserID, advertiserID)
	if err != nil || row.IsEmpty() {
		return ""
	}
	return row["advertiser_name"].String()
}

func normalizeQianchuanSourceAccountNames(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\r' || r == '\t'
	})
	seen := make(map[string]struct{}, len(parts))
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, name)
	}
	return strings.Join(names, ",")
}
