package system

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-crawler/internal/service"
	"go-crawler/internal/service/platform/wechatshop"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// WechatShopOrderPull 手动触发微信小店订单拉取 POST /api/wechat-shop-order/trigger-pull
func (c *ShipinhaoOrderController) WechatShopOrderPull(r *ghttp.Request) {
	params := wechatshop.PullParams{
		StartTime:   r.Get("startTime").String(),
		EndTime:     r.Get("endTime").String(),
		StoreId:     r.Get("storeId", 0).Int(),
		TimeType:    r.Get("timeType", "create").String(),
		Concurrency: r.Get("concurrency", 5).Int(),
	}

	reqParams, _ := json.Marshal(params)
	logId, err := service.TaskLogStart(r.Context(), "微信小店订单手动拉取", "wechat_shop_order_pull", "/api/wechat-shop-order/trigger-pull", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[WechatShopOrder] 写入任务日志失败: %v", err)
	}

	g.Log().Infof(r.Context(), "[WechatShopOrder] 收到手动拉取请求: startTime=%s endTime=%s storeId=%d timeType=%s logId=%d",
		params.StartTime, params.EndTime, params.StoreId, params.TimeType, logId)

	go c.runWechatShopOrderPull(context.Background(), params, logId, r.GetCtxVar("userId").Int64())

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logId}, "msg": "任务已启动，后台拉取中"})
}

func (c *ShipinhaoOrderController) runWechatShopOrderPull(ctx context.Context, params wechatshop.PullParams, logId, userId int64) {
	start := time.Now()
	result, err := wechatshop.NewService().PullOrders(ctx, params)

	if err != nil {
		durationMs := int(time.Since(start).Milliseconds())
		if logId > 0 {
			service.TaskLogFailed(ctx, logId, err.Error(), durationMs)
			service.CreateNotification(ctx, "微信小店订单拉取失败", fmt.Sprintf(`{"error":"%s"}`, err.Error()), "task_result", "wechat_shop_order_pull", logId, service.NotifyLevelImportant, userId)
		}
		return
	}

	matchSvc := service.NewOrderMatchService()
	accountMatchResult, accountMatchErr := matchSvc.BackfillShipinhaoTalentFromAccounts(ctx, params.StartTime, params.EndTime, params.StoreId, params.TimeType)
	autoMatchResult, matchErr := matchSvc.AutoMatchShipinhaoPulledOrders(ctx, params.StartTime, params.EndTime, params.StoreId, params.TimeType)
	resp := g.Map{
		"pull":         result,
		"accountMatch": accountMatchResult,
		"autoMatch":    autoMatchResult,
	}
	if accountMatchErr != nil {
		g.Log().Warningf(ctx, "[WechatShopOrder] 微信小店订单账号表达人回填失败: %v", accountMatchErr)
		resp["accountMatchError"] = accountMatchErr.Error()
	}
	if matchErr != nil {
		g.Log().Warningf(ctx, "[WechatShopOrder] 微信小店订单自动达人匹配失败: %v", matchErr)
		resp["autoMatchError"] = matchErr.Error()
	}
	respBody, _ := json.Marshal(resp)
	if logId > 0 {
		durationMs := int(time.Since(start).Milliseconds())
		service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
		level := service.NotifyLevelNormal
		if result.FailedCount > 0 || accountMatchErr != nil || matchErr != nil {
			level = service.NotifyLevelImportant
		}
		service.CreateNotification(ctx, "微信小店订单拉取完成", string(respBody), "task_result", "wechat_shop_order_pull", logId, level, userId)
	}
}
