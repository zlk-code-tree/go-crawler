package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service/platform"

	afterSaleReq "doudian.com/open/sdk_golang/api/afterSale_List/request"
	afterSaleResp "doudian.com/open/sdk_golang/api/afterSale_List/response"
	allianceReq "doudian.com/open/sdk_golang/api/alliance_getOrderList/request"
	buyinReq "doudian.com/open/sdk_golang/api/buyin_queryShopAllianceOrder/request"
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

	cst := time.FixedZone("CST", 8*3600)
	if params.StartTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", params.StartTime, cst)
		if err != nil {
			return nil, fmt.Errorf("开始时间格式错误: %w", err)
		}
		ts := t.Unix()
		if params.UseUpdateTime {
			req.GetParams().UpdateTimeStart = &ts
		} else {
			req.GetParams().CreateTimeStart = &ts
		}
	}
	if params.EndTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", params.EndTime, cst)
		if err != nil {
			return nil, fmt.Errorf("结束时间格式错误: %w", err)
		}
		ts := t.Unix()
		if params.UseUpdateTime {
			req.GetParams().UpdateTimeEnd = &ts
		} else {
			req.GetParams().CreateTimeEnd = &ts
		}
	}
	if params.UseUpdateTime {
		orderBy := "update_time"
		orderAsc := true
		req.GetParams().OrderBy = &orderBy
		req.GetParams().OrderAsc = &orderAsc
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

// GetAllianceOrderList 获取精选联盟订单列表
func (s *Service) GetAllianceOrderList(ctx context.Context, params model.PlatformAllianceOrderListParams) (*model.PlatformAllianceOrderListResult, error) {
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

	req := allianceReq.New()
	req.SetConfig(s.config)
	req.GetParams().Size = int64(params.Size)
	req.GetParams().Cursor = params.Cursor
	if params.OrderIds != "" {
		req.GetParams().OrderIds = strings.Split(params.OrderIds, ",")
	}
	if params.StartTime != "" {
		req.GetParams().StartTime = &params.StartTime
	}
	if params.EndTime != "" {
		req.GetParams().EndTime = &params.EndTime
	}
	if params.TimeType != "" {
		req.GetParams().TimeType = &params.TimeType
	}

	g.Log().Debugf(ctx, "[AllianceAPI] cursor=%s start=%s end=%s orderIds=%s size=%d",
		req.GetParams().Cursor, params.StartTime, params.EndTime, params.OrderIds, params.Size)

	resp, err := req.Execute(token)
	if err != nil {
		return nil, fmt.Errorf("精选联盟订单查询失败: %w", err)
	}
	if resp.Code != 10000 {
		return nil, fmt.Errorf("精选联盟返回错误: code=%d, msg=%s, sub_msg=%s",
			resp.Code, resp.Msg, resp.SubMsg)
	}

	data := resp.Data
	g.Log().Debugf(ctx, "[AllianceAPI] code=%s, code_msg=%s, datas=%d, cursor=%s",
		data.Code, data.CodeMsg, len(data.Orders), data.Cursor)

	result := &model.PlatformAllianceOrderListResult{
		Cursor: data.Cursor,
	}

	for _, o := range data.Orders {
		result.Orders = append(result.Orders, model.PlatformAllianceOrderItem{
			Platform:           "douyin",
			OrderId:            o.OrderId,
			ProductId:          o.ProductId,
			AuthorAccount:      o.AuthorAccount,
			CommissionRate:     o.CommissionRate,
			EstimatedComission: o.EstimatedComission,
			RealComission:      o.RealComission,
			TotalPayAmount:     o.TotalPayAmount,
			OrderStatus:        o.OrderStatus,
			ShopId:             o.ShopId,
			AllianceBizType:    o.AllianceBizType,
			PhaseId:            o.PhaseId,
			ShortId:            o.ShortId,
		})
	}

	return result, nil
}

// GetRefundOrderList 获取抖店售后/退款订单列表。
func (s *Service) GetRefundOrderList(ctx context.Context, params model.PlatformRefundOrderListParams) (*model.PlatformRefundOrderListResult, error) {
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

	req := afterSaleReq.New()
	req.SetConfig(s.config)
	req.GetParams().Page = int64(params.Page)
	req.GetParams().Size = int64(params.PageSize)
	if req.GetParams().Size <= 0 || req.GetParams().Size > 100 {
		req.GetParams().Size = 100
	}
	if params.OrderNo != "" {
		req.GetParams().OrderId = &params.OrderNo
	}
	if params.AftersaleId != "" {
		req.GetParams().AftersaleId = &params.AftersaleId
	}
	if params.AftersaleType != nil {
		req.GetParams().AftersaleType = params.AftersaleType
	}
	if len(params.StatusList) > 0 {
		req.GetParams().StandardAftersaleStatus = params.StatusList
	}

	cst := time.FixedZone("CST", 8*3600)
	if params.StartTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", params.StartTime, cst)
		if err != nil {
			return nil, fmt.Errorf("开始时间格式错误: %w", err)
		}
		ts := t.Unix()
		if params.UseUpdateTime {
			req.GetParams().UpdateStartTime = &ts
		} else {
			req.GetParams().StartTime = &ts
		}
	}
	if params.EndTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", params.EndTime, cst)
		if err != nil {
			return nil, fmt.Errorf("结束时间格式错误: %w", err)
		}
		ts := t.Unix()
		if params.UseUpdateTime {
			req.GetParams().UpdateEndTime = &ts
		} else {
			req.GetParams().EndTime = &ts
		}
	}
	if params.UseUpdateTime {
		req.GetParams().OrderBy = []string{"update_time asc"}
	} else {
		req.GetParams().OrderBy = []string{"apply_time asc"}
	}

	resp, err := req.Execute(token)
	if err != nil {
		return nil, fmt.Errorf("抖店售后/退款列表查询失败: %w", err)
	}
	if resp.Code != 10000 {
		return nil, fmt.Errorf("抖店售后/退款返回错误: code=%d, msg=%s, sub_msg=%s",
			resp.Code, resp.Msg, resp.SubMsg)
	}
	if resp.Data == nil {
		return &model.PlatformRefundOrderListResult{Page: params.Page, Size: int(req.GetParams().Size)}, nil
	}

	result := &model.PlatformRefundOrderListResult{
		Total:   int(resp.Data.Total),
		Page:    int(resp.Data.Page),
		Size:    int(resp.Data.Size),
		HasMore: resp.Data.HasMore,
	}
	for _, item := range resp.Data.Items {
		result.Orders = append(result.Orders, afterSaleItemToRefundOrders(item)...)
	}
	return result, nil
}

