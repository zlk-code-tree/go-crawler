package douyin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service/platform"

	orderDetailReq "doudian.com/open/sdk_golang/api/order_orderDetail/request"
	orderSearchReq "doudian.com/open/sdk_golang/api/order_searchList/request"
	orderSearchResp "doudian.com/open/sdk_golang/api/order_searchList/response"
	doudian "doudian.com/open/sdk_golang/core"

	"github.com/gogf/gf/v2/frame/g"
)

func init() {
	platform.Register(NewService())
}

// Service 抖店开放平台服务
type Service struct {
	config     *doudian.DoudianOpConfig
	tokenCache map[int64]*doudian.AccessToken // key: douyin shop_id
	mu         sync.RWMutex
}

func NewService() *Service {
	return &Service{
		tokenCache: make(map[int64]*doudian.AccessToken),
	}
}

func (s *Service) Platform() string {
	return "douyin"
}

// initConfig 初始化SDK配置（从配置文件读取）
func (s *Service) initConfig() {
	if s.config != nil {
		return
	}
	cfg := g.Cfg()
	s.config = &doudian.DoudianOpConfig{
		AppKey:          cfg.MustGet(context.Background(), "platform.douyin.appKey").String(),
		AppSecret:       cfg.MustGet(context.Background(), "platform.douyin.appSecret").String(),
		OpenRequestUrl:  cfg.MustGet(context.Background(), "platform.douyin.baseUrl", "https://openapi-fxg.jinritemai.com").String(),
		HttpReadTimeout: cfg.MustGet(context.Background(), "platform.douyin.timeout", 15000).Int64(),
	}
}

// getTokenForShop 根据抖店 shop_id 获取或创建 access_token
func (s *Service) getTokenForShop(shopId int64) (*doudian.AccessToken, error) {
	s.mu.RLock()
	if token, ok := s.tokenCache[shopId]; ok {
		s.mu.RUnlock()
		return token, nil
	}
	s.mu.RUnlock()

	s.initConfig()
	token, err := doudian.BuildAccessToken(&doudian.BuildAccessTokenParam{
		ShopId: &shopId,
		Config: s.config,
	})
	if err != nil {
		return nil, fmt.Errorf("获取抖店 access_token 失败(shopId=%d): %w", shopId, err)
	}
	if !token.IsSuccess() {
		return nil, fmt.Errorf("抖店 access_token 获取失败(shopId=%d)", shopId)
	}

	s.mu.Lock()
	s.tokenCache[shopId] = token
	s.mu.Unlock()

	return token, nil
}

// GetTokenForStore 根据 store 表记录获取 token（公开方法，供外部使用）
func (s *Service) GetTokenForStore(ctx context.Context, storeId int) (*doudian.AccessToken, int64, error) {
	store, err := g.DB().Model("store").Ctx(ctx).Where("id", storeId).One()
	if err != nil {
		return nil, 0, fmt.Errorf("查询店铺信息失败: %w", err)
	}
	if store.IsEmpty() {
		return nil, 0, fmt.Errorf("店铺不存在: id=%d", storeId)
	}

	shopId := store["store_id"].Int64()
	token, err := s.getTokenForShop(shopId)
	if err != nil {
		return nil, 0, err
	}
	return token, shopId, nil
}

// strPtr 返回字符串指针
func strPtr(s string) *string {
	return &s
}

