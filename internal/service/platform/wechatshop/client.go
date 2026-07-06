package wechatshop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

const (
	baseURL              = "https://api.weixin.qq.com"
	ordersTable          = "shipinhao_orders"
	authTable            = "wechat_shop_auth"
	maxOrderListDuration = 7 * 24 * time.Hour
)

type PullParams struct {
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	StoreId     int    `json:"storeId"`
	TimeType    string `json:"timeType"`
	Concurrency int    `json:"concurrency"`
}

type PullResult struct {
	StoreCount  int      `json:"storeCount"`
	TotalOrders int      `json:"totalOrders"`
	TotalSaved  int      `json:"totalSaved"`
	FailedCount int      `json:"failedCount"`
	Errors      []string `json:"errors,omitempty"`
}

type Service struct {
	httpClient *http.Client
}

func NewService() *Service {
	return &Service{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type authConfig struct {
	AuthId      int64
	StorePk     int
	StoreId     string
	StoreName   string
	AppId       string
	AppSecret   string
	AccessToken string
	ExpiresAt   *time.Time
}

type wxBaseResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type tokenResp struct {
	wxBaseResp
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type orderListResp struct {
	wxBaseResp
	OrderIdList []string `json:"order_id_list"`
	NextKey     string   `json:"next_key"`
	HasMore     bool     `json:"has_more"`
}

type orderDetailResp struct {
	wxBaseResp
	Order map[string]interface{} `json:"order"`
}

// EnsureSchema creates the auth table and adds API-specific fields when the SQL
// migration has not been applied yet.
func (s *Service) EnsureSchema(ctx context.Context) error {
	_, err := g.DB().Exec(ctx, `
CREATE TABLE IF NOT EXISTS wechat_shop_auth (
  id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  store_id INT NOT NULL COMMENT '关联 store.id',
  appid VARCHAR(128) NOT NULL DEFAULT '' COMMENT '微信小店/服务商调用 appid',
  app_secret VARCHAR(255) NOT NULL DEFAULT '' COMMENT '微信小店/服务商调用 appsecret',
  access_token VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '接口调用凭证',
  expires_at DATETIME DEFAULT NULL COMMENT 'access_token 过期时间',
  last_pull_time DATETIME DEFAULT NULL COMMENT '最近成功拉取时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_wechat_shop_auth_store (store_id),
  KEY idx_wechat_shop_auth_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='微信小店接口授权配置'`)
	if err != nil {
		return err
	}

	columns := []struct {
		name       string
		definition string
	}{
		{"sku_id", "`sku_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '微信小店SKU ID' AFTER `product_id`"},
		{"out_product_id", "`out_product_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '外部商品ID' AFTER `sku_id`"},
		{"out_sku_id", "`out_sku_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '外部SKU ID' AFTER `out_product_id`"},
		{"openid", "`openid` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '买家openid' AFTER `receiver_phone`"},
		{"wechat_order_status", "`wechat_order_status` INT NOT NULL DEFAULT 0 COMMENT '微信小店订单状态码' AFTER `order_status`"},
		{"wechat_order_status_desc", "`wechat_order_status_desc` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '微信小店订单状态描述' AFTER `wechat_order_status`"},
		{"order_update_time", "`order_update_time` DATETIME DEFAULT NULL COMMENT '订单更新时间' AFTER `order_submit_time`"},
		{"source_info", "`source_info` JSON DEFAULT NULL COMMENT '订单来源信息' AFTER `flow_channel`"},
		{"commission_info", "`commission_info` JSON DEFAULT NULL COMMENT '分账/佣金信息' AFTER `source_info`"},
		{"raw_json", "`raw_json` JSON DEFAULT NULL COMMENT '微信小店订单原始详情JSON' AFTER `expect_arrive_time`"},
		{"api_sync_time", "`api_sync_time` DATETIME DEFAULT NULL COMMENT 'API同步时间' AFTER `raw_json`"},
	}
	for _, column := range columns {
		if err := s.ensureColumn(ctx, ordersTable, column.name, column.definition); err != nil {
			return err
		}
	}

	for name, column := range map[string]string{
		"idx_sku_id":              "sku_id",
		"idx_wechat_order_status": "wechat_order_status",
		"idx_api_sync_time":       "api_sync_time",
	} {
		if err := s.ensureIndex(ctx, name, column); err != nil {
			g.Log().Warningf(ctx, "[WechatShop] 创建索引%s失败: %v", name, err)
		}
	}
	return nil
}

func (s *Service) ensureColumn(ctx context.Context, table, column, definition string) error {
	row, err := g.DB().GetOne(ctx, `
SELECT COUNT(*) AS cnt
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, column)
	if err != nil {
		return err
	}
	if row["cnt"].Int() > 0 {
		return nil
	}
	_, err = g.DB().Exec(ctx, fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", table, definition))
	return err
}

func (s *Service) ensureIndex(ctx context.Context, name, column string) error {
	row, err := g.DB().GetOne(ctx, `
SELECT COUNT(*) AS cnt
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`, ordersTable, name)
	if err != nil {
		return err
	}
	if row["cnt"].Int() > 0 {
		return nil
	}
	_, err = g.DB().Exec(ctx, fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` (`%s`)", ordersTable, name, column))
	return err
}

func (s *Service) PullOrders(ctx context.Context, params PullParams) (*PullResult, error) {
	if err := s.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("初始化微信小店订单表结构失败: %w", err)
	}

	start, end, err := parseTimeRange(params.StartTime, params.EndTime)
	if err != nil {
		return nil, err
	}
	if params.Concurrency <= 0 {
		params.Concurrency = 5
	}
	if params.Concurrency > 20 {
		params.Concurrency = 20
	}
	if params.TimeType == "" {
		params.TimeType = "create"
	}
	if params.TimeType != "create" && params.TimeType != "update" {
		return nil, fmt.Errorf("timeType 仅支持 create 或 update")
	}

	auths, err := s.loadAuthConfigs(ctx, params.StoreId)
	if err != nil {
		return nil, err
	}
	if len(auths) == 0 {
		return nil, fmt.Errorf("未找到可用的微信小店授权配置，请先在 %s 表配置 appid/app_secret", authTable)
	}

	result := &PullResult{StoreCount: len(auths), Errors: make([]string, 0)}
	for _, auth := range auths {
		token, err := s.getAccessToken(ctx, auth)
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: 获取access_token失败: %v", auth.StoreName, err))
			continue
		}

		storeResult := s.pullStoreOrders(ctx, auth, token, start, end, params.TimeType, params.Concurrency)
		result.TotalOrders += storeResult.TotalOrders
		result.TotalSaved += storeResult.TotalSaved
		result.FailedCount += storeResult.FailedCount
		result.Errors = append(result.Errors, storeResult.Errors...)
		if storeResult.FailedCount == 0 {
			_, _ = g.DB().Model(authTable).Ctx(ctx).Where("store_id", auth.StorePk).
				Data(g.Map{"last_pull_time": time.Now().Format("2006-01-02 15:04:05")}).
				Update()
		}
	}

	if result.FailedCount > 0 && result.TotalSaved == 0 {
		return result, fmt.Errorf("微信小店订单拉取失败，失败数: %d", result.FailedCount)
	}
	return result, nil
}

func (s *Service) pullStoreOrders(ctx context.Context, auth authConfig, token string, start, end time.Time, timeType string, concurrency int) *PullResult {
	result := &PullResult{StoreCount: 1, Errors: make([]string, 0)}
	for _, tr := range splitTimeRange(start, end, maxOrderListDuration) {
		nextKey := ""
		for {
			ids, newNextKey, hasMore, err := s.getOrderIdList(ctx, token, tr.Start, tr.End, timeType, nextKey)
			if err != nil {
				result.FailedCount++
				result.Errors = append(result.Errors, fmt.Sprintf("%s %s~%s: 拉取订单列表失败: %v", auth.StoreName, tr.Start.Format(time.DateTime), tr.End.Format(time.DateTime), err))
				break
			}
			result.TotalOrders += len(ids)
			saved, failed, errors := s.fetchAndSaveDetails(ctx, auth, token, ids, concurrency)
			result.TotalSaved += saved
			result.FailedCount += failed
			result.Errors = append(result.Errors, errors...)
			if !hasMore || newNextKey == "" {
				break
			}
			nextKey = newNextKey
		}
	}
	return result
}

func (s *Service) fetchAndSaveDetails(ctx context.Context, auth authConfig, token string, ids []string, concurrency int) (int, int, []string) {
	if len(ids) == 0 {
		return 0, 0, nil
	}

	var mu sync.Mutex
	var saved, failed int
	errors := make([]string, 0)
	jobs := make(chan string)
	wg := sync.WaitGroup{}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for orderId := range jobs {
				order, err := s.getOrderDetail(ctx, token, orderId)
				if err != nil {
					mu.Lock()
					failed++
					errors = append(errors, fmt.Sprintf("%s: 订单%s详情失败: %v", auth.StoreName, orderId, err))
					mu.Unlock()
					continue
				}
				count, err := s.saveOrderDetail(ctx, auth, order)
				mu.Lock()
				if err != nil {
					failed++
					errors = append(errors, fmt.Sprintf("%s: 订单%s入库失败: %v", auth.StoreName, orderId, err))
				} else {
					saved += count
				}
				mu.Unlock()
			}
		}()
	}

	for _, id := range ids {
		select {
		case <-ctx.Done():
			break
		case jobs <- id:
		}
	}
	close(jobs)
	wg.Wait()
	return saved, failed, errors
}

