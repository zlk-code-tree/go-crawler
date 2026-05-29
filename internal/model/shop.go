package model

import "time"

// Shop 店铺/账号信息（支持抖音/快手/视频号等多平台）
type Shop struct {
	Id                int       `json:"id"`
	Platform          string    `json:"platform"`          // 平台：douyin/kuaishou/shipinhao
	Uid string    `json:"uid"` // 达人ID
	AccountName       string    `json:"accountName"`       // 账号名称
	PlatformNickname  string    `json:"platformNickname"`  // 平台昵称（抖音号等）
	Operator          string    `json:"operator"`          // 运营者
	CompanyId         int64     `json:"companyId"`         // 所属公司ID（关联sys_company）
	DepartmentId      int64     `json:"departmentId"`      // 所属部门ID（关联sys_department）
	GroupType         string    `json:"groupType"`         // 组类型：invest(投放组)/shop(店铺组)
	MemberName        string    `json:"memberName"`        // 成员名称
	TalentImage       string    `json:"talentImage"`       // 达人形象（视频号）
	Extra             string    `json:"extra"`             // JSON扩展字段
	BindStatus        int       `json:"bindStatus"`        // 绑定状态：0=未绑定，1=被绑定
	Status            int       `json:"status"`            // 状态：0-禁用 1-启用
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// ShopListReq 店铺列表请求
type ShopListReq struct {
	Page     int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	Keyword  string `json:"keyword"`
	Platform string `json:"platform"` // 平台筛选
}

// ShopSaveReq 保存店铺请求
type ShopSaveReq struct {
	Id                int    `json:"id"`
	Platform          string `json:"platform" v:"required#平台不能为空"`
	Uid string `json:"uid" v:"required#达人ID不能为空"`
	AccountName       string `json:"accountName" v:"required#账号名称不能为空"`
	PlatformNickname  string `json:"platformNickname"`
	Operator          string `json:"operator" v:"required#运营者不能为空"`
}

// ShopExcelRow Excel导入行数据
type ShopExcelRow struct {
	Platform          string `json:"platform"`
	Uid string `json:"uid"`
	AccountName       string `json:"accountName"`
	PlatformNickname  string `json:"platformNickname"`
	Operator          string `json:"operator"`
}
