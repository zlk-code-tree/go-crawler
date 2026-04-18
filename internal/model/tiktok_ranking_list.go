package model

import "time"

// TikTokRankingListInput 商品榜单数据（用于数据库插入）
type TikTokRankingListInput struct {
	TaskId           string    `json:"task_id" db:"task_id"`
	ProductId        string    `json:"product_id" db:"product_id"`                 // 商品 ID
	Rank             int       `json:"rank" db:"rank"`                             // 排名
	ProductName      string    `json:"product_name" db:"product_name"`             // 商品名称
	ShopName         string    `json:"shop_name" db:"shop_name"`                   // 店铺名称
	PaymentAmount    string    `json:"payment_amount" db:"payment_amount"`         // 短视频用户支付金额
	ClickCount       string    `json:"click_count" db:"click_count"`               // 短视频点击次数
	SalesVolume      string    `json:"sales_volume" db:"sales_volume"`             // 短视频成交件数
	DeliveryVideoNum string    `json:"delivery_video_num" db:"delivery_video_num"` // 带货视频数
	FirstShelfTime   string    `json:"first_shelf_time" db:"first_shelf_time"`     // 首次上架时间
	MinPrice         string    `json:"min_price" db:"min_price"`                   // 最低到手价
	PriceRange       string    `json:"price_range" db:"price_range"`               // 价格区间（价格带）
	FirstList        int       `json:"first_list" db:"first_list"`                 // 是否首次上榜：0-否 1-是
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// TikTokRankingListEntity 数据库实体
type TiktokRankingList struct {
	Id               uint      `json:"id" db:"id"`
	TaskId           string    `json:"task_id" db:"task_id"`
	ProductId        string    `json:"product_id" db:"product_id"`
	Rank             int       `json:"rank" db:"rank"`
	ProductName      string    `json:"product_name" db:"product_name"`
	ShopName         string    `json:"shop_name" db:"shop_name"`
	PaymentAmount    string    `json:"payment_amount" db:"payment_amount"`
	ClickCount       string    `json:"click_count" db:"click_count"`
	SalesVolume      string    `json:"sales_volume" db:"sales_volume"`
	DeliveryVideoNum string    `json:"delivery_video_num" db:"delivery_video_num"`
	FirstShelfTime   string    `json:"first_shelf_time" db:"first_shelf_time"`
	MinPrice         string    `json:"min_price" db:"min_price"`
	PriceRange       string    `json:"price_range" db:"price_range"`
	FirstList        int       `json:"first_list" db:"first_list"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// VideoAnalysisData 视频分析数据（用于数据库插入）
type VideoAnalysisData struct {
	TaskId            string    `json:"task_id" db:"task_id"`
	ProductId         string    `json:"product_id" db:"product_id"`                   // 商品 ID
	ProductName       string    `json:"product_name" db:"product_name"`               // 商品名称
	VideoId           string    `json:"video_id" db:"video_id"`                       // 视频 ID
	VideoDesc         string    `json:"video_desc" db:"video_desc"`                   // 短视频信息
	PaymentAmount     string    `json:"payment_amount" db:"payment_amount"`           // 用户支付金额
	ExposeUserCount   string    `json:"expose_user_count" db:"expose_user_count"`     // 商品曝光人数
	ClickUserCount    string    `json:"click_user_count" db:"click_user_count"`       // 商品点击人数
	ConvertUserCount  string    `json:"convert_user_count" db:"convert_user_count"`   // 商品成交人数
	ClickRate         string    `json:"click_rate" db:"click_rate"`                   // 商品点击率 (人数)
	ClickConvertRate  string    `json:"click_convert_rate" db:"click_convert_rate"`   // 商品点击 - 成交率 (人数)
	ExposeConvertRate string    `json:"expose_convert_rate" db:"expose_convert_rate"` // 商品曝光 - 成交率 (人数)
	OrderCount        string    `json:"order_count" db:"order_count"`                 // 商品成交订单数
	SalesVolume       string    `json:"sales_volume" db:"sales_volume"`               // 商品成交件数
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// VideoAnalysisEntity 视频分析数据库实体
type VideoAnalysis struct {
	Id                uint      `json:"id" db:"id"`
	TaskId            string    `json:"task_id" db:"task_id"`
	ProductId         string    `json:"product_id" db:"product_id"`
	ProductName       string    `json:"product_name" db:"product_name"`
	VideoId           string    `json:"video_id" db:"video_id"`
	VideoDesc         string    `json:"video_desc" db:"video_desc"`
	PaymentAmount     string    `json:"payment_amount" db:"payment_amount"`
	ExposeUserCount   string    `json:"expose_user_count" db:"expose_user_count"`
	ClickUserCount    string    `json:"click_user_count" db:"click_user_count"`
	ConvertUserCount  string    `json:"convert_user_count" db:"convert_user_count"`
	ClickRate         string    `json:"click_rate" db:"click_rate"`
	ClickConvertRate  string    `json:"click_convert_rate" db:"click_convert_rate"`
	ExposeConvertRate string    `json:"expose_convert_rate" db:"expose_convert_rate"`
	OrderCount        string    `json:"order_count" db:"order_count"`
	SalesVolume       string    `json:"sales_volume" db:"sales_volume"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}