func (s *Service) loadAuthConfigs(ctx context.Context, storeId int) ([]authConfig, error) {
	authTableExists, err := s.tableExists(ctx, authTable)
	if err != nil {
		return nil, err
	}

	cfgAppId := g.Cfg().MustGet(ctx, "platform.shipinhao.appKey", "").String()
	cfgSecret := g.Cfg().MustGet(ctx, "platform.shipinhao.appSecret", "").String()

	m := g.DB().Model("store s").Ctx(ctx).
		Where("s.status", 1).
		Where("s.platform", "shipinhao")
	if storeId > 0 {
		m = m.Where("s.id", storeId)
	}
	if authTableExists {
		m = m.LeftJoin(authTable+" a", "a.store_id=s.id").
			Fields("s.id AS store_pk, s.store_id, s.store_name, a.id AS auth_id, a.appid, a.app_secret, a.access_token, a.expires_at")
	} else {
		m = m.Fields("s.id AS store_pk, s.store_id, s.store_name")
	}

	rows, err := m.All()
	if err != nil {
		return nil, err
	}

	auths := make([]authConfig, 0, len(rows))
	for _, row := range rows {
		appId := strings.TrimSpace(row["appid"].String())
		secret := strings.TrimSpace(row["app_secret"].String())
		if appId == "" {
			appId = cfgAppId
		}
		if secret == "" {
			secret = cfgSecret
		}
		if appId == "" || secret == "" {
			continue
		}
		var expiresAt *time.Time
		if s := row["expires_at"].String(); s != "" {
			if t, err := time.ParseInLocation(time.DateTime, s, time.Local); err == nil {
				expiresAt = &t
			}
		}
		auths = append(auths, authConfig{
			AuthId:      row["auth_id"].Int64(),
			StorePk:     row["store_pk"].Int(),
			StoreId:     row["store_id"].String(),
			StoreName:   row["store_name"].String(),
			AppId:       appId,
			AppSecret:   secret,
			AccessToken: row["access_token"].String(),
			ExpiresAt:   expiresAt,
		})
	}
	return auths, nil
}

