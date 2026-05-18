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
	UserId         int64
	Username       string
	Module         string
	Action         string
	Method         string
	Url            string
	Params         string
	Ip             string
	Location       string
	UserAgent      string
	Status         int
	ErrorMsg       string
	Duration       int
	PermissionCode string
}

func (s *OperationLogService) Record(ctx context.Context, input OperationLogInput) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		_, err := g.DB().Model("sys_operation_log").Insert(g.Map{
			"user_id":         input.UserId,
			"username":        input.Username,
			"module":          input.Module,
			"action":          input.Action,
			"method":          input.Method,
			"url":             input.Url,
			"params":          input.Params,
			"ip":              input.Ip,
			"location":        input.Location,
			"user_agent":      input.UserAgent,
			"status":          input.Status,
			"error_msg":       input.ErrorMsg,
			"duration":        input.Duration,
			"permission_code": input.PermissionCode,
			"created_at":      time.Now(),
		})
		if err == nil {
			return
		}
		// 如果是连接错误，重试
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
	}
}

func (s *OperationLogService) GetList(ctx context.Context, page, pageSize int, keyword string) ([]g.Map, int, error) {
	// 构建查询条件
	whereModel := g.DB().Model("sys_operation_log ol").
		LeftJoin("sys_user u", "ol.user_id = u.id")

	if keyword != "" {
		whereModel = whereModel.Where("ol.username LIKE ? OR u.real_name LIKE ? OR ol.module LIKE ? OR ol.action LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 统计总数
	var total int
	count, err := whereModel.Count()
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

	// 查询列表数据
	result, err := whereModel.
		Fields("ol.*, u.real_name").
		OrderDesc("ol.created_at").
		Limit((page-1)*pageSize, pageSize).
		All()
	if err != nil {
		return nil, 0, err
	}

	rows := make([]g.Map, 0, len(result))
	for _, row := range result {
		realName := row["real_name"].String()
		if realName == "" {
			realName = row["username"].String()
		}

		rows = append(rows, g.Map{
			"id":              row["id"].Int64(),
			"user_id":         row["user_id"].Int64(),
			"username":        row["username"].String(),
			"real_name":       realName,
			"module":          row["module"].String(),
			"action":          row["action"].String(),
			"method":          row["method"].String(),
			"url":             row["url"].String(),
			"params":          row["params"].String(),
			"ip":              row["ip"].String(),
			"location":        row["location"].String(),
			"user_agent":      row["user_agent"].String(),
			"status":          row["status"].Int(),
			"error_msg":       row["error_msg"].String(),
			"duration":        row["duration"].Int(),
			"permission_code": row["permission_code"].String(),
			"created_at":      row["created_at"].String(),
		})
	}

	return rows, total, nil
}

func (s *OperationLogService) GetDetail(ctx context.Context, id int64) (g.Map, error) {
	result, err := g.DB().Model("sys_operation_log ol").
		LeftJoin("sys_user u", "ol.user_id = u.id").
		Fields("ol.*, u.real_name, u.email, u.phone").
		Where("ol.id", id).
		One()

	if err != nil {
		return nil, err
	}

	if result.IsEmpty() {
		return nil, nil
	}

	realName := result["real_name"].String()
	if realName == "" {
		realName = result["username"].String()
	}

	return g.Map{
		"id":              result["id"].Int64(),
		"user_id":         result["user_id"].Int64(),
		"username":        result["username"].String(),
		"real_name":       realName,
		"email":           result["email"].String(),
		"phone":           result["phone"].String(),
		"module":          result["module"].String(),
		"action":          result["action"].String(),
		"method":          result["method"].String(),
		"url":             result["url"].String(),
		"params":          result["params"].String(),
		"ip":              result["ip"].String(),
		"location":        result["location"].String(),
		"user_agent":      result["user_agent"].String(),
		"status":          result["status"].Int(),
		"error_msg":       result["error_msg"].String(),
		"duration":        result["duration"].Int(),
		"permission_code": result["permission_code"].String(),
		"created_at":      result["created_at"].String(),
	}, nil
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
