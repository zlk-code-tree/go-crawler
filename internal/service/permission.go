package service

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

type PermissionService struct{}

func NewPermissionService() *PermissionService {
	return &PermissionService{}
}

type permissionRow struct {
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
	ExtUrl    string
}

type AdminPermissionItem struct {
	Id        int64                 `json:"id"`
	Name      string                `json:"name"`
	Code      string                `json:"code"`
	Type      int                   `json:"type"`
	ParentId  int64                 `json:"parentId"`
	Path      string                `json:"path"`
	Component string                `json:"component"`
	Icon      string                `json:"icon"`
	Method    string                `json:"method,omitempty"`
	Api       string                `json:"api,omitempty"`
	Status    int                   `json:"status"`
	Hide      int                   `json:"hide"`
	Sort      int                   `json:"sort"`
	ExtUrl    string                `json:"extUrl"`
	Children  []AdminPermissionItem `json:"children,omitempty"`
}

func (s *PermissionService) GetPermissionTree(ctx context.Context) (interface{}, error) {
	result, err := g.DB().Model("sys_permission").OrderAsc("sort").OrderAsc("id").All()
	if err != nil {
		return nil, err
	}
	var rows []permissionRow
	for _, row := range result {
		rows = append(rows, permissionRow{
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
			ExtUrl:    row["ext_url"].String(),
		})
	}
	return buildAdminPermissionTree(rows, 0), nil
}

func buildAdminPermissionTree(rows []permissionRow, parentId int64) []AdminPermissionItem {
	var items []AdminPermissionItem
	for _, r := range rows {
		if r.ParentId == parentId {
			item := AdminPermissionItem{
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
				ExtUrl:    r.ExtUrl,
			}
			children := buildAdminPermissionTree(rows, r.Id)
			if len(children) > 0 {
				item.Children = children
			}
			items = append(items, item)
		}
	}
	return items
}

func (s *PermissionService) GetDetail(ctx context.Context, id int64) (interface{}, error) {
	row, err := g.DB().Model("sys_permission").Where("id", id).One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, nil
	}
	return row, nil
}

func (s *PermissionService) Create(ctx context.Context, data g.Map) (int64, error) {
	result, err := g.DB().Model("sys_permission").Insert(data)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return id, nil
}

func (s *PermissionService) Update(ctx context.Context, id int64, data g.Map) error {
	_, err := g.DB().Model("sys_permission").Where("id", id).Update(data)
	return err
}

func (s *PermissionService) Delete(ctx context.Context, id int64) error {
	// 检查是否有子权限
	count, err := g.DB().Model("sys_permission").Where("parent_id", id).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
			childIds, _ := tx.Model("sys_permission").Where("parent_id", id).Array("id")
			for _, cid := range childIds {
				_, _ = tx.Model("sys_role_permission").Where("permission_id", cid.Int64()).Delete()
				_, _ = tx.Model("sys_permission").Where("id", cid.Int64()).Delete()
			}
			_, _ = tx.Model("sys_role_permission").Where("permission_id", id).Delete()
			_, err := tx.Model("sys_permission").Where("id", id).Delete()
			return err
		})
	}
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		_, _ = tx.Model("sys_role_permission").Where("permission_id", id).Delete()
		_, err := tx.Model("sys_permission").Where("id", id).Delete()
		return err
	})
}

// CheckPermission 检查用户是否有特定权限
func (s *PermissionService) CheckPermission(ctx context.Context, userId int64, permissionCode string) (bool, error) {
	count, err := g.DB().Model("v_user_permissions").
		Where("user_id", userId).
		Where("permission_code", permissionCode).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserPermissions 获取用户所有权限编码
func (s *PermissionService) GetUserPermissions(ctx context.Context, userId int64) ([]string, error) {
	result, err := g.DB().Model("v_user_permissions").
		Where("user_id", userId).
		Array("permission_code")
	if err != nil {
		return nil, err
	}
	permissions := make([]string, 0, len(result))
	for _, v := range result {
		permissions = append(permissions, v.String())
	}
	return permissions, nil
}