func (s *Service) tableExists(ctx context.Context, table string) (bool, error) {
	row, err := g.DB().GetOne(ctx, `
SELECT COUNT(*) AS cnt
FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = ?`, table)
	if err != nil {
		return false, err
	}
	return row["cnt"].Int() > 0, nil
}

func (s *Service) getAccessToken(ctx context.Context, auth authConfig) (string, error) {
	if auth.AccessToken != "" && auth.ExpiresAt != nil && time.Until(*auth.ExpiresAt) > 5*time.Minute {
		return auth.AccessToken, nil
	}

	reqBody := map[string]interface{}{
		"grant_type":    "client_credential",
		"appid":         auth.AppId,
		"secret":        auth.AppSecret,
		"force_refresh": false,
	}
	var resp tokenResp
	if err := s.postJSON(ctx, baseURL+"/cgi-bin/stable_token", reqBody, &resp); err != nil {
		return "", err
	}
	if resp.ErrCode != 0 {
		return "", fmt.Errorf("errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("微信返回access_token为空")
	}

	expiresAt := time.Now().Add(time.Duration(resp.ExpiresIn-300) * time.Second)
	if auth.AuthId > 0 {
		_, _ = g.DB().Model(authTable).Ctx(ctx).Where("id", auth.AuthId).Data(g.Map{
			"access_token": resp.AccessToken,
			"expires_at":   expiresAt.Format(time.DateTime),
		}).Update()
	}
	return resp.AccessToken, nil
}

func (s *Service) getOrderIdList(ctx context.Context, token string, start, end time.Time, timeType, nextKey string) ([]string, string, bool, error) {
	body := map[string]interface{}{
		"page_size": 100,
	}
	rangeBody := map[string]int64{
		"start_time": start.Unix(),
		"end_time":   end.Unix(),
	}
	if timeType == "update" {
		body["update_time_range"] = rangeBody
	} else {
		body["create_time_range"] = rangeBody
	}
	if nextKey != "" {
		body["next_key"] = nextKey
	}

	var resp orderListResp
	url := fmt.Sprintf("%s/channels/ec/order/list/get?access_token=%s", baseURL, token)
	if err := s.postJSON(ctx, url, body, &resp); err != nil {
		return nil, "", false, err
	}
	if resp.ErrCode != 0 {
		return nil, "", false, fmt.Errorf("errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return resp.OrderIdList, resp.NextKey, resp.HasMore, nil
}

func (s *Service) getOrderDetail(ctx context.Context, token, orderId string) (map[string]interface{}, error) {
	var resp orderDetailResp
	url := fmt.Sprintf("%s/channels/ec/order/get?access_token=%s", baseURL, token)
	if err := s.postJSON(ctx, url, map[string]interface{}{"order_id": orderId}, &resp); err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("errcode=%d errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	if len(resp.Order) == 0 {
		return nil, fmt.Errorf("微信返回订单为空")
	}
	return resp.Order, nil
}

func (s *Service) postJSON(ctx context.Context, url string, payload interface{}, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("解析微信响应失败: %w, body=%s", err, string(respBody))
	}
	return nil
}

func (s *Service) saveOrderDetail(ctx context.Context, auth authConfig, order map[string]interface{}) (int, error) {
	orderId := getString(order, "order_id")
	if orderId == "" {
		return 0, fmt.Errorf("订单缺少order_id")
	}

	detail := getMap(order, "order_detail")
	payInfo := getMap(detail, "pay_info")
	priceInfo := getMap(detail, "price_info")
	deliveryInfo := getMap(detail, "delivery_info")
	addressInfo := getMap(deliveryInfo, "address_info")
	extInfo := getMap(detail, "ext_info")
	aftersaleDetail := getMap(order, "aftersale_detail")
	sourceInfos := getSlice(detail, "source_infos")
	commissionInfos := getSlice(detail, "commission_infos")
	products := getSlice(detail, "product_infos")
	if len(products) == 0 {
		products = []interface{}{map[string]interface{}{}}
	}

	rawJSON, _ := json.Marshal(order)
	orderStatus := getInt64(order, "status")
	now := time.Now().Format(time.DateTime)
	rows := make([]g.Map, 0, len(products))
	for _, productRaw := range products {
		product, _ := productRaw.(map[string]interface{})
		if product == nil {
			product = map[string]interface{}{}
		}
		skuId := getString(product, "sku_id")
		source := findBySku(sourceInfos, skuId)
		commission := findBySku(commissionInfos, skuId)
		sourceJSON, _ := json.Marshal(source)
		commissionJSON, _ := json.Marshal(commission)
		specJSON, _ := json.Marshal(getSlice(product, "sku_attrs"))

		data := g.Map{
			"shop_name":                truncateLocal(auth.StoreName, 255),
			"order_no":                 truncateLocal(orderId, 100),
			"product_name":             truncateLocal(firstString(product, "title", "product_name"), 500),
			"product_spec":             truncateLocal(string(specJSON), 255),
			"product_num":              intOrDefault(getInt64(product, "sku_cnt"), 1),
			"product_id":               truncateLocal(getString(product, "product_id"), 100),
			"sku_id":                   truncateLocal(skuId, 100),
			"out_product_id":           truncateLocal(getString(product, "out_product_id"), 100),
			"out_sku_id":               truncateLocal(getString(product, "out_sku_id"), 100),
			"seller_code":              truncateLocal(getString(product, "sku_code"), 100),
			"product_price":            centsToYuan(getInt64(product, "sale_price")),
			"order_pay_amount":         centsToYuan(productPayAmount(product)),
			"shipping_fee":             centsToYuan(getInt64(priceInfo, "freight")),
			"total_discount":           centsToYuan(totalDiscount(product, priceInfo)),
			"platform_discount":        centsToYuan(firstInt64(product, "cash_coupon_discounted_price", "platform_activity_merchant_discounted_price")),
			"seller_discount":          centsToYuan(firstInt64(product, "merchant_discounted_price", "discounted_price")),
			"talent_discount":          centsToYuan(getInt64(product, "finder_discounted_price")),
			"seller_change_price":      centsToYuan(getInt64(product, "change_price")),
			"pay_discount":             centsToYuan(getInt64(product, "deduction_price")),
			"red_deduction":            centsToYuan(getInt64(product, "deduction_price")),
			"pay_type":                 paymentMethodDesc(getInt64(payInfo, "payment_method")),
			"receiver_name":            truncateLocal(getString(addressInfo, "user_name"), 100),
			"receiver_phone":           truncateLocal(getString(addressInfo, "tel_number"), 50),
			"openid":                   truncateLocal(getString(order, "openid"), 128),
			"province":                 truncateLocal(getString(addressInfo, "province_name"), 50),
			"city":                     truncateLocal(getString(addressInfo, "city_name"), 50),
			"district":                 truncateLocal(getString(addressInfo, "county_name"), 255),
			"street":                   truncateLocal(getString(addressInfo, "house_number"), 255),
			"address_detail":           truncateLocal(getString(addressInfo, "detail_info"), 500),
			"buyer_note":               truncateLocal(getString(detail, "buyer_notes"), 500),
			"seller_note":              truncateLocal(getString(detail, "merchant_notes"), 500),
			"order_submit_time":        unixTimeOrNil(getInt64(order, "create_time")),
			"order_update_time":        unixTimeOrNil(getInt64(order, "update_time")),
			"order_finish_time":        unixTimeOrNil(firstInt64(order, "finish_time", "settle_time")),
			"pay_finish_time":          unixTimeOrNil(getInt64(payInfo, "pay_time")),
			"flow_source":              truncateLocal(sourceDesc(source), 50),
			"order_status":             orderStatusDesc(orderStatus),
			"wechat_order_status":      orderStatus,
			"wechat_order_status_desc": orderStatusDesc(orderStatus),
			"promise_send_time":        unixTimeOrNil(getInt64(product, "delivery_deadline")),
			"order_type":               truncateLocal(orderSceneDesc(getInt64(extInfo, "order_scene")), 50),
			"talent_id":                truncateLocal(firstString(source, "account_id", "finder_id"), 100),
			"talent_name":              truncateLocal(getString(source, "account_nickname"), 100),
			"account_type":             truncateLocal(accountTypeDesc(getInt64(source, "account_type")), 50),
			"store_id":                 truncateLocal(auth.StoreId, 100),
			"after_sale_status":        truncateLocal(afterSaleStatus(product, aftersaleDetail), 50),
			"cancel_reason":            truncateLocal(getString(order, "cancel_reason"), 255),
			"send_user":                truncateLocal(firstDeliveryString(deliveryInfo, "delivery_name"), 50),
			"send_user_detail":         truncateLocal(firstDeliveryString(deliveryInfo, "delivery_id"), 100),
			"send_time":                unixTimeOrNil(firstInt64(deliveryInfo, "ship_done_time")),
			"expect_arrive_time":       unixTimeOrNil(getInt64(getMap(order, "intra_city_order_info"), "predict_arrive_time")),
			"source_info":              string(sourceJSON),
			"commission_info":          string(commissionJSON),
			"raw_json":                 string(rawJSON),
			"api_sync_time":            now,
			"create_time":              now,
		}
		rows = append(rows, data)
	}

	return len(rows), g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(ordersTable).Ctx(ctx).Where("order_no", orderId).Delete(); err != nil {
			return err
		}
		_, err := tx.Model(ordersTable).Ctx(ctx).Data(rows).Insert()
		return err
	})
}

