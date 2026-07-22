package model

import "time"

// Shop 账号信息（支持抖音/快手/视频号等多平台）
type Shop struct {
	Id                  int       `json:"id"`
	Platform            string    `json:"platform"`            // 平台：douyin/kuaishou/shipinhao
	Uid                 string    `json:"uid"`                 // 达人ID
	AccountName         string    `json:"accountName"`         // 账号名称
	PlatformNickname    string    `json:"platformNickname"`    // 平台昵称（抖音号等）
	Operator            string    `json:"operator"`            // 运营者
	CompanyId           int64     `json:"companyId"`           // 所属公司ID（关联sys_company）
	DepartmentId        int64     `json:"departmentId"`        // 所属部门ID（关联sys_department）
	GroupType           string    `json:"groupType"`           // 组类型：invest(投放组)/shop(账号组)
	MemberName          string    `json:"memberName"`          // 成员名称
	TalentImage         string    `json:"talentImage"`         // 达人形象（视频号）
	Extra               string    `json:"extra"`               // JSON扩展字段
	BindStatus          int       `json:"bindStatus"`          // 绑定状态：0=未绑定，1=被绑定
	BlankOrderEnabled   int       `json:"blankOrderEnabled"`   // 是否参与空白单归属：0=否 1=是
	BlankOrderOwnerName string    `json:"blankOrderOwnerName"` // 空白单归属人；为空默认使用运营者
	Status              int       `json:"status"`              // 状态：0-禁用 1-启用
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// ShopListReq 账号列表请求
type ShopListReq struct {
	Page     int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	Keyword  string `json:"keyword"`
	Platform string `json:"platform"` // 平台筛选
}

// ShopSaveReq 保存账号请求
type ShopSaveReq struct {
	Id                  int     `json:"id"`
	Platform            string  `json:"platform" v:"required#平台不能为空"`
	Uid                 string  `json:"uid"`
	AccountName         string  `json:"accountName" v:"required#账号昵称不能为空"`
	PlatformNickname    string  `json:"platformNickname"`
	Operator            string  `json:"operator" v:"required#运营者不能为空"`
	BlankOrderEnabled   *int    `json:"blankOrderEnabled"`
	BlankOrderOwnerName *string `json:"blankOrderOwnerName"`
}

// AccountQianchuanMapping 内部账号与千川广告主映射
type AccountQianchuanMapping struct {
	Id                      int64     `json:"id"`
	AccountId               int       `json:"accountId"`
	CompanyId               int64     `json:"companyId"`
	QianchuanAdvertiserId   int64     `json:"qianchuanAdvertiserId"`
	QianchuanAdvertiserName string    `json:"qianchuanAdvertiserName"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

// QianchuanMappingSaveReq 保存千川映射请求
type QianchuanMappingSaveReq struct {
	AccountId               int    `json:"accountId" v:"required#账号ID不能为空"`
	CompanyId               int64  `json:"companyId" v:"required#公司ID不能为空"`
	QianchuanAdvertiserId   int64  `json:"qianchuanAdvertiserId" v:"required#千川广告主ID不能为空"`
	QianchuanAdvertiserName string `json:"qianchuanAdvertiserName"`
}

// QianchuanMappingListReq 千川映射列表请求
type QianchuanMappingListReq struct {
	Page      int   `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize  int   `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	AccountId int   `json:"accountId"` // 按账号筛选
	CompanyId int64 `json:"companyId"` // 按公司筛选
}

// ShopExcelRow Excel导入行数据
type ShopExcelRow struct {
	Platform         string `json:"platform"`
	Uid              string `json:"uid"`
	AccountName      string `json:"accountName"`
	PlatformNickname string `json:"platformNickname"`
	Operator         string `json:"operator"`
}
