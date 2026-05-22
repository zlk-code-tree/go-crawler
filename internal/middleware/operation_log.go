package middleware

import (
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	"go-crawler/internal/service"
)

// excludePaths 不需要记录日志的路径（精确匹配）
var excludePaths = map[string]bool{
	"/api/health":            true,
	"/api/ping":              true,
	"/api/system/dict":       true,
	"/api/system/dict/list":  true,
	"/api/org/tree":          true,
	"/api/org/list":          true,
	"/api/org/children":      true,
	"/api/org/ancestors":     true,
	"/api/org/detail":        true,
	"/api/org/shop-path":     true,
}

// excludePathPrefixes 不需要记录日志的路径前缀
var excludePathPrefixes = []string{
	"/api/system/dict",
	"/api/health",
	"/static/",
	"/assets/",
	"/favicon.ico",
}

func OperationLog(r *ghttp.Request) {
	startTime := time.Now()

	userId := r.GetCtxVar("userId").Int64()
	username := r.GetCtxVar("username").String()
	permissionCode := r.GetCtxVar("permissionCode").String()

	r.Middleware.Next()

	duration := int(time.Since(startTime).Milliseconds())

	method := r.Method
	url := r.URL.Path

	// 优化：跳过GET请求（只记录写操作：POST/PUT/DELETE）
	if method == "GET" {
		return
	}

	// 优化：跳过排除的路径
	if shouldExcludePath(url) {
		return
	}

	ip := r.GetClientIp()
	location := service.NewIpLocationService().GetLocationByIP(ip)
	userAgent := r.UserAgent()
	params := string(r.GetBody())
	if len(params) > 2000 {
		params = params[:2000] + "..."
	}

	module := extractModuleFromUrl(url)
	action := extractActionFromMethod(method, url)

	status := 1
	errorMsg := ""

	if r.Response.Status > 0 && r.Response.Status != 200 {
		status = 0
		// 尝试获取错误信息
		buffer := r.Response.Buffer()
		if buffer != nil {
			errorMsg = string(buffer)
			if len(errorMsg) > 500 {
				errorMsg = errorMsg[:500] + "..."
			}
		}
	}

	logInput := service.OperationLogInput{
		UserId:         userId,
		Username:       username,
		Module:         module,
		Action:         action,
		Method:         method,
		Url:            url,
		Params:         params,
		Ip:             ip,
		Location:       location,
		UserAgent:      userAgent,
		Status:         status,
		ErrorMsg:       errorMsg,
		Duration:       duration,
		PermissionCode: permissionCode,
	}

	go func() {
		svc := service.NewOperationLogService()
		svc.Record(r.Context(), logInput)
	}()
}