type timeRange struct {
	Start time.Time
	End   time.Time
}

func splitTimeRange(start, end time.Time, maxDuration time.Duration) []timeRange {
	ranges := make([]timeRange, 0)
	for cur := start; cur.Before(end); {
		next := cur.Add(maxDuration)
		if next.After(end) {
			next = end
		}
		ranges = append(ranges, timeRange{Start: cur, End: next})
		cur = next.Add(time.Second)
	}
	return ranges
}

func parseTimeRange(startRaw, endRaw string) (time.Time, time.Time, error) {
	start, err := parseFlexibleTime(startRaw, false)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("开始时间不正确: %w", err)
	}
	end, err := parseFlexibleTime(endRaw, true)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("结束时间不正确: %w", err)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("结束时间不能早于开始时间")
	}
	return start, end, nil
}

func parseFlexibleTime(raw string, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("不能为空")
	}
	layouts := []string{time.DateTime, "2006-01-02 15:04", time.DateOnly}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			if layout == time.DateOnly && endOfDay {
				t = t.Add(24*time.Hour - time.Second)
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("请使用 YYYY-MM-DD 或 YYYY-MM-DD HH:mm:ss")
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return map[string]interface{}{}
}

func getSlice(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}

func getString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if math.Trunc(x) == x {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	case json.Number:
		return x.String()
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func getInt64(m map[string]interface{}, key string) int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		i, _ := strconv.ParseInt(x, 10, 64)
		return i
	case json.Number:
		i, _ := x.Int64()
		return i
	default:
		return 0
	}
}

