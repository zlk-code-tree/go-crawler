package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// CrawlReq 爬虫请求
type CrawlReq struct {
	g.Meta    `path:"/crawl" tags:"Crawler" method:"post" summary:"启动抖音电商罗盘爬虫任务"`
	TargetUrl string `json:"target_url" dc:"目标 URL（可选，默认使用商品榜单页面）"`
	LoginType string `json:"login_type" dc:"登录方式：qrcode(扫码) / password(账号密码)"`
	Username  string `json:"username" dc:"账号（仅 password 模式需要）"`
	Password  string `json:"password" dc:"密码（仅 password 模式需要）"`
}

// Result 爬虫结果数据
type Result struct {
	URL            string            `json:"url" dc:"抓取页面 URL"`
	Title          string            `json:"title" dc:"页面标题"`
	Content        string            `json:"content" dc:"抓取内容摘要"`
	ProductCount   int               `json:"product_count" dc:"商品数量"`
	VideoCount     int               `json:"video_count" dc:"视频分析数量"`
	Products       []ProductInfo     `json:"products,omitempty" dc:"商品列表"`
	ProductDetails []ProductDetail   `json:"product_details,omitempty" dc:"商品详情"`
	VideoAnalysis  []VideoAnalysis   `json:"video_analysis,omitempty" dc:"视频分析数据"`
	ExportFiles    map[string]string `json:"export_files,omitempty" dc:"导出文件路径"`
}

// ProductInfo 商品基本信息
type ProductInfo struct {
	Rank             int    `json:"rank" dc:"排名"`
	ProductName      string `json:"product_name" dc:"商品名称"`
	ShopName         string `json:"shop_name" dc:"账号名称"`
	PaymentAmount    string `json:"payment_amount" dc:"短视频用户支付金额"`
	ClickCount       string `json:"click_count" dc:"短视频点击次数"`
	SalesVolume      string `json:"sales_volume" dc:"短视频成交件数"`
	DeliveryVideoNum string `json:"delivery_video_num" dc:"带货视频数"`
	ProductURL       string `json:"product_url" dc:"商品详情页 URL"`
	FirstList        int    `json:"first_list" dc:"是否首次上榜：0-否 1-是"`
}

// ProductDetail 商品详情数据
type ProductDetail struct {
	ProductName    string `json:"product_name" dc:"商品名称"`
	FirstShelfTime string `json:"first_shelf_time" dc:"首次上架时间"`
	MinPrice       string `json:"min_price" dc:"最低到手价"`
}

// VideoAnalysis 视频分析数据
type VideoAnalysis struct {
	VideoID           string `json:"video_id" dc:"视频 ID"`
	VideoDesc         string `json:"video_desc" dc:"短视频信息"`
	PaymentAmount     string `json:"payment_amount" dc:"用户支付金额"`
	ExposeUserCount   string `json:"expose_user_count" dc:"商品曝光人数"`
	ClickUserCount    string `json:"click_user_count" dc:"商品点击人数"`
	ConvertUserCount  string `json:"convert_user_count" dc:"商品成交人数"`
	ClickRate         string `json:"click_rate" dc:"商品点击率 (人数)"`
	ClickConvertRate  string `json:"click_convert_rate" dc:"商品点击 - 成交率 (人数)"`
	ExposeConvertRate string `json:"expose_convert_rate" dc:"商品曝光 - 成交率 (人数)"`
	OrderCount        string `json:"order_count" dc:"商品成交订单数"`
	SalesVolume       string `json:"sales_volume" dc:"商品成交件数"`
}

// CrawlRes 爬虫响应
type CrawlRes struct {
	g.Meta  `mime:"application/json" example:"json"`
	Success bool     `json:"success" dc:"是否成功"`
	Message string   `json:"message" dc:"响应消息"`
	Data    []Result `json:"data" dc:"爬虫数据"`
}
