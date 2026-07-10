package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

type ExternalTalentService struct{}

func NewExternalTalentService() *ExternalTalentService {
	return &ExternalTalentService{}
}

func normalizeExternalPlatformType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "抖音", "douyin":
		return "douyin"
	case "快手", "kuaishou", "kuaishow":
		return "kuaishou"
	case "视频号", "shipinhao", "wechat", "wechatshop":
		return "shipinhao"
	case "小红书", "xiaohongshu", "xhs":
		return "xiaohongshu"
	default:
		return value
	}
}

func platformTypeDisplayName(value string) string {
	switch normalizeExternalPlatformType(value) {
	case "douyin":
		return "抖音"
	case "kuaishou":
		return "快手"
	case "shipinhao":
		return "视频号"
	case "xiaohongshu":
		return "小红书"
	default:
		return strings.TrimSpace(value)
	}
}

func parsePlatformShopLabel(label string) (string, string) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", ""
	}

	for _, sep := range []string{"-", "－", "—", "_", "/"} {
		parts := strings.SplitN(label, sep, 2)
		if len(parts) == 2 {
			return normalizeExternalPlatformType(parts[0]), strings.TrimSpace(parts[1])
		}
	}

	return normalizeExternalPlatformType(label), ""
}

func (s *ExternalTalentService) normalizeSaveReq(req *model.ExternalTalentSaveReq) {
	req.TeamName = strings.TrimSpace(req.TeamName)
	req.PlatformType = normalizeExternalPlatformType(req.PlatformType)
	req.ShopName = strings.TrimSpace(req.ShopName)
	req.PlatformShopLabel = strings.TrimSpace(req.PlatformShopLabel)
	req.Uid = strings.TrimSpace(req.Uid)
	req.AccountName = strings.TrimSpace(req.AccountName)
	req.ShowId = strings.TrimSpace(req.ShowId)
	req.LinkCode = strings.TrimSpace(req.LinkCode)
	req.TalentName = strings.TrimSpace(req.TalentName)
	req.ProductId = strings.TrimSpace(req.ProductId)
	req.GoodsCode = strings.TrimSpace(req.GoodsCode)
	req.Remark = strings.TrimSpace(req.Remark)
	req.SourceFile = strings.TrimSpace(req.SourceFile)
	req.SourceSheet = strings.TrimSpace(req.SourceSheet)

	if req.PlatformShopLabel != "" && (req.PlatformType == "" || req.ShopName == "") {
		platformType, shopName := parsePlatformShopLabel(req.PlatformShopLabel)
		if req.PlatformType == "" {
			req.PlatformType = platformType
		}
		if req.ShopName == "" {
			req.ShopName = shopName
		}
	}

	if req.PlatformShopLabel == "" && req.PlatformType != "" && req.ShopName != "" {
		req.PlatformShopLabel = platformTypeDisplayName(req.PlatformType) + "-" + req.ShopName
	}

	if req.TalentName == "" {
		switch {
		case req.AccountName != "":
			req.TalentName = req.AccountName
		case req.ShowId != "":
			req.TalentName = req.ShowId
		case req.Uid != "":
			req.TalentName = req.Uid
		}
	}

}

func (s *ExternalTalentService) List(ctx context.Context, req *model.ExternalTalentListReq) ([]g.Map, int64, error) {
	query := g.DB().Model("external_talent_registry")

	if req.Keyword != "" {
		like := "%" + strings.TrimSpace(req.Keyword) + "%"
		query = query.Where(`(
			team_name LIKE ? OR shop_name LIKE ? OR platform_shop_label LIKE ? OR uid LIKE ? OR
			account_name LIKE ? OR show_id LIKE ? OR talent_name LIKE ? OR product_id LIKE ? OR goods_code LIKE ?
		)`, like, like, like, like, like, like, like, like, like)
	}
	if req.TeamName != "" {
		query = query.Where("team_name", strings.TrimSpace(req.TeamName))
	}
	if req.PlatformType != "" {
		query = query.Where("platform_type", normalizeExternalPlatformType(req.PlatformType))
	}
	if req.Status >= 0 {
		query = query.Where("status", req.Status)
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
			"id":                    row["id"],
			"teamName":              row["team_name"],
			"platformType":          row["platform_type"],
			"shopName":              row["shop_name"],
			"platformShopLabel":     row["platform_shop_label"],
			"uid":                   row["uid"],
			"accountName":           row["account_name"],
			"showId":                row["show_id"],
			"linkCode":              row["link_code"],
			"talentName":            row["talent_name"],
			"talentType":            row["talent_type"],
			"productId":             row["product_id"],
			"goodsCode":             row["goods_code"],
			"customerPrice":         row["customer_price"],
			"onlineCommissionRate":  row["online_commission_rate"],
			"offlineCommissionRate": row["offline_commission_rate"],
			"retainRate":            row["retain_rate"],
			"remark":                row["remark"],
			"status":                row["status"],
			"sourceFile":            row["source_file"],
			"sourceSheet":           row["source_sheet"],
			"sourceRowNo":           row["source_row_no"],
			"createdAt":             row["created_at"],
			"updatedAt":             row["updated_at"],
		}
	}

	return items, int64(total), nil
}