func firstString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v := getString(m, key); v != "" {
			return v
		}
	}
	return ""
}

func firstInt64(m map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		if v := getInt64(m, key); v != 0 {
			return v
		}
	}
	return 0
}

func findBySku(items []interface{}, skuId string) map[string]interface{} {
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if skuId == "" || getString(m, "sku_id") == skuId {
			return m
		}
	}
	return map[string]interface{}{}
}

func centsToYuan(cents int64) float64 {
	return float64(cents) / 100
}

func intOrDefault(v int64, def int) int {
	if v == 0 {
		return def
	}
	return int(v)
}

func unixTimeOrNil(ts int64) interface{} {
	if ts <= 0 {
		return nil
	}
	return time.Unix(ts, 0).Format(time.DateTime)
}

func totalDiscount(product, priceInfo map[string]interface{}) int64 {
	keys := []string{
		"merchant_discounted_price",
		"finder_discounted_price",
		"deduction_price",
		"bulkbuy_discounted_price",
		"national_subsidy_discounted_price",
		"cash_coupon_discounted_price",
		"limited_discount_discounted_price",
		"normal_friends_like_discounted_price",
	}
	var total int64
	for _, key := range keys {
		total += getInt64(product, key)
	}
	if total == 0 {
		total = getInt64(priceInfo, "discounted_price") +
			getInt64(priceInfo, "merchant_discounted_price") +
			getInt64(priceInfo, "finder_discounted_price")
	}
	return total
}