func afterSaleItemToRefundOrders(item afterSaleResp.ItemsItem) []model.PlatformRefundOrderItem {
	if item.AftersaleInfo == nil {
		return nil
	}

	rawJSON, _ := json.Marshal(item)
	base := model.PlatformRefundOrderItem{
		Platform:                   "douyin",
		AftersaleId:                item.AftersaleInfo.AftersaleId,
		AftersaleOrderType:         item.AftersaleInfo.AftersaleOrderType,
		AftersaleType:              item.AftersaleInfo.AftersaleType,
		AftersaleStatus:            item.AftersaleInfo.AftersaleStatus,
		AftersaleRefundType:        item.AftersaleInfo.AftersaleRefundType,
		RefundType:                 item.AftersaleInfo.RefundType,
		RefundStatus:               item.AftersaleInfo.RefundStatus,
		RefundAmount:               centsToYuan(item.AftersaleInfo.RefundAmount),
		RefundPostAmount:           centsToYuan(item.AftersaleInfo.RefundPostAmount),
		RefundTaxAmount:            centsToYuan(item.AftersaleInfo.RefundTaxAmount),
		ReturnPromotionAmount:      centsToYuan(item.AftersaleInfo.ReturnPromotionAmount),
		AftersaleNum:               item.AftersaleInfo.AftersaleNum,
		PartType:                   item.AftersaleInfo.PartType,
		ApplyTime:                  unixToTime(item.AftersaleInfo.ApplyTime),
		UpdateTime:                 unixToTime(item.AftersaleInfo.UpdateTime),
		CreateTime:                 unixToTime(item.AftersaleInfo.CreateTime),
		StatusDeadline:             unixToTime(item.AftersaleInfo.StatusDeadline),
		AftersaleStatusToFinalTime: unixToTime(item.AftersaleInfo.AftersaleStatusToFinalTime),
		ReturnLogisticsCode:        item.AftersaleInfo.ReturnLogisticsCode,
		ReturnLogisticsCompanyName: item.AftersaleInfo.ReturnLogisticsCompanyName,
		RiskDecisionCode:           item.AftersaleInfo.RiskDecisionCode,
		RiskDecisionReason:         item.AftersaleInfo.RiskDecisionReason,
		RiskDecisionDescription:    item.AftersaleInfo.RiskDecisionDescription,
		ArbitrateStatus:            item.AftersaleInfo.ArbitrateStatus,
		Remark:                     item.AftersaleInfo.Remark,
		RawJSON:                    string(rawJSON),
	}
	if item.TextPart != nil {
		base.AftersaleTypeText = item.TextPart.AftersaleTypeText
		base.AftersaleStatusText = item.TextPart.AftersaleStatusText
		base.AftersaleRefundTypeText = item.TextPart.AftersaleRefundTypeText
		base.ReasonText = item.TextPart.ReasonText
	}
	if item.OrderInfo != nil {
		base.ShopOrderId = item.OrderInfo.ShopOrderId
		base.OrderFlag = item.OrderInfo.OrderFlag
		if len(item.OrderInfo.RelatedOrderInfo) == 0 {
			return []model.PlatformRefundOrderItem{base}
		}
		orders := make([]model.PlatformRefundOrderItem, 0, len(item.OrderInfo.RelatedOrderInfo))
		for _, related := range item.OrderInfo.RelatedOrderInfo {
			refund := base
			fillRefundOrderRelatedInfo(&refund, related)
			orders = append(orders, refund)
		}
		return orders
	}
	return []model.PlatformRefundOrderItem{base}
}

