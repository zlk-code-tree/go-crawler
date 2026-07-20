package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

type ShopAccountAttributionService struct{}

func NewShopAccountAttributionService() *ShopAccountAttributionService {
	return &ShopAccountAttributionService{}
}

// List 分页查询
func (s *ShopAccountAttributionService) List(ctx context.Context, req *model.ShopAccountAttributionListReq) ([]g.Map, int64, error) {
	query := g.DB().Model("shop_account_attribution")

	if req.Keyword != "" {
		like := "%" + req.Keyword + "%"
		query = query.Where("(shop_name LIKE ? OR CAST(shop_id AS CHAR) LIKE ? OR source_account_name LIKE ? OR source_account_uid LIKE ? OR target_value LIKE ?)",
			like, like, like, like, like)
	}
	req.ShopId = strings.TrimSpace(req.ShopId)
	if req.ShopId != "" {
		query = query.Where("shop_id", req.ShopId)
	}
	if req.ShopName != "" {
		query = query.Where("shop_name", req.ShopName)
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	rows, err := query.Page(req.Page, req.PageSize).OrderDesc("id").All()
	if err != nil {
		return nil, 0, err
	}

	items := make([]g.Map, len(rows))
	for i, row := range rows {
		items[i] = g.Map{
			"id":                row["id"],
			"shopId":            row["shop_id"],
			"shopName":          row["shop_name"],
			"sourceAccountUid":  row["source_account_uid"],
			"sourceAccountName": row["source_account_name"],
			"targetType":        row["target_type"],
			"targetValue":       row["target_value"],
			"remark":            row["remark"],
			"status":            row["status"],
			"createdAt":         row["created_at"],
			"updatedAt":         row["updated_at"],
		}
	}

	return items, int64(total), nil
}

// GetById 根据ID查询
func (s *ShopAccountAttributionService) GetById(ctx context.Context, id int) (g.Map, error) {
	row, err := g.DB().Model("shop_account_attribution").Where("id", id).One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, fmt.Errorf("记录不存在")
	}
	return row.Map(), nil
}

// Save 新增或更新
func (s *ShopAccountAttributionService) Save(ctx context.Context, req *model.ShopAccountAttributionSaveReq) (int64, error) {
	req.ShopId = strings.TrimSpace(req.ShopId)
	shopName := strings.TrimSpace(req.ShopName)
	if req.ShopId != "" {
		store, err := g.DB().Model("store").Ctx(ctx).
			Where("store_id", req.ShopId).
			Where("platform", "douyin").
			Fields("store_name").
			One()
		if err != nil {
			return 0, err
		}
		if store.IsEmpty() {
			return 0, fmt.Errorf("店铺ID不存在: %s", req.ShopId)
		}
		shopName = store["store_name"].String()
	} else if shopName != "" {
		store, err := g.DB().Model("store").Ctx(ctx).
			Where("store_name", shopName).
			Where("platform", "douyin").
			Fields("store_id").
			OrderAsc("id").
			One()
		if err != nil {
			return 0, err
		}
		if !store.IsEmpty() {
			req.ShopId = store["store_id"].String()
		}
	}
	if shopName == "" {
		return 0, fmt.Errorf("店铺不能为空")
	}
	status := req.Status
	if status != 0 {
		status = 1
	}

	data := g.Map{
		"shop_id":             req.ShopId,
		"shop_name":           shopName,
		"source_account_uid":  req.SourceAccountUid,
		"source_account_name": req.SourceAccountName,
		"target_type":         req.TargetType,
		"target_value":        req.TargetValue,
		"remark":              req.Remark,
		"status":              status,
	}

	if req.Id > 0 {
		// 更新
		result, err := g.DB().Model("shop_account_attribution").Where("id", req.Id).Data(data).Update()
		if err != nil {
			return 0, err
		}
		ClearEnrichedDouyinOrderCache()
		return result.RowsAffected()
	}

	// 新增
	result, err := g.DB().Model("shop_account_attribution").Data(data).Insert()
	if err != nil {
		return 0, err
	}
	ClearEnrichedDouyinOrderCache()
	return result.RowsAffected()
}

// Delete 删除
func (s *ShopAccountAttributionService) Delete(ctx context.Context, id int) (int64, error) {
	result, err := g.DB().Model("shop_account_attribution").Where("id", id).Delete()
	if err != nil {
		return 0, err
	}
	ClearEnrichedDouyinOrderCache()
	return result.RowsAffected()
}

// GetShops 获取所有店铺名称列表
func (s *ShopAccountAttributionService) GetShops(ctx context.Context) ([]g.Map, error) {
	rows, err := g.DB().Model("store").Ctx(ctx).
		Fields("id, store_id, store_name").
		Where("platform", "douyin").
		Where("status", 1).
		Order("store_name").
		All()
	if err != nil {
		return nil, err
	}
	shops := make([]g.Map, 0, len(rows))
	for _, row := range rows {
		shops = append(shops, g.Map{
			"id":       row["id"],
			"shopId":   row["store_id"],
			"storeId":  row["store_id"],
			"shopName": row["store_name"],
		})
	}
	return shops, nil
}

// GetOperators 获取所有运营者列表
func (s *ShopAccountAttributionService) GetOperators(ctx context.Context) ([]string, error) {
	rows, err := g.DB().Model("account").Fields("DISTINCT operator").Where("operator IS NOT NULL AND operator != ''").Where("status", 1).Order("operator").All()
	if err != nil {
		return nil, err
	}
	var operators []string
	for _, row := range rows {
		operators = append(operators, row["operator"].String())
	}
	return operators, nil
}

// SearchAccounts 搜索账号（用于选择达人）
func (s *ShopAccountAttributionService) SearchAccounts(ctx context.Context, keyword string) ([]g.Map, error) {
	like := "%" + keyword + "%"
	rows, err := g.DB().Model("account").
		Fields("uid, account_name, platform_nickname, operator").
		Where("platform", "douyin").
		Where("status", 1).
		Where("(account_name LIKE ? OR platform_nickname LIKE ? OR uid LIKE ?)", like, like, like).
		Limit(20).
		All()
	if err != nil {
		return nil, err
	}
	var items []g.Map
	for _, row := range rows {
		items = append(items, g.Map{
			"uid":              row["uid"],
			"accountName":      row["account_name"],
			"platformNickname": row["platform_nickname"],
			"operator":         row["operator"],
		})
	}
	return items, nil
}

// GetAttributionSQL 获取归属规则的 SQL 片段（用于统计查询）
// 返回 COALESCE 表达式，优先使用特殊规则，兜底使用默认 operator
func (s *ShopAccountAttributionService) GetAttributionSQL() string {
	return fmt.Sprintf(`COALESCE(
		(SELECT saa.target_value
		 FROM shop_account_attribution saa
		 WHERE %s
		 ORDER BY CASE WHEN COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') != '' THEN 0 ELSE 1 END
		 LIMIT 1),
		a.operator
	)`, shopAccountAttributionRuleSQL("o", "o.talent_id"))
}