func productPayAmount(product map[string]interface{}) int64 {
	if amount := firstInt64(product, "real_price", "estimate_price", "change_price"); amount > 0 {
		return amount
	}
	return getInt64(product, "sale_price") * int64(intOrDefault(getInt64(product, "sku_cnt"), 1))
}

func paymentMethodDesc(v int64) string {
	switch v {
	case 1:
		return "微信支付"
	case 2:
		return "先用后付"
	case 3:
		return "抽奖0元"
	case 4:
		return "会员积分兑换"
	default:
		if v == 0 {
			return ""
		}
		return strconv.FormatInt(v, 10)
	}
}

func orderStatusDesc(v int64) string {
	switch v {
	case 10:
		return "待付款"
	case 12:
		return "礼物待收下"
	case 13:
		return "一起买待成团"
	case 20:
		return "待发货"
	case 21:
		return "部分发货"
	case 30:
		return "待收货"
	case 100:
		return "完成"
	case 250:
		return "订单取消"
	default:
		if v == 0 {
			return ""
		}
		return strconv.FormatInt(v, 10)
	}
}

func accountTypeDesc(v int64) string {
	switch v {
	case 1:
		return "视频号"
	case 2:
		return "公众号"
	case 3:
		return "小程序"
	case 4:
		return "企业微信"
	case 5:
		return "带货达人"
	case 6:
		return "服务号"
	case 1000:
		return "带货机构"
	default:
		if v == 0 {
			return ""
		}
		return strconv.FormatInt(v, 10)
	}
}