// GetOrderList 获取订单列表
func (s *Service) GetOrderList(ctx context.Context, params model.PlatformOrderListParams) (*model.PlatformOrderListResult, error) {
	s.initConfig()

	var token *doudian.AccessToken
	if params.StoreId > 0 {
		var err error
		token, _, err = s.GetTokenForStore(ctx, params.StoreId)
		if err != nil {
			return nil, err
		}
	} else {
		cfg := g.Cfg()
		shopId := cfg.MustGet(ctx, "platform.douyin.shopId").Int64()
		if shopId <= 0 {
			return nil, fmt.Errorf("请提供 storeId 或在配置文件中设置 platform.douyin.shopId")
		}
		var err error
		token, err = s.getTokenForShop(shopId)
		if err != nil {
			return nil, err
		}
	}

	req := orderSearchReq.New()
	req.SetConfig(s.config)
	req.GetParams().Size = int64(params.PageSize)
	req.GetParams().Page = int64(params.Page)

	if params.StartTime != "" {
		t, err := time.Parse("2006-01-02 15:04:05", params.StartTime)
		if err != nil {
			return nil, fmt.Errorf("开始时间格式错误: %w", err)
		}
		ts := t.Unix()
		req.GetParams().CreateTimeStart = &ts
	}
	if params.EndTime != "" {
		t, err := time.Parse("2006-01-02 15:04:05", params.EndTime)
		if err != nil {
			return nil, fmt.Errorf("结束时间格式错误: %w", err)
		}
		ts := t.Unix()
		req.GetParams().CreateTimeEnd = &ts
	}

	if params.OrderStatus != "" {
		req.GetParams().CombineStatus = []orderSearchReq.CombineStatusItem{
			{OrderStatus: strPtr(params.OrderStatus)},
		}
	}

	resp, err := req.Execute(token)
	if err != nil {
		return nil, fmt.Errorf("抖店订单列表查询失败: %w", err)
	}
	if resp.Code != 10000 {
		return nil, fmt.Errorf("抖店返回错误: code=%d, msg=%s, sub_msg=%s",
			resp.Code, resp.Msg, resp.SubMsg)
	}

	data := resp.Data
	result := &model.PlatformOrderListResult{
		Total: int(data.Total),
		Page:  int(data.Page),
		Size:  int(data.Size),
	}

	for _, item := range data.ShopOrderList {
		si := shopItemToOrderItem(item)
		for _, sku := range item.SkuOrderList {
			oi := si
			oi = fillSkuFields(oi, sku)
			result.Orders = append(result.Orders, oi)
		}
	}

	return result, nil
}

// shopItemToOrderItem 将店铺级字段映射到 PlatformOrderItem
func shopItemToOrderItem(item orderSearchResp.ShopOrderListItem) model.PlatformOrderItem {
	p := model.PlatformOrderItem{
		Platform:                "douyin",
		OrderNo:                 item.OrderId,
		ShopId:                  item.ShopId,
		ShopName:                item.ShopName,
		OrderStatus:             item.OrderStatus,
		OrderStatusDesc:         item.OrderStatusDesc,
		MainStatus:              item.MainStatus,
		MainStatusDesc:          item.MainStatusDesc,
		PayAmount:               float64(item.PayAmount) / 100,
		OrderAmount:             float64(item.OrderAmount) / 100,
		PostAmount:              float64(item.PostAmount) / 100,
		PostInsuranceAmount:     float64(item.PostInsuranceAmount) / 100,
		PromotionAmount:         float64(item.PromotionAmount) / 100,
		PromotionShopAmount:     float64(item.PromotionShopAmount) / 100,
		PromotionPlatformAmount: float64(item.PromotionPlatformAmount) / 100,
		PromotionTalentAmount:   float64(item.PromotionTalentAmount) / 100,
		PromotionPayAmount:      float64(item.PromotionPayAmount) / 100,
		ShopCostAmount:          float64(item.ShopCostAmount) / 100,
		PlatformCostAmount:      float64(item.PlatformCostAmount) / 100,
		AuthorCostAmount:        float64(item.AuthorCostAmount) / 100,
		ModifyAmount:            float64(item.ModifyAmount) / 100,
		ModifyPostAmount:        float64(item.ModifyPostAmount) / 100,
		BuyerWords:              item.BuyerWords,
		SellerWords:             item.SellerWords,
		PayType:                 item.PayType,
		BType:                   item.BType,
		BTypeDesc:               item.BTypeDesc,
		CancelReason:            item.CancelReason,
		OpenId:                  item.DoudianOpenId,
		PayTime:                 unixToTime(item.PayTime),
		CreateTime:              unixToTime(item.CreateTime),
		FinishTime:              unixToTime(item.FinishTime),
		UpdateTime:              unixToTime(item.UpdateTime),
		ShipTime:                unixToTime(item.ShipTime),
		PostReceiver:            item.MaskPostReceiver,
		PostTel:                 item.MaskPostTel,
	}

	if item.MaskPostAddr != nil {
		p.PostProvince = item.MaskPostAddr.Province.Name
		p.PostCity = item.MaskPostAddr.City.Name
		p.PostTown = item.MaskPostAddr.Town.Name
		p.PostDetail = item.MaskPostAddr.Detail
		p.PostAddr = item.MaskPostAddr.Province.Name +
			item.MaskPostAddr.City.Name +
			item.MaskPostAddr.Town.Name +
			item.MaskPostAddr.Detail
	}
	if len(item.LogisticsInfo) > 0 {
		p.TrackingNo = item.LogisticsInfo[0].TrackingNo
		p.CompanyName = item.LogisticsInfo[0].CompanyName
	}

	return p
}

