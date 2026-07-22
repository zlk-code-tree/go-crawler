package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

type QianchuanAccountOwnerService struct{}

func NewQianchuanAccountOwnerService() *QianchuanAccountOwnerService {
	return &QianchuanAccountOwnerService{}
}

func (s *QianchuanAccountOwnerService) List(ctx context.Context, req *model.QianchuanAccountOwnerListReq) ([]g.Map, int64, error) {
	query := g.DB().Model("qianchuan_account_owner_mapping q").Ctx(ctx).
		LeftJoin("trader t", "t.id = q.trader_id").
		LeftJoin("sys_company c", "c.id = q.company_id").
		LeftJoin("sys_department d", "d.id = q.department_id")

	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(`(
			CAST(q.advertiser_id AS CHAR) LIKE ?
			OR q.advertiser_name LIKE ?
			OR q.owner_name LIKE ?
			OR t.name LIKE ?
			OR q.remark LIKE ?
		)`, like, like, like, like, like)
	}
	if ownerName := strings.TrimSpace(req.OwnerName); ownerName != "" {
		query = query.Where("q.owner_name", ownerName)
	}
	if req.AdvertiserId > 0 {
		query = query.Where("q.advertiser_id", req.AdvertiserId)
	}
	if req.Status == 0 || req.Status == 1 {
		query = query.Where("q.status", req.Status)
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	rows, err := query.Fields(`
		q.id,
		q.advertiser_id,
		q.advertiser_name,
		q.owner_name,
		q.trader_id,
		COALESCE(t.name, '') AS trader_name,
		q.company_id,
		COALESCE(c.name, '') AS company_name,
		q.department_id,
		COALESCE(d.name, '') AS department_name,
		q.remark,
		q.status,
		q.source_file,
		q.source_sheet,
		q.source_row_no,
		q.created_at,
		q.updated_at
	`).Page(req.Page, req.PageSize).OrderAsc("q.owner_name, q.source_row_no, q.id").All()
	if err != nil {
		return nil, 0, err
	}

	items := make([]g.Map, 0, len(rows))
	for _, row := range rows {
		items = append(items, g.Map{
			"id":             row["id"],
			"advertiserId":   row["advertiser_id"],
			"advertiserName": row["advertiser_name"],
			"ownerName":      row["owner_name"],
			"traderId":       row["trader_id"],
			"traderName":     row["trader_name"],
			"companyId":      row["company_id"],
			"companyName":    row["company_name"],
			"departmentId":   row["department_id"],
			"departmentName": row["department_name"],
			"remark":         row["remark"],
			"status":         row["status"],
			"sourceFile":     row["source_file"],
			"sourceSheet":    row["source_sheet"],
			"sourceRowNo":    row["source_row_no"],
			"createdAt":      row["created_at"],
			"updatedAt":      row["updated_at"],
		})
	}
	return items, int64(total), nil
}

func (s *QianchuanAccountOwnerService) Save(ctx context.Context, req *model.QianchuanAccountOwnerSaveReq) (int64, error) {
	req.AdvertiserName = strings.TrimSpace(req.AdvertiserName)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.AdvertiserId <= 0 {
		return 0, fmt.Errorf("千川户号不能为空")
	}
	if req.AdvertiserName == "" {
		return 0, fmt.Errorf("千川户名称不能为空")
	}
	if req.OwnerName == "" {
		return 0, fmt.Errorf("达人/归属不能为空")
	}

	traderID, companyID, departmentID, err := s.resolveOwnerTrader(ctx, req.OwnerName, req.TraderId)
	if err != nil {
		return 0, err
	}
	status := req.Status
	if status != 0 {
		status = 1
	}

	data := g.Map{
		"advertiser_id":   req.AdvertiserId,
		"advertiser_name": req.AdvertiserName,
		"owner_name":      req.OwnerName,
		"trader_id":       traderID,
		"company_id":      companyID,
		"department_id":   departmentID,
		"remark":          req.Remark,
		"status":          status,
	}
	if req.Id > 0 {
		result, err := g.DB().Model("qianchuan_account_owner_mapping").Ctx(ctx).
			Where("id", req.Id).
			Data(data).
			Update()
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	}

	result, err := g.DB().Model("qianchuan_account_owner_mapping").Ctx(ctx).Data(data).Insert()
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *QianchuanAccountOwnerService) Delete(ctx context.Context, id int64) (int64, error) {
	result, err := g.DB().Model("qianchuan_account_owner_mapping").Ctx(ctx).Where("id", id).Delete()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *QianchuanAccountOwnerService) Toggle(ctx context.Context, id int64, status int) (int64, error) {
	if status != 0 {
		status = 1
	}
	result, err := g.DB().Model("qianchuan_account_owner_mapping").Ctx(ctx).
		Where("id", id).
		Data(g.Map{"status": status}).
		Update()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *QianchuanAccountOwnerService) GetOwners(ctx context.Context) ([]g.Map, error) {
	rows, err := g.DB().Model("qianchuan_account_owner_mapping").Ctx(ctx).
		Fields("owner_name, COUNT(1) AS account_count, SUM(CASE WHEN trader_id > 0 THEN 1 ELSE 0 END) AS matched_count").
		Group("owner_name").
		OrderAsc("owner_name").
		All()
	if err != nil {
		return nil, err
	}
	items := make([]g.Map, 0, len(rows))
	for _, row := range rows {
		items = append(items, g.Map{
			"ownerName":    row["owner_name"],
			"accountCount": row["account_count"],
			"matchedCount": row["matched_count"],
		})
	}
	return items, nil
}

