package address_getAftersaleStrategy_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AddressGetAftersaleStrategyResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *Data `json:"data"`
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
type CompanyPhone struct {
	// 联系方式号码;
	PhoneNum string `json:"phone_num"`
	// 拼接之后的联系方式;
	Num string `json:"num"`
}
type Address struct {
	// 自增ID, 也是地址的ID;
	Id int64 `json:"id"`
	// 街道名称;
	StreetName string `json:"street_name"`
	// 是否是默认地址，0-不是默认地址，1-是默认地址;
	IsDefault int64 `json:"is_default"`
	// 省的名称;
	ProvinceName string `json:"province_name"`
	// 街道ID;
	StreetId int64 `json:"street_id"`
	// 联系方式类型;
	LinkType int32 `json:"link_type"`
	// 普通座机;22: CommonPhone Comm
	CommonPhone *CommonPhone `json:"common_phone"`
	// 创建时间;
	CreateTime string `json:"create_time"`
	// 城市ID;
	CityId int64 `json:"city_id"`
	// 业务类型 0:抖店店铺 1:门店
	BizType int32 `json:"biz_type"`
	// 售后地址联系人;
	UserName string `json:"user_name"`
	// 默认联系方式，如果是手机，这里展示手机号；如果默认联系方式选择固定电话，这里是固定电话;
	Mobile string `json:"mobile"`
	// 城市名称;
	CityName string `json:"city_name"`
	// 售后备注
	AfterSaleNote string `json:"after_sale_note"`
	// 店铺ID;
	ShopId int64 `json:"shop_id"`
	// 是否是默认发货地址;
	IsSendDefault int64 `json:"is_send_default"`
	// 更新时间;
	UpdateTime string `json:"update_time"`
	// 详细地址;
	Detail string `json:"detail"`
	// 省ID;
	ProvinceId int64 `json:"province_id"`
	// 区\县ID;
	TownId int64 `json:"town_id"`
	// 区\县名称;
	TownName string `json:"town_name"`
	// 业务id，如门店ID
	BizId int64 `json:"biz_id"`
	// 地址,详细地址;
	Address string `json:"address"`
	// 企业座机;
	CompanyPhone *CompanyPhone `json:"company_phone"`
	// 手机号，商家后台填写的手机号;
	Phone string `json:"phone"`
}
type AddressListItem struct {
	// 动态退货地址策略类型下，该地址绑定的用户收货地址
	BindAddressList []BindAddressListItem `json:"bind_address_list"`
	// 具体地址信息;
	Address *Address `json:"address"`
}
type Data struct {
	// 策略类型描述;
	StrategyTypeName string `json:"strategy_type_name"`
	// 策略名称;
	StrategyName string `json:"strategy_name"`
	// 售后策略绑定的地址列表;
	AddressList []AddressListItem `json:"address_list"`
	// 更新时间;
	UpdateTime int64 `json:"update_time"`
	// 店铺ID;
	ShopId int64 `json:"shop_id"`
	// 策略id;
	StrategyId string `json:"strategy_id"`
	// 创建时间;
	Createtime int64 `json:"createtime"`
	// 已关联商品数量
	RelatedProductCount int64 `json:"related_product_count"`
	// 策略类型;
	StrategyType int32 `json:"strategy_type"`
	// 策略描述;
	StrategyDescription string `json:"strategy_description"`
}
type AddressGetAftersaleStrategyData struct {
	// 返回数据
	Data *Data `json:"data"`
}
type BindAddressListItem struct {
	// 用户收货地址的省Id
	ProvinceId int64 `json:"province_id"`
	// 用户收货地址的省名称
	ProvinceName string `json:"province_name"`
	// 用户收货地址的市ID
	CityId int64 `json:"city_id"`
	// 用户收货地址的市名称
	CityName string `json:"city_name"`
}