func (s *ExternalTalentService) Stats(ctx context.Context) (*model.ExternalTalentStats, error) {
	row, err := g.DB().Model("external_talent_registry").
		Fields(`
			COUNT(*) AS total_count,
			SUM(CASE WHEN platform_type = 'douyin' THEN 1 ELSE 0 END) AS douyin_count,
			SUM(CASE WHEN platform_type = 'kuaishou' THEN 1 ELSE 0 END) AS kuaishou_count,
			SUM(CASE WHEN platform_type = 'shipinhao' THEN 1 ELSE 0 END) AS shipinhao_count,
			SUM(CASE WHEN platform_type = 'xiaohongshu' THEN 1 ELSE 0 END) AS xiaohongshu_count,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS enabled_count
		`).
		One()
	if err != nil {
		return nil, err
	}

	return &model.ExternalTalentStats{
		TotalCount:       row["total_count"].Int64(),
		DouyinCount:      row["douyin_count"].Int64(),
		KuaishouCount:    row["kuaishou_count"].Int64(),
		ShipinhaoCount:   row["shipinhao_count"].Int64(),
		XiaohongshuCount: row["xiaohongshu_count"].Int64(),
		EnabledCount:     row["enabled_count"].Int64(),
	}, nil
}

func (s *ExternalTalentService) GetById(ctx context.Context, id int) (g.Map, error) {
	row, err := g.DB().Model("external_talent_registry").Where("id", id).One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, fmt.Errorf("记录不存在")
	}

	return g.Map{
		"id":                    row["id"],
		"teamName":              row["team_name"],
		"platformType":          row["platform_type"],
		"shopName":              row["shop_name"],
		"platformShopLabel":     row["platform_shop_label"],
		"uid":                   row["uid"],
		"accountName":           row["account_name"],
		"showId":                row["show_id"],
		"linkCode":              row["link_code"],
		"talentName":            row["talent_name"],
		"talentType":            row["talent_type"],
		"productId":             row["product_id"],
		"goodsCode":             row["goods_code"],
		"customerPrice":         row["customer_price"],
		"onlineCommissionRate":  row["online_commission_rate"],
		"offlineCommissionRate": row["offline_commission_rate"],
		"retainRate":            row["retain_rate"],
		"remark":                row["remark"],
		"status":                row["status"],
		"sourceFile":            row["source_file"],
		"sourceSheet":           row["source_sheet"],
		"sourceRowNo":           row["source_row_no"],
		"createdAt":             row["created_at"],
		"updatedAt":             row["updated_at"],
	}, nil
}

func (s *ExternalTalentService) Save(ctx context.Context, req *model.ExternalTalentSaveReq) (int64, error) {
	s.normalizeSaveReq(req)

	data := g.Map{
		"team_name":               req.TeamName,
		"platform_type":           req.PlatformType,
		"shop_name":               req.ShopName,
		"platform_shop_label":     req.PlatformShopLabel,
		"uid":                     req.Uid,
		"account_name":            req.AccountName,
		"show_id":                 req.ShowId,
		"link_code":               req.LinkCode,
		"talent_name":             req.TalentName,
		"talent_type":             "external",
		"product_id":              req.ProductId,
		"goods_code":              req.GoodsCode,
		"customer_price":          req.CustomerPrice,
		"online_commission_rate":  req.OnlineCommissionRate,
		"offline_commission_rate": req.OfflineCommissionRate,
		"retain_rate":             req.RetainRate,
		"remark":                  req.Remark,
		"status":                  req.Status,
		"source_file":             req.SourceFile,
		"source_sheet":            req.SourceSheet,
		"source_row_no":           req.SourceRowNo,
	}

	if req.Id > 0 {
		result, err := g.DB().Model("external_talent_registry").Where("id", req.Id).Data(data).Update()
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	}

	result, err := g.DB().Model("external_talent_registry").Data(data).Insert()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *ExternalTalentService) Delete(ctx context.Context, id int) (int64, error) {
	result, err := g.DB().Model("external_talent_registry").Where("id", id).Delete()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