func (s *QianchuanAccountOwnerService) GetTraders(ctx context.Context) ([]g.Map, error) {
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

func (s *QianchuanAccountOwnerService) SyncReportAttribution(ctx context.Context) (g.Map, error) {
	adRows, err := s.syncReportTable(ctx, "qianchuan_ad_report", true)
	if err != nil {
		return nil, err
	}
	authorRows, err := s.syncReportTable(ctx, "qianchuan_author_daily_report", false)
	if err != nil {
		return nil, err
	}
	overallRows, err := s.syncReportTable(ctx, "qianchuan_overall_promotion_daily_report", false)
	if err != nil {
		return nil, err
	}
	return g.Map{
		"qianchuanAdReportRows":               adRows,
		"qianchuanAuthorDailyReportRows":      authorRows,
		"qianchuanOverallPromotionReportRows": overallRows,
		"message":                             "同步完成，已按启用且匹配到达人组的千川账户刷新报表归属",
	}, nil
}

func (s *QianchuanAccountOwnerService) syncReportTable(ctx context.Context, table string, hasOrganizationID bool) (int64, error) {
	allowed := map[string]bool{
		"qianchuan_ad_report":                      true,
		"qianchuan_author_daily_report":            true,
		"qianchuan_overall_promotion_daily_report": true,
	}
	if !allowed[table] {
		return 0, nil
	}
	setOrganization := ""
	if hasOrganizationID {
		setOrganization = ", report.organization_id = owner.company_id"
	}
	result, err := g.DB().Ctx(ctx).Exec(ctx, `UPDATE `+table+` report
		JOIN qianchuan_account_owner_mapping owner
			ON owner.advertiser_id = report.advertiser_id
			AND owner.status = 1
			AND owner.trader_id > 0
		SET report.trader_id = owner.trader_id,
			report.company_id = owner.company_id,
			report.department_id = owner.department_id`+setOrganization)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *QianchuanAccountOwnerService) resolveOwnerTrader(ctx context.Context, ownerName string, traderID int64) (int64, int64, int64, error) {
	if traderID > 0 {
		row, err := g.DB().Model("trader").Ctx(ctx).
			Where("id", traderID).
			Where("status", 1).
			Fields("id, company_id, group_id").
			One()
		if err != nil {
			return 0, 0, 0, err
		}
		if row.IsEmpty() {
			return 0, 0, 0, fmt.Errorf("选择的达人/组不存在或已停用")
		}
		return row["id"].Int64(), row["company_id"].Int64(), row["group_id"].Int64(), nil
	}

	name := strings.TrimSpace(ownerName)
	if name == "" || name == "备用" {
		return 0, 0, 0, nil
	}
	candidates := []string{name}
	if strings.HasSuffix(name, "组") {
		candidates = append(candidates, strings.TrimSuffix(name, "组"))
	} else {
		candidates = append(candidates, name+"组")
	}
	row, err := g.DB().Model("trader").Ctx(ctx).
		WhereIn("name", candidates).
		Where("status", 1).
		Fields("id, company_id, group_id, name").
		OrderAsc("id").
		One()
	if err != nil {
		return 0, 0, 0, err
	}
	if row.IsEmpty() {
		return 0, 0, 0, nil
	}
	return row["id"].Int64(), row["company_id"].Int64(), row["group_id"].Int64(), nil
}
