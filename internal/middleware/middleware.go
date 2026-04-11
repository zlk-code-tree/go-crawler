package middleware

import (
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

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
