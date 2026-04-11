package service

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type OperationLogService struct{}

func NewOperationLogService() *OperationLogService {
	return &OperationLogService{}
}

type OperationLogInput struct {
	UserId    int64
	Username  string
	Module    string
	Action    string
	Method    string
	Url       string
	Params    string
	Ip        string
	UserAgent string
	Status    int
	ErrorMsg  string
	Duration  int
}

func (s *OperationLogService) Record(ctx context.Context, input OperationLogInput) {
	_, _ = g.DB().Model("sys_operation_log").Insert(g.Map{
		"user_id":    input.UserId,
		"username":   input.Username,
		"module":     input.Module,
		"action":     input.Action,
		"method":     input.Method,
		"url":        input.Url,
		"params":     input.Params,
		"ip":         input.Ip,
		"user_agent": input.UserAgent,
		"status":     input.Status,
		"error_msg":  input.ErrorMsg,
		"duration":   input.Duration,
		"created_at": time.Now(),
	})
}

func (s *OperationLogService) GetList(ctx context.Context, page, pageSize int, keyword string) ([]g.Map, int, error) {
	model := g.DB().Model("sys_operation_log").OrderDesc("created_at")

	if keyword != "" {
		model = model.Where("username LIKE ? OR module LIKE ? OR action LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
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
			"module":     row["module"].String(),
			"action":     row["action"].String(),
			"method":     row["method"].String(),
			"url":        row["url"].String(),
			"params":     row["params"].String(),
			"ip":         row["ip"].String(),
			"user_agent": row["user_agent"].String(),
			"status":     row["status"].Int(),
			"error_msg":  row["error_msg"].String(),
			"duration":   row["duration"].Int(),
			"created_at": row["created_at"].String(),
		})
	}

	return rows, total, nil
}

func extractModuleFromUrl(url string) string {
	parts := strings.Split(strings.TrimPrefix(url, "/api/"), "/")
	if len(parts) > 0 {
		module := parts[0]
		switch {
		case strings.HasPrefix(module, "system/"):
			return strings.Replace(module, "system/", "系统管理 - ", 1)
		case module == "user":
			return "用户管理"
		case module == "role":
			return "角色管理"
		case module == "company":
			return "公司管理"
		case module == "permission":
			return "权限管理"
		case module == "dept":
			return "部门管理"
		case module == "menu":
			return "菜单管理"
		case module == "profile":
			return "个人设置"
		default:
			return module
		}
	}
	return "未知模块"
}

func extractActionFromMethod(method, url string) string {
	upperMethod := strings.ToUpper(method)
	switch upperMethod {
	case "GET":
		return "查询"
	case "POST":
		return "新增"
	case "PUT":
		return "更新"
	case "DELETE":
		return "删除"
	default:
		return upperMethod
	}
}
