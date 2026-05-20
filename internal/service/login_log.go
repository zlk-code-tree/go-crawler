package service

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

type LoginLogService struct{}

func NewLoginLogService() *LoginLogService {
	return &LoginLogService{}
}

func (s *LoginLogService) GetList(ctx context.Context, page, pageSize int, keyword string) ([]g.Map, int, error) {
	model := g.DB().Model("sys_login_log").OrderDesc("created_at")

	if keyword != "" {
		model = model.Where("username LIKE ? OR msg LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int
	count, err := model.Count()
	if err != nil {
		return nil, 0, err
	}
	total = count

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	result, err := model.Limit((page-1)*pageSize, pageSize).All()
	if err != nil {
		return nil, 0, err
	}

	rows := make([]g.Map, 0, len(result))
	for _, row := range result {
		// 状态转文字 (1=登录成功, 2=登录失败)
		statusVal := row["status"].Int()
		statusText := "登录失败"
		if statusVal == 1 {
			statusText = "登录成功"
		}

		rows = append(rows, g.Map{
			"id":         row["id"].Int64(),
			"user_id":    row["user_id"].Int64(),
			"username":   row["username"].String(),
			"ip":         row["ip"].String(),
			"location":   row["location"].String(),
			"browser":    row["browser"].String(),
			"os":         row["os"].String(),
			"user_agent": row["user_agent"].String(),
			"status":     statusText,
			"status_val": statusVal,
			"msg":        row["msg"].String(),
			"created_at": row["created_at"].String(),
		})
	}

	return rows, total, nil
}

// GetDetail 获取登录日志详情
func (s *LoginLogService) GetDetail(ctx context.Context, id int64) (g.Map, error) {
	result, err := g.DB().Model("sys_login_log ll").
		LeftJoin("sys_user u", "ll.user_id = u.id").
		Fields("ll.*, u.real_name, u.email, u.phone").
		Where("ll.id", id).
		One()

	if err != nil {
		return nil, err
	}

	if result.IsEmpty() {
		return nil, nil
	}

	// 状态转文字 (1=登录成功, 2=登录失败)
	statusVal := result["status"].Int()
	statusText := "登录失败"
	if statusVal == 1 {
		statusText = "登录成功"
	}

	realName := result["real_name"].String()
	if realName == "" {
		realName = result["username"].String()
	}

	return g.Map{
		"id":         result["id"].Int64(),
		"user_id":    result["user_id"].Int64(),
		"username":   result["username"].String(),
		"real_name":  realName,
		"email":      result["email"].String(),
		"phone":      result["phone"].String(),
		"ip":         result["ip"].String(),
		"location":   result["location"].String(),
		"browser":    result["browser"].String(),
		"os":         result["os"].String(),
		"user_agent": result["user_agent"].String(),
		"status":     statusText,
		"status_val": statusVal,
		"msg":        result["msg"].String(),
		"created_at": result["created_at"].String(),
	}, nil
}
