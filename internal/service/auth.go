package service

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/model"
)

// AuthService 认证服务
type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// GetUserByUsername 根据用户名查询用户
func (s *AuthService) GetUserByUsername(ctx context.Context, username string) (*model.SysUser, error) {
	var user model.SysUser
	err := g.DB().Model("sys_user").
		Where("username", username).
		Where("status", 1).
		Scan(&user)
	if err != nil {
		return nil, err
	}
	if user.Id == 0 {
		return nil, nil
	}
	// 需要单独查密码字段（因为 Scan 不会映射 json:"-" 的字段）
	password, err := g.DB().Model("sys_user").
		Where("username", username).
		Value("password")
	if err != nil {
		return nil, err
	}
	user.Password = password.String()

	// 获取用户的主公司ID
	companyId, err := g.DB().Model("sys_user_company").
		Where("user_id", user.Id).
		Where("is_primary", 1).
		Value("company_id")
	if err == nil && !companyId.IsNil() {
		user.CompanyId = companyId.Int64()
	}
	return &user, nil
}

// GetUserRoles 获取用户角色编码列表
func (s *AuthService) GetUserRoles(ctx context.Context, userId int64) ([]string, error) {
	result, err := g.DB().Model("sys_user_role ur").
		LeftJoin("sys_role r", "ur.role_id = r.id").
		Where("ur.user_id", userId).
		Where("r.status", 1).
		Array("r.code")
	if err != nil {
		return nil, err
	}
	roles := make([]string, 0, len(result))
	for _, v := range result {
		roles = append(roles, v.String())
	}
	return roles, nil
}

// GetUserPermissionCodes 获取用户权限编码列表
func (s *AuthService) GetUserPermissionCodes(ctx context.Context, userId int64) ([]string, error) {
	result, err := g.DB().GetAll(ctx,
		`SELECT DISTINCT p.code
		 FROM sys_user_role ur
		 JOIN sys_role_permission rp ON ur.role_id = rp.role_id
		 JOIN sys_permission p ON rp.permission_id = p.id
		 WHERE ur.user_id = ? AND p.status = 1`, userId)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(result))
	for _, row := range result {
		codes = append(codes, row["code"].String())
	}
	return codes, nil
}

// menuPermRow 内部结构，用于构建菜单树
type menuPermRow struct {
	Id        int64
	Name      string
	Code      string
	ParentId  int64
	Path      string
	Component string
	Icon      string
	Sort      int
	ExtUrl    string
}

// GetUserMenus 获取用户菜单树 (适配 Vben Admin 格式)
func (s *AuthService) GetUserMenus(ctx context.Context, userId int64) ([]model.MenuItem, error) {
	result, err := g.DB().GetAll(ctx,
		`SELECT DISTINCT p.id, p.name, p.code, p.parent_id, p.path, p.component, p.icon, p.sort, p.ext_url
		 FROM sys_user_role ur
		 JOIN sys_role_permission rp ON ur.role_id = rp.role_id
		 JOIN sys_permission p ON rp.permission_id = p.id
		 WHERE ur.user_id = ? AND p.type = 1 AND p.status = 1
		 ORDER BY p.sort ASC`, userId)
	if err != nil {
		return nil, err
	}

	allPerms := make([]menuPermRow, 0, len(result))
	for _, row := range result {
		allPerms = append(allPerms, menuPermRow{
			Id:        row["id"].Int64(),
			Name:      row["name"].String(),
			Code:      row["code"].String(),
			ParentId:  row["parent_id"].Int64(),
			Path:      row["path"].String(),
			Component: row["component"].String(),
			Icon:      row["icon"].String(),
			Sort:      row["sort"].Int(),
			ExtUrl:    row["ext_url"].String(),
		})
	}

	return buildMenuTree(allPerms, 0), nil
}

func buildMenuTree(perms []menuPermRow, parentId int64) []model.MenuItem {
	var items []model.MenuItem
	for _, p := range perms {
		if p.ParentId == parentId {
			component := p.Component
			if component == "Layout" {
				component = "BasicLayout"
			}
			item := model.MenuItem{
				Name:      p.Code,
				Path:      p.Path,
				Component: component,
				Meta: model.MenuMeta{
					Title: p.Name,
					Icon:  p.Icon,
					Order: p.Sort,
					Link:  p.ExtUrl,
				},
			}
			children := buildMenuTree(perms, p.Id)
			if len(children) > 0 {
				item.Children = children
			}
			items = append(items, item)
		}
	}
	return items
}

// UpdateLastLogin 更新最后登录信息
func (s *AuthService) UpdateLastLogin(ctx context.Context, userId int64, ip string) error {
	_, err := g.DB().Model("sys_user").
		Where("id", userId).
		Data(g.Map{
			"last_login_at": "NOW()",
			"last_login_ip": ip,
		}).
		Update()
	return err
}

// RecordLoginLog 记录登录日志
func (s *AuthService) RecordLoginLog(ctx context.Context, username string, userId int64, ip string, status int, msg string) {
	_, _ = g.DB().Model("sys_login_log").Insert(g.Map{
		"user_id":  userId,
		"username": username,
		"ip":       ip,
		"status":   status,
		"msg":      msg,
	})
}
