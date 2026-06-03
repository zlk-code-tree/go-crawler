package v1

import (
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== 投手列表 ====================

// TraderListReq 投手列表请求
type TraderListReq struct {
	g.Meta    `path:"/trader/list" method:"get" tags:"Trader" summary:"获取投手列表"`
	Page      int    `json:"page" in:"query" d:"1" v:"required|min:1#页码不能为空|页码最小为1" dc:"页码"`
	PageSize  int    `json:"pageSize" in:"query" d:"20" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100" dc:"每页数量"`
	Keyword   string `json:"keyword" in:"query" d:"" dc:"搜索关键词（匹配名称）"`
	Role      int    `json:"role" in:"query" d:"0" dc:"身份筛选：0=全部，1=投手，2=运营"`
	CompanyId int64  `json:"companyId" in:"query" d:"0" dc:"归属公司ID：0=全部"`
	GroupId   int64  `json:"groupId" in:"query" d:"0" dc:"归属组ID：0=全部"`
}

// TraderListRes 投手列表响应
type TraderListRes struct {
	g.Meta `mime:"application/json"`
	List   []model.Trader `json:"list" dc:"投手列表"`
	Total  int            `json:"total" dc:"总数"`
	Page   int            `json:"page" dc:"当前页码"`
}

// ==================== 投手详情 ====================

// TraderDetailReq 投手详情请求
type TraderDetailReq struct {
	g.Meta `path:"/trader/detail" method:"get" tags:"Trader" summary:"获取投手详情"`
	Id     int64 `json:"id" in:"query" v:"required#投手ID不能为空" dc:"投手ID"`
}

// TraderDetailRes 投手详情响应
type TraderDetailRes = model.TraderWithShops

// ==================== 保存投手 ====================

// TraderSaveReq 保存投手请求
type TraderSaveReq struct {
	g.Meta    `path:"/trader/save" method:"post" tags:"Trader" summary:"新增/编辑投手"`
	Id        int64  `json:"id" dc:"投手ID，编辑时必填"`
	Name      string `json:"name" v:"required#投手名称不能为空" dc:"投手名称"`
	CompanyId int64  `json:"companyId" dc:"归属公司ID（关联sys_company.id）"`
	GroupId   int64  `json:"groupId" dc:"归属部门ID（关联sys_department.id）"`
	Role      int    `json:"role" v:"required|in:1,2#身份不能为空|身份值非法" dc:"身份：1=投手，2=运营"`
	Remark    string `json:"remark" dc:"备注"`
}

// TraderSaveRes 保存投手响应
type TraderSaveRes struct {
	g.Meta `mime:"application/json"`
	Id     int64 `json:"id" dc:"投手ID"`
}

// ==================== 删除投手 ====================

// TraderDeleteReq 删除投手请求
type TraderDeleteReq struct {
	g.Meta `path:"/trader/delete" method:"post" tags:"Trader" summary:"删除投手"`
	Id     int64 `json:"id" v:"required#投手ID不能为空" dc:"投手ID"`
}

// TraderDeleteRes 删除投手响应
type TraderDeleteRes struct {
	g.Meta `mime:"application/json"`
}

// ==================== 批量绑定账号 ====================

// TraderBindShopReq 批量绑定账号请求
type TraderBindShopReq struct {
	g.Meta   `path:"/trader/bind-shop" method:"post" tags:"Trader" summary:"批量绑定账号到投手"`
	TraderId int64   `json:"traderId" v:"required#投手ID不能为空" dc:"投手ID"`
	AccountIds []int64 `json:"accountIds" v:"required#账号ID列表不能为空" dc:"账号ID列表"`
}

// TraderBindShopRes 批量绑定账号响应
type TraderBindShopRes struct {
	g.Meta     `mime:"application/json"`
	BoundCount int `json:"boundCount" dc:"成功绑定数量"`
}

// ==================== 解绑账号 ====================

// TraderUnbindShopReq 解绑账号请求
type TraderUnbindShopReq struct {
	g.Meta   `path:"/trader/unbind-shop" method:"post" tags:"Trader" summary:"解绑账号"`
	TraderId int64 `json:"traderId" v:"required#投手ID不能为空" dc:"投手ID"`
	AccountId int   `json:"accountId" v:"required#账号ID不能为空" dc:"账号ID"`
}

// TraderUnbindShopRes 解绑账号响应
type TraderUnbindShopRes struct {
	g.Meta `mime:"application/json"`
}

// ==================== 获取投手管理的店铺 ====================

// TraderGetShopsReq 获取投手管理的店铺请求
type TraderGetShopsReq struct {
	g.Meta   `path:"/trader/get-shops" method:"get" tags:"Trader" summary:"获取投手管理的账号列表"`
	TraderId int64 `json:"traderId" in:"query" v:"required#投手ID不能为空" dc:"投手ID"`
}

// TraderGetShopsRes 获取投手管理的店铺响应
type TraderGetShopsRes struct {
	g.Meta `mime:"application/json"`
	Shops  []model.TraderShop `json:"shops" dc:"账号列表"`
}

// ==================== 获取可绑定账号 ====================

// TraderBindableShopsReq 获取可绑定账号请求
type TraderBindableShopsReq struct {
	g.Meta   `path:"/trader/bindable-shops" method:"get" tags:"Trader" summary:"获取可绑定的账号列表"`
	TraderId int64  `json:"traderId" in:"query" v:"required#投手ID不能为空" dc:"投手ID"`
	Keyword  string `json:"keyword" in:"query" d:"" dc:"搜索关键词（匹配账号名称、UID）"`
	Platform string `json:"platform" in:"query" d:"" dc:"平台筛选：抖音/快手/视频号"`
}

// TraderBindableShopsRes 获取可绑定账号响应
type TraderBindableShopsRes struct {
	g.Meta `mime:"application/json"`
	Shops  []model.BindableShop `json:"shops" dc:"可绑定的账号列表"`
}