// fillSkuFields 将SKU级字段填充到 PlatformOrderItem
func fillSkuFields(p model.PlatformOrderItem, sku orderSearchResp.SkuOrderListItem) model.PlatformOrderItem {
	p.SubOrderNo = sku.OrderId
	p.ProductId = sku.ProductId
	p.ProductIdStr = sku.ProductIdStr
	p.SkuId = sku.SkuId
	p.Code = sku.Code
	p.OutSkuId = sku.OutSkuId
	p.OutProductId = sku.OutProductId
	p.ProductName = sku.ProductName
	p.ProductPic = sku.ProductPic
	p.ProductNum = int(sku.ItemNum)
	p.ProductPrice = float64(sku.OriginAmount) / 100
	p.SumAmount = float64(sku.SumAmount) / 100
	p.GoodsType = sku.GoodsType
	p.AuthorId = sku.AuthorId
	p.AuthorName = sku.AuthorName
	p.Biz = sku.Biz
	p.BizDesc = sku.BizDesc
	p.CBiz = sku.CBiz
	p.CBizDesc = sku.CBizDesc
	p.AdEnvType = sku.AdEnvType

	if len(sku.Spec) > 0 {
		var parts []string
		for _, s := range sku.Spec {
			parts = append(parts, s.Name+":"+s.Value)
		}
		p.Spec = fmt.Sprintf("[%s]", joinStrings(parts))
	}
	if sku.AfterSaleInfo != nil {
		p.AfterSaleStatus = sku.AfterSaleInfo.AfterSaleStatus
		p.AfterSaleType = sku.AfterSaleInfo.AfterSaleType
	}

	return p
}

