package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/model"
)

// FeishuAuthService 飞书认证服务
type FeishuAuthService struct{}

func NewFeishuAuthService() *FeishuAuthService {
	return &FeishuAuthService{}
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	AppId       string
	AppSecret   string
	RedirectUri string
}

// GetFeishuConfig 获取飞书配置
func (s *FeishuAuthService) GetFeishuConfig(ctx context.Context) FeishuConfig {
	appId, _ := g.Cfg().Get(ctx, "feishu.appId")
	appSecret, _ := g.Cfg().Get(ctx, "feishu.appSecret")
	redirectUri, _ := g.Cfg().Get(ctx, "feishu.redirectUri")
	return FeishuConfig{
		AppId:       appId.String(),
		AppSecret:   appSecret.String(),
		RedirectUri: redirectUri.String(),
	}
}

// GetAuthorizationUrl 获取飞书授权链接
func (s *FeishuAuthService) GetAuthorizationUrl(ctx context.Context, state string) string {
	config := s.GetFeishuConfig(ctx)
	return fmt.Sprintf(
		"https://open.feishu.cn/open-apis/authen/v1/index?app_id=%s&redirect_uri=%s&state=%s",
		config.AppId, config.RedirectUri, state,
	)
}

// FeishuTokenResponse 飞书token响应
type FeishuTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	UserAccessToken   string `json:"user_access_token"`
	Expire            int    `json:"expire"`
	RefreshToken      string `json:"refresh_token"`
}

// FeishuUserInfo 飞书用户信息
type FeishuUserInfo struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AccessToken  string `json:"access_token"`
		Avatar       string `json:"avatar"`
		AvatarBig    string `json:"avatar_big"`
		AvatarThumb  string `json:"avatar_thumb"`
		AvatarMiddle string `json:"avatar_middle"`
		EnName       string `json:"en_name"`
		Name         string `json:"name"`
		OpenId       string `json:"open_id"`
		UnionId      string `json:"union_id"`
		Email        string `json:"email"`
		UserId       string `json:"user_id"`
		TenantKey    string `json:"tenant_key"`
		Mobile       string `json:"mobile"`
	} `json:"data"`
}

// GetUserAccessToken 通过code获取用户access_token
func (s *FeishuAuthService) GetUserAccessToken(ctx context.Context, code string) (*FeishuTokenResponse, error) {
	config := s.GetFeishuConfig(ctx)

	url := "https://open.feishu.cn/open-apis/authen/v1/access_token"
	data := map[string]string{
		"app_id":     config.AppId,
		"app_secret": config.AppSecret,
		"code":       code,
	}

	client := &http.Client{}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(string(jsonData)))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp FeishuTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.Code != 0 {
		return nil, fmt.Errorf("获取token失败: %s", tokenResp.Msg)
	}

	return &tokenResp, nil
}

// GetUserInfo 获取用户信息
func (s *FeishuAuthService) GetUserInfo(ctx context.Context, accessToken string) (*FeishuUserInfo, error) {
	url := "https://open.feishu.cn/open-apis/authen/v1/user_info"

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo FeishuUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	if userInfo.Code != 0 {
		return nil, fmt.Errorf("获取用户信息失败: %s", userInfo.Msg)
	}

	return &userInfo, nil
}

// GetOrCreateUser 根据飞书用户信息获取或创建系统用户
func (s *FeishuAuthService) GetOrCreateUser(ctx context.Context, userInfo *FeishuUserInfo) (*model.SysUser, error) {
	// 先尝试通过飞书open_id查询用户
	var user model.SysUser
	err := g.DB().Model("sys_user").
		Where("feishu_open_id", userInfo.Data.OpenId).
		Scan(&user)

	if err != nil {
		return nil, err
	}

	// 如果用户不存在，创建新用户
	if user.Id == 0 {
		username := userInfo.Data.Email
		if username == "" {
			username = userInfo.Data.Mobile
		}
		if username == "" {
			username = userInfo.Data.OpenId
		}

		result, err := g.DB().Model("sys_user").Insert(g.Map{
			"username":       username,
			"real_name":      userInfo.Data.Name,
			"avatar":         userInfo.Data.Avatar,
			"feishu_open_id": userInfo.Data.OpenId,
			"feishu_user_id": userInfo.Data.UserId,
			"status":         1,
			"password":       "", // 飞书登录用户不需要密码
		})

		if err != nil {
			return nil, err
		}

		lastId, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		user.Id = lastId
		user.Username = username
		user.RealName = userInfo.Data.Name
		user.Avatar = userInfo.Data.Avatar
		user.FeishuOpenId = userInfo.Data.OpenId
		user.FeishuUserId = userInfo.Data.UserId
		user.Status = 1
	}

	return &user, nil
}
