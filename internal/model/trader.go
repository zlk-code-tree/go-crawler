package model

import "time"

// Trader 投手
type Trader struct {
	Id          int64     `json:"id"`
	Name        string    `json:"name"`
	CompanyId   int64     `json:"companyId"`   // 归属公司ID（关联sys_company.id）
	CompanyName string    `json:"companyName"` // 归属公司名称
	GroupId     int64     `json:"groupId"`     // 归属部门ID（关联sys_department.id）
	GroupName   string    `json:"groupName"`   // 归属部门名称
	Role        int       `json:"role"`        // 1=投手，2=运营
	Status      int       `json:"status"`      // 0=禁用，1=启用
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TraderWithShops 投手详情（含管理的店铺）
type TraderWithShops struct {
	Id          int64         `json:"id"`
	Name        string        `json:"name"`
	CompanyId   int64         `json:"companyId"`   // 归属公司ID
	CompanyName string        `json:"companyName"` // 归属公司名称
	GroupId     int64         `json:"groupId"`     // 归属组ID
	GroupName   string        `json:"groupName"`   // 归属组名称
	Role        int           `json:"role"`
	Status      int           `json:"status"`
	Remark      string        `json:"remark"`
	Shops       []TraderShop  `json:"shops"`       // 管理的店铺列表
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

// TraderShop 投手关联的店铺信息
type TraderShop struct {
	ShopId   int    `json:"shopId"`
	ShopName string `json:"shopName"` // 店铺名称
	Uid      string `json:"uid"`      // 店铺UID
}

// TraderShopRelation 投手-店铺关联
type TraderShopRelation struct {
	Id        int64     `json:"id"`
	TraderId  int64     `json:"traderId"`
	ShopId    int       `json:"shopId"`
	CreatedAt time.Time `json:"createdAt"`
}

// TraderListReq 投手列表请求
type TraderListReq struct {
	Page      int    `json:"page" v:"required|min:1#页码不能为空|页码最小为1"`
	PageSize  int    `json:"pageSize" v:"required|min:1|max:100#每页数量不能为空|每页数量最小为1|每页数量最大为100"`
	Keyword   string `json:"keyword"`  // 搜索关键词（匹配名称）
	Role      int    `json:"role"`     // 身份筛选：0=全部，1=投手，2=运营
	CompanyId int64  `json:"companyId"` // 归属公司筛选：0=全部
	GroupId   int64  `json:"groupId"`  // 归属组筛选：0=全部
}

// TraderSaveReq 保存投手请求
type TraderSaveReq struct {
	Id        int64  `json:"id"`                        // ID，编辑时必填
	Name      string `json:"name" v:"required#投手名称不能为空"`
	CompanyId int64  `json:"companyId"`                 // 归属公司ID
	GroupId   int64  `json:"groupId"`                   // 归属组ID
	Role      int    `json:"role" v:"required|in:1,2#身份不能为空|身份值非法"`
	Remark    string `json:"remark"`
}

// TraderBindShopReq 绑定店铺请求（支持批量）
type TraderBindShopReq struct {
	TraderId int64   `json:"traderId" v:"required#投手ID不能为空"`
	ShopIds  []int64 `json:"shopIds" v:"required#店铺ID列表不能为空"`
}

// TraderUnbindShopReq 解绑店铺请求
type TraderUnbindShopReq struct {
	TraderId int64 `json:"traderId" v:"required#投手ID不能为空"`
	ShopId   int   `json:"shopId" v:"required#店铺ID不能为空"`
}

// BindableShop 可绑定的店铺信息
type BindableShop struct {
	ShopId   int    `json:"shopId"`
	ShopName string `json:"shopName"`
	Uid      string `json:"uid"`
	Platform string `json:"platform"` // 平台：抖音/快手/视频号
}