func fillRefundOrderRelatedInfo(refund *model.PlatformRefundOrderItem, related afterSaleResp.RelatedOrderInfoItem) {
	refund.SkuOrderId = related.SkuOrderId
	refund.OrderStatus = related.OrderStatus
	refund.PayAmount = centsToYuan(related.PayAmount)
	refund.PostAmount = centsToYuan(related.PostAmount)
	refund.ProductNum = related.AftersaleItemNum
	if refund.ProductNum == 0 {
		refund.ProductNum = related.ItemNum
	}
	refund.OrderCreateTime = unixToTime(related.CreateTime)
	refund.ProductName = related.ProductName
	if related.ProductId > 0 {
		refund.ProductId = fmt.Sprintf("%d", related.ProductId)
	}
	refund.ProductImage = related.ProductImage
	refund.ShopSkuCode = related.ShopSkuCode
	refund.ProductPrice = centsToYuan(related.Price)
	if related.AftersalePayAmount > 0 {
		refund.RefundAmount = centsToYuan(related.AftersalePayAmount)
	}
	if related.AftersalePostAmount > 0 {
		refund.RefundPostAmount = centsToYuan(related.AftersalePostAmount)
	}
	if related.AftersaleTaxAmount > 0 {
		refund.RefundTaxAmount = centsToYuan(related.AftersaleTaxAmount)
	}
	if len(related.SkuSpec) > 0 {
		parts := make([]string, 0, len(related.SkuSpec))
		for _, spec := range related.SkuSpec {
			parts = append(parts, spec.Name+":"+spec.Value)
		}
		refund.ProductSpec = fmt.Sprintf("[%s]", joinStrings(parts))
	}
}