// shouldExcludePath 判断路径是否应该排除
func shouldExcludePath(url string) bool {
	// 精确匹配
	if excludePaths[url] {
		return true
	}
	// 前缀匹配
	for _, prefix := range excludePathPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

func extractModuleFromUrl(url string) string {
	parts := strings.Split(strings.TrimPrefix(url, "/api/"), "/")
	if len(parts) == 0 {
		return "系统管理"
	}

	// 解析模块层级
	var firstLevel, secondLevel string
	var operation string
	
	// 处理外链菜单访问 - /system/external/{code}
	if len(parts) >= 2 && parts[0] == "system" && parts[1] == "external" {
		return "系统管理 - 外链菜单 - 访问"
	}
	
	// 处理 /api/system/user/list 格式
	if strings.HasPrefix(parts[0], "system/") {
		firstLevel = "系统管理"
		if len(parts) >= 2 {
			secondLevel = parseSecondLevel(parts[0])
			if len(parts) >= 3 {
				operation = parseOperation(parts[1], parts[2])
			}
		}
	} else {
		// 处理 /api/user/list 格式（向后兼容）
		firstLevel = parseFirstLevel(parts[0])
		if len(parts) >= 2 {
			secondLevel = parseSecondLevel(parts[0])
			if len(parts) >= 3 {
				operation = parseOperation(parts[1], parts[2])
			}
		}
	}

	// 构建模块名称
	if operation != "" {
		return firstLevel + " - " + secondLevel + " - " + operation
	} else if secondLevel != "" {
		return firstLevel + " - " + secondLevel
	} else {
		return firstLevel
	}
}

// 解析一级菜单
func parseFirstLevel(module string) string {
	switch module {
	case "system":
		return "系统管理"
	case "douyin":
		return "抖音盯盘"
	case "product-pool":
		return "商品池"
	case "video-pool":
		return "视频池"
	case "user", "user-list", "user-save", "user-delete":
		return "用户管理"
	case "role", "assign-roles", "assign-menus", "data-scope":
		return "角色管理"
	case "company":
		return "公司管理"
	case "permission":
		return "权限管理"
	case "dept":
		return "部门管理"
	case "menu":
		return "菜单管理"
	case "profile", "change-password", "update-profile", "upload-avatar":
		return "个人设置"
	case "external":
		return "外链菜单"
	default:
		// 智能推断
		if strings.Contains(module, "user") {
			return "用户管理"
		} else if strings.Contains(module, "role") {
			return "角色管理"
		} else if strings.Contains(module, "company") {
			return "公司管理"
		} else if strings.Contains(module, "permission") {
			return "权限管理"
		} else if strings.Contains(module, "dept") {
			return "部门管理"
		} else if strings.Contains(module, "menu") {
			return "菜单管理"
		} else if strings.Contains(module, "profile") {
			return "个人设置"
		} else if strings.Contains(module, "log") {
			return "系统管理"
		} else if strings.Contains(module, "douyin") {
			return "抖音盯盘"
		} else if strings.Contains(module, "product") {
			return "商品池"
		} else if strings.Contains(module, "video") {
			return "视频池"
		} else if strings.Contains(module, "external") {
			return "外链菜单"
		}
		return "系统管理"
	}
}

// 解析二级菜单
func parseSecondLevel(module string) string {
	// 处理 system/user 格式
	if strings.HasPrefix(module, "system/") {
		subModule := strings.TrimPrefix(module, "system/")
		switch subModule {
		case "user", "user-list", "user-save", "user-delete":
			return "用户管理"
		case "role", "assign-roles", "assign-menus", "data-scope":
			return "角色管理"
		case "company":
			return "公司管理"
		case "permission":
			return "权限管理"
		case "dept":
			return "部门管理"
		case "menu":
			return "菜单管理"
		case "profile", "change-password", "update-profile", "upload-avatar":
			return "个人设置"
		case "log-operation":
			return "操作日志"
		case "log-login":
			return "登录日志"
		case "external":
			return "外链菜单"
		default:
			return "系统管理"
		}
	}
	
	// 处理 douyin/product-pool 格式
	if strings.HasPrefix(module, "douyin/") {
		subModule := strings.TrimPrefix(module, "douyin/")
		switch subModule {
		case "product-pool":
			return "商品池"
		case "video-pool":
			return "视频池"
		default:
			return "抖音盯盘"
		}
	}
	
	// 处理直接模块名
	return parseFirstLevel(module)
}

// 解析操作类型（三级）
func parseOperation(controller, action string) string {
	// 特殊按钮操作优先判断
	fullPath := controller
	if action != "" {
		fullPath = controller + "/" + action
	}
	
	switch fullPath {
	case "assign-roles":
		return "分配角色"
	case "assign-menus":
		return "分配权限"
	case "change-password":
		return "修改密码"
	case "upload-avatar":
		return "上传头像"
	case "data-scope":
		return "数据权限"
	}
	
	// 根据HTTP方法和常见操作名称判断
	upperAction := strings.ToUpper(action)
	
	// 常见的CRUD操作映射
	if action == "list" {
		return "查询"
	} else if action == "detail" || action == "{id}" {
		return "详情"
	} else if action == "save" {
		return "新增/编辑"
	} else if strings.Contains(action, "delete") {
		return "删除"
	} else if action == "create" {
		return "新增"
	} else if action == "update" {
		return "更新"
	}
	
	// 根据HTTP方法判断
	switch upperAction {
	case "GET":
		return "查看"
	case "POST":
		return "提交"
	case "PUT":
		return "更新"
	case "DELETE":
		return "删除"
	default:
		return action
	}
}

func extractActionFromMethod(method, url string) string {
	upperMethod := strings.ToUpper(method)
	switch upperMethod {
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
