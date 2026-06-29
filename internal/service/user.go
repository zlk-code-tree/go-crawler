package service

import (
	"context"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户相关业务逻辑

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

// List 查询用户列表
func (s *UserService) List(ctx context.Context, page, pageSize int, keyword string) ([]model.SysUser, int, error) {
	m := g.DB().Model("sys_user")
	if keyword != "" {
		m = m.WhereLike("username", "%"+keyword+"%")
	}
	total, err := m.Count()
	if err != nil {
		return nil, 0, err
	}
	var users []model.SysUser
	err = m.OrderDesc("id").Page(page, pageSize).Scan(&users)
	return users, total, err
}

// AssignRoles 分配角色
func (s *UserService) AssignRoles(ctx context.Context, userId int64, roleIds []int64) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		_, err := tx.Model("sys_user_role").Where("user_id", userId).Delete()
		if err != nil {
			return err
		}
		var data []g.Map
		for _, rid := range roleIds {
			data = append(data, g.Map{"user_id": userId, "role_id": rid})
		}
		if len(data) > 0 {
			_, err = tx.Model("sys_user_role").Data(data).Insert()
		}
		return err
	})
}

// Save 保存用户，同时同步 sys_user_role
func (s *UserService) Save(ctx context.Context, user *model.SysUser) (int64, error) {
	data := g.Map{
		"username":   user.Username,
		"real_name":  user.RealName,
		"nickname":   user.Nickname,
		"email":      user.Email,
		"phone":      user.Phone,
		"avatar":     user.Avatar,
		"gender":     user.Gender,
		"position":   user.Position,
		"remark":     user.Remark,
		"status":     user.Status,
		"dept_id":    user.DeptId,
		"role_id":    user.RoleId,
		"company_id": user.CompanyId,
	}

	if user.Username == "" {
		delete(data, "username")
	}

	if user.Id == 0 || user.Password != "" {
		password := user.Password
		if password == "" {
			password = "123456"
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return 0, err
		}
		data["password"] = string(hashedPassword)
	}

	var userId int64
	err := g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if user.Id > 0 {
			_, err := tx.Model("sys_user").Where("id", user.Id).Data(data).Update()
			if err != nil {
				return err
			}
			userId = user.Id
		} else {
			id, err := tx.Model("sys_user").Data(data).InsertAndGetId()
			if err != nil {
				return err
			}
			userId = id
		}

		// 同步 sys_user_role
		_, err := tx.Model("sys_user_role").Where("user_id", userId).Delete()
		if err != nil {
			return err
		}
		if user.RoleId > 0 {
			_, err = tx.Model("sys_user_role").Insert(g.Map{"user_id": userId, "role_id": user.RoleId})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return userId, err
}

// Delete 删除用户
func (s *UserService) Delete(ctx context.Context, userId int64) error {
	// 先删除用户角色关联
	_, err := g.DB().Model("sys_user_role").Where("user_id", userId).Delete()
	if err != nil {
		return err
	}
	// 再删除用户
	_, err = g.DB().Model("sys_user").Where("id", userId).Delete()
	return err
}

// Get 获取用户详情
func (s *UserService) Get(ctx context.Context, userId int64) (*model.SysUser, error) {
	var user model.SysUser
	err := g.DB().Model("sys_user").Where("id", userId).Scan(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