func sourceDesc(source map[string]interface{}) string {
	accountType := accountTypeDesc(getInt64(source, "account_type"))
	saleChannel := getInt64(source, "sale_channel")
	if accountType == "" && saleChannel == 0 {
		return ""
	}
	if saleChannel == 100 {
		return accountType + "联盟带货"
	}
	if saleChannel == 101 {
		return accountType + "机构推广"
	}
	return accountType
}

func orderSceneDesc(v int64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(v, 10)
}

func afterSaleStatus(product, aftersale map[string]interface{}) string {
	if list := getSlice(aftersale, "aftersale_order_list"); len(list) > 0 {
		if m, ok := list[0].(map[string]interface{}); ok {
			return getString(m, "status")
		}
	}
	onCnt := getInt64(product, "on_aftersale_sku_cnt")
	finishCnt := getInt64(product, "finish_aftersale_sku_cnt")
	if onCnt > 0 {
		return "售后中"
	}
	if finishCnt > 0 {
		return "售后完成"
	}
	return ""
}

func firstDeliveryString(deliveryInfo map[string]interface{}, key string) string {
	items := getSlice(deliveryInfo, "delivery_product_info")
	if len(items) == 0 {
		return ""
	}
	if m, ok := items[0].(map[string]interface{}); ok {
		return getString(m, key)
	}
	return ""
}

func truncateLocal(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
