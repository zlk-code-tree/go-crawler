package system

import (
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ProfileController 个人设置相关接口
// 路径前缀：/system/profile

type ProfileController struct{}

func NewProfileController() *ProfileController {
	return &ProfileController{}
}

// GetProfile 获取个人资料详情 GET /system/profile
func (c *ProfileController) GetProfile(r *ghttp.Request) {
	// 从 JWT token 中获取当前用户ID
	userId := r.GetCtxVar("userId").Int64()
	if userId <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 401, "data": nil, "msg": "未登录或登录已过期"})
		return
	}

	// 调用服务层方法获取个人资料
	profileService := service.NewProfileService()
	user, err := profileService.GetProfile(r.Context(), userId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取个人资料失败"})
		return
	}

	// 返回用户信息（排除敏感字段）
	profileData := g.Map{
		"id":       user.Id,
		"username": user.Username,
		"realName": user.RealName,
		"nickname": user.Nickname,
		"email":    user.Email,
		"phone":    user.Phone,
		"avatar":   user.Avatar,
		"gender":   user.Gender,
		"deptId":   user.DeptId,
		"position": user.Position,
		"remark":   user.Remark,
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": profileData, "msg": "ok"})
}

// UpdateProfile 更新个人资料 PUT /system/profile/update
func (c *ProfileController) UpdateProfile(r *ghttp.Request) {
	// 从 JWT token 中获取当前用户ID
	userId := r.GetCtxVar("userId").Int64()
	if userId <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 401, "data": nil, "msg": "未登录或登录已过期"})
		return
	}

	// 解析请求参数
	var req struct {
		Nickname string `json:"nickname"`
		RealName string `json:"realName"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Avatar   string `json:"avatar"`
		Gender   int    `json:"gender"`
		Position string `json:"position"`
		Remark   string `json:"remark"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	// 构建用户模型

	// 调用服务层方法更新个人资料
	profileService := service.NewProfileService()
	err := profileService.UpdateProfile(r.Context(), userId, g.Map{
		"real_name": req.RealName,
		"nickname":  req.Nickname,
		"email":     req.Email,
		"phone":     req.Phone,
		"avatar":    req.Avatar,
		"gender":    req.Gender,
		"position":  req.Position,
		"remark":    req.Remark,
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "更新个人资料失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "更新成功"})
}

// ChangePassword 修改密码 POST /system/profile/change-password
func (c *ProfileController) ChangePassword(r *ghttp.Request) {
	// 从 JWT token 中获取当前用户ID
	userId := r.GetCtxVar("userId").Int64()
	if userId <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 401, "data": nil, "msg": "未登录或登录已过期"})
		return
	}

	// 解析请求参数
	var req struct {
		OldPassword string `json:"oldPassword" v:"required#旧密码不能为空"`
		NewPassword string `json:"newPassword" v:"required|min-length:6#新密码不能为空|密码长度不能少于6位"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	// 调用服务层方法修改密码
	profileService := service.NewProfileService()
	err := profileService.ChangePassword(r.Context(), userId, req.OldPassword, req.NewPassword)
	if err != nil {
		// 检查是否是密码验证错误
		if err == service.ErrInvalidOldPassword {
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "旧密码错误"})
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "密码修改失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "密码修改成功"})
}

// UploadAvatar 上传头像 POST /system/profile/upload-avatar
func (c *ProfileController) UploadAvatar(r *ghttp.Request) {
	// 从 JWT token 中获取当前用户ID
	userId := r.GetCtxVar("userId").Int64()
	if userId <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 401, "data": nil, "msg": "未登录或登录已过期"})
		return
	}

	// 获取上传的文件
	file := r.GetUploadFile("avatar")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请选择头像文件"})
		return
	}

	// 检查文件类型
	fileType := file.Header.Get("Content-Type")
	if fileType != "image/jpeg" && fileType != "image/png" && fileType != "image/gif" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "只支持 jpg、png、gif 格式的图片"})
		return
	}

	// 检查文件大小（限制为 2MB）
	if file.Size > 2*1024*1024 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "头像文件大小不能超过 2MB"})
		return
	}

	// 保存文件到指定目录
	fileName, err := file.Save("./uploads/avatars/")
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "头像上传失败"})
		return
	}

	// 构建头像URL
	avatarUrl := "/uploads/avatars/" + fileName

	// 更新用户头像
	updateData := g.Map{
		"avatar": avatarUrl,
	}
	_, err = g.DB().Model("sys_user").Where("id", userId).Data(updateData).Update()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "头像更新失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"avatar": avatarUrl}, "msg": "头像上传成功"})
}
