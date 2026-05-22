package v1

import (
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// --- Org Structure APIs ---

// OrgTreeReq 获取组织架构树请求
type OrgTreeReq struct {
	g.Meta `path:"/org/tree" method:"get" tags:"OrgStructure" summary:"获取组织架构树"`
}

// OrgTreeRes 获取组织架构树响应
type OrgTreeRes struct {
	g.Meta `mime:"application/json"`
	Tree   []model.OrgTreeNode `json:"tree" dc:"组织架构树"`
}

// OrgListReq 获取组织节点列表请求
type OrgListReq struct {
	g.Meta   `path:"/org/list" method:"get" tags:"OrgStructure" summary:"获取组织节点列表"`
	ParentId int64  `json:"parentId" in:"query" d:"0" dc:"父节点ID，0表示根节点"`
	Type     string `json:"type" in:"query" d:"" dc:"节点类型筛选"`
	Keyword  string `json:"keyword" in:"query" d:"" dc:"搜索关键词"`
}

// OrgListRes 获取组织节点列表响应
type OrgListRes struct {
	g.Meta `mime:"application/json"`
	List   []model.OrgStructure `json:"list" dc:"组织节点列表"`
}

// OrgDetailReq 获取组织节点详情请求
type OrgDetailReq struct {
	g.Meta `path:"/org/detail" method:"get" tags:"OrgStructure" summary:"获取组织节点详情"`
	Id     int64 `json:"id" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
}

// OrgDetailRes 获取组织节点详情响应
type OrgDetailRes struct {
	g.Meta   `mime:"application/json"`
	Data     *model.OrgStructure `json:"data" dc:"节点详情"`
	PathName string              `json:"pathName" dc:"完整路径名称"`
}

// OrgSaveReq 保存组织节点请求
type OrgSaveReq struct {
	g.Meta   `path:"/org/save" method:"post" tags:"OrgStructure" summary:"保存组织节点（新增或编辑）"`
	Id       int64  `json:"id" dc:"节点ID，编辑时必填"`
	ParentId int64  `json:"parentId" dc:"父节点ID"`
	Name     string `json:"name" v:"required#节点名称不能为空" dc:"节点名称"`
	Type     string `json:"type" v:"required#节点类型不能为空" dc:"节点类型：company/group/employee"`
	Code     string `json:"code" dc:"节点编码"`
	Sort     int    `json:"sort" dc:"排序值"`
	Remark   string `json:"remark" dc:"备注"`
}

// OrgSaveRes 保存组织节点响应
type OrgSaveRes struct {
	g.Meta `mime:"application/json"`
	Id     int64 `json:"id" dc:"节点ID"`
}

// OrgDeleteReq 删除组织节点请求
type OrgDeleteReq struct {
	g.Meta `path:"/org/delete" method:"post" tags:"OrgStructure" summary:"删除组织节点"`
	Id     int64 `json:"id" v:"required#节点ID不能为空" dc:"节点ID"`
}

// OrgDeleteRes 删除组织节点响应
type OrgDeleteRes struct {
	g.Meta `mime:"application/json"`
}

// OrgChildrenReq 获取子节点列表请求
type OrgChildrenReq struct {
	g.Meta   `path:"/org/children" method:"get" tags:"OrgStructure" summary:"获取子节点列表"`
	ParentId int64 `json:"parentId" in:"query" v:"required#父节点ID不能为空" dc:"父节点ID"`
}

// OrgChildrenRes 获取子节点列表响应
type OrgChildrenRes struct {
	g.Meta `mime:"application/json"`
	List   []model.OrgStructure `json:"list" dc:"子节点列表"`
}

// OrgAncestorsReq 获取祖先节点路径请求
type OrgAncestorsReq struct {
	g.Meta `path:"/org/ancestors" method:"get" tags:"OrgStructure" summary:"获取祖先节点路径"`
	Id     int64 `json:"id" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
}

// OrgAncestorsRes 获取祖先节点路径响应
type OrgAncestorsRes struct {
	g.Meta `mime:"application/json"`
	List   []model.OrgStructure `json:"list" dc:"祖先节点列表（从根到当前节点）"`
}

// OrgBindShopReq 绑定店铺到组织节点请求
type OrgBindShopReq struct {
	g.Meta `path:"/org/bind-shop" method:"post" tags:"OrgStructure" summary:"绑定店铺到组织节点"`
	ShopId int   `json:"shopId" v:"required#店铺ID不能为空" dc:"店铺ID"`
	OrgId  int64 `json:"orgId" v:"required#组织节点ID不能为空" dc:"组织节点ID（必须是投手/employee级别）"`
}

// OrgBindShopRes 绑定店铺到组织节点响应
type OrgBindShopRes struct {
	g.Meta `mime:"application/json"`
}

// OrgUnbindShopReq 解绑店铺与组织节点请求
type OrgUnbindShopReq struct {
	g.Meta `path:"/org/unbind-shop" method:"post" tags:"OrgStructure" summary:"解绑店铺与组织节点"`
	ShopId int   `json:"shopId" v:"required#店铺ID不能为空" dc:"店铺ID"`
	OrgId  int64 `json:"orgId" v:"required#组织节点ID不能为空" dc:"组织节点ID"`
}

// OrgUnbindShopRes 解绑店铺与组织节点响应
type OrgUnbindShopRes struct {
	g.Meta `mime:"application/json"`
}

// OrgShopPathReq 获取店铺的组织路径请求
type OrgShopPathReq struct {
	g.Meta `path:"/org/shop-path" method:"get" tags:"OrgStructure" summary:"获取店铺的组织路径"`
	ShopId int `json:"shopId" in:"query" v:"required#店铺ID不能为空" dc:"店铺ID"`
}

// OrgShopPathRes 获取店铺的组织路径响应
type OrgShopPathRes struct {
	g.Meta   `mime:"application/json"`
	Path     []model.OrgStructure `json:"path" dc:"组织路径"`
	PathName string               `json:"pathName" dc:"路径名称（如：总公司 > 济宁 > 李艳雨 > 投放一组 > 李百川 > 抖音 > 账号1）"`
}

// --- Org Stats API ---

// OrgStatsReq 组织节点佣金统计请求
type OrgStatsReq struct {
	g.Meta    `path:"/org/stats" method:"get" tags:"OrgStructure" summary:"组织节点佣金统计"`
	OrgId     int64  `json:"orgId" in:"query" v:"required#节点ID不能为空" dc:"节点ID"`
	Preset    string `json:"preset" in:"query" d:"thisMonth" dc:"预设时间：today/yesterday/last7/last30/thisMonth/all"`
	StartDate string `json:"startDate" in:"query" dc:"自定义开始日期"`
	EndDate   string `json:"endDate" in:"query" dc:"自定义结束日期"`
}

// OrgStatsRes 组织节点佣金统计响应
type OrgStatsRes struct {
	g.Meta    `mime:"application/json"`
	OrgId     int64                `json:"orgId" dc:"节点ID"`
	OrgName   string               `json:"orgName" dc:"节点名称"`
	OrgType   string               `json:"orgType" dc:"节点类型"`
	StartDate string               `json:"startDate" dc:"开始日期"`
	EndDate   string               `json:"endDate" dc:"结束日期"`
	Summary   model.StatsSummary   `json:"summary" dc:"该节点汇总统计"`
	Children  []model.OrgStatsNode `json:"children,omitempty" dc:"直接子节点统计"`
}
