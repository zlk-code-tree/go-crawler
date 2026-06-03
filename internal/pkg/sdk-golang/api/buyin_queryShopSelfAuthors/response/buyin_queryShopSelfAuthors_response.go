package buyin_queryShopSelfAuthors_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinQueryShopSelfAuthorsResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinQueryShopSelfAuthorsData `json:"data"`
}
type AuthorsItem struct {
	// 达人帐号的客户端1128：抖音2329：抖音极速版1112：火山13：头条
	AuthorApp int64 `json:"author_app"`
	// 达人uid
	AuthorId int64 `json:"author_id"`
	// 0:店铺官方帐号 1:店铺渠道授权帐号 2: 普通达人帐号
	AuthorType int32 `json:"author_type"`
	// 达人的百应id
	BuyinId int64 `json:"buyin_id"`
	// 账户名称
	AccountName string `json:"account_name"`
	// 达人头像
	AvatarUrl string `json:"avatar_url"`
}
type BuyinQueryShopSelfAuthorsData struct {
	// 总数量
	Total int64 `json:"total"`
	// 达人信息
	Authors []AuthorsItem `json:"authors"`
}
