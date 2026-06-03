package service

import (
	"context"
	"database/sql"
	"fmt"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// TraderService 投手管理服务
type TraderService struct{}

func NewTraderService() *TraderService {
	return &TraderService{}
}

// GetList 获取投手列表（分页+搜索，含公司名和组名）
func (s *TraderService) GetList(ctx context.Context, req model.TraderListReq) ([]model.Trader, int, error) {
	// 构建基础查询（不设置 Fields，先用于 Count）
	query := g.DB().Model("trader t").
		LeftJoin("sys_company oc", "t.company_id = oc.id").
		LeftJoin("sys_department og", "t.group_id = og.id").
		Where("t.status", 1)

	if req.Keyword != "" {
		query = query.WhereLike("t.name", "%"+req.Keyword+"%")
	}
	if req.Role > 0 {
		query = query.Where("t.role", req.Role)
	}
	if req.CompanyId > 0 {
		query = query.Where("t.company_id", req.CompanyId)
	}
	if req.GroupId > 0 {
		query = query.Where("t.group_id", req.GroupId)
	}

	// 查询总数
	total, err := query.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("查询投手总数失败: %v", err)
	}

	// 分页查询（追加 Fields）
	var list []model.Trader
	err = query.
		Fields("t.*, oc.name AS company_name, og.name AS group_name").
		Order("t.id DESC").
		Page(req.Page, req.PageSize).
		Scan(&list)
	if err != nil {
		return nil, 0, fmt.Errorf("查询投手列表失败: %v", err)
	}

	return list, total, nil
}

// GetById 根据ID获取投手，查不到返回 nil, nil
func (s *TraderService) GetById(ctx context.Context, id int64) (*model.Trader, error) {
	var trader model.Trader
	err := g.DB().Model("trader").Where("id", id).Scan(&trader)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询投手失败: %v", err)
	}
	return &trader, nil
}

// GetDetail 获取投手详情（含管理的店铺、公司名、组名）
func (s *TraderService) GetDetail(ctx context.Context, id int64) (*model.TraderWithShops, error) {
	// 通过 JOIN 一次查询投手信息 + 公司名 + 部门名
	var detail model.TraderWithShops
	err := g.DB().Model("trader t").
		LeftJoin("sys_company oc", "t.company_id = oc.id").
		LeftJoin("sys_department og", "t.group_id = og.id").
		Where("t.id", id).
		Where("t.status", 1).
		Fields("t.*, oc.name AS company_name, og.name AS group_name").
		Scan(&detail)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("投手不存在")
		}
		return nil, fmt.Errorf("查询投手详情失败: %v", err)
	}
	if detail.Id == 0 {
		return nil, fmt.Errorf("投手不存在")
	}

	// 获取管理的账号列表
	shops, err := s.GetShops(ctx, id)
	if err != nil {
		return nil, err
	}
	detail.Shops = shops

	return &detail, nil
}

// Save 保存投手（新增或编辑）
func (s *TraderService) Save(ctx context.Context, req model.TraderSaveReq) (int64, error) {
	if req.Id > 0 {
		// 编辑 - 先验证投手存在
		trader, err := s.GetById(ctx, req.Id)
		if err != nil {
			return 0, err
		}
		if trader == nil || trader.Id == 0 {
			return 0, fmt.Errorf("投手不存在")
		}
		_, err = g.DB().Model("trader").
			Where("id", req.Id).
			Data(g.Map{
				"name":       req.Name,
				"company_id": req.CompanyId,
				"group_id":   req.GroupId,
				"role":       req.Role,
				"remark":     req.Remark,
			}).
			Update()
		if err != nil {
			return 0, fmt.Errorf("更新投手失败: %v", err)
		}
		return req.Id, nil
	}

	// 新增
	lastInsertId, err := g.DB().Model("trader").
		Data(g.Map{
			"name":       req.Name,
			"company_id": req.CompanyId,
			"group_id":   req.GroupId,
			"role":       req.Role,
			"remark":     req.Remark,
			"status":     1,
		}).
		InsertAndGetId()
	if err != nil {
		return 0, fmt.Errorf("新增投手失败: %v", err)
	}
	return lastInsertId, nil
}

// Delete 删除投手（软删除）
func (s *TraderService) Delete(ctx context.Context, id int64) error {
	// 验证投手存在
	trader, err := s.GetById(ctx, id)
	if err != nil {
		return err
	}
	if trader == nil || trader.Id == 0 {
		return fmt.Errorf("投手不存在")
	}
	if trader.Status == 0 {
		return fmt.Errorf("投手已被禁用")
	}

	// 获取该投手绑定的店铺ID列表
	var accountIds []int64
	err = g.DB().Model("trader_account_relation").
		Where("trader_id", id).
		Fields("account_id").
		Scan(&accountIds)
	if err != nil {
		return fmt.Errorf("查询投手账号关联失败: %v", err)
	}

	// 删除关联的店铺关系
	_, err = g.DB().Model("trader_account_relation").
		Where("trader_id", id).
		Delete()
	if err != nil {
		return fmt.Errorf("删除投手账号关联失败: %v", err)
	}

	// 更新账号绑定状态（仅更新没有被其他投手绑定的账号）
	for _, accountId := range accountIds {
		otherCount, err := g.DB().Model("trader_account_relation").
			Where("account_id", accountId).
			Count()
		if err != nil {
			continue
		}
		if otherCount == 0 {
			_, _ = g.DB().Model("account").
				Where("id", accountId).
				Data(g.Map{"bind_status": 0}).
				Update()
		}
	}

	// 软删除投手
	_, err = g.DB().Model("trader").
		Where("id", id).
		Data(g.Map{
			"status":     0,
			"updated_at": gtime.Now(),
		}).
		Update()
	if err != nil {
		return fmt.Errorf("删除投手失败: %v", err)
	}
	return nil
}

