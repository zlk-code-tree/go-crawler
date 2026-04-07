package auth

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/crypto/bcrypt"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"
	"go-crawler/utility"
)

// PublicController 公开认证接口（无需 JWT）
type PublicController struct {
	authSvc *service.AuthService
}

func NewPublic() *PublicController {
	return &PublicController{
		authSvc: service.NewAuthService(),
	}
}

// Login 用户登录
func (c *PublicController) Login(ctx context.Context, req *v1.AuthLoginReq) (res *v1.AuthLoginRes, err error) {
	r := g.RequestFromCtx(ctx)

	user, e := c.authSvc.GetUserByUsername(ctx, req.Username)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "系统错误", "data": nil})
		return
	}
	if user == nil {
		c.authSvc.RecordLoginLog(ctx, req.Username, 0, r.GetClientIp(), 0, "用户不存在")
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "用户名或密码错误", "data": nil})
		return
	}

	if e = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); e != nil {
		c.authSvc.RecordLoginLog(ctx, req.Username, user.Id, r.GetClientIp(), 0, "密码错误")
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "用户名或密码错误", "data": nil})
		return
	}

	secret := getJwtSecret(ctx)
	token, e := utility.GenerateToken(user.Id, user.Username, secret, 86400)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "生成token失败", "data": nil})
		return
	}

	_ = c.authSvc.UpdateLastLogin(ctx, user.Id, r.GetClientIp())
	c.authSvc.RecordLoginLog(ctx, req.Username, user.Id, r.GetClientIp(), 1, "登录成功")

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "ok",
		"data": g.Map{
			"accessToken": token,
		},
	})
	return
}

// Logout 退出登录
func (c *PublicController) Logout(ctx context.Context, req *v1.AuthLogoutReq) (res *v1.AuthLogoutRes, err error) {
	r := g.RequestFromCtx(ctx)
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}
