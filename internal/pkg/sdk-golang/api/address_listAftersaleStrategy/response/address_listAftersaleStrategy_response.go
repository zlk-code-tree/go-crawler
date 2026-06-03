package address_listAftersaleStrategy_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AddressListAftersaleStrategyResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *AddressListAftersaleStrategyData `json:"data"`
}
type AddressListAftersaleStrategyData struct {
	// 每页数量;
	Size int64 `json:"size"`
	// 策略列表;
	StrategyList []StrategyListItem `json:"strategy_list"`
	// 总量;
	Total int64 `json:"total"`
	// 页数;
	Page int64 `json:"page"`
}
type CompanyPhone struct {
	// 联系方式号码;
	PhoneNum string `json:"phone_num"`
	// 拼接之后的联系方式;
	Num string `json:"num"`
}
type CommonPhone struct {
	// 区号;
	AreaNum string `json:"area_num"`
	// 主机号;
	MainNum string `json:"main_num"`
	// 分机号;
	AuxiliaryNum string `json:"auxiliary_num"`
	// 拼接之后的联系方式;
	Num string `json:"num"`
}
type Address struct {
	// 地址更新时间;
	UpdateTime string `json:"update_time"`
	// 企业座机;
	CompanyPhone *CompanyPhone `json:"company_phone"`
	// 自增ID, 也是地址的ID;
	Id int64 `json:"id"`
	// 是否是默认地址，0-不是默认地址，1-是默认地址;
	IsDefault int64 `json:"is_default"`
	// 城市名称;
	CityName string `json:"city_name"`
	// 默认联系方式，如果是手机，这里展示手机号；如果默认联系方式选择固定电话，这里是固定电话;
	Mobile string `json:"mobile"`
	// 省ID;
	ProvinceId int64 `json:"province_id"`
	// 地址异常信息
	ErrorType int64 `json:"error_type"`
	// 业务类型 0:抖店店铺 1:门店
	BizType int32 `json:"biz_type"`
	// 手机号，商家后台填写的手机号;
	Phone string `json:"phone"`
	// 售后备注
	AfterSaleNote string `json:"after_sale_note"`
	// 店铺ID;
	ShopId int64 `json:"shop_id"`
	// 地址,详细地址;
	Address string `json:"address"`
	// 街道ID;
	StreetId int64 `json:"street_id"`
	// 街道名称;
	StreetName string `json:"street_name"`
	// 区\县ID;
	TownId int64 `json:"town_id"`
	// 业务id，如门店ID
	BizId int64 `json:"biz_id"`
	// 是否是默认发货地址;0-不是，1-是
	IsSendDefault int64 `json:"is_send_default"`
	// 地址创建时间;
	CreateTime string `json:"create_time"`
	// 省的名称;
	ProvinceName string `json:"province_name"`
	// 区\县名称;
	TownName string `json:"town_name"`
	// 联系方式类型;0-手机，1-普通座机，2-企业座机
	LinkType int32 `json:"link_type"`
	// 售后地址联系人;
	UserName string `json:"user_name"`
	// 城市ID;
	CityId int64 `json:"city_id"`
	// 详细地址;
	Detail string `json:"detail"`
	// 普通座机;22: CommonPhone Comm
	CommonPhone *CommonPhone `json:"common_phone"`
}
type AddressListItem struct {
	// 具体地址信息;
	Address *Address `json:"address"`
}
type StrategyListItem struct {
	// 策略id;
	StrategyId string `json:"strategy_id"`
	// 售后策略绑定的地址列表;
	AddressList []AddressListItem `json:"address_list"`
	// 已关联商品数量
	RelatedProductCount int64 `json:"related_product_count"`
	// 策略名称
	StrategyName string `json:"strategy_name"`
	// 策略创建时间;
	Createtime int64 `json:"createtime"`
	// 策略更新时间;
	UpdateTime int64 `json:"update_time"`
	// 店铺ID;
	ShopId int64 `json:"shop_id"`
	// 策略类型;1-固定退货地址，2-动态退货地址
	StrategyType int32 `json:"strategy_type"`
	// 策略类型描述;
	StrategyTypeName string `json:"strategy_type_name"`
	// 策略描述;
	StrategyDescription string `json:"strategy_description"`
}
