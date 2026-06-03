package address_list_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AddressListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *AddressListData `json:"data"`
}
type AddressListItem struct {
	// 即时零售商家：biz_type = 2代表发货仓地址，biz_type = 3代表自建地址
	BizType int64 `json:"biz_type"`
	// 业务id即时零售商家：如果biz_type = 2则是发货仓id，如果biz_type = 3则是门店id
	BizId int64 `json:"biz_id"`
	// 普通座机
	CommonPhone *CommonPhone `json:"common_phone"`
	// 企业座机信息
	CompanyPhone *CompanyPhone `json:"company_phone"`
	// 联系方式类型;0-手机，1-普通座机，2-企业座机
	LinkType int32 `json:"link_type"`
	// 手机号，商家后台填写的手机号;
	Phone string `json:"phone"`
	// 默认联系方式，如果是手机，这里展示手机号；如果默认联系方式选择固定电话，这里是固定电话;
	Mobile string `json:"mobile"`
	// 地址库id
	AddressId int64 `json:"address_id"`
	// 收/发件人
	RecieverName string `json:"reciever_name"`
	// 是否为退货默认
	IsDefault int64 `json:"is_default"`
	// 是否为发货默认
	IsSendDefault int64 `json:"is_send_default"`
	// 创建时间，时间戳，秒
	CreateTime string `json:"create_time"`
	// 更新时间，时间戳，秒
	UpdateTime string `json:"update_time"`
	// 收件人省份
	ReceiverProvinc string `json:"receiver_provinc"`
	// 收件人城市
	ReceiverCity string `json:"receiver_city"`
	// 收件人地区
	ReceiverDistrict string `json:"receiver_district"`
	// 收件人详情地址
	ReceiverDetail string `json:"receiver_detail"`
	// 收件人街道
	ReceiverStreet string `json:"receiver_street"`
	// 地址备注信息
	Remark string `json:"remark"`
}
type AddressListData struct {
	// 地址总数
	Total int64 `json:"total"`
	// 页码
	PageSize int64 `json:"page_size"`
	// 每页数量
	PageNo int64 `json:"page_no"`
	// 地址列表
	AddressList []AddressListItem `json:"address_list"`
}
type CommonPhone struct {
	// 区号
	AreaNum string `json:"area_num"`
	// 主机号
	MainNum string `json:"main_num"`
	// 分机号
	AuxiliaryNum string `json:"auxiliary_num"`
	// 拼接之后的联系方式;
	Num string `json:"num"`
}
type CompanyPhone struct {
	// 拼接之后的联系方式;
	Num string `json:"num"`
	// 联系方式号码;
	PhoneNum string `json:"phone_num"`
}