func centsToYuan(value int64) float64 {
	return float64(value) / 100
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

// AllianceOrderQuery 精选联盟订单查询参数（透传）
type AllianceOrderQuery struct {
	Size      int64
	Cursor    string
	StartTime string
	EndTime   string
	OrderIds  string
	TimeType  string
}

// QueryShopAllianceOrder 调用 /buyin/queryShopAllianceOrder，返回原始JSON字符串
func (s *Service) QueryShopAllianceOrder(ctx context.Context, token *doudian.AccessToken, q AllianceOrderQuery) (string, error) {
	s.initConfig()
	req := buyinReq.New()
	req.SetConfig(s.config)

	if q.Size > 0 {
		req.GetParams().Size = &q.Size
	}
	if q.Cursor != "" {
		cursor := q.Cursor
		req.GetParams().Cursor = &cursor
	}
	if q.StartTime != "" {
		st := q.StartTime
		req.GetParams().StartTime = &st
	}
	if q.EndTime != "" {
		et := q.EndTime
		req.GetParams().EndTime = &et
	}
	if q.OrderIds != "" {
		oi := q.OrderIds
		req.GetParams().OrderIds = &oi
	}
	if q.TimeType != "" {
		tt := q.TimeType
		req.GetParams().TimeType = &tt
	}

	rawJSON, err := req.GetClient().Request(req, token)
	if err != nil {
		return "", err
	}
	return rawJSON, nil
}

// DailyPullAllianceOrders 每天凌晨拉取昨天精选联盟订单
func (s *Service) DailyPullAllianceOrders(ctx context.Context) error {
	token, _, err := s.GetTokenForStore(ctx, 1)
	if err != nil {
		return fmt.Errorf("获取token失败: %w", err)
	}

	yesterday := time.Now().AddDate(0, 0, -1)
	startTime := yesterday.Format("2006-01-02") + " 00:00:00"
	endTime := yesterday.Format("2006-01-02") + " 23:59:59"

	cursor := "0"
	totalSaved := 0

	for {
		rawJSON, err := s.QueryShopAllianceOrder(ctx, token, AllianceOrderQuery{
			Size:      20,
			Cursor:    cursor,
			StartTime: startTime,
			EndTime:   endTime,
		})
		if err != nil {
			return fmt.Errorf("API调用失败: %w", err)
		}

		// 解析为通用结构（SDK struct 字段名和 API 实际返回不一致）
		var rawResp struct {
			Data struct {
				Cursor string                   `json:"cursor"`
				Orders []map[string]interface{} `json:"orders"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(rawJSON), &rawResp); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
		if len(rawResp.Data.Orders) == 0 {
			break
		}

		for _, o := range rawResp.Data.Orders {
			orderId := getStrVal(o, "order_id")
			if orderId == "" {
				continue
			}
			data := g.Map{
				"platform":            "douyin",
				"order_id":            orderId,
				"product_id":          getStrVal(o, "product_id"),
				"product_name":        getStrVal(o, "product_name"),
				"product_img":         getStrVal(o, "product_img"),
				"product_num":         getIntVal(o, "item_num"),
				"product_price":       float64(getIntVal(o, "pay_goods_amount")) / 100,
				"pay_amount":          float64(getIntVal(o, "total_pay_amount")) / 100,
				"commission_rate":     fmt.Sprintf("%d", getIntVal(o, "commission_rate")),
				"commission":          float64(getIntVal(o, "real_commission")) / 100,
				"order_status":        getStrVal(o, "flow_point"),
				"author_id":           getStrVal(o, "author_buyin_id"),
				"author_buyin_id":     getStrVal(o, "author_buyin_id"),
				"author_name":         getStrVal(o, "author_account"),
				"author_account":      getStrVal(o, "author_account"),
				"shop_name":           getStrVal(o, "shop_name"),
				"shop_id":             getStrVal(o, "shop_id"),
				"pay_time":            getStrVal(o, "pay_success_time"),
				"create_time":         getStrVal(o, "pay_success_time"),
				"finish_time":         getStrVal(o, "confirm_time"),
				"estimated_comission": getIntVal(o, "estimated_total_commission"),
				"real_comission":      getIntVal(o, "real_commission"),
				"total_pay_amount":    getIntVal(o, "total_pay_amount"),
				"alliance_biz_type":   getStrVal(o, "media_type"),
				"short_id":            getStrVal(o, "author_short_id"),
			}
			// upsert by order_id
			count, _ := g.DB().Model("alliance_orders").Ctx(ctx).Where("platform", "douyin").Where("order_id", orderId).Count()
			if count > 0 {
				g.DB().Model("alliance_orders").Ctx(ctx).Where("platform", "douyin").Where("order_id", orderId).Data(data).Update()
			} else {
				g.DB().Model("alliance_orders").Ctx(ctx).Data(data).Insert()
			}
			totalSaved++
		}

		cursor = rawResp.Data.Cursor
		if cursor == "" {
			break
		}
	}

	g.Log().Infof(ctx, "[AlliancePuller] 拉取完成: %s, 保存 %d 条", startTime, totalSaved)
	return nil
}

func getStrVal(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return fmt.Sprintf("%.0f", s)
	default:
		return fmt.Sprint(s)
	}
}

func getIntVal(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		return 0
	default:
		return 0
	}
}