// GetOrderDetail 获取订单详情
func (s *Service) GetOrderDetail(ctx context.Context, params model.PlatformOrderDetailParams) (*model.PlatformOrderItem, error) {
	s.initConfig()

	var token *doudian.AccessToken
	if params.StoreId > 0 {
		var err error
		token, _, err = s.GetTokenForStore(ctx, params.StoreId)
		if err != nil {
			return nil, err
		}
	} else {
		cfg := g.Cfg()
		shopId := cfg.MustGet(ctx, "platform.douyin.shopId").Int64()
		if shopId <= 0 {
			return nil, fmt.Errorf("请提供 storeId 或在配置文件中设置 platform.douyin.shopId")
		}
		var err error
		token, err = s.getTokenForShop(shopId)
		if err != nil {
			return nil, err
		}
	}

	req := orderDetailReq.New()
	req.SetConfig(s.config)
	req.GetParams().ShopOrderId = params.OrderNo

	resp, err := req.Execute(token)
	if err != nil {
		return nil, fmt.Errorf("抖店订单详情查询失败: %w", err)
	}
	if resp.Code != 10000 {
		return nil, fmt.Errorf("抖店返回错误: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	detail := resp.Data.ShopOrderDetail

	result := &model.PlatformOrderItem{
		Platform:                "douyin",
		OrderNo:                 detail.OrderId,
		ShopId:                  detail.ShopId,
		ShopName:                detail.ShopName,
		OrderStatus:             detail.OrderStatus,
		OrderStatusDesc:         detail.OrderStatusDesc,
		MainStatus:              detail.MainStatus,
		MainStatusDesc:          detail.MainStatusDesc,
		PayAmount:               float64(detail.PayAmount) / 100,
		OrderAmount:             float64(detail.OrderAmount) / 100,
		PostAmount:              float64(detail.PostAmount) / 100,
		PostInsuranceAmount:     float64(detail.PostInsuranceAmount) / 100,
		PromotionAmount:         float64(detail.PromotionAmount) / 100,
		PromotionShopAmount:     float64(detail.PromotionShopAmount) / 100,
		PromotionPlatformAmount: float64(detail.PromotionPlatformAmount) / 100,
		PromotionTalentAmount:   float64(detail.PromotionTalentAmount) / 100,
		ShopCostAmount:          float64(detail.ShopCostAmount) / 100,
		PlatformCostAmount:      float64(detail.PlatformCostAmount) / 100,
		AuthorCostAmount:        float64(detail.AuthorCostAmount) / 100,
		BuyerWords:              detail.BuyerWords,
		SellerWords:             detail.SellerWords,
		PayType:                 detail.PayType,
		BType:                   detail.BType,
		BTypeDesc:               detail.BTypeDesc,
		CancelReason:            detail.CancelReason,
		OpenId:                  detail.DoudianOpenId,
		PayTime:                 unixToTime(detail.PayTime),
		CreateTime:              unixToTime(detail.CreateTime),
		FinishTime:              unixToTime(detail.FinishTime),
		UpdateTime:              unixToTime(detail.UpdateTime),
		ShipTime:                unixToTime(detail.ShipTime),
		PostReceiver:            detail.MaskPostReceiver,
		PostTel:                 detail.MaskPostTel,
	}

	if detail.MaskPostAddr != nil {
		result.PostProvince = detail.MaskPostAddr.Province.Name
		result.PostCity = detail.MaskPostAddr.City.Name
		result.PostTown = detail.MaskPostAddr.Town.Name
		result.PostDetail = detail.MaskPostAddr.Detail
		result.PostAddr = detail.MaskPostAddr.Province.Name +
			detail.MaskPostAddr.City.Name +
			detail.MaskPostAddr.Town.Name +
			detail.MaskPostAddr.Detail
	}
	if len(detail.LogisticsInfo) > 0 {
		result.TrackingNo = detail.LogisticsInfo[0].TrackingNo
		result.CompanyName = detail.LogisticsInfo[0].CompanyName
	}
	if len(detail.SkuOrderList) > 0 {
		sku := detail.SkuOrderList[0]
		result.SubOrderNo = sku.OrderId
		result.ProductId = sku.ProductId
		result.ProductIdStr = sku.ProductIdStr
		result.SkuId = sku.SkuId
		result.Code = sku.Code
		result.OutSkuId = sku.OutSkuId
		result.OutProductId = sku.OutProductId
		result.ProductName = sku.ProductName
		result.ProductPic = sku.ProductPic
		result.ProductNum = int(sku.ItemNum)
		result.ProductPrice = float64(sku.OriginAmount) / 100
		result.SumAmount = float64(sku.SumAmount) / 100
		result.GoodsType = sku.GoodsType
		result.AuthorId = sku.AuthorId
		result.AuthorName = sku.AuthorName
		result.Biz = sku.Biz
		result.BizDesc = sku.BizDesc
		result.CBiz = sku.CBiz
		result.CBizDesc = sku.CBizDesc
		if len(sku.Spec) > 0 {
			var parts []string
			for _, s := range sku.Spec {
				parts = append(parts, s.Name+":"+s.Value)
			}
			result.Spec = fmt.Sprintf("[%s]", joinStrings(parts))
		}
		if sku.AfterSaleInfo != nil {
			result.AfterSaleStatus = sku.AfterSaleInfo.AfterSaleStatus
			result.AfterSaleType = sku.AfterSaleInfo.AfterSaleType
		}
	}

	return result, nil
}

// ==================== 辅助函数 ====================

func unixToTime(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

func joinStrings(elems []string) string {
	s := ""
	for i, e := range elems {
		if i > 0 {
			s += ", "
		}
		s += e
	}
	return s
}
