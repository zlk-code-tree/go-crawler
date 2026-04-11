package service

import (
	"context"
	"go-crawler/internal/model"

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
	_, err := g.DB().Model("sys_user_role").Where("user_id", userId).Delete()
	if err != nil {
		return err
	}
	var data []g.Map
	for _, rid := range roleIds {
		data = append(data, g.Map{"user_id": userId, "role_id": rid})
	}
	if len(data) > 0 {
		_, err = g.DB().Model("sys_user_role").Data(data).Insert()
	}
	return err
}

// Save 保存用户
func (s *UserService) Save(ctx context.Context, user *model.SysUser) (int64, error) {
	// 构建基础数据映射（不含密码）
	data := g.Map{
		"real_name": user.RealName,
		"nickname":  user.Nickname,
		"email":     user.Email,
		"phone":     user.Phone,
		"avatar":    user.Avatar,
		"gender":    user.Gender,
		"position":  user.Position,
		"remark":    user.Remark,
	}

	// 如果是新增用户或提供了密码，则设置密码
	if user.Id == 0 || user.Password != "" {
		password := user.Password
		if password == "" {
			password = "123456" // 设置默认密码
		}
		// 加密密码
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return 0, err
		}
		data["password"] = string(hashedPassword)

		// 对于新增用户，还需要其他必填字段
		if user.Id == 0 {
			data["username"] = user.Username
			data["status"] = user.Status
			data["dept_id"] = user.DeptId
			data["role_id"] = user.RoleId
		}
	}

	if user.Id > 0 {
		// 更新用户（不包含用户名等不可更改的字段）
		_, err := g.DB().Model("sys_user").Where("id", user.Id).Data(data).Update()
		return user.Id, err
	} else {
		// 新增用户
		id, err := g.DB().Model("sys_user").Data(data).InsertAndGetId()
		return id, err
	}
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
