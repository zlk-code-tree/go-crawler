package service

import (
	"context"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// RoleService 角色相关业务逻辑
// 包含角色的增删改查、菜单分配、数据权限分配等

type RoleService struct{}

func NewRoleService() *RoleService {
	return &RoleService{}
}

// List 查询角色列表
func (s *RoleService) List(ctx context.Context, page, pageSize int, keyword string) ([]model.SysRole, int, error) {
	m := g.DB().Model("sys_role")
	if keyword != "" {
		m = m.WhereLike("name", "%"+keyword+"%")
	}
	total, err := m.Count()
	if err != nil {
		return nil, 0, err
	}
	var roles []model.SysRole
	err = m.OrderDesc("id").Page(page, pageSize).Scan(&roles)
	return roles, total, err
}

// Create 创建角色并分配权限
func (s *RoleService) Create(ctx context.Context, name, code string, sort int, status int, remark string, menuIds []int64) (int64, error) {
	var roleId int64
	err := g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		result, err := tx.Model("sys_role").Insert(g.Map{
			"name":   name,
			"code":   code,
			"sort":   sort,
			"status": status,
			"remark": remark,
			"type":   3,
		})
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		roleId = id
		if len(menuIds) > 0 {
			var data []g.Map
			for _, mid := range menuIds {
				data = append(data, g.Map{"role_id": roleId, "permission_id": mid})
			}
			_, err = tx.Model("sys_role_permission").Data(data).Insert()
			if err != nil {
				return err
			}
		}
		return nil
	})
	return roleId, err
}

// Save 新增或编辑角色
func (s *RoleService) Save(ctx context.Context, role *model.SysRole) (int64, error) {
	if role.Id == 0 {
		result, err := g.DB().Model("sys_role").Insert(role)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	} else {
		_, err := g.DB().Model("sys_role").Where("id", role.Id).Update(role)
		return role.Id, err
	}
}

// Delete 删除角色
func (s *RoleService) Delete(ctx context.Context, id int64) error {
	_, err := g.DB().Model("sys_role").Where("id", id).Delete()
	return err
}

// GetById 根据ID获取角色详情
func (s *RoleService) GetById(ctx context.Context, id int64) (*model.SysRole, error) {
	var role model.SysRole
	err := g.DB().Model("sys_role").Where("id", id).Scan(&role)
	if err != nil {
		return nil, err
	}
	if role.Id == 0 {
		return nil, nil
	}
	return &role, nil
}

// Update 更新角色信息
func (s *RoleService) Update(ctx context.Context, id int64, name, code string, sort int, status int, remark string, menuIds []int64) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 更新角色基本信息
		_, err := tx.Model("sys_role").Where("id", id).Update(g.Map{
			"name":   name,
			"code":   code,
			"sort":   sort,
			"status": status,
			"remark": remark,
		})
		if err != nil {
			return err
		}

		// 更新角色权限
		_, err = tx.Model("sys_role_permission").Where("role_id", id).Delete()
		if err != nil {
			return err
		}

		if len(menuIds) > 0 {
			var data []g.Map
			for _, mid := range menuIds {
				data = append(data, g.Map{"role_id": id, "permission_id": mid})
			}
			_, err = tx.Model("sys_role_permission").Data(data).Insert()
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// GetMenuIds 获取角色已分配菜单ID
func (s *RoleService) GetMenuIds(ctx context.Context, roleId int64) ([]int64, error) {
	result, err := g.DB().Model("sys_role_permission").
		Fields("permission_id").
		Where("role_id", roleId).
		Array()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(result))
	for _, v := range result {
		ids = append(ids, v.Int64())
	}
	return ids, nil
}

// AssignMenus 分配菜单
func (s *RoleService) AssignMenus(ctx context.Context, roleId int64, menuIds []int64) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		_, err := tx.Model("sys_role_permission").Where("role_id", roleId).Delete()
		if err != nil {
			return err
		}
		var data []g.Map
		for _, mid := range menuIds {
			data = append(data, g.Map{"role_id": roleId, "permission_id": mid})
		}
		if len(data) > 0 {
			_, err = tx.Model("sys_role_permission").Data(data).Insert()
		}
		return err
	})
}

// GetDataScope 获取角色数据权限
func (s *RoleService) GetDataScope(ctx context.Context, roleId int64) ([]int64, error) {
	var ids []int64
	_, err := g.DB().Model("sys_role_dept").Where("role_id", roleId).Array(&ids, "dept_id")
	return ids, err
}

// AssignDataScope 分配数据权限
func (s *RoleService) AssignDataScope(ctx context.Context, roleId int64, deptIds []int64) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		_, err := tx.Model("sys_role_dept").Where("role_id", roleId).Delete()
		if err != nil {
			return err
		}
		var data []g.Map
		for _, did := range deptIds {
			data = append(data, g.Map{"role_id": roleId, "dept_id": did})
		}
		if len(data) > 0 {
			_, err = tx.Model("sys_role_dept").Data(data).Insert()
		}
		return err
	})
}
