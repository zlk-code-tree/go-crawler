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
		rows = append(rows, g.Map{
			"id":         row["id"].Int64(),
			"user_id":    row["user_id"].Int64(),
			"username":   row["username"].String(),
			"ip":         row["ip"].String(),
			"location":   row["location"].String(),
			"browser":    row["browser"].String(),
			"os":         row["os"].String(),
			"status":     row["status"].Int(),
			"msg":        row["msg"].String(),
			"created_at": row["created_at"].String(),
		})
	}

	return rows, total, nil
}