// BindShop 批量绑定账号到投手
func (s *TraderService) BindShop(ctx context.Context, traderId int64, accountIds []int64) (int, error) {
	g.Log().Infof(ctx, "[BindShop-Service] traderId=%d, accountIds=%v, len=%d", traderId, accountIds, len(accountIds))
	// 验证投手是否存在
	trader, err := s.GetById(ctx, traderId)
	if err != nil {
		return 0, err
	}
	if trader == nil || trader.Id == 0 || trader.Status == 0 {
		return 0, fmt.Errorf("投手不存在或已禁用")
	}

	boundCount := 0
	for i, accountId := range accountIds {
		g.Log().Infof(ctx, "[BindShop-Service] 处理第%d个: accountId=%d", i+1, accountId)
		// 验证店铺是否存在
		shopCount, err := g.DB().Model("account").Where("id", accountId).Count()
		if err != nil {
			g.Log().Warningf(ctx, "[BindShop-Service] 查询账号失败: accountId=%d, err=%v", accountId, err)
			continue
		}
		if shopCount == 0 {
			g.Log().Warningf(ctx, "[BindShop-Service] 账号不存在: accountId=%d", accountId)
			continue
		}

		// 检查是否已绑定
		count, err := g.DB().Model("trader_account_relation").
			Where("trader_id", traderId).
			Where("account_id", accountId).
			Count()
		if err != nil {
			g.Log().Warningf(ctx, "[BindShop-Service] 查询绑定关系失败: accountId=%d, err=%v", accountId, err)
			continue
		}
		if count > 0 {
			g.Log().Infof(ctx, "[BindShop-Service] 已绑定，跳过: accountId=%d", accountId)
			continue
		}

		// 写入关联关系
		_, err = g.DB().Model("trader_account_relation").
			Data(g.Map{
				"trader_id": traderId,
				"account_id":   accountId,
			}).
			Insert()
		if err != nil {
			g.Log().Warningf(ctx, "[BindShop-Service] 写入关联失败: accountId=%d, err=%v", accountId, err)
			continue
		}

		// 更新账号绑定状态(bind_status字段可能不存在，忽略错误)
		_, _ = g.DB().Model("account").
			Where("id", accountId).
			Data(g.Map{"bind_status": 1}).
			Update()

		g.Log().Infof(ctx, "[BindShop-Service] 绑定成功: accountId=%d", accountId)
		boundCount++
	}
	return boundCount, nil
}

// UnbindShop 解绑账号
func (s *TraderService) UnbindShop(ctx context.Context, traderId int64, accountId int) error {
	result, err := g.DB().Model("trader_account_relation").
		Where("trader_id", traderId).
		Where("account_id", accountId).
		Delete()
	if err != nil {
		return fmt.Errorf("解绑账号失败: %v", err)
	}

	// 检查删除是否生效
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("绑定关系不存在")
	}

	// 检查该账号是否还被其他投手绑定
	otherCount, err := g.DB().Model("trader_account_relation").
		Where("account_id", accountId).
		Count()
	if err != nil {
		return fmt.Errorf("查询账号绑定状态失败: %v", err)
	}

	// 如果没有其他投手绑定，更新账号为未绑定
	if otherCount == 0 {
		_, err = g.DB().Model("account").
			Where("id", accountId).
			Data(g.Map{"bind_status": 0}).
			Update()
		if err != nil {
			return fmt.Errorf("更新账号绑定状态失败: %v", err)
		}
	}

	return nil
}

// GetBindableShops 获取可绑定的账号列表（未被当前投手绑定的店铺）
func (s *TraderService) GetBindableShops(ctx context.Context, traderId int64, keyword string, platform string) ([]model.BindableShop, error) {
	query := g.DB().Model("account a").
		Where("a.id NOT IN (?)", g.DB().Raw("SELECT account_id FROM trader_account_relation WHERE trader_id = ?", traderId))

	if keyword != "" {
		query = query.Where("a.account_name LIKE ? OR a.uid LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if platform != "" {
		query = query.Where("a.platform", platform)
	}

	var shops []model.BindableShop
	err := query.Fields("a.id AS account_id, a.account_name AS shop_name, a.uid, a.platform").Scan(&shops)
	if err != nil {
		return nil, fmt.Errorf("查询可绑定账号失败: %v", err)
	}
	return shops, nil
}

// GetShops 获取投手管理的账号列表
func (s *TraderService) GetShops(ctx context.Context, traderId int64) ([]model.TraderShop, error) {
	var shops []model.TraderShop
	err := g.DB().Model("trader_account_relation r").
		LeftJoin("account a", "r.account_id = a.id").
		Where("r.trader_id", traderId).
		Fields("r.account_id, a.account_name AS shop_name, a.uid").
		Scan(&shops)
	if err != nil {
		return nil, fmt.Errorf("查询投手账号失败: %v", err)
	}
	return shops, nil
}
