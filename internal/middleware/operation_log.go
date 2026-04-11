package middleware

import (
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	"go-crawler/internal/service"
)

func OperationLog(r *ghttp.Request) {
	startTime := time.Now()

	userId := r.GetCtxVar("userId").Int64()
	username := r.GetCtxVar("username").String()

	r.Middleware.Next()

	duration := int(time.Since(startTime).Milliseconds())

	ip := r.GetClientIp()
	userAgent := r.UserAgent()
	method := r.Method
	url := r.URL.Path
	params := ""

	if r.Method == "GET" {
		params = r.URL.Query().Encode()
	} else {
		params = string(r.GetBody())
	}
	if len(params) > 2000 {
		params = params[:2000] + "..."
	}

	module := extractModuleFromUrl(url)
	action := extractActionFromMethod(method, url)

	status := 1
	errorMsg := ""

	if r.Response.Status > 0 && r.Response.Status != 200 {
		status = 0
	}

	logInput := service.OperationLogInput{
		UserId:    userId,
		Username:  username,
		Module:    module,
		Action:    action,
		Method:    method,
		Url:       url,
		Params:    params,
		Ip:        ip,
		UserAgent: userAgent,
		Status:    status,
		ErrorMsg:  errorMsg,
		Duration:  duration,
	}

	go func() {
		svc := service.NewOperationLogService()
		svc.Record(r.Context(), logInput)
	}()
}

func extractModuleFromUrl(url string) string {
	parts := strings.Split(strings.TrimPrefix(url, "/api/"), "/")
	if len(parts) > 0 {
		module := parts[0]
		switch {
		case strings.HasPrefix(module, "system/"):
			return strings.Replace(module, "system/", "系统管理 - ", 1)
		case module == "user" || module == "user-list" || module == "user-save" || module == "user-delete":
			return "用户管理"
		case module == "role" || module == "assign-roles" || module == "assign-menus" || module == "data-scope":
			return "角色管理"
		case module == "company":
			return "公司管理"
		case module == "permission":
			return "权限管理"
		case module == "dept":
			return "部门管理"
		case module == "menu":
			return "菜单管理"
		case module == "profile" || module == "change-password" || module == "update-profile" || module == "upload-avatar":
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
