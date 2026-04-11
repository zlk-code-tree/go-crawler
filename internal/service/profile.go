package service

import (
	"context"
	"errors"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/crypto/bcrypt"
)

// ProfileService 个人设置相关业务逻辑

type ProfileService struct{}

func NewProfileService() *ProfileService {
	return &ProfileService{}
}

// UpdateProfile 更新个人资料
func (s *ProfileService) UpdateProfile(ctx context.Context, userId int64, profileData g.Map) error {
	// 只更新允许修改的个人资料字段
	updateData := g.Map{}

	// 安全地复制允许更新的字段
	allowedFields := []string{"real_name", "nickname", "email", "phone", "avatar", "gender", "position", "remark"}

	for _, field := range allowedFields {
		if val, exists := profileData[field]; exists {
			updateData[field] = val
		}
	}

	// 更新用户资料
	_, err := g.DB().Model("sys_user").Where("id", userId).Data(updateData).Update()
	return err
}

// GetProfile 获取个人资料
func (s *ProfileService) GetProfile(ctx context.Context, userId int64) (*model.SysUser, error) {
	var user model.SysUser
	err := g.DB().Model("sys_user").Where("id", userId).Scan(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 自定义错误类型
var ErrInvalidOldPassword = errors.New("invalid old password")

// ChangePassword 修改密码
func (s *ProfileService) ChangePassword(ctx context.Context, userId int64, oldPassword, newPassword string) error {
	// 单独查询密码字段
	password, err := g.DB().Model("sys_user").Where("id", userId).Value("password")
	if err != nil {
		return err
	}
	if password.IsNil() {
		return ErrInvalidOldPassword
	}

	// 验证旧密码
	err = bcrypt.CompareHashAndPassword([]byte(password.String()), []byte(oldPassword))
	if err != nil {
		return ErrInvalidOldPassword
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 更新密码
	_, err = g.DB().Model("sys_user").Where("id", userId).Data(g.Map{
		"password": string(hashedPassword),
	}).Update()
	return err
}
