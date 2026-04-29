package auth

import (
	"math/rand"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"

	"go-crawler/internal/service"
	"go-crawler/utility"
)

// FeishuAuthController 飞书认证控制器
type FeishuAuthController struct {
	feishuAuthSvc *service.FeishuAuthService
	authSvc       *service.AuthService
}

func NewFeishuAuthController() *FeishuAuthController {
	return &FeishuAuthController{
		feishuAuthSvc: service.NewFeishuAuthService(),
		authSvc:       service.NewAuthService(),
	}
}

// FeishuLogin 飞书登录
func (c *FeishuAuthController) FeishuLogin(req *ghttp.Request) {
	// 生成state参数，用于防止CSRF攻击
	rand.Seed(time.Now().UnixNano())
	state := gconv.String(rand.Intn(1000000))

	// 获取飞书授权链接
	authUrl := c.feishuAuthSvc.GetAuthorizationUrl(req.Context(), state)

	// 重定向到飞书授权页面
	req.Response.RedirectTo(authUrl)
}

// FeishuCallback 飞书回调
func (c *FeishuAuthController) FeishuCallback(req *ghttp.Request) {
	// 获取code和state参数
	code := req.GetQuery("code").String()
	_ = req.GetQuery("state").String() // 暂时不使用state参数

	if code == "" {
		req.Response.WriteJsonExit(g.Map{
			"code": 400,
			"msg":  "缺少code参数",
		})
		return
	}

	ctx := req.Context()

	// 通过code获取用户access_token
	tokenResp, err := c.feishuAuthSvc.GetUserAccessToken(ctx, code)
	if err != nil {
		req.Response.WriteJsonExit(g.Map{
			"code": 500,
			"msg":  "获取token失败: " + err.Error(),
		})
		return
	}

	// 获取用户信息
	userInfo, err := c.feishuAuthSvc.GetUserInfo(ctx, tokenResp.UserAccessToken)
	if err != nil {
		req.Response.WriteJsonExit(g.Map{
			"code": 500,
			"msg":  "获取用户信息失败: " + err.Error(),
		})
		return
	}

	// 获取或创建系统用户
	sysUser, err := c.feishuAuthSvc.GetOrCreateUser(ctx, userInfo)
	if err != nil {
		req.Response.WriteJsonExit(g.Map{
			"code": 500,
			"msg":  "创建用户失败: " + err.Error(),
		})
		return
	}

	// 生成JWT token
	secret := getJwtSecret(ctx)
	token, err := utility.GenerateToken(sysUser.Id, sysUser.Username, sysUser.CompanyId, secret, 86400)
	if err != nil {
		req.Response.WriteJsonExit(g.Map{
			"code": 500,
			"msg":  "生成token失败: " + err.Error(),
		})
		return
	}

	// 记录登录日志
	c.authSvc.RecordLoginLog(ctx, sysUser.Username, sysUser.Id, req.GetClientIp(), 1, "飞书登录成功")

	// 更新最后登录信息
	c.authSvc.UpdateLastLogin(ctx, sysUser.Id, req.GetClientIp())

	// 返回登录成功信息
	req.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "登录成功",
		"data": g.Map{
			"accessToken": token,
			"userId":      sysUser.Id,
			"username":    sysUser.Username,
			"realName":    sysUser.RealName,
			"avatar":      sysUser.Avatar,
		},
	})
}
