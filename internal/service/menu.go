package service

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/model"
)

type MenuService struct{}

func NewMenuService() *MenuService {
	return &MenuService{}
}

// menuRow 数据库行映射
type menuRow struct {
	Id        int64
	Name      string
	Code      string
	Type      int
	ParentId  int64
	Path      string
	Component string
	Icon      string
	Method    string
	Api       string
	Status    int
	Hide      int
	Sort      int
}

// GetMenuTree 获取完整菜单树（管理用，包含所有类型）
func (s *MenuService) GetMenuTree(ctx context.Context) (interface{}, error) {
	result, err := g.DB().Model("sys_permission").
		OrderAsc("sort").OrderAsc("id").
		All()
	if err != nil {
		return nil, err
	}

	var rows []menuRow
	for _, row := range result {
		rows = append(rows, menuRow{
			Id:        row["id"].Int64(),
			Name:      row["name"].String(),
			Code:      row["code"].String(),
			Type:      row["type"].Int(),
			ParentId:  row["parent_id"].Int64(),
			Path:      row["path"].String(),
			Component: row["component"].String(),
			Icon:      row["icon"].String(),
			Method:    row["method"].String(),
			Api:       row["api"].String(),
			Status:    row["status"].Int(),
			Hide:      row["hide"].Int(),
			Sort:      row["sort"].Int(),
		})
	}

	return buildAdminMenuTree(rows, 0), nil
}

// AdminMenuItem 管理端菜单项（比前端路由多返回 id、type 等字段）
type AdminMenuItem struct {
	Id        int64           `json:"id"`
	Name      string          `json:"name"`
	Code      string          `json:"code"`
	Type      int             `json:"type"`
	ParentId  int64           `json:"parentId"`
	Path      string          `json:"path"`
	Component string          `json:"component"`
	Icon      string          `json:"icon"`
	Method    string          `json:"method,omitempty"`
	Api       string          `json:"api,omitempty"`
	Status    int             `json:"status"`
	Hide      int             `json:"hide"`
	Sort      int             `json:"sort"`
	Children  []AdminMenuItem `json:"children,omitempty"`
}

func buildAdminMenuTree(rows []menuRow, parentId int64) []AdminMenuItem {
	var items []AdminMenuItem
	for _, r := range rows {
		if r.ParentId == parentId {
			item := AdminMenuItem{
				Id:        r.Id,
				Name:      r.Name,
				Code:      r.Code,
				Type:      r.Type,
				ParentId:  r.ParentId,
				Path:      r.Path,
				Component: r.Component,
				Icon:      r.Icon,
				Method:    r.Method,
				Api:       r.Api,
				Status:    r.Status,
				Hide:      r.Hide,
				Sort:      r.Sort,
			}
			children := buildAdminMenuTree(rows, r.Id)
			if len(children) > 0 {
				item.Children = children
			}
			items = append(items, item)
		}
	}
	return items
}

// GetDetail 获取菜单详情
func (s *MenuService) GetDetail(ctx context.Context, id int64) (interface{}, error) {
	row, err := g.DB().Model("sys_permission").Where("id", id).One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, nil
	}
	return row, nil
}

// Create 创建菜单
func (s *MenuService) Create(ctx context.Context, data g.Map) (int64, error) {
	result, err := g.DB().Model("sys_permission").Insert(data)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// Update 更新菜单
func (s *MenuService) Update(ctx context.Context, id int64, data g.Map) error {
	_, err := g.DB().Model("sys_permission").Where("id", id).Update(data)
	return err
}

// Delete 删除菜单（同时删除子菜单和关联的角色权限）
func (s *MenuService) Delete(ctx context.Context, id int64) error {
	// 检查是否有子菜单
	count, err := g.DB().Model("sys_permission").Where("parent_id", id).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
			// 递归删除子菜单的角色关联
			childIds, _ := tx.Model("sys_permission").Where("parent_id", id).Array("id")
			for _, cid := range childIds {
				_, _ = tx.Model("sys_role_permission").Where("permission_id", cid.Int64()).Delete()
				_, _ = tx.Model("sys_permission").Where("id", cid.Int64()).Delete()
			}
			// 删除自身的角色关联和记录
			_, _ = tx.Model("sys_role_permission").Where("permission_id", id).Delete()
			_, err := tx.Model("sys_permission").Where("id", id).Delete()
			return err
		})
	}

	// 无子菜单，直接删除
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		_, _ = tx.Model("sys_role_permission").Where("permission_id", id).Delete()
		_, err := tx.Model("sys_permission").Where("id", id).Delete()
		return err
	})
}

// GetUserMenuTree 获取用户菜单树（前端路由用，只返回菜单类型且启用的）
func (s *MenuService) GetUserMenuTree(ctx context.Context, userId int64) ([]model.MenuItem, error) {
	result, err := g.DB().GetAll(ctx,
		`SELECT DISTINCT p.id, p.name, p.code, p.parent_id, p.path, p.component, p.icon, p.sort, p.hide
		 FROM sys_user_role ur
		 JOIN sys_role_permission rp ON ur.role_id = rp.role_id
		 JOIN sys_permission p ON rp.permission_id = p.id
		 WHERE ur.user_id = ? AND p.type = 1 AND p.status = 1
		 ORDER BY p.sort ASC, p.id ASC`, userId)
	if err != nil {
		return nil, err
	}

	var rows []menuRow
	for _, row := range result {
		rows = append(rows, menuRow{
			Id:        row["id"].Int64(),
			Name:      row["name"].String(),
			Code:      row["code"].String(),
			ParentId:  row["parent_id"].Int64(),
			Path:      row["path"].String(),
			Component: row["component"].String(),
			Icon:      row["icon"].String(),
			Sort:      row["sort"].Int(),
			Hide:      row["hide"].Int(),
		})
	}

	return buildVbenMenuTree(rows, 0), nil
}

func buildVbenMenuTree(rows []menuRow, parentId int64) []model.MenuItem {
	var items []model.MenuItem
	for _, r := range rows {
		if r.ParentId == parentId {
			component := r.Component
			if component == "Layout" {
				component = "BasicLayout"
			}
			item := model.MenuItem{
				Name:      r.Code,
				Path:      r.Path,
				Component: component,
				Meta: model.MenuMeta{
					Title:      r.Name,
					Icon:       r.Icon,
					Order:      r.Sort,
					HideInMenu: r.Hide == 1,
				},
			}
			children := buildVbenMenuTree(rows, r.Id)
			if len(children) > 0 {
				item.Children = children
			}
			items = append(items, item)
		}
	}
	return items
}
