package model

import "time"

// Shop 店铺信息
type Shop struct {
	Id               int       `json:"id"`
	Uid              string    `json:"uid"`
	DouyinName       string    `json:"douyinName"`
	DouyinId         string    `json:"douyinId"`
	OperationLeader  string    `json:"operationLeader"`
	CompanyId        int64     `json:"companyId"`        // 所属公司ID（关联sys_company）
	DepartmentId     int64     `json:"departmentId"`     // 所属部门ID（关联sys_department）
	GroupType        string    `json:"groupType"`        // 组类型：invest(投放组)/shop(店铺组)
	MemberName       string    `json:"memberName"`       // 成员名称
	AccountOwnership string    `json:"accountOwnership"`
	ShopPlatform     string    `json:"shopPlatform"`
	BindStatus       int       `json:"bindStatus"`       // 绑定状态：0=未绑定，1=被绑定
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// ShopListReq 店铺列表请求
type ShopListReq struct {
	Page     int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	Keyword  string `json:"keyword"`
}

// ShopSaveReq 保存店铺请求
type ShopSaveReq struct {
	Id               int    `json:"id"`
	Uid              string `json:"uid" v:"required#店铺唯一标识不能为空"`
	DouyinName       string `json:"douyinName" v:"required#抖音号名称不能为空"`
	DouyinId         string `json:"douyinId"`
	OperationLeader  string `json:"operationLeader" v:"required#运营组长不能为空"`
	AccountOwnership string `json:"accountOwnership"`
	ShopPlatform     string `json:"shopPlatform" v:"required#店铺平台不能为空"`
}

// ShopExcelRow Excel导入行数据
type ShopExcelRow struct {
	Uid              string `json:"uid"`
	DouyinName       string `json:"douyinName"`
	DouyinId         string `json:"douyinId"`
	OperationLeader  string `json:"operationLeader"`
	AccountOwnership string `json:"accountOwnership"`
	ShopPlatform     string `json:"shopPlatform"`
}
