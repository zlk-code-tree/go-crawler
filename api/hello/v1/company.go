package v1

import "github.com/gogf/gf/v2/frame/g"

// --- System Company APIs ---

type SystemCompanyListReq struct {
	g.Meta   `path:"/system/company/list" method:"get" tags:"System-Company" summary:"公司列表"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"20" dc:"每页条数"`
	Keyword  string `json:"keyword" in:"query" dc:"搜索关键词"`
	Status   string `json:"status" in:"query" dc:"状态"`
}

type SystemCompanyListRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"公司列表"`
	Total    int         `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"pageSize" dc:"每页条数"`
}

type SystemCompanyCreateReq struct {
	g.Meta   `path:"/system/company/create" method:"post" tags:"System-Company" summary:"创建公司"`
	Name     string `json:"name" v:"required" dc:"公司名称"`
	Code     string `json:"code" v:"required" dc:"公司编码"`
	ParentId int64  `json:"parent_id" d:"0" dc:"父公司ID"`
	Level    int    `json:"level" d:"1" dc:"公司层级（1-集团，2-子公司，3-分公司）"`
	Sort     int    `json:"sort" d:"0" dc:"排序值"`
	Remark   string `json:"remark" dc:"备注"`
}

type SystemCompanyCreateRes struct {
	g.Meta `mime:"application/json"`
	Id     int64 `json:"id" dc:"公司ID"`
}

type SystemCompanyUpdateReq struct {
	g.Meta   `path:"/system/company/update" method:"put" tags:"System-Company" summary:"更新公司"`
	Id       int64  `json:"id" v:"required" dc:"公司ID"`
	Name     string `json:"name" dc:"公司名称"`
	ParentId *int64 `json:"parent_id" dc:"父公司ID"`
	Level    *int   `json:"level" dc:"公司层级"`
	Status   *int   `json:"status" dc:"状态"`
	Sort     *int   `json:"sort" dc:"排序值"`
	Remark   string `json:"remark" dc:"备注"`
}

type SystemCompanyUpdateRes struct {
	g.Meta `mime:"application/json"`
}

type SystemCompanyDetailReq struct {
	g.Meta `path:"/system/company/detail" method:"get" tags:"System-Company" summary:"公司详情"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"公司ID"`
}

type SystemCompanyDetailRes struct {
	g.Meta `mime:"application/json"`
}

type SystemCompanyDeleteReq struct {
	g.Meta `path:"/system/company/delete" method:"delete" tags:"System-Company" summary:"删除公司"`
	Id     int64 `json:"id" v:"required" dc:"公司ID"`
}

type SystemCompanyDeleteRes struct {
	g.Meta `mime:"application/json"`
}
