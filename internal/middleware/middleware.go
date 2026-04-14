package middleware

import (
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"

	"go-crawler/internal/service"
	"go-crawler/utility"
)

// CORS 跨域中间件
func CORS(r *ghttp.Request) {
	r.Response.CORSDefault()
	r.Middleware.Next()
}

// JwtAuth JWT 认证中间件
func JwtAuth(r *ghttp.Request) {
	authHeader := r.GetHeader("Authorization")
	if authHeader == "" {
		r.Response.WriteStatus(401, `{"code":401,"message":"未提供认证信息","data":null}`)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == authHeader {
		r.Response.WriteStatus(401, `{"code":401,"message":"认证格式错误","data":null}`)
		return
	}

	secret, _ := g.Cfg().Get(r.Context(), "jwt.secret", "go-crawler-jwt-secret-2024")
	claims, err := utility.ParseToken(tokenStr, secret.String())
	if err != nil {
		r.Response.WriteStatus(401, `{"code":401,"message":"认证已过期，请重新登录","data":null}`)
		return
	}

	// 将用户信息存入 context
	r.SetCtxVar("userId", claims.UserId)
	r.SetCtxVar("username", claims.Username)
	r.SetCtxVar("companyId", claims.CompanyId)

	r.Middleware.Next()
}

// PermissionAuth 权限验证中间件
func PermissionAuth(r *ghttp.Request) {
	// 获取用户ID和用户名
	userId := r.GetCtxVar("userId").Int64()
	username := r.GetCtxVar("username").String()
	if userId <= 0 {
		r.Response.WriteStatus(401, `{"code":401,"message":"未登录或登录已过期","data":null}`)
		return
	}

	// 检查用户是否在白名单中
	if isUserInWhitelist(username) {
		// 白名单用户跳过权限验证
		r.Middleware.Next()
		return
	}

	// 获取请求信息
	method := r.Method
	url := r.URL.Path

	// 映射URL到权限编码
	permissionCode := mapUrlToPermission(method, url)
	if permissionCode == "" {
		// 无需权限验证的接口
		r.Middleware.Next()
		return
	}

	// 检查用户是否有该权限
	permissionService := service.NewPermissionService()
	hasPermission, err := permissionService.CheckPermission(r.Context(), userId, permissionCode)
	if err != nil {
		r.Response.WriteStatus(500, `{"code":500,"message":"权限检查失败","data":null}`)
		return
	}

	if !hasPermission {
		r.Response.WriteStatus(403, `{"code":403,"message":"权限不足","data":null}`)
		return
	}

	// 将权限信息存入context，用于操作日志
	r.SetCtxVar("permissionCode", permissionCode)

	r.Middleware.Next()
}

// isUserInWhitelist 检查用户是否在权限白名单中
func isUserInWhitelist(username string) bool {
	// 从配置文件读取白名单
	whitelist := getPermissionWhitelist()

	for _, user := range whitelist {
		if user == username {
			return true
		}
	}

	return false
}

// getPermissionWhitelist 从配置文件获取权限白名单
func getPermissionWhitelist() []string {
	// 默认白名单
	defaultWhitelist := []string{
		"admin", // 超级用户
	}

	// 尝试从配置文件读取白名单
	cfg, err := g.Cfg().Get(gctx.GetInitCtx(), "permission.whitelist")
	if err == nil {
		whitelist := cfg.Strings()
		if len(whitelist) > 0 {
			return whitelist
		}
	}

	// 如果配置文件中没有设置或读取失败，返回默认白名单
	return defaultWhitelist
}

// mapUrlToPermission 映射URL到权限编码
// 路由由 GoFrame Bind 自动注册，方法名转 kebab-case，前缀为 /api
func mapUrlToPermission(method, url string) string {
	// 去掉 /api 前缀，统一小写
	url = strings.ToLower(strings.TrimPrefix(url, "/api"))

	switch {
	// ── 用户管理 ──────────────────────────────────────────────
	case url == "/system/user/list" && method == "GET":
		return "user:list"
	case url == "/system/user/save" && method == "POST":
		return "user:add"
	case url == "/system/user/save" && method == "PUT":
		return "user:edit"
	case url == "/system/user/delete" && method == "POST":
		return "user:delete"
	case url == "/system/user/assign-roles" && method == "POST":
		return "user:assign-roles"

	// ── 角色管理 ──────────────────────────────────────────────
	case url == "/system/role/list" && method == "GET":
		return "role:list"
	case url == "/system/role/save" && method == "POST":
		return "role:add"
	case url == "/system/role/detail" && method == "POST":
		return "role:edit"
	case url == "/system/role/delete" && method == "POST":
		return "role:delete"
	case url == "/system/role/assign-menus" && method == "POST":
		return "role:permission"
	case url == "/system/role/assign-data-scope" && method == "POST":
		return "role:permission"

	// ── 部门管理 ──────────────────────────────────────────────
	case url == "/system/dept/list" && method == "GET":
		return "dept:list"
	case url == "/system/dept/save" && method == "POST":
		return "dept:add"
	case url == "/system/dept/update" && method == "PUT":
		return "dept:edit"
	case url == "/system/dept/delete" && method == "DELETE":
		return "dept:delete"
	case url == "/system/dept/tree" && method == "GET":
		return "dept:tree"

	// ── 公司管理 ──────────────────────────────────────────────
	case url == "/system/company/list" && method == "GET":
		return "company:list"
	case url == "/system/company/save" && method == "POST":
		return "company:add"
	case url == "/system/company/update" && method == "PUT":
		return "company:edit"
	case url == "/system/company/delete" && method == "DELETE":
		return "company:delete"
	case url == "/system/company/detail" && method == "GET":
		return "company:detail"

	// ── 权限管理 ──────────────────────────────────────────────
	case url == "/system/permission/list" && method == "GET":
		return "permission:list"
	case url == "/system/permission/create" && method == "POST":
		return "permission:add"
	case url == "/system/permission/update" && method == "PUT":
		return "permission:edit"
	case url == "/system/permission/delete" && method == "DELETE":
		return "permission:delete"

	// ── 菜单管理 ──────────────────────────────────────────────
	case url == "/system/menu/list" && method == "GET":
		return "menu:list"
	case url == "/system/menu/save" && method == "POST":
		return "menu:add"
	case url == "/system/menu/update" && method == "PUT":
		return "menu:edit"
	case url == "/system/menu/delete" && method == "DELETE":
		return "menu:delete"

	// ── 操作日志 / 登录日志（只读，需要 list 权限）───────────
	case url == "/system/operation-log-list" && method == "GET":
		return "log:operation"
	case url == "/system/login-log-list" && method == "GET":
		return "log:login"

	// ── 任务管理 ──────────────────────────────────────────────
	case url == "/task/list" && method == "GET":
		return "task:list"
	case url == "/task/create" && method == "POST":
		return "task:create"
	case strings.HasPrefix(url, "/task/delete/") && method == "DELETE":
		return "task:delete"
	case strings.HasPrefix(url, "/task/stop/") && method == "POST":
		return "task:stop"

	// ── 个人中心（自身操作，无需额外权限校验）────────────────
	case url == "/system/profile/change-password" && method == "POST":
		return ""
	case url == "/system/profile/update" && method == "POST":
		return ""
	case url == "/system/profile/upload-avatar" && method == "POST":
		return ""

	default:
		return ""
	}
}
