package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/xuri/excelize/v2"
)

type OrderService struct{}

type externalTalentInfo struct {
	uid                   string
	accountName           string
	talentName            string
	showID                string
	teamName              string
	onlineCommissionRate  float64
	offlineCommissionRate float64
}

type enrichedDouyinOrderCacheEntry struct {
	expiresAt time.Time
	orders    []g.Map
}

type enrichedDouyinOrderLoad struct {
	done     chan struct{}
	revision uint64
	orders   []g.Map
	err      error
}

var enrichedDouyinOrderCache sync.Map
var enrichedDouyinOrderLoadMu sync.Mutex
var enrichedDouyinOrderLoads = make(map[string]*enrichedDouyinOrderLoad)
var enrichedDouyinOrderCacheRevision uint64

const (
	enrichedDouyinOrderCacheTTL     = time.Minute
	enrichedDouyinOrderBuildTimeout = 2 * time.Minute
)

const orderListInternalCommissionRate = 0.05

func NewOrderService() *OrderService {
	return &OrderService{}
}

func (s *OrderService) loadEnrichedDouyinOrders(ctx context.Context, startDate, endDate string) ([]g.Map, error) {
	qianchuanBlankOrderInferenceEnabled := IsQianchuanPlanBlankOrderInferenceEnabled(ctx)
	ctx = withQianchuanPlanBlankOrderInferenceSetting(ctx, qianchuanBlankOrderInferenceEnabled)
	cacheKey := startDate + "\x00" + endDate + "\x00qianchuanBlankInference=" + strconv.FormatBool(qianchuanBlankOrderInferenceEnabled)
	cacheRevision := atomic.LoadUint64(&enrichedDouyinOrderCacheRevision)
	if orders, ok := getCachedEnrichedDouyinOrders(cacheKey); ok {
		return orders, nil
	}

	enrichedDouyinOrderLoadMu.Lock()
	if orders, ok := getCachedEnrichedDouyinOrders(cacheKey); ok {
		enrichedDouyinOrderLoadMu.Unlock()
		return orders, nil
	}
	load, loading := enrichedDouyinOrderLoads[cacheKey]
	if !loading || load.revision != cacheRevision {
		load = &enrichedDouyinOrderLoad{done: make(chan struct{}), revision: cacheRevision}
		enrichedDouyinOrderLoads[cacheKey] = load
		go s.buildEnrichedDouyinOrderCache(context.WithoutCancel(ctx), cacheKey, startDate, endDate, load)
	}
	enrichedDouyinOrderLoadMu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-load.done:
		return load.orders, load.err
	}
}

func getCachedEnrichedDouyinOrders(cacheKey string) ([]g.Map, bool) {
	cached, ok := enrichedDouyinOrderCache.Load(cacheKey)
	if !ok {
		return nil, false
	}
	entry := cached.(enrichedDouyinOrderCacheEntry)
	if time.Now().Before(entry.expiresAt) {
		return entry.orders, true
	}
	enrichedDouyinOrderCache.Delete(cacheKey)
	return nil, false
}

// ClearEnrichedDouyinOrderCache 清空抖音订单增强缓存。
// 财务归属口径配置变更后需要立即清理缓存，避免短时间内继续使用旧口径。
func ClearEnrichedDouyinOrderCache() {
	atomic.AddUint64(&enrichedDouyinOrderCacheRevision, 1)
	enrichedDouyinOrderCache.Range(func(key, _ interface{}) bool {
		enrichedDouyinOrderCache.Delete(key)
		return true
	})
}

func (s *OrderService) buildEnrichedDouyinOrderCache(parent context.Context, cacheKey, startDate, endDate string, load *enrichedDouyinOrderLoad) {
	buildCtx, cancel := context.WithTimeout(parent, enrichedDouyinOrderBuildTimeout)
	defer cancel()

	load.orders, load.err = s.queryEnrichedDouyinOrders(buildCtx, startDate, endDate)
	if load.err == nil && atomic.LoadUint64(&enrichedDouyinOrderCacheRevision) == load.revision {
		enrichedDouyinOrderCache.Store(cacheKey, enrichedDouyinOrderCacheEntry{
			expiresAt: time.Now().Add(enrichedDouyinOrderCacheTTL),
			orders:    load.orders,
		})
	}

	enrichedDouyinOrderLoadMu.Lock()
	if enrichedDouyinOrderLoads[cacheKey] == load {
		delete(enrichedDouyinOrderLoads, cacheKey)
	}
	close(load.done)
	enrichedDouyinOrderLoadMu.Unlock()
}

func (s *OrderService) queryEnrichedDouyinOrders(ctx context.Context, startDate, endDate string) ([]g.Map, error) {
	rows, err := g.DB().GetAll(ctx, `
		SELECT o.*, COALESCE(g.base_price, 0) AS base_price, COALESCE(store_match.store_id, '') AS order_store_id
		FROM order_info o
		LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) g ON o.seller_code = g.goods_code
		LEFT JOIN (
			SELECT s.store_id, s.store_name
			FROM store s
			INNER JOIN (
				SELECT MIN(id) AS id
				FROM store
				WHERE platform = 'douyin' AND status = 1
				GROUP BY store_name
			) picked ON picked.id = s.id
		) store_match ON store_match.store_name COLLATE utf8mb4_unicode_ci = o.shop_name COLLATE utf8mb4_unicode_ci
		WHERE o.order_submit_time >= ?
		  AND o.order_submit_time < ?
		  AND o.main_order_no IS NOT NULL
		  AND o.main_order_no != ''
		ORDER BY o.id ASC
	`, startDate+" 00:00:00", CalcEndDateNext(endDate)+" 00:00:00")
	if err != nil {
		return nil, err
	}
	orders, err := s.enrichDouyinOrderRows(ctx, rows)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func orderListTalentUIDExpr(alias string) string {
	return fmt.Sprintf(
		"COALESCE(NULLIF(NULLIF(%s.talent_id, ''), '0'), NULLIF(an.uid, ''), NULLIF(au.uid, ''))",
		alias,
	)
}

func orderListTalentNameExpr(alias string) string {
	return fmt.Sprintf(
		"COALESCE(NULLIF(%s.talent_name, ''), NULLIF(au.account_name, ''), NULLIF(an.account_name, ''), NULLIF(ao.author_account, ''))",
		alias,
	)
}

func orderListDouyinAccountExpr() string {
	return "COALESCE(NULLIF(an.platform_nickname, ''), NULLIF(au.platform_nickname, ''), NULLIF(ao.short_id, ''))"
}

func orderListTeamOwnerExpr(alias string) string {
	talentExpr := orderListTalentUIDExpr(alias)
	return fmt.Sprintf(`COALESCE(
		(SELECT saa.target_value
		 FROM shop_account_attribution saa
		 WHERE %s
		 ORDER BY CASE WHEN COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') != '' THEN 0 ELSE 1 END
		 LIMIT 1),
		(SELECT saa.target_value
		 FROM shop_account_attribution saa
		 WHERE COALESCE(%s, '') = ''
		   AND saa.status = 1
		   AND saa.source_account_uid = '__blank__'
		   AND %s
		 ORDER BY CASE WHEN COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') != '' THEN 0 ELSE 1 END
		 LIMIT 1),
		(SELECT MIN(saa.target_value)
		 FROM shop_account_attribution saa
		 WHERE COALESCE(%s, '') = ''
		   AND %s
		 GROUP BY COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), saa.shop_name)
		 HAVING COUNT(DISTINCT CONCAT(saa.target_type, ':', TRIM(saa.target_value))) = 1
		 LIMIT 1),
		NULLIF(au.operator, ''),
		NULLIF(an.operator, ''),
		''
	)`, shopAccountAttributionRuleSQL(alias, talentExpr), talentExpr, shopAccountAttributionShopRuleSQL(alias), talentExpr, shopAccountAttributionShopRuleSQL(alias))
}

func orderListCommissionExpr(alias string) string {
	validCond := orderListValidCondition(alias)
	return fmt.Sprintf(
		"CASE WHEN %s THEN ROUND((%s.product_price * %s.product_num) - (COALESCE(g.base_price, 0) * %s.product_num) - (%s.product_price * %s.product_num * 0.03) - (%s.product_price * %s.product_num * 0.02), 2) ELSE 0 END",
		validCond, alias, alias, alias, alias, alias, alias, alias,
	)
}

func orderListValidCondition(alias string) string {
	return financeValidOrderCondition(alias)
}

func orderListCountKeywordCondition(alias string) string {
	return fmt.Sprintf(`(%s.sub_order_no LIKE ? OR %s.main_order_no LIKE ? OR %s.product_name LIKE ? OR %s.talent_name LIKE ? OR EXISTS (
		SELECT 1
		FROM alliance_orders ao
		WHERE ao.platform = 'douyin'
		  AND ao.order_id = %s.sub_order_no
		  AND (ao.author_account LIKE ? OR ao.short_id LIKE ?)
	))`, alias, alias, alias, alias, alias)
}

func orderListCountTalentCondition(alias string) string {
	return fmt.Sprintf(`(
		NULLIF(NULLIF(%s.talent_id, ''), '0') = ?
		OR (
			(%s.talent_id IS NULL OR %s.talent_id = '' OR %s.talent_id = '0')
			AND EXISTS (
				SELECT 1
				FROM alliance_orders ao
				INNER JOIN account a
					ON a.platform = 'douyin'
					AND a.status = 1
					AND a.uid = ?
					AND a.platform_nickname IS NOT NULL
					AND a.platform_nickname != ''
					AND ao.short_id COLLATE utf8mb4_unicode_ci = a.platform_nickname COLLATE utf8mb4_unicode_ci
				WHERE ao.platform = 'douyin'
				  AND ao.order_id = %s.sub_order_no
			)
		)
		OR (
			(%s.talent_id IS NULL OR %s.talent_id = '' OR %s.talent_id = '0')
			AND NOT EXISTS (
				SELECT 1
				FROM alliance_orders ao
				WHERE ao.platform = 'douyin'
				  AND ao.order_id = %s.sub_order_no
				  AND COALESCE(NULLIF(TRIM(ao.short_id), ''), NULLIF(TRIM(ao.author_account), '')) IS NOT NULL
			)
			AND EXISTS (
				SELECT 1
				FROM account shop_account
				WHERE shop_account.platform = 'douyin'
				  AND shop_account.status = 1
				  AND shop_account.uid = ?
				  AND REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(LOWER(shop_account.account_name), '官方旗舰店', ''), '旗舰店', ''), '专卖店', ''), '专营店', ''), '店', '')
					  COLLATE utf8mb4_unicode_ci =
					  REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(LOWER(%s.shop_name), '官方旗舰店', ''), '旗舰店', ''), '专卖店', ''), '专营店', ''), '店', '')
					  COLLATE utf8mb4_unicode_ci
			)
		)
		OR (
			(%s.talent_id IS NULL OR %s.talent_id = '' OR %s.talent_id = '0')
			AND NOT EXISTS (
				SELECT 1
				FROM alliance_orders ao
				WHERE ao.platform = 'douyin'
				  AND ao.order_id = %s.sub_order_no
				  AND ao.short_id IS NOT NULL
				  AND ao.short_id != ''
			)
			AND EXISTS (
				SELECT 1
				FROM qianchuan_ad_report qar
				LEFT JOIN account legacy_shop ON legacy_shop.id = qar.account_id
				LEFT JOIN (
					SELECT sqm.qianchuan_advertiser_id, MAX(s.store_name) AS shop_name
					FROM store_qianchuan_mapping sqm
					INNER JOIN store s ON s.id = sqm.store_id
					GROUP BY sqm.qianchuan_advertiser_id
				) mapped_shop ON mapped_shop.qianchuan_advertiser_id = qar.advertiser_id
				LEFT JOIN account target_account
					ON target_account.platform = 'douyin'
					AND target_account.status = 1
					AND target_account.uid = ?
					AND target_account.platform_nickname COLLATE utf8mb4_unicode_ci = qar.aweme_show_id COLLATE utf8mb4_unicode_ci
				WHERE COALESCE(mapped_shop.shop_name, legacy_shop.account_name) COLLATE utf8mb4_unicode_ci = %s.shop_name COLLATE utf8mb4_unicode_ci
				  AND CAST(qar.product_id AS CHAR) COLLATE utf8mb4_unicode_ci = CAST(%s.product_id AS CHAR) COLLATE utf8mb4_unicode_ci
				  AND LEFT(qar.stat_datetime, 10) = LEFT(COALESCE(%s.pay_finish_time, %s.order_submit_time), 10)
				  AND qar.aweme_show_id IS NOT NULL
				  AND qar.aweme_show_id != ''
				  AND (qar.stat_cost > 0 OR qar.pay_order_count > 0 OR qar.create_order_count > 0)
				GROUP BY LEFT(qar.stat_datetime, 10), COALESCE(mapped_shop.shop_name, legacy_shop.account_name), CAST(qar.product_id AS CHAR)
				HAVING COUNT(DISTINCT target_account.id) > 0
			)
		)
	)`, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias)
}

func orderListFromSQL() string {
	return `
		FROM order_info o
		LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) g ON o.seller_code = g.goods_code
		LEFT JOIN alliance_orders ao ON ao.platform = 'douyin' AND ao.order_id = o.sub_order_no
		LEFT JOIN (
			SELECT
				uid,
				MAX(account_name) AS account_name,
				MAX(platform_nickname) AS platform_nickname,
				MAX(operator) AS operator
			FROM account
			WHERE platform = 'douyin' AND status = 1 AND uid IS NOT NULL AND uid != ''
			GROUP BY uid
		) au ON o.talent_id COLLATE utf8mb4_unicode_ci = au.uid COLLATE utf8mb4_unicode_ci
		LEFT JOIN (
			SELECT
				platform_nickname,
				MAX(uid) AS uid,
				MAX(account_name) AS account_name,
				MAX(operator) AS operator
			FROM account
			WHERE platform = 'douyin' AND status = 1 AND platform_nickname IS NOT NULL AND platform_nickname != ''
			GROUP BY platform_nickname
		) an ON ao.short_id COLLATE utf8mb4_unicode_ci = an.platform_nickname COLLATE utf8mb4_unicode_ci
		WHERE 1=1`
}

func roundFloatOrder(val float64, precision int) float64 {
	pow := math.Pow10(precision)
	return math.Round(val*pow) / pow
}

func orderListFormatRatePercent(rate float64) string {
	if rate <= 0 {
		return ""
	}
	return fmt.Sprintf("%.2f%%", roundFloatOrder(rate*100, 2))
}

func orderListNormalizeRateValue(rate float64) float64 {
	switch {
	case rate <= 0:
		return 0
	case rate <= 1:
		return rate
	case rate > 100:
		return rate / 10000
	default:
		return rate / 100
	}
}

func orderListParseRate(raw string) float64 {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	switch {
	case value <= 0:
		return 0
	case value <= 1:
		return value
	case value > 100:
		return value / 10000
	default:
		return value / 100
	}
}

func orderListResolveCommissionRate(rawAllianceRate string, orderOwnershipCode string, externalOnlineRate, externalOfflineRate float64) string {
	externalOnlineRate = orderListNormalizeRateValue(externalOnlineRate)
	externalOfflineRate = orderListNormalizeRateValue(externalOfflineRate)
	switch orderOwnershipCode {
	case "self_sale":
		return "0.00%"
	case "internal":
		return orderListFormatRatePercent(orderListInternalCommissionRate)
	case "external":
		if externalOnlineRate > 0 {
			return orderListFormatRatePercent(externalOnlineRate)
		}
		if externalOfflineRate > 0 {
			return orderListFormatRatePercent(externalOfflineRate)
		}
	}

	if parsed := orderListParseRate(rawAllianceRate); parsed > 0 {
		return orderListFormatRatePercent(parsed)
	}

	if orderOwnershipCode == "conflict" {
		if externalOnlineRate > 0 {
			return orderListFormatRatePercent(externalOnlineRate)
		}
		return orderListFormatRatePercent(orderListInternalCommissionRate)
	}

	return ""
}

func orderListPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func orderListNormalizedTalentID(val string) string {
	val = strings.TrimSpace(val)
	if val == "" || val == "0" {
		return ""
	}
	return val
}

func orderListFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func orderListNormalizedLookupKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "0", "null", "nil", "-":
		return ""
	default:
		return value
	}
}

func orderListMatchFieldLabel(field string) string {
	switch field {
	case "uid":
		return "UID"
	case "show_id":
		return "抖音号"
	case "shop_account":
		return "店铺账号"
	case "shop_account_blank":
		return "店铺借号空白单"
	case "qianchuan_plan":
		return "千川计划"
	case "qianchuan_unique":
		return "千川计划"
	case "talent_name":
		return "达人名称"
	case "shop_self_sale":
		return "小店自营"
	default:
		return ""
	}
}

func orderListResolveOwnership(isShopSelfSale, hasInternal, hasExternal bool, internalField, externalField string) (string, string, string, string, string) {
	if isShopSelfSale {
		matchedBy := orderListMatchFieldLabel("shop_self_sale")
		if label := orderListMatchFieldLabel(internalField); label != "" {
			matchedBy = label
		}
		return "self_sale", "小店自营", "self_sale", "小店自营", matchedBy
	}
	if hasInternal && hasExternal {
		matchedBy := "内部:" + orderListMatchFieldLabel(internalField)
		if label := orderListMatchFieldLabel(externalField); label != "" {
			matchedBy += " / 外部:" + label
		}
		return "conflict", "归属冲突", "conflict", "归属冲突", matchedBy
	}
	if hasExternal {
		return "external", "外部订单", "external", "外部达人", orderListMatchFieldLabel(externalField)
	}
	if hasInternal {
		return "internal", "内部订单", "internal", "内部达人", orderListMatchFieldLabel(internalField)
	}
	return "unknown", "未归类", "unknown", "未归类", ""
}

func orderListIsShopSelfSale(row gdb.Record, talentUID, talentName, douyinAccount string) bool {
	return orderListNormalizedLookupKey(talentUID) == "" &&
		orderListNormalizedLookupKey(talentName) == "" &&
		orderListNormalizedLookupKey(douyinAccount) == ""
}

func orderListValidForCommission(orderStatus, afterSaleStatus string) bool {
	return IsValidDouyinOrderForCommission(orderStatus, afterSaleStatus)
}

// IsValidDouyinOrderForCommission returns the project-wide effective Douyin order rule.
// Valid orders include normal paid-flow orders without active/refunded after-sale,
// plus closed orders whose after-sale was rejected, failed, or cancelled.
func IsValidDouyinOrderForCommission(orderStatus, afterSaleStatus string) bool {
	return IsValidOrderForStats(orderStatus, afterSaleStatus)
}

func orderListCommissionAmount(row gdb.Record) float64 {
	orderStatus := row["order_status"].String()
	afterSaleStatus := row["after_sale_status"].String()
	if !orderListValidForCommission(orderStatus, afterSaleStatus) {
		return 0
	}

	productNum := float64(row["product_num"].Int())
	payAmount := row["product_price"].Float64() * productNum
	productCost := row["base_price"].Float64() * productNum
	return roundFloatOrder(payAmount-productCost-payAmount*0.03-payAmount*0.02, 2)
}

func orderListSalesAmount(row gdb.Record) float64 {
	return roundFloatOrder(row["product_price"].Float64()*float64(row["product_num"].Int()), 2)
}

func orderListActualCommissionRate(row gdb.Record, commission float64) string {
	payAmount := orderListSalesAmount(row)
	if payAmount <= 0 {
		return ""
	}
	rate := commission / payAmount
	if rate < 0 {
		rate = 0
	}
	return orderListFormatRatePercent(rate)
}

func orderListOfflineCommissionAmount(order g.Map) float64 {
	if gconv.String(order["orderOwnershipCode"]) != "external" {
		return 0
	}
	rate := orderListNormalizeRateValue(gconv.Float64(order["externalOfflineCommissionRate"]))
	if rate <= 0 {
		return 0
	}
	productNum := gconv.Float64(order["productNum"])
	if productNum <= 0 {
		return 0
	}
	amount := gconv.Float64(order["productPrice"]) * productNum * rate
	return roundFloatOrder(amount, 2)
}

// orderListApplyCommissionScope applies the order-type card semantics. Online
// commission represents the talent-order margin and is therefore shown only
// for the combined talent view. The external-talent view shows its separately
// configured offline commission only.
func orderListApplyCommissionScope(stats *OrderStats, orderType string) {
	if stats == nil {
		return
	}
	if normalizeOrderListType(orderType) != "talent" {
		stats.OnlineCommission = 0
	}
}

func orderListAfterSaleStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "6":
		return "售后申请"
	case "7":
		return "售后退货中"
	case "11":
		return "售后已发货"
	case "12":
		return "售后成功"
	case "27":
		return "售后拒绝"
	case "28":
		return "售后失败"
	case "29":
		return "售后退货拒绝"
	case "51":
		return "售后取消"
	default:
		return ""
	}
}

func orderListDisplayStatus(orderStatus, afterSaleStatus string) string {
	afterLabel := orderListAfterSaleStatusLabel(afterSaleStatus)
	if afterLabel == "" {
		return orderStatus
	}
	return strings.TrimSpace(orderStatus + "/" + afterLabel)
}

func orderListCompositeKey(shopName, talentUID string) string {
	return shopName + "\x00" + talentUID
}

type shopAccountAttributionTarget struct {
	TargetValue      string
	TraderID         int64
	AccountID        int64
	AccountUID       string
	AccountName      string
	PlatformNickname string
	Operator         string
	CompanyName      string
}

func orderListCompositeShopIDKey(shopID, talentUID string) string {
	return "shop_id:" + strings.TrimSpace(shopID) + "\x00" + talentUID
}

func orderListShopAttributionIDKey(shopID string) string {
	return "shop_id:" + strings.TrimSpace(shopID)
}

func orderListShopAttributionNameKey(shopName string) string {
	return "shop_name:" + orderListNormalizedLookupKey(shopName)
}

func orderListLookupAttributionTarget(attributionMap map[string]shopAccountAttributionTarget, shopID, shopName, talentUID string) (shopAccountAttributionTarget, bool) {
	shopID = strings.TrimSpace(shopID)
	if shopID != "" {
		if target, ok := attributionMap[orderListCompositeShopIDKey(shopID, talentUID)]; ok {
			return target, true
		}
	}
	target, ok := attributionMap[orderListCompositeKey(shopName, talentUID)]
	return target, ok
}

func orderListLookupShopAttributionTarget(attributionMap map[string]shopAccountAttributionTarget, shopID, shopName string) (shopAccountAttributionTarget, bool) {
	shopID = strings.TrimSpace(shopID)
	if shopID != "" {
		if target, ok := attributionMap[orderListShopAttributionIDKey(shopID)]; ok {
			return target, true
		}
	}
	target, ok := attributionMap[orderListShopAttributionNameKey(shopName)]
	return target, ok
}

func shopAccountAttributionStoreIDExpr(orderAlias string) string {
	return fmt.Sprintf(`COALESCE((
		SELECT st.store_id
		FROM store st
		WHERE st.platform = 'douyin'
		  AND st.status = 1
		  AND st.store_name COLLATE utf8mb4_unicode_ci = %s.shop_name COLLATE utf8mb4_unicode_ci
		ORDER BY st.id ASC
		LIMIT 1
	), '')`, orderAlias)
}

func shopAccountAttributionShopRuleSQL(orderAlias string) string {
	storeIDExpr := shopAccountAttributionStoreIDExpr(orderAlias)
	return fmt.Sprintf(`(
		        (COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') != '' AND saa.shop_id COLLATE utf8mb4_unicode_ci = %s COLLATE utf8mb4_unicode_ci)
		        OR (COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') = '' AND saa.shop_name COLLATE utf8mb4_unicode_ci = %s.shop_name COLLATE utf8mb4_unicode_ci)
		   )`, storeIDExpr, orderAlias)
}

func shopAccountAttributionRuleSQL(orderAlias, talentExpr string) string {
	return fmt.Sprintf(`saa.status = 1
		   AND saa.source_account_uid COLLATE utf8mb4_unicode_ci = %s COLLATE utf8mb4_unicode_ci
		   AND %s`, talentExpr, shopAccountAttributionShopRuleSQL(orderAlias))
}

func orderListExternalMatchKey(shopName, value string) string {
	valueKey := orderListNormalizedLookupKey(value)
	if valueKey == "" {
		return ""
	}
	return orderListNormalizedLookupKey(shopName) + "\x00" + valueKey
}

func orderListLookupExternalTalent(shopName, value string, index map[string]externalTalentInfo) (externalTalentInfo, bool) {
	if key := orderListExternalMatchKey(shopName, value); key != "" {
		if info, ok := index[key]; ok {
			return info, true
		}
	}
	if key := orderListExternalMatchKey("", value); key != "" {
		if info, ok := index[key]; ok {
			return info, true
		}
	}
	return externalTalentInfo{}, false
}

func orderListResolveExternalTalent(shopName, uid, showID, talentName string, externalByUID, externalByShowID, externalByName map[string]externalTalentInfo) (externalTalentInfo, string, bool) {
	if info, ok := orderListLookupExternalTalent(shopName, uid, externalByUID); ok {
		return info, "uid", true
	}
	if info, ok := orderListLookupExternalTalent(shopName, showID, externalByShowID); ok {
		return info, "show_id", true
	}
	if info, ok := orderListLookupExternalTalent(shopName, talentName, externalByName); ok {
		return info, "talent_name", true
	}
	return externalTalentInfo{}, "", false
}

func orderListQianchuanMatchKey(statDate, shopName, productID string) string {
	return strings.TrimSpace(statDate) + "\x00" + strings.TrimSpace(shopName) + "\x00" + strings.TrimSpace(productID)
}

func orderListQianchuanProductMatchKey(statDate, productID string) string {
	return strings.TrimSpace(statDate) + "\x00" + strings.TrimSpace(productID)
}

func orderListUsesAllocatedBlankRefundRate(order g.Map) bool {
	switch strings.TrimSpace(gconv.String(order["profitAttributionSource"])) {
	case "qianchuan_plan", "shop_account_blank":
		return strings.TrimSpace(gconv.String(order["originalTalentId"])) == ""
	default:
		return false
	}
}

func orderListAllocatedBlankRefundCost(order g.Map) float64 {
	if !orderListUsesAllocatedBlankRefundRate(order) {
		return 0
	}
	rate := gconv.Float64(order["linkRefundCostRate"])
	if rate <= 0 {
		return 0
	}
	return gconv.Float64(order["productCost"]) * rate
}

func orderListStatTimeExpr(alias string) string {
	return alias + ".order_submit_time"
}

func orderListShopNameMatchKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "", "（", "(", "）", ")")
	value = replacer.Replace(value)
	for {
		origin := value
		for _, suffix := range []string{"官方旗舰店", "旗舰店", "专卖店", "专营店", "旗舰", "店"} {
			value = strings.TrimSuffix(value, suffix)
		}
		if value == origin {
			break
		}
	}
	return value
}

func orderListStatDateFromRow(row gdb.Record) string {
	for _, field := range []string{"order_submit_time", "pay_finish_time"} {
		value := strings.TrimSpace(row[field].String())
		if len(value) < 10 {
			continue
		}
		datePart := value[:10]
		if datePart == "0000-00-00" {
			continue
		}
		return datePart
	}
	return ""
}

func orderListWeightedRatio(totalCost, statCost float64) float64 {
	if totalCost <= 0 || statCost <= 0 {
		return 0
	}
	return statCost / totalCost
}

func orderListStableWeight(orderKey string) float64 {
	digest := sha256.Sum256([]byte(orderKey))
	return float64(binary.BigEndian.Uint64(digest[:8])) / float64(^uint64(0))
}

func orderListWeightedIndex(orderKey string, ratios []float64) int {
	point := orderListStableWeight(orderKey)
	cumulative := 0.0
	lastPositive := -1
	for i, ratio := range ratios {
		if ratio <= 0 {
			continue
		}
		lastPositive = i
		cumulative += ratio
		if point < cumulative {
			return i
		}
	}
	return lastPositive
}

// orderListExactWeightedAssignments 使用最大余数法生成稳定的整数订单配额。
// 同一批订单和权重无论查询多少次都会得到相同结果，同时保证订单数严格守恒。
func orderListExactWeightedAssignments(scopeKey string, orderKeys []string, ratios []float64) map[string]int {
	assignments := make(map[string]int)
	if len(orderKeys) == 0 || len(ratios) == 0 {
		return assignments
	}

	uniqueKeys := make([]string, 0, len(orderKeys))
	seen := make(map[string]struct{}, len(orderKeys))
	for _, orderKey := range orderKeys {
		if orderKey == "" {
			continue
		}
		if _, exists := seen[orderKey]; exists {
			continue
		}
		seen[orderKey] = struct{}{}
		uniqueKeys = append(uniqueKeys, orderKey)
	}
	if len(uniqueKeys) == 0 {
		return assignments
	}

	totalWeight := 0.0
	for _, ratio := range ratios {
		if ratio > 0 {
			totalWeight += ratio
		}
	}
	if totalWeight <= 0 {
		return assignments
	}

	type remainderItem struct {
		index     int
		remainder float64
	}
	quotas := make([]int, len(ratios))
	remainders := make([]remainderItem, 0, len(ratios))
	allocated := 0
	for i, ratio := range ratios {
		if ratio < 0 {
			ratio = 0
		}
		exactQuota := float64(len(uniqueKeys)) * ratio / totalWeight
		baseQuota := int(math.Floor(exactQuota))
		quotas[i] = baseQuota
		allocated += baseQuota
		remainders = append(remainders, remainderItem{index: i, remainder: exactQuota - float64(baseQuota)})
	}
	sort.SliceStable(remainders, func(i, j int) bool {
		if math.Abs(remainders[i].remainder-remainders[j].remainder) > 1e-12 {
			return remainders[i].remainder > remainders[j].remainder
		}
		return remainders[i].index < remainders[j].index
	})
	for i := 0; allocated < len(uniqueKeys); i++ {
		quotas[remainders[i%len(remainders)].index]++
		allocated++
	}

	sort.SliceStable(uniqueKeys, func(i, j int) bool {
		left := sha256.Sum256([]byte(scopeKey + "\x00" + uniqueKeys[i]))
		right := sha256.Sum256([]byte(scopeKey + "\x00" + uniqueKeys[j]))
		if left == right {
			return uniqueKeys[i] < uniqueKeys[j]
		}
		return string(left[:]) < string(right[:])
	})

	position := 0
	for ownerIndex, quota := range quotas {
		for count := 0; count < quota && position < len(uniqueKeys); count++ {
			assignments[uniqueKeys[position]] = ownerIndex
			position++
		}
	}
	return assignments
}

func orderListExactWeightedAssignmentsWithCaps(scopeKey string, orderKeys []string, ratios []float64, caps []int) map[string]int {
	assignments := make(map[string]int)
	if len(orderKeys) == 0 || len(ratios) == 0 || len(ratios) != len(caps) {
		return assignments
	}

	uniqueKeys := make([]string, 0, len(orderKeys))
	seen := make(map[string]struct{}, len(orderKeys))
	for _, orderKey := range orderKeys {
		if orderKey == "" {
			continue
		}
		if _, exists := seen[orderKey]; exists {
			continue
		}
		seen[orderKey] = struct{}{}
		uniqueKeys = append(uniqueKeys, orderKey)
	}
	if len(uniqueKeys) == 0 {
		return assignments
	}

	totalWeight := 0.0
	totalCap := 0
	for i, ratio := range ratios {
		if ratio <= 0 || caps[i] <= 0 {
			continue
		}
		totalWeight += ratio
		totalCap += caps[i]
	}
	if totalWeight <= 0 || totalCap <= 0 {
		return assignments
	}

	target := len(uniqueKeys)
	if totalCap < target {
		target = totalCap
	}

	type remainderItem struct {
		index     int
		remainder float64
	}
	quotas := make([]int, len(ratios))
	remainders := make([]remainderItem, 0, len(ratios))
	allocated := 0
	for i, ratio := range ratios {
		if ratio <= 0 || caps[i] <= 0 {
			continue
		}
		exactQuota := float64(target) * ratio / totalWeight
		baseQuota := int(math.Floor(exactQuota))
		if baseQuota > caps[i] {
			baseQuota = caps[i]
		}
		quotas[i] = baseQuota
		allocated += baseQuota
		if quotas[i] < caps[i] {
			remainders = append(remainders, remainderItem{index: i, remainder: exactQuota - math.Floor(exactQuota)})
		}
	}

	sort.SliceStable(remainders, func(i, j int) bool {
		if math.Abs(remainders[i].remainder-remainders[j].remainder) > 1e-12 {
			return remainders[i].remainder > remainders[j].remainder
		}
		return remainders[i].index < remainders[j].index
	})
	for allocated < target {
		progressed := false
		for _, item := range remainders {
			if quotas[item.index] >= caps[item.index] {
				continue
			}
			quotas[item.index]++
			allocated++
			progressed = true
			if allocated >= target {
				break
			}
		}
		if !progressed {
			break
		}
	}

	sort.SliceStable(uniqueKeys, func(i, j int) bool {
		left := sha256.Sum256([]byte(scopeKey + "\x00" + uniqueKeys[i]))
		right := sha256.Sum256([]byte(scopeKey + "\x00" + uniqueKeys[j]))
		if left == right {
			return uniqueKeys[i] < uniqueKeys[j]
		}
		return string(left[:]) < string(right[:])
	})

	position := 0
	for ownerIndex, quota := range quotas {
		for count := 0; count < quota && position < len(uniqueKeys); count++ {
			assignments[uniqueKeys[position]] = ownerIndex
			position++
		}
	}
	return assignments
}

func orderListAllocationPercent(ratio float64) string {
	if ratio <= 0 {
		return ""
	}
	return fmt.Sprintf("%.2f%%", roundFloatOrder(ratio*100, 2))
}

func orderListFilterByTalent(orders []g.Map, talentID string) []g.Map {
	talentID = orderListNormalizedTalentID(talentID)
	if talentID == "" {
		return orders
	}
	filtered := make([]g.Map, 0, len(orders))
	for _, order := range orders {
		if orderListNormalizedTalentID(gconv.String(order["talentId"])) == talentID {
			filtered = append(filtered, order)
		}
	}
	return filtered
}

func normalizeOrderListType(orderType string) string {
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "self_sale":
		return "self_sale"
	case "external":
		return "external"
	case "talent":
		return "talent"
	case "refund_after_sale":
		return "refund_after_sale"
	default:
		return ""
	}
}

func orderListIsSelfSale(order g.Map) bool {
	return gconv.Bool(order["isShopSelfSale"]) || gconv.String(order["orderOwnershipCode"]) == "self_sale"
}

func orderListHasRefundAfterSale(order g.Map) bool {
	status := strings.ToLower(strings.TrimSpace(gconv.String(order["afterSaleStatus"])))
	switch status {
	case "", "0", "none", "null", "无", "未申请", "未发起":
		return false
	default:
		return true
	}
}

func orderListFilterByType(orders []g.Map, orderType string) []g.Map {
	orderType = normalizeOrderListType(orderType)
	if orderType == "" {
		return orders
	}
	filtered := make([]g.Map, 0, len(orders))
	for _, order := range orders {
		if orderType == "refund_after_sale" {
			if orderListHasRefundAfterSale(order) {
				filtered = append(filtered, order)
			}
			continue
		}
		if orderType == "external" {
			if gconv.String(order["orderOwnershipCode"]) == "external" {
				filtered = append(filtered, order)
			}
			continue
		}
		isSelfSale := orderListIsSelfSale(order)
		if (orderType == "self_sale" && isSelfSale) || (orderType == "talent" && !isSelfSale) {
			filtered = append(filtered, order)
		}
	}
	return filtered
}

func orderListQueryDateRange(startTime, endTime string) (string, string) {
	today := time.Now().Format("2006-01-02")
	startDate, endDate := today, today
	if len(startTime) >= 10 {
		startDate = startTime[:10]
	}
	if len(endTime) >= 10 {
		endDate = endTime[:10]
	} else if len(startTime) >= 10 {
		endDate = startDate
	}
	return startDate, endDate
}

// orderAdCostDateRange converts an order time range to the inclusive dates used
// by Qianchuan daily reports. A midnight end is exclusive for the ending day.
func orderAdCostDateRange(startTime, endTime string) (string, string) {
	startDate, endDate := orderListQueryDateRange(startTime, endTime)
	if len(endTime) < len("2006-01-02 15:04:05") || endTime[10:] != " 00:00:00" {
		return startDate, endDate
	}

	end, err := time.ParseInLocation("2006-01-02 15:04:05", endTime, time.Local)
	if err != nil {
		return startDate, endDate
	}
	if len(startTime) >= 10 {
		start, startErr := time.ParseInLocation("2006-01-02 15:04:05", startTime, time.Local)
		if startErr == nil && !end.After(start) {
			return startDate, endDate
		}
	}

	return startDate, end.AddDate(0, 0, -1).Format("2006-01-02")
}

func orderListFilterEnriched(orders []g.Map, keyword, shopName, orderStatus, talentID, orderType, startTime, endTime string) []g.Map {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	filtered := make([]g.Map, 0, len(orders))
	for _, order := range orders {
		if shopName != "" && gconv.String(order["shopName"]) != shopName {
			continue
		}
		if orderStatus != "" && gconv.String(order["rawOrderStatus"]) != orderStatus && gconv.String(order["orderStatus"]) != orderStatus {
			continue
		}
		if keyword != "" {
			matched := false
			for _, field := range []string{"subOrderNo", "mainOrderNo", "productName", "talentName", "douyinAccount", "talentId"} {
				if strings.Contains(strings.ToLower(gconv.String(order[field])), keyword) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		statTime := formatTime(order["orderSubmitTime"])
		if startTime != "" && statTime < startTime {
			continue
		}
		if endTime != "" && statTime > endTime {
			continue
		}
		filtered = append(filtered, order)
	}
	filtered = orderListFilterByTalent(filtered, talentID)
	return orderListFilterByType(filtered, orderType)
}

func orderListPage(orders []g.Map, page, pageSize int) []g.Map {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		return orders
	}
	start := (page - 1) * pageSize
	if start >= len(orders) {
		return []g.Map{}
	}
	end := start + pageSize
	if end > len(orders) {
		end = len(orders)
	}
	return orders[start:end]
}

func buildTimeRangeWhere(column, startTime, endTime string) (string, []interface{}) {
	if startTime == "" && endTime == "" {
		return fmt.Sprintf("%s >= CURDATE() AND %s < DATE_ADD(CURDATE(), INTERVAL 1 DAY)", column, column), nil
	}

	whereParts := make([]string, 0, 2)
	params := make([]interface{}, 0, 2)
	if startTime != "" {
		whereParts = append(whereParts, column+" >= ?")
		params = append(params, startTime)
	}
	if endTime != "" {
		whereParts = append(whereParts, column+" <= ?")
		params = append(params, endTime)
	}
	return strings.Join(whereParts, " AND "), params
}

func (s *OrderService) loadDouyinLinkRefundCostRates(ctx context.Context, rows gdb.Result) (map[string]float64, error) {
	rates := make(map[string]float64)
	if len(rows) == 0 {
		return rates, nil
	}

	productSet := make(map[string]struct{})
	products := make([]string, 0)
	startDate := ""
	endExclusive := ""
	for _, row := range rows {
		statDate := orderListStatDateFromRow(row)
		productID := strings.TrimSpace(row["product_id"].String())
		if statDate == "" || productID == "" {
			continue
		}
		parsed, err := time.Parse("2006-01-02", statDate)
		if err != nil {
			continue
		}
		nextDate := parsed.AddDate(0, 0, 1).Format("2006-01-02")
		if startDate == "" || statDate < startDate {
			startDate = statDate
		}
		if endExclusive == "" || nextDate > endExclusive {
			endExclusive = nextDate
		}
		if _, ok := productSet[productID]; !ok {
			productSet[productID] = struct{}{}
			products = append(products, productID)
		}
	}
	if startDate == "" || endExclusive == "" || len(products) == 0 {
		return rates, nil
	}

	validCondition := FinanceValidOrderCondition("o")
	refundCondition := financeRefundOrderCondition("o")
	params := make([]interface{}, 0, 2+len(products))
	params = append(params, startDate+" 00:00:00", endExclusive+" 00:00:00")
	for _, productID := range products {
		params = append(params, productID)
	}
	sql := fmt.Sprintf(`
		SELECT
			LEFT(o.order_submit_time, 10) AS stat_date,
			o.shop_name,
			CAST(o.product_id AS CHAR) AS product_id,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS valid_cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount
		FROM order_info o
		LEFT JOIN (
			SELECT goods_code, MAX(base_price) AS base_price
			FROM goods
			GROUP BY goods_code
		) g ON o.seller_code = g.goods_code
		WHERE o.order_submit_time >= ?
		  AND o.order_submit_time < ?
		  AND o.product_id IN (%s)
		  AND o.shop_name IS NOT NULL
		  AND o.shop_name != ''
		GROUP BY LEFT(o.order_submit_time, 10), o.shop_name, CAST(o.product_id AS CHAR)`,
		validCondition, refundCondition, orderListPlaceholders(len(products)))
	result, err := g.DB().GetAll(ctx, sql, params...)
	if err != nil {
		return nil, fmt.Errorf("查询链接整体退货成本率失败: %w", err)
	}
	for _, row := range result {
		validCost := row["valid_cost_amount"].Float64()
		if validCost <= 0 {
			continue
		}
		key := orderListQianchuanMatchKey(row["stat_date"].String(), row["shop_name"].String(), row["product_id"].String())
		rates[key] = row["refund_cost_amount"].Float64() / validCost
	}
	return rates, nil
}

func (s *OrderService) enrichDouyinOrderRows(ctx context.Context, result gdb.Result) ([]g.Map, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	linkRefundCostRateByScope, err := s.loadDouyinLinkRefundCostRates(ctx, result)
	if err != nil {
		return nil, err
	}
	qianchuanBlankOrderInferenceEnabled := IsQianchuanPlanBlankOrderInferenceEnabled(ctx)

	type accountInfo struct {
		id               int64
		traderID         int64
		uid              string
		accountName      string
		platformNickname string
		operator         string
		companyName      string
	}

	allianceByOrder := make(map[string]gdb.Record)
	subOrderNos := make([]string, 0, len(result))
	subOrderSet := make(map[string]struct{}, len(result))
	for _, row := range result {
		subOrderNo := strings.TrimSpace(row["sub_order_no"].String())
		if subOrderNo == "" {
			continue
		}
		if _, ok := subOrderSet[subOrderNo]; ok {
			continue
		}
		subOrderSet[subOrderNo] = struct{}{}
		subOrderNos = append(subOrderNos, subOrderNo)
	}

	if len(subOrderNos) > 0 {
		// 联盟订单按大批次查询，避免财务统计对数万订单产生几十次串行数据库往返。
		const allianceBatchSize = 10000
		for start := 0; start < len(subOrderNos); start += allianceBatchSize {
			end := start + allianceBatchSize
			if end > len(subOrderNos) {
				end = len(subOrderNos)
			}
			batch := subOrderNos[start:end]
			params := make([]interface{}, 0, len(batch))
			for _, subOrderNo := range batch {
				params = append(params, subOrderNo)
			}
			sql := fmt.Sprintf(
				"SELECT order_id, author_account, short_id, commission_rate FROM alliance_orders WHERE platform = 'douyin' AND order_id IN (%s)",
				orderListPlaceholders(len(batch)),
			)
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			rows, err := g.DB().GetAll(ctx, sql, params...)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				orderID := row["order_id"].String()
				if orderID == "" {
					continue
				}
				if _, exists := allianceByOrder[orderID]; !exists {
					allianceByOrder[orderID] = row
				}
			}
		}
	}

	talentIDs := make([]string, 0, len(result))
	talentIDSet := make(map[string]struct{}, len(result))
	shortIDs := make([]string, 0, len(allianceByOrder))
	shortIDSet := make(map[string]struct{}, len(allianceByOrder))
	talentNames := make([]string, 0, len(result))
	talentNameSet := make(map[string]struct{}, len(result))
	resolveShopNames := make([]string, 0, len(result))
	resolveShopNameSet := make(map[string]struct{}, len(result))
	for _, row := range result {
		if talentID := orderListNormalizedTalentID(row["talent_id"].String()); talentID != "" {
			if _, ok := talentIDSet[talentID]; !ok {
				talentIDSet[talentID] = struct{}{}
				talentIDs = append(talentIDs, talentID)
			}
		}
		alliance := allianceByOrder[row["sub_order_no"].String()]
		if shortID := strings.TrimSpace(alliance["short_id"].String()); shortID != "" {
			if _, ok := shortIDSet[shortID]; !ok {
				shortIDSet[shortID] = struct{}{}
				shortIDs = append(shortIDs, shortID)
			}
		}
		for _, name := range []string{row["talent_name"].String(), alliance["author_account"].String()} {
			if normalized := orderListNormalizedLookupKey(name); normalized != "" {
				if _, ok := talentNameSet[normalized]; !ok {
					talentNameSet[normalized] = struct{}{}
					talentNames = append(talentNames, strings.TrimSpace(name))
				}
			}
		}
		if shopName := strings.TrimSpace(row["shop_name"].String()); shopName != "" {
			key := orderListNormalizedLookupKey(shopName)
			if _, ok := resolveShopNameSet[key]; !ok {
				resolveShopNameSet[key] = struct{}{}
				resolveShopNames = append(resolveShopNames, shopName)
			}
		}
	}

	accountByUID := make(map[string]accountInfo)
	accountByNickname := make(map[string]accountInfo)
	accountByName := make(map[string]accountInfo)
	accountByShopNameKey := make(map[string]accountInfo)
	accountByShopNameKeyCount := make(map[string]int)
	if len(talentIDs) > 0 || len(shortIDs) > 0 || len(talentNames) > 0 || len(resolveShopNames) > 0 {
		params := make([]interface{}, 0, len(talentIDs)+len(shortIDs)+len(talentNames))
		conds := make([]string, 0, 3)
		if len(talentIDs) > 0 {
			conds = append(conds, fmt.Sprintf("uid IN (%s)", orderListPlaceholders(len(talentIDs))))
			for _, talentID := range talentIDs {
				params = append(params, talentID)
			}
		}
		if len(shortIDs) > 0 {
			conds = append(conds, fmt.Sprintf("platform_nickname IN (%s)", orderListPlaceholders(len(shortIDs))))
			for _, shortID := range shortIDs {
				params = append(params, shortID)
			}
		}
		if len(talentNames) > 0 {
			conds = append(conds, fmt.Sprintf("account_name IN (%s)", orderListPlaceholders(len(talentNames))))
			for _, talentName := range talentNames {
				params = append(params, talentName)
			}
		}

		sql := "SELECT a.id, a.uid, a.account_name, a.platform_nickname, a.operator, c.name AS company_name, (SELECT MIN(tar.trader_id) FROM trader_account_relation tar WHERE tar.account_id = a.id) AS trader_id FROM account a LEFT JOIN sys_company c ON a.company_id = c.id WHERE a.platform = 'douyin' AND a.status = 1"
		if len(resolveShopNames) > 0 {
			params = nil
		} else if len(conds) > 0 {
			sql += " AND (" + strings.Join(conds, " OR ") + ")"
		}
		sql += " ORDER BY a.id DESC"
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := g.DB().GetAll(ctx, sql, params...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			info := accountInfo{
				id:               row["id"].Int64(),
				traderID:         row["trader_id"].Int64(),
				uid:              row["uid"].String(),
				accountName:      row["account_name"].String(),
				platformNickname: row["platform_nickname"].String(),
				operator:         row["operator"].String(),
				companyName:      row["company_name"].String(),
			}
			if info.uid != "" {
				key := orderListNormalizedLookupKey(info.uid)
				if _, exists := accountByUID[key]; !exists {
					accountByUID[key] = info
				}
			}
			if info.platformNickname != "" {
				key := orderListNormalizedLookupKey(info.platformNickname)
				if _, exists := accountByNickname[key]; !exists {
					accountByNickname[key] = info
				}
			}
			if info.accountName != "" {
				key := orderListNormalizedLookupKey(info.accountName)
				if _, exists := accountByName[key]; !exists {
					accountByName[key] = info
				}
			}
			if shopKey := orderListShopNameMatchKey(info.accountName); shopKey != "" {
				accountByShopNameKeyCount[shopKey]++
				if _, exists := accountByShopNameKey[shopKey]; !exists {
					accountByShopNameKey[shopKey] = info
				}
			}
		}
	}

	externalByUID := make(map[string]externalTalentInfo)
	externalByShowID := make(map[string]externalTalentInfo)
	externalByName := make(map[string]externalTalentInfo)
	if len(talentIDs) > 0 || len(shortIDs) > 0 || len(talentNames) > 0 {
		params := make([]interface{}, 0, len(talentIDs)+len(shortIDs)+len(talentNames)*2+len(resolveShopNames))
		conds := make([]string, 0, 4)
		if len(talentIDs) > 0 {
			conds = append(conds, fmt.Sprintf("uid IN (%s)", orderListPlaceholders(len(talentIDs))))
			for _, talentID := range talentIDs {
				params = append(params, talentID)
			}
		}
		if len(shortIDs) > 0 {
			conds = append(conds, fmt.Sprintf("show_id IN (%s)", orderListPlaceholders(len(shortIDs))))
			for _, shortID := range shortIDs {
				params = append(params, shortID)
			}
		}
		if len(talentNames) > 0 {
			holders := orderListPlaceholders(len(talentNames))
			conds = append(conds, fmt.Sprintf("(talent_name IN (%s) OR account_name IN (%s))", holders, holders))
			for _, talentName := range talentNames {
				params = append(params, talentName)
			}
			for _, talentName := range talentNames {
				params = append(params, talentName)
			}
		}

		sql := "SELECT id, uid, account_name, talent_name, show_id, team_name, online_commission_rate, offline_commission_rate, shop_name FROM external_talent_registry WHERE platform_type = 'douyin' AND status = 1 AND (" + strings.Join(conds, " OR ") + ")"
		if len(resolveShopNames) > 0 {
			sql += fmt.Sprintf(" AND (shop_name IN (%s) OR shop_name = '')", orderListPlaceholders(len(resolveShopNames)))
			for _, shopName := range resolveShopNames {
				params = append(params, shopName)
			}
		}
		sql += " ORDER BY id DESC"
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := g.DB().GetAll(ctx, sql, params...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			info := externalTalentInfo{
				uid:                   row["uid"].String(),
				accountName:           row["account_name"].String(),
				talentName:            row["talent_name"].String(),
				showID:                row["show_id"].String(),
				teamName:              row["team_name"].String(),
				onlineCommissionRate:  row["online_commission_rate"].Float64(),
				offlineCommissionRate: row["offline_commission_rate"].Float64(),
			}
			if info.uid != "" {
				key := orderListExternalMatchKey(row["shop_name"].String(), info.uid)
				if _, exists := externalByUID[key]; !exists {
					externalByUID[key] = info
				}
			}
			if info.showID != "" {
				key := orderListExternalMatchKey(row["shop_name"].String(), info.showID)
				if _, exists := externalByShowID[key]; !exists {
					externalByShowID[key] = info
				}
			}
			for _, name := range []string{info.talentName, info.accountName} {
				key := orderListExternalMatchKey(row["shop_name"].String(), name)
				if key == "" {
					continue
				}
				if _, exists := externalByName[key]; !exists {
					externalByName[key] = info
				}
			}
		}
	}

	resolveInternal := func(uid, showID, talentName string) (accountInfo, string, bool) {
		if key := orderListNormalizedLookupKey(uid); key != "" {
			if info, ok := accountByUID[key]; ok {
				return info, "uid", true
			}
		}
		if key := orderListNormalizedLookupKey(showID); key != "" {
			if info, ok := accountByNickname[key]; ok {
				return info, "show_id", true
			}
		}
		if key := orderListNormalizedLookupKey(talentName); key != "" {
			if info, ok := accountByName[key]; ok {
				return info, "talent_name", true
			}
		}
		return accountInfo{}, "", false
	}

	resolveExternal := func(shopName, uid, showID, talentName string) (externalTalentInfo, string, bool) {
		return orderListResolveExternalTalent(shopName, uid, showID, talentName, externalByUID, externalByShowID, externalByName)
	}

	blankShopAttributionMap := make(map[string]shopAccountAttributionTarget)
	if len(resolveShopNames) > 0 {
		params := make([]interface{}, 0, len(resolveShopNames))
		for _, shopName := range resolveShopNames {
			params = append(params, shopName)
		}
		sql := fmt.Sprintf(
			`SELECT saa.shop_id, saa.shop_name, saa.source_account_uid, saa.target_type, saa.target_value,
					COALESCE(t.id, 0) AS trader_id,
					COALESCE(a.id, 0) AS account_id,
					COALESCE(a.uid, '') AS account_uid,
					COALESCE(a.account_name, '') AS account_name,
					COALESCE(a.platform_nickname, '') AS platform_nickname,
					COALESCE(NULLIF(a.operator, ''), saa.target_value) AS operator,
					COALESCE(c.name, '') AS company_name
			 FROM shop_account_attribution saa
			 LEFT JOIN trader t
			   ON t.status = 1
			  AND (
				(saa.target_type = 'operator' AND (
					saa.target_value COLLATE utf8mb4_unicode_ci = t.name COLLATE utf8mb4_unicode_ci
					OR saa.target_value COLLATE utf8mb4_unicode_ci = CONCAT(t.name, '组') COLLATE utf8mb4_unicode_ci
					OR (saa.target_value LIKE '%%组' AND TRIM(TRAILING '组' FROM saa.target_value) COLLATE utf8mb4_unicode_ci = t.name COLLATE utf8mb4_unicode_ci)
				))
				OR (saa.target_type = 'trader' AND saa.target_value COLLATE utf8mb4_unicode_ci = CAST(t.id AS CHAR) COLLATE utf8mb4_unicode_ci)
			  )
			 LEFT JOIN (
				SELECT tar.trader_id, MIN(a.id) AS account_id
				FROM trader_account_relation tar
				INNER JOIN account a ON a.id = tar.account_id AND a.platform = 'douyin' AND a.status = 1
				GROUP BY tar.trader_id
			 ) first_account ON first_account.trader_id = t.id
			 LEFT JOIN account a ON a.id = first_account.account_id
			 LEFT JOIN sys_company c ON c.id = a.company_id
			 WHERE saa.status = 1
			   AND saa.shop_name IN (%s)
			 ORDER BY CASE WHEN COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') != '' THEN 0 ELSE 1 END, saa.id ASC`,
			orderListPlaceholders(len(resolveShopNames)),
		)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := g.DB().GetAll(ctx, sql, params...)
		if err != nil {
			return nil, err
		}
		type blankShopAggregate struct {
			target   shopAccountAttributionTarget
			key      string
			conflict bool
		}
		aggregates := make(map[string]*blankShopAggregate)
		addBlankShopTarget := func(key string, target shopAccountAttributionTarget, targetKey string) {
			if key == "" || targetKey == "" {
				return
			}
			agg := aggregates[key]
			if agg == nil {
				aggregates[key] = &blankShopAggregate{target: target, key: targetKey}
				return
			}
			if agg.key != targetKey {
				agg.conflict = true
			}
		}
		for _, row := range rows {
			target := shopAccountAttributionTarget{
				TargetValue:      row["target_value"].String(),
				TraderID:         row["trader_id"].Int64(),
				AccountID:        row["account_id"].Int64(),
				AccountUID:       row["account_uid"].String(),
				AccountName:      row["account_name"].String(),
				PlatformNickname: row["platform_nickname"].String(),
				Operator:         row["operator"].String(),
				CompanyName:      row["company_name"].String(),
			}
			if row["source_account_uid"].String() == "__blank__" {
				if target.TraderID <= 0 {
					continue
				}
				if shopID := strings.TrimSpace(row["shop_id"].String()); shopID != "" {
					blankShopAttributionMap[orderListShopAttributionIDKey(shopID)] = target
				}
				blankShopAttributionMap[orderListShopAttributionNameKey(row["shop_name"].String())] = target
				continue
			}
			targetKey := orderListNormalizedLookupKey(row["target_type"].String() + ":" + target.TargetValue)
			if target.TraderID > 0 {
				targetKey = "trader:" + strconv.FormatInt(target.TraderID, 10)
			}
			if shopID := strings.TrimSpace(row["shop_id"].String()); shopID != "" {
				addBlankShopTarget(orderListShopAttributionIDKey(shopID), target, targetKey)
			}
			addBlankShopTarget(orderListShopAttributionNameKey(row["shop_name"].String()), target, targetKey)
		}
		for key, agg := range aggregates {
			if agg == nil || agg.conflict || agg.target.TraderID <= 0 {
				continue
			}
			blankShopAttributionMap[key] = agg.target
		}
	}

	type qianchuanMatchRequest struct {
		statDate  string
		shopName  string
		productID string
	}
	type qianchuanMatchInfo struct {
		accountInfo
		planID              int64
		planName            string
		planCount           int
		awemeShowID         string
		allocatedStoreName  string
		blankOrderOwnerName string
		storeOrderCount     int64
		statCost            float64
		payOrderCount       int64
		allocationRatio     float64
		allocationBasis     string
	}
	type qianchuanCandidateStats struct {
		total         int
		zeroOrder     int
		zeroPlanOrder int
		disabled      int
		missingOwner  int
		zeroCost      int
		valid         int
		plans         []g.Map
		owners        []g.Map
	}
	mergeQianchuanMatchesByOwner := func(matches []qianchuanMatchInfo) []qianchuanMatchInfo {
		merged := make([]qianchuanMatchInfo, 0, len(matches))
		indexByOwner := make(map[string]int, len(matches))
		for _, match := range matches {
			traderKey := ""
			if match.traderID > 0 {
				traderKey = strconv.FormatInt(match.traderID, 10)
			}
			ownerKey := orderListFirstNonEmpty(traderKey, match.blankOrderOwnerName, match.uid, match.awemeShowID)
			if ownerKey == "" {
				continue
			}
			if index, ok := indexByOwner[ownerKey]; ok {
				merged[index].statCost += match.statCost
				merged[index].payOrderCount += match.payOrderCount
				merged[index].storeOrderCount += match.storeOrderCount
				merged[index].planCount++
				merged[index].planID = 0
				merged[index].planName = fmt.Sprintf("多个店铺/计划（%d个）", merged[index].planCount)
				if !strings.Contains(merged[index].allocatedStoreName, match.allocatedStoreName) && match.allocatedStoreName != "" {
					merged[index].allocatedStoreName = orderListFirstNonEmpty(merged[index].allocatedStoreName, match.allocatedStoreName)
				}
				continue
			}
			match.planCount = 1
			indexByOwner[ownerKey] = len(merged)
			merged = append(merged, match)
		}
		return merged
	}
	applyQianchuanAllocationRatio := func(matches []qianchuanMatchInfo) []qianchuanMatchInfo {
		if len(matches) == 0 {
			return matches
		}
		var totalPayOrders int64
		var totalCost float64
		for _, match := range matches {
			totalPayOrders += match.payOrderCount
			totalCost += match.statCost
		}
		for i := range matches {
			if totalCost > 0 {
				matches[i].allocationRatio = orderListWeightedRatio(totalCost, matches[i].statCost)
				matches[i].allocationBasis = "stat_cost"
			} else {
				matches[i].allocationRatio = orderListWeightedRatio(float64(totalPayOrders), float64(matches[i].payOrderCount))
				matches[i].allocationBasis = "pay_order_count_fallback"
			}
		}
		return matches
	}

	qianchuanRequests := make([]qianchuanMatchRequest, 0, len(result))
	qianchuanRequestSet := make(map[string]struct{}, len(result))
	if qianchuanBlankOrderInferenceEnabled {
		for _, row := range result {
			alliance := allianceByOrder[row["sub_order_no"].String()]
			rawTalentName := orderListFirstNonEmpty(row["talent_name"].String(), alliance["author_account"].String())
			if orderListNormalizedTalentID(row["talent_id"].String()) != "" ||
				strings.TrimSpace(alliance["short_id"].String()) != "" ||
				orderListNormalizedLookupKey(rawTalentName) != "" {
				continue
			}

			statDate := orderListStatDateFromRow(row)
			shopName := strings.TrimSpace(row["shop_name"].String())
			productID := strings.TrimSpace(row["product_id"].String())
			if statDate == "" || shopName == "" || productID == "" {
				continue
			}
			if _, matched := orderListLookupShopAttributionTarget(blankShopAttributionMap, row["order_store_id"].String(), shopName); matched {
				continue
			}

			key := orderListQianchuanMatchKey(statDate, shopName, productID)
			if _, exists := qianchuanRequestSet[key]; exists {
				continue
			}
			qianchuanRequestSet[key] = struct{}{}
			qianchuanRequests = append(qianchuanRequests, qianchuanMatchRequest{
				statDate:  statDate,
				shopName:  shopName,
				productID: productID,
			})
		}
	}

	qianchuanByKey := make(map[string][]qianchuanMatchInfo, len(qianchuanRequests))
	qianchuanCandidateByKey := make(map[string]*qianchuanCandidateStats, len(qianchuanRequests))
	if len(qianchuanRequests) > 0 {
		const qianchuanBatchSize = 50
		for start := 0; start < len(qianchuanRequests); start += qianchuanBatchSize {
			end := start + qianchuanBatchSize
			if end > len(qianchuanRequests) {
				end = len(qianchuanRequests)
			}

			batch := qianchuanRequests[start:end]
			reqSelects := make([]string, 0, len(batch))
			reqParams := make([]interface{}, 0, len(batch)*4)
			productIDs := make([]string, 0, len(batch))
			productIDSet := make(map[string]struct{}, len(batch))
			batchStartDate := ""
			batchEndExclusive := ""
			for _, req := range batch {
				statDate := strings.TrimSpace(req.statDate)
				shopName := strings.TrimSpace(req.shopName)
				productID := strings.TrimSpace(req.productID)
				parsedDate, parseErr := time.Parse("2006-01-02", statDate)
				if parseErr != nil || shopName == "" || productID == "" {
					continue
				}
				nextDate := parsedDate.AddDate(0, 0, 1).Format("2006-01-02")
				reqSelects = append(reqSelects, "SELECT ? AS stat_date, ? AS next_date, ? AS shop_name, ? AS product_id_text, CAST(? AS UNSIGNED) AS product_id_num")
				reqParams = append(reqParams, statDate, nextDate, shopName, productID, productID)
				if _, exists := productIDSet[productID]; !exists {
					productIDSet[productID] = struct{}{}
					productIDs = append(productIDs, productID)
				}
				if batchStartDate == "" || statDate < batchStartDate {
					batchStartDate = statDate
				}
				if batchEndExclusive == "" || nextDate > batchEndExclusive {
					batchEndExclusive = nextDate
				}
			}
			if len(reqSelects) == 0 {
				continue
			}
			productPlaceholders := orderListPlaceholders(len(productIDs))
			params := make([]interface{}, 0, len(reqParams)+2+len(productIDs))
			params = append(params, reqParams...)
			params = append(params, batchStartDate+" 00:00:00", batchEndExclusive+" 00:00:00")
			for _, productID := range productIDs {
				params = append(params, productID)
			}

			sql := fmt.Sprintf(`
						SELECT
							q.stat_date,
						q.shop_name AS allocated_store_name,
						q.product_id,
						q.ad_id,
						q.ad_name,
						q.advertiser_id,
						q.pay_order_count,
						q.stat_cost,
							q.store_order_count,
							q.blank_order_enabled,
							q.blank_order_owner_name,
							q.account_id,
							q.account_uid AS uid,
							q.account_name,
							q.platform_nickname,
							q.blank_order_owner_name AS operator,
							c.name AS company_name,
							COALESCE(t.id, 0) AS trader_id
						FROM (
							SELECT
									req.stat_date AS stat_date,
									COALESCE(mapped_shop.shop_name, legacy_shop.account_name) AS shop_name,
									req.product_id_text AS product_id,
									qar.account_id AS account_id,
									qar.ad_id AS ad_id,
									qar.advertiser_id,
									MAX(legacy_shop.uid) AS account_uid,
									MAX(legacy_shop.account_name) AS account_name,
									MAX(legacy_shop.platform_nickname) AS platform_nickname,
									MAX(legacy_shop.company_id) AS company_id,
									MAX(qar.ad_name) AS ad_name,
									COALESCE(SUM(qar.pay_order_count), 0) AS pay_order_count,
									COALESCE(SUM(qar.stat_cost), 0) AS stat_cost,
									COALESCE(MAX(legacy_shop.blank_order_enabled), 0) AS blank_order_enabled,
									COALESCE(
										NULLIF(MAX(legacy_shop.blank_order_owner_name), ''),
										NULLIF(MAX(legacy_shop.operator), ''),
										''
									) AS blank_order_owner_name,
									COALESCE(MAX(oi_stat.store_order_count), 0) AS store_order_count
							FROM qianchuan_ad_report qar
						INNER JOIN (
							%s
						) req
						  ON qar.product_id = req.product_id_num
						 AND qar.stat_datetime >= req.stat_date
						 AND qar.stat_datetime < req.next_date
							LEFT JOIN account legacy_shop ON legacy_shop.id = qar.account_id AND legacy_shop.platform = 'douyin' AND legacy_shop.status = 1
						LEFT JOIN (
							SELECT
								sqm.qianchuan_advertiser_id,
							MAX(s.store_name) AS shop_name,
							MAX(s.blank_order_enabled) AS blank_order_enabled,
							MAX(s.blank_order_owner_name) AS blank_order_owner_name
						FROM store_qianchuan_mapping sqm
							INNER JOIN store s ON s.id = sqm.store_id
							GROUP BY sqm.qianchuan_advertiser_id
						) mapped_shop ON mapped_shop.qianchuan_advertiser_id = qar.advertiser_id
						LEFT JOIN (
							SELECT
								LEFT(oi.order_submit_time, 10) AS stat_date,
								oi.shop_name,
								CAST(oi.product_id AS CHAR) AS product_id,
								COUNT(DISTINCT oi.main_order_no) AS store_order_count
							FROM order_info oi
							WHERE oi.order_submit_time >= ?
							  AND oi.order_submit_time < ?
							  AND oi.product_id IN (%s)
							  AND oi.main_order_no IS NOT NULL
							  AND oi.main_order_no != ''
							GROUP BY LEFT(oi.order_submit_time, 10), oi.shop_name, CAST(oi.product_id AS CHAR)
						) oi_stat
						  ON oi_stat.stat_date = req.stat_date
						 AND oi_stat.shop_name COLLATE utf8mb4_unicode_ci = COALESCE(mapped_shop.shop_name, legacy_shop.account_name) COLLATE utf8mb4_unicode_ci
						 AND oi_stat.product_id COLLATE utf8mb4_unicode_ci = req.product_id_text COLLATE utf8mb4_unicode_ci
							WHERE qar.product_id IS NOT NULL
							  AND qar.product_id != ''
								  AND (qar.stat_cost > 0 OR qar.pay_order_count > 0 OR qar.create_order_count > 0)
								  AND COALESCE(mapped_shop.shop_name, legacy_shop.account_name) IS NOT NULL
								  AND COALESCE(mapped_shop.shop_name, legacy_shop.account_name) COLLATE utf8mb4_unicode_ci = req.shop_name COLLATE utf8mb4_unicode_ci
								GROUP BY req.stat_date, COALESCE(mapped_shop.shop_name, legacy_shop.account_name), req.product_id_text, qar.account_id, qar.ad_id, qar.advertiser_id
						) q
						LEFT JOIN trader t
						  ON t.status = 1
				 AND q.blank_order_owner_name != ''
				 AND (
					t.name COLLATE utf8mb4_unicode_ci = q.blank_order_owner_name COLLATE utf8mb4_unicode_ci
						OR t.name COLLATE utf8mb4_unicode_ci = CONCAT(q.blank_order_owner_name, '组') COLLATE utf8mb4_unicode_ci
						OR (q.blank_order_owner_name LIKE '%%组' AND t.name COLLATE utf8mb4_unicode_ci = TRIM(TRAILING '组' FROM q.blank_order_owner_name) COLLATE utf8mb4_unicode_ci)
					 )
						LEFT JOIN sys_company c ON q.company_id = c.id
							ORDER BY q.stat_date, q.product_id, q.store_order_count DESC, q.stat_cost DESC, q.shop_name ASC, q.ad_id ASC`,
				strings.Join(reqSelects, " UNION ALL "),
				productPlaceholders,
			)
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			rows, err := g.DB().GetAll(ctx, sql, params...)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				key := orderListQianchuanMatchKey(row["stat_date"].String(), row["allocated_store_name"].String(), row["product_id"].String())
				if key == "\x00\x00" {
					continue
				}
				stats := qianchuanCandidateByKey[key]
				if stats == nil {
					stats = &qianchuanCandidateStats{}
					qianchuanCandidateByKey[key] = stats
				}
				stats.total++
				ownerName := strings.TrimSpace(row["blank_order_owner_name"].String())
				storeName := strings.TrimSpace(row["allocated_store_name"].String())
				stats.plans = append(stats.plans, g.Map{
					"storeName":       storeName,
					"planId":          row["ad_id"].Int64(),
					"planName":        row["ad_name"].String(),
					"statCost":        row["stat_cost"].Float64(),
					"payOrderCount":   row["pay_order_count"].Int64(),
					"storeOrderCount": row["store_order_count"].Int64(),
				})
				stats.owners = append(stats.owners, g.Map{
					"storeName":           storeName,
					"blankOrderEnabled":   row["blank_order_enabled"].Int(),
					"blankOrderOwnerName": ownerName,
					"matchedTraderId":     row["trader_id"].Int64(),
					"matchedAccountId":    row["account_id"].Int64(),
					"matchedAccountUid":   row["uid"].String(),
				})
				if row["stat_cost"].Float64() <= 0 {
					stats.zeroCost++
					continue
				}
				if row["pay_order_count"].Int64() <= 0 {
					stats.zeroPlanOrder++
					continue
				}
				if row["store_order_count"].Int64() <= 0 {
					stats.zeroOrder++
					continue
				}
				if row["blank_order_enabled"].Int() == 0 {
					stats.disabled++
					continue
				}
				if ownerName == "" || row["trader_id"].Int64() <= 0 || row["account_id"].Int64() <= 0 || orderListNormalizedTalentID(row["uid"].String()) == "" {
					stats.missingOwner++
					continue
				}
				stats.valid++
				qianchuanByKey[key] = append(qianchuanByKey[key], qianchuanMatchInfo{
					accountInfo: accountInfo{
						id:               row["account_id"].Int64(),
						traderID:         row["trader_id"].Int64(),
						uid:              row["uid"].String(),
						accountName:      row["account_name"].String(),
						platformNickname: row["platform_nickname"].String(),
						operator:         row["operator"].String(),
						companyName:      row["company_name"].String(),
					},
					planID:              row["ad_id"].Int64(),
					planName:            row["ad_name"].String(),
					allocatedStoreName:  storeName,
					blankOrderOwnerName: ownerName,
					storeOrderCount:     row["store_order_count"].Int64(),
					statCost:            row["stat_cost"].Float64(),
					payOrderCount:       row["pay_order_count"].Int64(),
				})
			}
		}

		for key, matches := range qianchuanByKey {
			qianchuanByKey[key] = applyQianchuanAllocationRatio(mergeQianchuanMatchesByOwner(matches))
		}
	}

	// 先按日期+店铺+商品收集真正无达人信息的小店自营订单，再按千川消耗权重生成严格整数配额。
	qianchuanOrderKeysByScope := make(map[string][]string)
	for _, row := range result {
		alliance := allianceByOrder[row["sub_order_no"].String()]
		rawTalentName := orderListFirstNonEmpty(row["talent_name"].String(), alliance["author_account"].String())
		if orderListNormalizedTalentID(row["talent_id"].String()) != "" ||
			strings.TrimSpace(alliance["short_id"].String()) != "" ||
			orderListNormalizedLookupKey(rawTalentName) != "" {
			continue
		}
		if !IsValidOrderForStats(row["order_status"].String(), row["after_sale_status"].String()) {
			continue
		}
		scopeKey := orderListQianchuanMatchKey(orderListStatDateFromRow(row), row["shop_name"].String(), row["product_id"].String())
		if len(qianchuanByKey[scopeKey]) == 0 {
			continue
		}
		if _, matched := orderListLookupShopAttributionTarget(blankShopAttributionMap, row["order_store_id"].String(), row["shop_name"].String()); matched {
			continue
		}
		orderKey := orderListFirstNonEmpty(row["main_order_no"].String(), row["sub_order_no"].String(), row["id"].String())
		if orderKey != "" {
			qianchuanOrderKeysByScope[scopeKey] = append(qianchuanOrderKeysByScope[scopeKey], orderKey)
		}
	}
	qianchuanAssignmentByOrder := make(map[string]qianchuanMatchInfo)
	scopeKeys := make([]string, 0, len(qianchuanOrderKeysByScope))
	for scopeKey := range qianchuanOrderKeysByScope {
		scopeKeys = append(scopeKeys, scopeKey)
	}
	sort.Strings(scopeKeys)
	for _, scopeKey := range scopeKeys {
		matches := qianchuanByKey[scopeKey]
		availableOrderKeys := make([]string, 0, len(qianchuanOrderKeysByScope[scopeKey]))
		for _, orderKey := range qianchuanOrderKeysByScope[scopeKey] {
			if _, assigned := qianchuanAssignmentByOrder[orderKey]; !assigned {
				availableOrderKeys = append(availableOrderKeys, orderKey)
			}
		}
		ratios := make([]float64, len(matches))
		caps := make([]int, len(matches))
		for i, match := range matches {
			ratios[i] = match.allocationRatio
			caps[i] = int(match.payOrderCount)
		}
		for orderKey, ownerIndex := range orderListExactWeightedAssignmentsWithCaps(scopeKey, availableOrderKeys, ratios, caps) {
			if ownerIndex >= 0 && ownerIndex < len(matches) {
				qianchuanAssignmentByOrder[orderKey] = matches[ownerIndex]
			}
		}
	}
	qianchuanExceptionByOrder := make(map[string]g.Map)
	qianchuanCandidatePlans := func(stats *qianchuanCandidateStats) []g.Map {
		if stats == nil {
			return nil
		}
		return stats.plans
	}
	qianchuanCandidateOwners := func(stats *qianchuanCandidateStats) []g.Map {
		if stats == nil {
			return nil
		}
		return stats.owners
	}
	buildQianchuanException := func(row gdb.Record, scopeKey string) g.Map {
		stats := qianchuanCandidateByKey[scopeKey]
		exceptionType := "NO_QIANCHUAN_PLAN"
		reason := "未匹配到同日期同商品的千川计划"
		if stats != nil && stats.total > 0 {
			switch {
			case stats.valid == 0 && stats.zeroCost == stats.total:
				exceptionType = "ZERO_QIANCHUAN_COST"
				reason = "匹配到千川计划，但候选计划消耗均为0"
			case stats.valid == 0 && stats.zeroPlanOrder+stats.zeroCost == stats.total && stats.zeroPlanOrder > 0:
				exceptionType = "ZERO_QIANCHUAN_ORDER_COUNT"
				reason = "匹配到千川计划，但候选计划成交订单数均为0"
			case stats.valid == 0 && stats.zeroOrder+stats.zeroPlanOrder+stats.zeroCost == stats.total && stats.zeroOrder > 0:
				exceptionType = "ALL_SHOPS_ZERO_ORDER"
				reason = "匹配到千川投流店铺，但候选店铺同日同商品出单量均为0"
			case stats.valid == 0 && stats.disabled+stats.zeroOrder+stats.zeroPlanOrder+stats.zeroCost == stats.total && stats.disabled > 0:
				exceptionType = "ALL_SHOPS_DISABLED"
				reason = "匹配到千川投流店铺，但候选店铺均未开启空白单归属"
			case stats.valid == 0 && stats.missingOwner > 0:
				exceptionType = "SHOP_OWNER_MISSING"
				reason = "匹配到千川投流店铺，但候选店铺缺少空白单归属人或归属人未匹配到可统计账号"
			case stats.valid > 0:
				exceptionType = "QIANCHUAN_ORDER_CAP_REACHED"
				reason = "空白单数量超过千川计划成交订单数上限，剩余订单不自动分配"
			default:
				exceptionType = "NO_VALID_SHOP"
				reason = "匹配到千川计划，但没有满足分配条件的有效店铺"
			}
		}
		return g.Map{
			"orderInfoId":           row["id"].Int64(),
			"mainOrderNo":           row["main_order_no"].String(),
			"subOrderNo":            row["sub_order_no"].String(),
			"statDate":              orderListStatDateFromRow(row),
			"shopName":              row["shop_name"].String(),
			"productId":             row["product_id"].String(),
			"exceptionType":         exceptionType,
			"exceptionReason":       reason,
			"qianchuanPlanSnapshot": orderListMarshalJSON(qianchuanCandidatePlans(stats)),
			"shopOwnerSnapshot":     orderListMarshalJSON(qianchuanCandidateOwners(stats)),
		}
	}

	type resolvedMeta struct {
		accountID                   int64
		traderID                    int64
		attributionSource           string
		talentUID                   string
		talentName                  string
		commissionRate              string
		douyinAccount               string
		operator                    string
		companyName                 string
		externalTeamName            string
		commission                  float64
		talentOwnership             string
		talentOwnershipTag          string
		orderOwnership              string
		orderOwnershipTag           string
		matchedBy                   string
		isShopSelfSale              bool
		allocationRatio             float64
		qianchuanPlanID             int64
		qianchuanPlanName           string
		qianchuanPlanCount          int
		qianchuanAwemeID            string
		qianchuanAllocatedStoreName string
		blankOrderOwnerName         string
		qianchuanStatCost           float64
		qianchuanPayOrders          int64
		allocationBasis             string
		externalOnlineRate          float64
		externalOfflineRate         float64
	}

	resolved := make([][]resolvedMeta, len(result))
	shopIDs := make([]string, 0, len(result))
	shopIDSet := make(map[string]struct{}, len(result))
	shopNames := make([]string, 0, len(result))
	shopNameSet := make(map[string]struct{}, len(result))
	resolvedUIDs := make([]string, 0, len(result))
	resolvedUIDSet := make(map[string]struct{}, len(result))

	for i, row := range result {
		alliance := allianceByOrder[row["sub_order_no"].String()]
		shortID := strings.TrimSpace(alliance["short_id"].String())
		originTalentID := orderListNormalizedTalentID(row["talent_id"].String())
		rawTalentName := orderListFirstNonEmpty(row["talent_name"].String(), alliance["author_account"].String())
		internalInfo, internalField, hasInternal := resolveInternal(originTalentID, shortID, rawTalentName)
		externalInfo, externalField, hasExternal := resolveExternal(row["shop_name"].String(), originTalentID, shortID, rawTalentName)
		appendResolved := func(internalInfo accountInfo, internalField string, hasInternal bool, qianchuanMatch qianchuanMatchInfo) {
			allocationRatio := qianchuanMatch.allocationRatio
			if allocationRatio <= 0 {
				allocationRatio = 1
			}

			talentUID := orderListFirstNonEmpty(originTalentID, internalInfo.uid, externalInfo.uid)
			talentName := orderListFirstNonEmpty(row["talent_name"].String(), internalInfo.accountName, externalInfo.talentName, externalInfo.accountName, alliance["author_account"].String())
			douyinAccount := orderListFirstNonEmpty(internalInfo.platformNickname, shortID, externalInfo.showID)
			operator := strings.TrimSpace(internalInfo.operator)
			isShopSelfSale := orderListIsShopSelfSale(row, talentUID, talentName, douyinAccount)
			if internalField == "shop_account" {
				isShopSelfSale = true
			}
			orderOwnershipCode, orderOwnership, talentOwnershipCode, talentOwnership, matchedBy := orderListResolveOwnership(
				isShopSelfSale,
				hasInternal,
				hasExternal,
				internalField,
				externalField,
			)
			commission := orderListCommissionAmount(row)

			resolved[i] = append(resolved[i], resolvedMeta{
				accountID:                   internalInfo.id,
				traderID:                    internalInfo.traderID,
				attributionSource:           internalField,
				talentUID:                   talentUID,
				talentName:                  talentName,
				commissionRate:              orderListActualCommissionRate(row, commission),
				douyinAccount:               douyinAccount,
				operator:                    operator,
				companyName:                 internalInfo.companyName,
				externalTeamName:            externalInfo.teamName,
				commission:                  commission,
				talentOwnership:             talentOwnership,
				talentOwnershipTag:          talentOwnershipCode,
				orderOwnership:              orderOwnership,
				orderOwnershipTag:           orderOwnershipCode,
				matchedBy:                   matchedBy,
				isShopSelfSale:              isShopSelfSale,
				allocationRatio:             allocationRatio,
				qianchuanPlanID:             qianchuanMatch.planID,
				qianchuanPlanName:           qianchuanMatch.planName,
				qianchuanPlanCount:          qianchuanMatch.planCount,
				qianchuanAwemeID:            qianchuanMatch.awemeShowID,
				qianchuanAllocatedStoreName: qianchuanMatch.allocatedStoreName,
				blankOrderOwnerName:         qianchuanMatch.blankOrderOwnerName,
				qianchuanStatCost:           qianchuanMatch.statCost,
				qianchuanPayOrders:          qianchuanMatch.payOrderCount,
				allocationBasis:             qianchuanMatch.allocationBasis,
				externalOnlineRate:          externalInfo.onlineCommissionRate,
				externalOfflineRate:         externalInfo.offlineCommissionRate,
			})
		}

		if !hasInternal && !hasExternal && originTalentID == "" && shortID == "" && orderListNormalizedLookupKey(rawTalentName) == "" {
			if target, matched := orderListLookupShopAttributionTarget(blankShopAttributionMap, row["order_store_id"].String(), row["shop_name"].String()); matched {
				appendResolved(accountInfo{
					id:               target.AccountID,
					traderID:         target.TraderID,
					uid:              target.AccountUID,
					accountName:      target.AccountName,
					platformNickname: target.PlatformNickname,
					operator:         orderListFirstNonEmpty(target.Operator, target.TargetValue),
					companyName:      target.CompanyName,
				}, "shop_account_blank", true, qianchuanMatchInfo{})
			}
		}
		if qianchuanBlankOrderInferenceEnabled && len(resolved[i]) == 0 && !hasInternal && !hasExternal && originTalentID == "" && shortID == "" && orderListNormalizedLookupKey(rawTalentName) == "" {
			orderKey := orderListFirstNonEmpty(row["main_order_no"].String(), row["sub_order_no"].String(), row["id"].String())
			if match, matched := qianchuanAssignmentByOrder[orderKey]; matched {
				appendResolved(match.accountInfo, "qianchuan_plan", true, match)
			} else if orderKey != "" {
				scopeKey := orderListQianchuanMatchKey(orderListStatDateFromRow(row), row["shop_name"].String(), row["product_id"].String())
				qianchuanExceptionByOrder[orderKey] = buildQianchuanException(row, scopeKey)
			}
		}
		if len(resolved[i]) == 0 && !hasInternal && !hasExternal && originTalentID == "" && shortID == "" && rawTalentName == "" {
			shopKey := orderListShopNameMatchKey(row["shop_name"].String())
			if info, ok := accountByShopNameKey[shopKey]; ok && accountByShopNameKeyCount[shopKey] == 1 {
				internalInfo = info
				internalField = "shop_account"
				hasInternal = true
			}
		}
		if len(resolved[i]) == 0 {
			appendResolved(internalInfo, internalField, hasInternal, qianchuanMatchInfo{})
		}

		if shopID := strings.TrimSpace(row["order_store_id"].String()); shopID != "" {
			if _, ok := shopIDSet[shopID]; !ok {
				shopIDSet[shopID] = struct{}{}
				shopIDs = append(shopIDs, shopID)
			}
		}
		if shopName := row["shop_name"].String(); shopName != "" {
			if _, ok := shopNameSet[shopName]; !ok {
				shopNameSet[shopName] = struct{}{}
				shopNames = append(shopNames, shopName)
			}
		}
		for _, item := range resolved[i] {
			if item.talentUID != "" {
				if _, ok := resolvedUIDSet[item.talentUID]; !ok {
					resolvedUIDSet[item.talentUID] = struct{}{}
					resolvedUIDs = append(resolvedUIDs, item.talentUID)
				}
			}
		}
	}

	attributionMap := make(map[string]shopAccountAttributionTarget)
	if len(shopNames) > 0 && len(resolvedUIDs) > 0 {
		params := make([]interface{}, 0, len(shopIDs)+len(shopNames)+len(resolvedUIDs))
		shopConds := make([]string, 0, 2)
		if len(shopIDs) > 0 {
			shopConds = append(shopConds, fmt.Sprintf("saa.shop_id IN (%s)", orderListPlaceholders(len(shopIDs))))
			for _, shopID := range shopIDs {
				params = append(params, shopID)
			}
		}
		if len(shopNames) > 0 {
			shopConds = append(shopConds, fmt.Sprintf("(COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') = '' AND saa.shop_name IN (%s))", orderListPlaceholders(len(shopNames))))
			for _, shopName := range shopNames {
				params = append(params, shopName)
			}
		}
		for _, talentUID := range resolvedUIDs {
			params = append(params, talentUID)
		}
		sql := fmt.Sprintf(
			`SELECT saa.shop_id, saa.shop_name, saa.source_account_uid, saa.target_value, COALESCE(t.id, 0) AS trader_id
			 FROM shop_account_attribution saa
			 LEFT JOIN trader t
			   ON t.status = 1
			  AND ((saa.target_type = 'operator' AND saa.target_value COLLATE utf8mb4_unicode_ci = t.name COLLATE utf8mb4_unicode_ci)
			       OR (saa.target_type = 'trader' AND saa.target_value COLLATE utf8mb4_unicode_ci = CAST(t.id AS CHAR) COLLATE utf8mb4_unicode_ci))
			 WHERE saa.status = 1
			   AND (%s)
			   AND saa.source_account_uid IN (%s)`,
			strings.Join(shopConds, " OR "),
			orderListPlaceholders(len(resolvedUIDs)),
		)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := g.DB().GetAll(ctx, sql, params...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			target := shopAccountAttributionTarget{
				TargetValue: row["target_value"].String(),
				TraderID:    row["trader_id"].Int64(),
			}
			if shopID := strings.TrimSpace(row["shop_id"].String()); shopID != "" {
				key := orderListCompositeShopIDKey(shopID, row["source_account_uid"].String())
				if _, exists := attributionMap[key]; !exists {
					attributionMap[key] = target
				}
				continue
			}
			key := orderListCompositeKey(row["shop_name"].String(), row["source_account_uid"].String())
			if _, exists := attributionMap[key]; !exists {
				attributionMap[key] = target
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	orders := make([]g.Map, 0, len(result))
	for i, row := range result {
		isValidOrder := orderListValidForCommission(row["order_status"].String(), row["after_sale_status"].String())
		for allocationIndex, item := range resolved[i] {
			allocationRatio := item.allocationRatio
			if allocationRatio <= 0 {
				allocationRatio = 1
			}
			productCost := roundFloatOrder(row["base_price"].Float64()*float64(row["product_num"].Int()), 2)
			attributionTarget, attributionMatched := orderListLookupAttributionTarget(attributionMap, row["order_store_id"].String(), row["shop_name"].String(), item.talentUID)
			teamOwner := orderListFirstNonEmpty(
				attributionTarget.TargetValue,
				item.operator,
				item.externalTeamName,
			)
			attributedTraderID := item.traderID
			if attributionMatched && attributionTarget.TraderID > 0 {
				attributedTraderID = attributionTarget.TraderID
			}
			allocationRowID := row["id"].String()
			if len(resolved[i]) > 1 || item.qianchuanPlanID != 0 {
				allocationRowID = fmt.Sprintf("%s_%d_%d", row["id"].String(), item.qianchuanPlanID, allocationIndex+1)
			}
			orderKey := orderListFirstNonEmpty(row["main_order_no"].String(), row["sub_order_no"].String(), row["id"].String())
			qianchuanException := qianchuanExceptionByOrder[orderKey]
			attributionStatDate := orderListStatDateFromRow(row)
			linkRefundCostRate := linkRefundCostRateByScope[orderListQianchuanMatchKey(attributionStatDate, row["shop_name"].String(), row["product_id"].String())]

			// talentId 是列表展示使用的归属结果，可能由千川计划反推；利润归属必须保留订单原始 UID。
			orders = append(orders, g.Map{
				"id":                            row["id"],
				"allocationRowId":               allocationRowID,
				"shopId":                        row["order_store_id"],
				"shopName":                      row["shop_name"],
				"mainOrderNo":                   row["main_order_no"],
				"subOrderNo":                    row["sub_order_no"],
				"productName":                   row["product_name"],
				"productSpec":                   row["product_spec"],
				"productNum":                    row["product_num"],
				"originalProductNum":            row["product_num"],
				"productId":                     row["product_id"],
				"attributionStatDate":           attributionStatDate,
				"productPrice":                  row["product_price"],
				"orderPayAmount":                row["order_pay_amount"],
				"originalOrderPayAmount":        row["order_pay_amount"],
				"shippingFee":                   row["shipping_fee"],
				"totalDiscount":                 row["total_discount"],
				"orderStatus":                   orderListDisplayStatus(row["order_status"].String(), row["after_sale_status"].String()),
				"rawOrderStatus":                row["order_status"],
				"orderSubmitTime":               row["order_submit_time"],
				"sendTime":                      row["send_time"],
				"payFinishTime":                 row["pay_finish_time"],
				"orderFinishTime":               row["order_finish_time"],
				"receiverName":                  row["receiver_name"],
				"receiverPhone":                 row["receiver_phone"],
				"province":                      row["province"],
				"city":                          row["city"],
				"district":                      row["district"],
				"talentName":                    item.talentName,
				"flowSource":                    row["flow_source"],
				"afterSaleStatus":               row["after_sale_status"],
				"afterSaleStatusLabel":          orderListAfterSaleStatusLabel(row["after_sale_status"].String()),
				"talentId":                      item.talentUID,
				"originalTalentId":              row["talent_id"],
				"profitAttributionSource":       item.attributionSource,
				"attributedAccountId":           item.accountID,
				"attributedTraderId":            attributedTraderID,
				"specialAttributionMatched":     attributionMatched || item.attributionSource == "qianchuan_plan" || item.attributionSource == "shop_account_blank",
				"productCost":                   productCost,
				"originalProductCost":           roundFloatOrder(row["base_price"].Float64()*float64(row["product_num"].Int()), 2),
				"linkRefundCostRate":            roundFloatOrder(linkRefundCostRate, 8),
				"allocatedRefundCost":           roundFloatOrder(productCost*linkRefundCostRate, 2),
				"shopServiceFee":                row["shop_service_fee"],
				"originalShopServiceFee":        row["shop_service_fee"],
				"commissionRate":                item.commissionRate,
				"douyinAccount":                 item.douyinAccount,
				"commission":                    item.commission,
				"isValidOrder":                  isValidOrder,
				"uid":                           item.talentUID,
				"teamOwner":                     teamOwner,
				"companyName":                   item.companyName,
				"externalTeamName":              item.externalTeamName,
				"externalOnlineCommissionRate":  roundFloatOrder(orderListNormalizeRateValue(item.externalOnlineRate), 6),
				"externalOfflineCommissionRate": roundFloatOrder(orderListNormalizeRateValue(item.externalOfflineRate), 6),
				"orderType":                     item.orderOwnership,
				"orderOwnershipCode":            item.orderOwnershipTag,
				"talentType":                    item.talentOwnership,
				"talentOwnershipCode":           item.talentOwnershipTag,
				"matchedBy":                     item.matchedBy,
				"isShopSelfSale":                item.isShopSelfSale,
				"allocationRatio":               roundFloatOrder(allocationRatio, 6),
				"allocationPercent":             orderListAllocationPercent(allocationRatio),
				"isAllocated":                   allocationRatio < 0.999999 || allocationRatio > 1.000001,
				"allocationMode":                orderListFirstNonEmpty(item.allocationBasis, "direct"),
				"qianchuanPlanId":               item.qianchuanPlanID,
				"qianchuanPlanName":             item.qianchuanPlanName,
				"qianchuanPlanCount":            item.qianchuanPlanCount,
				"qianchuanAwemeShowId":          item.qianchuanAwemeID,
				"qianchuanAllocatedStoreName":   item.qianchuanAllocatedStoreName,
				"blankOrderOwnerName":           item.blankOrderOwnerName,
				"qianchuanStatCost":             roundFloatOrder(item.qianchuanStatCost, 2),
				"qianchuanPayOrderCount":        item.qianchuanPayOrders,
				"qianchuanAllocationBasis":      item.allocationBasis,
				"qianchuanAllocationMatched":    item.qianchuanAllocatedStoreName != "" || item.qianchuanAwemeID != "",
				"qianchuanExceptionType":        qianchuanException["exceptionType"],
				"qianchuanExceptionReason":      qianchuanException["exceptionReason"],
				"qianchuanPlanSnapshot":         qianchuanException["qianchuanPlanSnapshot"],
				"shopOwnerSnapshot":             qianchuanException["shopOwnerSnapshot"],
			})
		}
	}

	persistDouyinQianchuanAttributions(ctx, orders)
	return orders, nil
}

// persistDouyinQianchuanAttributions 保存千川反推结果，便于对单个订单的利润归属进行复核。
// 归属表是审计辅助数据；表未迁移或写入失败不影响当次利润计算。
func persistDouyinQianchuanAttributions(ctx context.Context, orders []g.Map) {
	if len(orders) == 0 || ctx.Err() != nil {
		return
	}
	persistDouyinQianchuanAttributionExceptions(ctx, orders)
	exists, err := g.DB().GetOne(ctx, `
		SELECT COUNT(*) AS cnt
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = 'douyin_order_attribution'`)
	if err != nil {
		g.Log().Warningf(ctx, "[DouyinOrderAttribution] 检查归属表失败: %v", err)
		return
	}
	if exists["cnt"].Int() == 0 {
		return
	}

	// 审计归属批量落库，减少统计请求中的串行写入次数。
	const batchSize = 2000
	resolvedOrderIDs := make([]int64, 0)
	for start := 0; start < len(orders); {
		values := make([]string, 0, batchSize)
		params := make([]interface{}, 0, batchSize*18)
		for start < len(orders) && len(values) < batchSize {
			order := orders[start]
			start++
			if strings.TrimSpace(gconv.String(order["profitAttributionSource"])) != "qianchuan_plan" ||
				gconv.Int64(order["id"]) <= 0 ||
				strings.TrimSpace(gconv.String(order["attributionStatDate"])) == "" {
				continue
			}
			resolvedOrderIDs = append(resolvedOrderIDs, gconv.Int64(order["id"]))
			values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			params = append(params,
				gconv.Int64(order["id"]),
				gconv.String(order["mainOrderNo"]),
				gconv.String(order["subOrderNo"]),
				gconv.String(order["attributionStatDate"]),
				gconv.String(order["shopName"]),
				gconv.String(order["qianchuanAllocatedStoreName"]),
				gconv.String(order["blankOrderOwnerName"]),
				gconv.String(order["productId"]),
				"qianchuan_plan",
				gconv.Int64(order["attributedAccountId"]),
				gconv.Int64(order["attributedTraderId"]),
				gconv.String(order["talentId"]),
				gconv.Int64(order["qianchuanPlanId"]),
				gconv.Int(order["qianchuanPlanCount"]),
				gconv.Float64(order["qianchuanStatCost"]),
				gconv.Float64(order["allocationRatio"]),
				gconv.String(order["qianchuanAllocationBasis"]),
				"qianchuan_cost_quota_v1",
			)
		}
		if len(values) == 0 {
			continue
		}
		sql := `INSERT INTO douyin_order_attribution (
			order_info_id, main_order_no, sub_order_no, stat_date, shop_name, allocated_store_name, blank_order_owner_name, product_id,
			source, account_id, trader_id, talent_uid, qianchuan_plan_id, qianchuan_plan_count,
			qianchuan_stat_cost, allocation_ratio, allocation_basis, rule_version
		) VALUES ` + strings.Join(values, ",") + `
		ON DUPLICATE KEY UPDATE
			main_order_no = VALUES(main_order_no),
			sub_order_no = VALUES(sub_order_no),
			stat_date = VALUES(stat_date),
			shop_name = VALUES(shop_name),
			allocated_store_name = VALUES(allocated_store_name),
			blank_order_owner_name = VALUES(blank_order_owner_name),
			product_id = VALUES(product_id),
			account_id = VALUES(account_id),
			trader_id = VALUES(trader_id),
			talent_uid = VALUES(talent_uid),
			qianchuan_plan_id = VALUES(qianchuan_plan_id),
			qianchuan_plan_count = VALUES(qianchuan_plan_count),
			qianchuan_stat_cost = VALUES(qianchuan_stat_cost),
			allocation_ratio = VALUES(allocation_ratio),
			allocation_basis = VALUES(allocation_basis),
			rule_version = VALUES(rule_version),
			calculated_at = CURRENT_TIMESTAMP`
		if _, err := g.DB().Exec(ctx, sql, params...); err != nil {
			g.Log().Warningf(ctx, "[DouyinOrderAttribution] 保存千川归属结果失败: %v", err)
			return
		}
	}
	markResolvedDouyinQianchuanAttributionExceptions(ctx, resolvedOrderIDs)
}

func persistDouyinQianchuanAttributionExceptions(ctx context.Context, orders []g.Map) {
	if err := ensureBlankOrderAllocationExceptionTable(ctx); err != nil {
		g.Log().Warningf(ctx, "[BlankOrderAllocationException] 创建异常池表失败: %v", err)
		return
	}

	const batchSize = 1000
	for start := 0; start < len(orders); {
		values := make([]string, 0, batchSize)
		params := make([]interface{}, 0, batchSize*11)
		for start < len(orders) && len(values) < batchSize {
			order := orders[start]
			start++
			exceptionType := strings.TrimSpace(gconv.String(order["qianchuanExceptionType"]))
			if exceptionType == "" || gconv.Int64(order["id"]) <= 0 {
				continue
			}
			values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')")
			params = append(params,
				gconv.Int64(order["id"]),
				gconv.String(order["mainOrderNo"]),
				gconv.String(order["subOrderNo"]),
				orderListNilIfEmpty(gconv.String(order["attributionStatDate"])),
				gconv.String(order["shopName"]),
				gconv.String(order["productId"]),
				exceptionType,
				gconv.String(order["qianchuanExceptionReason"]),
				orderListNilIfEmpty(gconv.String(order["qianchuanPlanSnapshot"])),
				orderListNilIfEmpty(gconv.String(order["shopOwnerSnapshot"])),
			)
		}
		if len(values) == 0 {
			continue
		}
		sql := `INSERT INTO blank_order_allocation_exception (
			order_info_id, main_order_no, sub_order_no, stat_date, shop_name, product_id,
			exception_type, exception_reason, qianchuan_plan_snapshot, shop_owner_snapshot, status
		) VALUES ` + strings.Join(values, ",") + `
		ON DUPLICATE KEY UPDATE
			main_order_no = VALUES(main_order_no),
			sub_order_no = VALUES(sub_order_no),
			stat_date = VALUES(stat_date),
			shop_name = VALUES(shop_name),
			product_id = VALUES(product_id),
			exception_reason = VALUES(exception_reason),
			qianchuan_plan_snapshot = VALUES(qianchuan_plan_snapshot),
			shop_owner_snapshot = VALUES(shop_owner_snapshot),
			status = IF(status IN ('resolved_manual', 'ignored'), status, 'pending')`
		if _, err := g.DB().Exec(ctx, sql, params...); err != nil {
			g.Log().Warningf(ctx, "[BlankOrderAllocationException] 保存异常池失败: %v", err)
			return
		}
	}
}

func markResolvedDouyinQianchuanAttributionExceptions(ctx context.Context, orderIDs []int64) {
	if len(orderIDs) == 0 {
		return
	}
	if err := ensureBlankOrderAllocationExceptionTable(ctx); err != nil {
		g.Log().Warningf(ctx, "[BlankOrderAllocationException] 创建异常池表失败: %v", err)
		return
	}
	uniqueIDs := make([]int64, 0, len(orderIDs))
	seen := make(map[int64]struct{}, len(orderIDs))
	for _, id := range orderIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	const batchSize = 1000
	for start := 0; start < len(uniqueIDs); start += batchSize {
		end := start + batchSize
		if end > len(uniqueIDs) {
			end = len(uniqueIDs)
		}
		batch := uniqueIDs[start:end]
		params := make([]interface{}, 0, len(batch))
		for _, id := range batch {
			params = append(params, id)
		}
		sql := fmt.Sprintf(`
			UPDATE blank_order_allocation_exception
			SET status = 'resolved_auto',
			    resolved_at = CURRENT_TIMESTAMP
			WHERE status = 'pending'
			  AND order_info_id IN (%s)`, orderListPlaceholders(len(batch)))
		if _, err := g.DB().Exec(ctx, sql, params...); err != nil {
			g.Log().Warningf(ctx, "[BlankOrderAllocationException] 自动解决异常失败: %v", err)
			return
		}
	}
}

func ensureBlankOrderAllocationExceptionTable(ctx context.Context) error {
	_, err := g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS blank_order_allocation_exception (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		order_info_id BIGINT NOT NULL DEFAULT 0 COMMENT '抖音订单明细ID',
		main_order_no VARCHAR(100) NOT NULL DEFAULT '' COMMENT '主订单号',
		sub_order_no VARCHAR(100) NOT NULL DEFAULT '' COMMENT '子订单号',
		stat_date DATE DEFAULT NULL COMMENT '归属日期',
		shop_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '空白单来源店铺',
		product_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '商品ID',
		exception_type VARCHAR(50) NOT NULL DEFAULT '' COMMENT '异常类型',
		exception_reason VARCHAR(500) NOT NULL DEFAULT '' COMMENT '异常原因',
		qianchuan_plan_snapshot JSON DEFAULT NULL COMMENT '千川计划候选快照',
		shop_owner_snapshot JSON DEFAULT NULL COMMENT '店铺归属快照',
		status VARCHAR(30) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/resolved_auto/resolved_manual/ignored',
		resolved_owner VARCHAR(100) NOT NULL DEFAULT '' COMMENT '人工指定归属人',
		resolved_by VARCHAR(100) NOT NULL DEFAULT '' COMMENT '处理人',
		resolved_at DATETIME DEFAULT NULL COMMENT '处理时间',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		UNIQUE KEY uk_order_source (order_info_id, exception_type),
		KEY idx_status_date (status, stat_date),
		KEY idx_shop_product_date (shop_name, product_id, stat_date),
		KEY idx_main_order_no (main_order_no)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空白单千川归属异常池'`)
	return err
}

func orderListMarshalJSON(value interface{}) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil || string(data) == "null" {
		return ""
	}
	return string(data)
}

func orderListNilIfEmpty(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

// OrderStats 订单统计数据
type OrderStats struct {
	TotalOrderCount      int64   `json:"totalOrderCount"`      // 总订单 = 有效 + 无效 + 未命中
	ValidOrderCount      int64   `json:"validOrderCount"`      // 有效订单
	InvalidOrderCount    int64   `json:"invalidOrderCount"`    // 无效订单
	UnmatchedOrderCount  int64   `json:"unmatchedOrderCount"`  // 未命中有效/无效规则的订单
	ValidPayAmount       float64 `json:"validPayAmount"`       // 有效定价总额 = product_price × num
	ProductCostAmount    float64 `json:"productCostAmount"`    // 货款 = base_price × num
	ReserveAmount        float64 `json:"reserveAmount"`        // 留底 = product_price × num × 3%
	RefundCostAmount     float64 `json:"refundCostAmount"`     // 发货后退款成本
	ShopServiceFee       float64 `json:"shopServiceFee"`       // 小店服务费 = 总商品金额 × 2%
	AdCostAmount         float64 `json:"adCostAmount"`         // 广告费消耗（千川）
	OnlineCommission     float64 `json:"onlineCommission"`     // 线上佣金支出
	OfflineCommission    float64 `json:"offlineCommission"`    // 线下佣金支出（外部达人）
	RealtimeProfit       float64 `json:"realtimeProfit"`       // 实时利润
	ValidOrderCostMatch  int64   `json:"validOrderCostMatch"`  // 有效订单已匹配数量
	RefundOrderCount     int64   `json:"refundOrderCount"`     // 退款订单数量
	CommissionOrderCount int64   `json:"commissionOrderCount"` // 有佣金的订单数量
	SyncDate             string  `json:"syncDate"`             // 最近同步日期
}

// Stats 查询订单统计
func (s *OrderService) Stats(ctx context.Context, keyword, shopName, orderStatus, talentId, orderType, startTime, endTime string) (*OrderStats, error) {
	stats := &OrderStats{
		SyncDate: time.Now().Format("2006-01-02"),
	}

	if normalizeOrderListType(orderType) != "" {
		startDate, endDate := orderListQueryDateRange(startTime, endTime)
		orders, err := s.loadEnrichedDouyinOrders(ctx, startDate, endDate)
		if err != nil {
			return nil, err
		}
		allTypeOrders := orderListFilterEnriched(orders, keyword, shopName, orderStatus, talentId, "", startTime, endTime)
		filteredOrders := orderListFilterByType(allTypeOrders, orderType)
		allTypeStats := s.statsFromEnrichedOrders(ctx, allTypeOrders, talentId, shopName, startTime, endTime)
		stats := s.statsFromEnrichedOrders(ctx, filteredOrders, talentId, shopName, startTime, endTime)
		orderListApplyCommissionScope(stats, orderType)
		stats.AdCostAmount = 0
		if allTypeStats.ValidPayAmount > 0 {
			stats.AdCostAmount = roundFloatOrder(allTypeStats.AdCostAmount*stats.ValidPayAmount/allTypeStats.ValidPayAmount, 2)
		}
		commissionDeduction := stats.ValidPayAmount * 0.005
		stats.RealtimeProfit = roundFloatOrder(CalculateOrderProfit(stats.ValidPayAmount, stats.ProductCostAmount, stats.ReserveAmount, stats.RefundCostAmount, stats.ShopServiceFee, commissionDeduction, 0, stats.AdCostAmount), 2)
		return stats, nil
	}

	// 统计与千川归属统一使用支付完成时间。
	dateWhere, dateParams := buildTimeRangeWhere(orderListStatTimeExpr("o"), startTime, endTime)

	whereClauses := []string{dateWhere}
	whereParams := append([]interface{}{}, dateParams...)

	// 关键词搜索（订单号、商品名称、达人名称、抖音号）
	if keyword != "" {
		searchKey := "%" + keyword + "%"
		whereClauses = append(whereClauses, orderListCountKeywordCondition("o"))
		whereParams = append(whereParams, searchKey, searchKey, searchKey, searchKey, searchKey, searchKey)
	}

	// 账号筛选
	if shopName != "" {
		whereClauses = append(whereClauses, "o.shop_name = ?")
		whereParams = append(whereParams, shopName)
	}

	// 订单状态筛选
	if orderStatus != "" {
		whereClauses = append(whereClauses, "o.order_status = ?")
		whereParams = append(whereParams, orderStatus)
	}

	// 达人 UID 筛选
	if talentId != "" {
		whereClauses = append(whereClauses, orderListCountTalentCondition("o"))
		whereParams = append(whereParams, talentId, talentId, talentId, talentId)
	}

	validJoinClause := financeValidOrderCondition("o")
	invalidJoinClause := financeInvalidOrderCondition("o")
	refundJoinClause := financeRefundOrderCondition("o")

	// 千川归属是稳定加权的整单结果。达人筛选必须先完成归属，再统计最终命中的订单。
	if talentId != "" {
		detailQuery := "SELECT o.*, COALESCE(g.base_price, 0) AS base_price FROM order_info o LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) g ON o.seller_code = g.goods_code WHERE " + strings.Join(whereClauses, " AND ") + " ORDER BY o.id DESC"
		rows, detailErr := g.DB().GetAll(ctx, detailQuery, whereParams...)
		if detailErr != nil {
			return nil, detailErr
		}
		orders, enrichErr := s.enrichDouyinOrderRows(ctx, rows)
		if enrichErr != nil {
			return nil, enrichErr
		}
		orders = orderListFilterByTalent(orders, talentId)
		return s.statsFromEnrichedOrders(ctx, orders, talentId, shopName, startTime, endTime), nil
	}

	aggregateQuery := `
		SELECT
			COUNT(*) AS total_order_count,
			COALESCE(SUM(CASE WHEN ` + invalidJoinClause + ` THEN 1 ELSE 0 END), 0) AS invalid_order_count,
			COALESCE(SUM(CASE WHEN ` + validJoinClause + ` THEN 1 ELSE 0 END), 0) AS valid_order_count,
			COALESCE(SUM(CASE WHEN ` + validJoinClause + ` THEN o.product_price * o.product_num ELSE 0 END), 0) AS valid_pay_amount,
			COALESCE(SUM(CASE WHEN ` + validJoinClause + ` THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS product_cost_amount,
			COALESCE(SUM(CASE WHEN ` + validJoinClause + ` THEN o.product_price * o.product_num * 0.03 ELSE 0 END), 0) AS reserve_amount,
			COALESCE(SUM(CASE WHEN ` + refundJoinClause + ` THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount,
			COALESCE(SUM(CASE WHEN ` + refundJoinClause + ` THEN 1 ELSE 0 END), 0) AS refund_order_count,
			COALESCE(SUM(CASE WHEN ` + validJoinClause + ` AND COALESCE(g.base_price, 0) > 0 THEN 1 ELSE 0 END), 0) AS valid_order_cost_match
		FROM order_info o
		LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) g ON o.seller_code = g.goods_code
		WHERE ` + strings.Join(whereClauses, " AND ")

	row, err := g.DB().GetOne(ctx, aggregateQuery, whereParams...)
	if err != nil {
		return nil, err
	}
	stats.TotalOrderCount = row["total_order_count"].Int64()
	stats.InvalidOrderCount = row["invalid_order_count"].Int64()
	stats.ValidOrderCount = row["valid_order_count"].Int64()
	stats.ValidPayAmount = row["valid_pay_amount"].Float64()
	stats.ProductCostAmount = row["product_cost_amount"].Float64()
	stats.ReserveAmount = CalculateReserveAmount(stats.ValidPayAmount)
	stats.RefundCostAmount = row["refund_cost_amount"].Float64()
	stats.RefundOrderCount = row["refund_order_count"].Int64()
	stats.ValidOrderCostMatch = row["valid_order_cost_match"].Int64()

	// 3. 未命中 = 总数 - 有效 - 无效
	stats.UnmatchedOrderCount = stats.TotalOrderCount - stats.ValidOrderCount - stats.InvalidOrderCount
	totalProductAmount := stats.ValidPayAmount

	// 7. 线上佣金 = 总商品金额 - 留底 - 货款（显示用）
	stats.OnlineCommission = totalProductAmount - stats.ReserveAmount - stats.ProductCostAmount
	if stats.OnlineCommission < 0 {
		stats.OnlineCommission = 0
	}

	// 8. 小店服务费 = 总商品金额 × 2%
	stats.ShopServiceFee = CalculateDouyinShopServiceFee(totalProductAmount)

	// 10. 广告费：从千川报表获取（按店铺过滤）
	stats.AdCostAmount = s.getAdCost(ctx, shopName, startTime, endTime)

	startDate, endDate := orderListQueryDateRange(startTime, endTime)
	enrichedOrders, enrichErr := s.loadEnrichedDouyinOrders(ctx, startDate, endDate)
	if enrichErr == nil {
		filteredOrders := orderListFilterEnriched(enrichedOrders, keyword, shopName, orderStatus, talentId, "", startTime, endTime)
		for _, order := range filteredOrders {
			orderStatus := gconv.String(order["rawOrderStatus"])
			afterSaleStatus := gconv.String(order["afterSaleStatus"])
			if !IsValidOrderForStats(orderStatus, afterSaleStatus) {
				continue
			}
			stats.OfflineCommission += orderListOfflineCommissionAmount(order)
		}
		stats.OfflineCommission = roundFloatOrder(stats.OfflineCommission, 2)
	}

	// 12. 有佣金的订单数：与当前筛选口径保持一致，只统计命中联盟订单的有效订单
	commissionQuery := `
		SELECT COUNT(*) AS total
		FROM order_info o
		WHERE ` + strings.Join(whereClauses, " AND ") + `
		  AND ` + validJoinClause + `
		  AND EXISTS (
			SELECT 1
			FROM alliance_orders ao
			WHERE ao.platform = 'douyin'
			  AND ao.order_id = o.sub_order_no
		  )`
	row, err = g.DB().GetOne(ctx, commissionQuery, whereParams...)
	if err == nil {
		stats.CommissionOrderCount = row["total"].Int64()
	}

	// 13. 实时利润 = 总商品金额 - 货款 - 留底 - 退款成本 - 小店服务费 - 佣金扣除 - 广告费
	// 佣金扣除 = 定价总额 × 0.5%（即 5% × 0.1）
	commissionDeduction := totalProductAmount * 0.005
	stats.RealtimeProfit = roundFloatOrder(CalculateOrderProfit(totalProductAmount, stats.ProductCostAmount, stats.ReserveAmount, stats.RefundCostAmount, stats.ShopServiceFee, commissionDeduction, 0, stats.AdCostAmount), 2)

	return stats, nil
}

func (s *OrderService) statsFromEnrichedOrders(ctx context.Context, orders []g.Map, talentID, shopName, startTime, endTime string) *OrderStats {
	stats := &OrderStats{SyncDate: time.Now().Format("2006-01-02")}
	for _, order := range orders {
		stats.TotalOrderCount++
		orderStatus := gconv.String(order["rawOrderStatus"])
		afterSaleStatus := gconv.String(order["afterSaleStatus"])
		if !IsValidOrderForStats(orderStatus, afterSaleStatus) {
			if IsInvalidOrderForStats(afterSaleStatus) {
				stats.InvalidOrderCount++
			}
			if !orderListUsesAllocatedBlankRefundRate(order) && IsRefundCostOrderForStats(afterSaleStatus, gconv.String(order["sendTime"])) {
				stats.RefundOrderCount++
				stats.RefundCostAmount += gconv.Float64(order["productCost"])
			}
			continue
		}

		stats.ValidOrderCount++
		productNum := gconv.Float64(order["productNum"])
		payAmount := gconv.Float64(order["productPrice"]) * productNum
		productCost := gconv.Float64(order["productCost"])
		stats.ValidPayAmount += payAmount
		stats.ProductCostAmount += productCost
		if productCost > 0 {
			stats.ValidOrderCostMatch++
		}
		if gconv.Float64(order["commission"]) != 0 {
			stats.CommissionOrderCount++
		}
		stats.OfflineCommission += orderListOfflineCommissionAmount(order)
		stats.RefundCostAmount += orderListAllocatedBlankRefundCost(order)
	}

	stats.UnmatchedOrderCount = stats.TotalOrderCount - stats.ValidOrderCount - stats.InvalidOrderCount
	stats.ValidPayAmount = roundFloatOrder(stats.ValidPayAmount, 2)
	stats.ProductCostAmount = roundFloatOrder(stats.ProductCostAmount, 2)
	stats.ReserveAmount = CalculateReserveAmount(stats.ValidPayAmount)
	stats.RefundCostAmount = roundFloatOrder(stats.RefundCostAmount, 2)
	stats.ShopServiceFee = CalculateDouyinShopServiceFee(stats.ValidPayAmount)
	stats.OnlineCommission = roundFloatOrder(stats.ValidPayAmount-stats.ReserveAmount-stats.ProductCostAmount, 2)
	stats.OfflineCommission = roundFloatOrder(stats.OfflineCommission, 2)
	if stats.OnlineCommission < 0 {
		stats.OnlineCommission = 0
	}
	if orderListNormalizedTalentID(talentID) == "" {
		stats.AdCostAmount = s.getAdCost(ctx, shopName, startTime, endTime)
	} else {
		stats.AdCostAmount = s.getAdCostByTalent(ctx, talentID, shopName, startTime, endTime)
	}
	commissionDeduction := stats.ValidPayAmount * 0.005
	stats.RealtimeProfit = roundFloatOrder(CalculateOrderProfit(stats.ValidPayAmount, stats.ProductCostAmount, stats.ReserveAmount, stats.RefundCostAmount, stats.ShopServiceFee, commissionDeduction, 0, stats.AdCostAmount), 2)
	return stats
}

func douyinOrderMatchesBoundUID(order g.Map, selectedUIDs map[string]struct{}) bool {
	rawTalentUID := orderListNormalizedTalentID(gconv.String(order["originalTalentId"]))
	if rawTalentUID != "" {
		_, selected := selectedUIDs[rawTalentUID]
		return selected
	}
	if strings.TrimSpace(gconv.String(order["profitAttributionSource"])) != "qianchuan_plan" {
		return false
	}
	inferredTalentUID := orderListNormalizedTalentID(gconv.String(order["talentId"]))
	_, selected := selectedUIDs[inferredTalentUID]
	return selected
}

// statsDouyinOrdersByOwner 按账号绑定 UID 汇总投手抖音订单。
// 原始 UID 始终优先；仅真正无达人信息且被千川计划明确分配的小店自营订单可用反推 UID。
func (s *OrderService) statsDouyinOrdersByOwnerLegacy(ctx context.Context, accountIDs []int, _ string, startDate, endDate string) (*model.FinanceStatsSummary, error) {
	if len(accountIDs) == 0 {
		return &model.FinanceStatsSummary{}, nil
	}

	uidRows, err := g.DB().Model("account").Ctx(ctx).
		WhereIn("id", accountIDs).
		Where("platform", "douyin").
		Where("status", 1).
		Fields("uid").All()
	if err != nil {
		return nil, err
	}
	selectedUIDs := make(map[string]struct{}, len(uidRows))
	for _, row := range uidRows {
		if uid := orderListNormalizedTalentID(row["uid"].String()); uid != "" {
			selectedUIDs[uid] = struct{}{}
		}
	}
	if len(selectedUIDs) == 0 {
		return &model.FinanceStatsSummary{}, nil
	}

	enriched, err := s.loadEnrichedDouyinOrders(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	stats := &model.FinanceStatsSummary{}
	totalOrders := make(map[string]struct{})
	validOrders := make(map[string]struct{})
	invalidOrders := make(map[string]struct{})
	refundOrders := make(map[string]struct{})
	validCostOrders := make(map[string]struct{})
	commissionOrders := make(map[string]struct{})
	for _, order := range enriched {
		if !douyinOrderMatchesBoundUID(order, selectedUIDs) {
			continue
		}

		orderKey := orderListFirstNonEmpty(
			gconv.String(order["mainOrderNo"]),
			gconv.String(order["subOrderNo"]),
			gconv.String(order["allocationRowId"]),
		)
		if orderKey == "" {
			continue
		}
		totalOrders[orderKey] = struct{}{}

		orderStatus := gconv.String(order["rawOrderStatus"])
		afterSaleStatus := gconv.String(order["afterSaleStatus"])
		if !IsValidOrderForStats(orderStatus, afterSaleStatus) {
			if IsInvalidOrderForStats(afterSaleStatus) {
				invalidOrders[orderKey] = struct{}{}
			}
			if !orderListUsesAllocatedBlankRefundRate(order) && IsRefundCostOrderForStats(afterSaleStatus, gconv.String(order["sendTime"])) {
				refundOrders[orderKey] = struct{}{}
				stats.RefundCostAmount += gconv.Float64(order["productCost"])
			}
			continue
		}

		validOrders[orderKey] = struct{}{}
		productNum := gconv.Int(order["productNum"])
		productCost := gconv.Float64(order["productCost"])
		stats.ProductNum += productNum
		stats.PayAmount += gconv.Float64(order["productPrice"]) * float64(productNum)
		stats.CostAmount += productCost
		if productCost > 0 {
			validCostOrders[orderKey] = struct{}{}
		}
		if gconv.Float64(order["commission"]) != 0 {
			commissionOrders[orderKey] = struct{}{}
		}
		stats.RefundCostAmount += orderListAllocatedBlankRefundCost(order)
	}

	stats.TotalOrderCount = len(totalOrders)
	stats.OrderCount = len(validOrders)
	stats.InvalidOrderCount = len(invalidOrders)
	stats.RefundOrderCount = len(refundOrders)
	stats.ValidOrderCostMatch = len(validCostOrders)
	stats.CommissionOrderCount = len(commissionOrders)
	stats.PayAmount = roundFloatFinance(stats.PayAmount, 2)
	stats.CostAmount = roundFloatFinance(stats.CostAmount, 2)
	stats.ReserveAmount = CalculateReserveAmount(stats.PayAmount)
	stats.RefundCostAmount = roundFloatFinance(stats.RefundCostAmount, 2)
	stats.ShopServiceFee = CalculateDouyinShopServiceFee(stats.PayAmount)
	// 与 statsOrderTable 保持返回契约一致，平台服务费由上层合并时统一扣除。
	stats.Commission = roundFloatFinance(stats.PayAmount-stats.ReserveAmount-stats.CostAmount, 2)
	commissionDeduction := roundFloatFinance(stats.PayAmount*0.005, 2)
	stats.RealtimeProfit = roundFloatFinance(CalculateOrderProfit(
		stats.PayAmount,
		stats.CostAmount,
		stats.ReserveAmount,
		stats.RefundCostAmount,
		stats.ShopServiceFee,
		commissionDeduction,
		0,
		0,
	), 2)
	return stats, nil
}

// statsDouyinOrdersByAccounts 一次扫描订单并按账号叶子分组，用于投手下钻时避免每个账号重复扫描全部订单。
func (s *OrderService) statsDouyinOrdersByAccounts(ctx context.Context, accountIDs []int, startDate, endDate string) (map[int]*model.FinanceStatsSummary, error) {
	result := make(map[int]*model.FinanceStatsSummary, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	rows, err := g.DB().Model("account").Ctx(ctx).
		LeftJoin("trader_account_relation tar", "tar.account_id = account.id").
		WhereIn("id", accountIDs).
		Where("platform", "douyin").
		Where("status", 1).
		Fields("account.id, account.uid, MIN(tar.trader_id) AS trader_id").
		Group("account.id, account.uid").
		All()
	if err != nil {
		return nil, err
	}
	accountByUID := make(map[string]int, len(rows))
	sourceTraderByAccountID := make(map[int]int64, len(rows))
	firstAccountByTraderID := make(map[int64]int, len(rows))
	for _, row := range rows {
		accountID := row["id"].Int()
		traderID := row["trader_id"].Int64()
		result[accountID] = &model.FinanceStatsSummary{}
		sourceTraderByAccountID[accountID] = traderID
		if traderID > 0 {
			if existing, ok := firstAccountByTraderID[traderID]; !ok || accountID < existing {
				firstAccountByTraderID[traderID] = accountID
			}
		}
		if uid := orderListNormalizedTalentID(row["uid"].String()); uid != "" {
			if existing, ok := accountByUID[uid]; !ok || accountID < existing {
				accountByUID[uid] = accountID
			}
		}
	}

	type accumulator struct {
		stats            *model.FinanceStatsSummary
		totalOrders      map[string]struct{}
		validOrders      map[string]struct{}
		invalidOrders    map[string]struct{}
		refundOrders     map[string]struct{}
		validCostOrders  map[string]struct{}
		commissionOrders map[string]struct{}
	}
	accumulators := make(map[int]*accumulator, len(result))
	for accountID, stats := range result {
		accumulators[accountID] = &accumulator{
			stats: stats, totalOrders: make(map[string]struct{}), validOrders: make(map[string]struct{}),
			invalidOrders: make(map[string]struct{}), refundOrders: make(map[string]struct{}),
			validCostOrders: make(map[string]struct{}), commissionOrders: make(map[string]struct{}),
		}
	}

	qianchuanBlankOrderInferenceEnabled := IsQianchuanPlanBlankOrderInferenceEnabled(ctx)
	ctx = withQianchuanPlanBlankOrderInferenceSetting(ctx, qianchuanBlankOrderInferenceEnabled)
	enriched, err := s.loadEnrichedDouyinOrders(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}
	for _, order := range enriched {
		if !qianchuanBlankOrderInferenceEnabled && strings.TrimSpace(gconv.String(order["profitAttributionSource"])) == "qianchuan_plan" {
			continue
		}
		uid := orderListNormalizedTalentID(gconv.String(order["originalTalentId"]))
		if uid == "" && strings.TrimSpace(gconv.String(order["profitAttributionSource"])) == "qianchuan_plan" {
			uid = orderListNormalizedTalentID(gconv.String(order["talentId"]))
		}
		accountID := 0
		if gconv.Bool(order["specialAttributionMatched"]) {
			targetTraderID := gconv.Int64(order["attributedTraderId"])
			if targetTraderID <= 0 {
				continue
			}
			targetAccountID, matched := firstAccountByTraderID[targetTraderID]
			if !matched {
				continue
			}
			accountID = targetAccountID
		} else {
			sourceAccountID, matched := accountByUID[uid]
			if !matched {
				continue
			}
			if targetTraderID := gconv.Int64(order["attributedTraderId"]); targetTraderID > 0 {
				if sourceTraderID := sourceTraderByAccountID[sourceAccountID]; sourceTraderID > 0 && sourceTraderID != targetTraderID {
					continue
				}
			}
			accountID = sourceAccountID
		}
		acc := accumulators[accountID]
		orderKey := orderListFirstNonEmpty(gconv.String(order["mainOrderNo"]), gconv.String(order["subOrderNo"]), gconv.String(order["allocationRowId"]))
		if orderKey == "" {
			continue
		}
		acc.totalOrders[orderKey] = struct{}{}
		orderStatus := gconv.String(order["rawOrderStatus"])
		afterSaleStatus := gconv.String(order["afterSaleStatus"])
		if !IsValidOrderForStats(orderStatus, afterSaleStatus) {
			if IsInvalidOrderForStats(afterSaleStatus) {
				acc.invalidOrders[orderKey] = struct{}{}
			}
			if !orderListUsesAllocatedBlankRefundRate(order) && IsRefundCostOrderForStats(afterSaleStatus, gconv.String(order["sendTime"])) {
				acc.refundOrders[orderKey] = struct{}{}
				acc.stats.RefundCostAmount += gconv.Float64(order["productCost"])
			}
			continue
		}
		acc.validOrders[orderKey] = struct{}{}
		productNum := gconv.Int(order["productNum"])
		productCost := gconv.Float64(order["productCost"])
		acc.stats.ProductNum += productNum
		acc.stats.PayAmount += gconv.Float64(order["productPrice"]) * float64(productNum)
		acc.stats.CostAmount += productCost
		if productCost > 0 {
			acc.validCostOrders[orderKey] = struct{}{}
		}
		if gconv.Float64(order["commission"]) != 0 {
			acc.commissionOrders[orderKey] = struct{}{}
		}
		acc.stats.RefundCostAmount += orderListAllocatedBlankRefundCost(order)
	}

	for _, acc := range accumulators {
		stats := acc.stats
		stats.TotalOrderCount = len(acc.totalOrders)
		stats.OrderCount = len(acc.validOrders)
		stats.InvalidOrderCount = len(acc.invalidOrders)
		stats.RefundOrderCount = len(acc.refundOrders)
		stats.ValidOrderCostMatch = len(acc.validCostOrders)
		stats.CommissionOrderCount = len(acc.commissionOrders)
		stats.PayAmount = roundFloatFinance(stats.PayAmount, 2)
		stats.CostAmount = roundFloatFinance(stats.CostAmount, 2)
		stats.ReserveAmount = CalculateReserveAmount(stats.PayAmount)
		stats.RefundCostAmount = roundFloatFinance(stats.RefundCostAmount, 2)
		stats.ShopServiceFee = CalculateDouyinShopServiceFee(stats.PayAmount)
		// 与 statsOrderTable 保持契约一致：返回小店服务费，但 Commission 先不扣服务费，由财务层统一扣除。
		stats.Commission = roundFloatFinance(stats.PayAmount-stats.ReserveAmount-stats.CostAmount, 2)
	}
	return result, nil
}

func (s *OrderService) statsDouyinOrdersByOwner(ctx context.Context, accountIDs []int, _ string, startDate, endDate string) (*model.FinanceStatsSummary, error) {
	leafStats, err := s.statsDouyinOrdersByAccounts(ctx, accountIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	adCosts, err := NewFinanceCommissionService().getAdCostsByAccountRange(ctx, accountIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	total := &model.FinanceStatsSummary{}
	for _, accountID := range uniqueIntSlice(accountIDs) {
		if stats := leafStats[accountID]; stats != nil {
			stats.AdCostAmount = roundFloatFinance(adCosts[accountID], 2)
			commissionDeduction := roundFloatFinance(stats.PayAmount*0.005, 2)
			stats.RealtimeProfit = roundFloatFinance(CalculateOrderProfit(
				stats.PayAmount, stats.CostAmount, stats.ReserveAmount, stats.RefundCostAmount,
				stats.ShopServiceFee, commissionDeduction, 0, stats.AdCostAmount,
			), 2)
			mergeFinanceStatsSummary(total, stats)
		}
	}
	return total, nil
}

func (s *OrderService) getAdCostByTalent(ctx context.Context, talentID, shopName, startTime, endTime string) float64 {
	if orderListNormalizedTalentID(talentID) == "" {
		return 0
	}
	query := `
		SELECT COALESCE(SUM(qar.stat_cost), 0) AS ad_cost
		FROM qianchuan_ad_report qar
		INNER JOIN (
			SELECT platform_nickname, MAX(uid) AS uid
			FROM account
			WHERE platform = 'douyin' AND status = 1
			  AND platform_nickname IS NOT NULL AND platform_nickname != ''
			GROUP BY platform_nickname
		) target ON target.platform_nickname COLLATE utf8mb4_unicode_ci = qar.aweme_show_id COLLATE utf8mb4_unicode_ci
		LEFT JOIN account legacy_shop ON legacy_shop.id = qar.account_id
		LEFT JOIN (
			SELECT sqm.qianchuan_advertiser_id, MAX(s.store_name) AS shop_name
			FROM store_qianchuan_mapping sqm
			INNER JOIN store s ON s.id = sqm.store_id
			GROUP BY sqm.qianchuan_advertiser_id
		) mapped_shop ON mapped_shop.qianchuan_advertiser_id = qar.advertiser_id
		WHERE target.uid = ?`
	params := []interface{}{talentID}
	if shopName != "" {
		query += " AND COALESCE(mapped_shop.shop_name, legacy_shop.account_name) = ?"
		params = append(params, shopName)
	}
	qStart, qEnd := orderAdCostDateRange(startTime, endTime)
	if qStart != "" {
		query += " AND LEFT(qar.stat_datetime, 10) >= ?"
		params = append(params, qStart)
	}
	if qEnd != "" {
		query += " AND LEFT(qar.stat_datetime, 10) <= ?"
		params = append(params, qEnd)
	}
	row, err := g.DB().GetOne(ctx, query, params...)
	if err != nil {
		g.Log().Warningf(ctx, "查询达人千川广告费失败: talentId=%s, err=%v", talentID, err)
		return 0
	}
	return roundFloatOrder(row["ad_cost"].Float64(), 2)
}

// getAdCost 从千川广告报表获取指定时间段内的广告费消耗
// 当 shopName 非空时，通过 account_qianchuan_mapping 关联到具体账号的广告主ID
func (s *OrderService) getAdCost(ctx context.Context, shopName string, startTime, endTime string) float64 {
	// stat_datetime 是日报日期；次日零点作为时间区间右边界时必须排除次日。
	qStart, qEnd := orderAdCostDateRange(startTime, endTime)

	if shopName != "" {
		// 按店铺关联千川广告主ID查询（通过 account 表关联 shopName）
		row, err := g.DB().Model("account_qianchuan_mapping aqm").
			LeftJoin("account a", "aqm.account_id = a.id").
			LeftJoin("qianchuan_advertiser_daily_report qar", qianchuanAdReportJoinCondRange(qStart, qEnd)).
			Where("a.account_name = ? OR a.platform_nickname = ?", shopName, shopName).
			Fields("COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
			One()
		if err != nil {
			g.Log().Warningf(ctx, "查询千川广告费(按店铺)失败: shopName=%s, err=%v", shopName, err)
			return 0
		}
		return row["ad_cost"].Float64()
	}

	// 无店铺过滤时全表汇总
	query := "SELECT COALESCE(SUM(stat_cost), 0) as cost FROM qianchuan_advertiser_daily_report WHERE 1=1"
	params := []interface{}{}
	if qStart != "" {
		query += " AND LEFT(stat_datetime, 10) >= ?"
		params = append(params, qStart)
	}
	if qEnd != "" {
		query += " AND LEFT(stat_datetime, 10) <= ?"
		params = append(params, qEnd)
	}

	row, err := g.DB().GetOne(ctx, query, params...)
	if err != nil {
		g.Log().Warningf(ctx, "查询千川广告费失败: %v", err)
		return 0
	}
	return row["cost"].Float64()
}

// List 查询订单列表
func (s *OrderService) List(ctx context.Context, page, pageSize int, keyword, shopName, orderStatus, talentId, orderType string, startTime, endTime string) ([]g.Map, int64, error) {
	if normalizeOrderListType(orderType) != "" {
		startDate, endDate := orderListQueryDateRange(startTime, endTime)
		orders, err := s.loadEnrichedDouyinOrders(ctx, startDate, endDate)
		if err != nil {
			return nil, 0, err
		}
		orders = orderListFilterEnriched(orders, keyword, shopName, orderStatus, talentId, orderType, startTime, endTime)
		return orderListPage(orders, page, pageSize), int64(len(orders)), nil
	}
	query := "SELECT o.*, COALESCE(g.base_price, 0) AS base_price FROM order_info o LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) g ON o.seller_code = g.goods_code WHERE 1=1"
	params := []interface{}{}
	countQuery := "SELECT COUNT(*) as total FROM order_info o WHERE 1=1"
	countParams := []interface{}{}

	// 关键词搜索（订单号、商品名称、达人名称、抖音号）
	if keyword != "" {
		searchKey := "%" + keyword + "%"
		query += " AND " + orderListCountKeywordCondition("o")
		params = append(params, searchKey, searchKey, searchKey, searchKey, searchKey, searchKey)
		countQuery += " AND " + orderListCountKeywordCondition("o")
		countParams = append(countParams, searchKey, searchKey, searchKey, searchKey, searchKey, searchKey)
	}

	// 账号筛选
	if shopName != "" {
		query += " AND o.shop_name = ?"
		params = append(params, shopName)
		countQuery += " AND o.shop_name = ?"
		countParams = append(countParams, shopName)
	}

	// 订单状态筛选
	if orderStatus != "" {
		query += " AND o.order_status = ?"
		params = append(params, orderStatus)
		countQuery += " AND o.order_status = ?"
		countParams = append(countParams, orderStatus)
	}

	// 达人UID筛选
	if talentId != "" {
		query += " AND " + orderListCountTalentCondition("o")
		params = append(params, talentId, talentId, talentId, talentId)
		countQuery += " AND " + orderListCountTalentCondition("o")
		countParams = append(countParams, talentId, talentId, talentId, talentId)
	}

	// 时间范围与千川归属统一使用订单创建时间。
	if startTime != "" {
		query += " AND " + orderListStatTimeExpr("o") + " >= ?"
		params = append(params, startTime)
		countQuery += " AND " + orderListStatTimeExpr("o") + " >= ?"
		countParams = append(countParams, startTime)
	}
	if endTime != "" {
		query += " AND " + orderListStatTimeExpr("o") + " <= ?"
		params = append(params, endTime)
		countQuery += " AND " + orderListStatTimeExpr("o") + " <= ?"
		countParams = append(countParams, endTime)
	}

	if talentId != "" {
		query += " ORDER BY o.id DESC"
		result, err := g.DB().GetAll(ctx, query, params...)
		if err != nil {
			return nil, 0, err
		}
		orders, err := s.enrichDouyinOrderRows(ctx, result)
		if err != nil {
			return nil, 0, err
		}
		orders = orderListFilterByTalent(orders, talentId)
		orders = orderListFilterByType(orders, orderType)
		return orderListPage(orders, page, pageSize), int64(len(orders)), nil
	}

	// 获取总数
	resultTotal, err := g.DB().GetOne(ctx, countQuery, countParams...)
	if err != nil {
		return nil, 0, err
	}
	total := resultTotal["total"].Int64()

	// 分页查询
	query += " ORDER BY o.id DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	params = append(params, pageSize, offset)

	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}

	orders, err := s.enrichDouyinOrderRows(ctx, result)
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ExportToExcel 导出订单数据到Excel
func (s *OrderService) ExportToExcel(ctx context.Context, keyword, shopName, orderStatus, talentId, orderType string, startTime, endTime string) (string, error) {
	if normalizeOrderListType(orderType) != "" && startTime == "" && endTime == "" {
		startDate, _ := orderListQueryDateRange(startTime, endTime)
		startTime = startDate + " 00:00:00"
		endTime = startDate + " 23:59:59"
	}
	query := "SELECT o.*, COALESCE(g.base_price, 0) AS base_price FROM order_info o LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) g ON o.seller_code = g.goods_code WHERE 1=1"
	params := []interface{}{}

	if keyword != "" {
		searchKey := "%" + keyword + "%"
		query += " AND " + orderListCountKeywordCondition("o")
		params = append(params, searchKey, searchKey, searchKey, searchKey, searchKey, searchKey)
	}

	if shopName != "" {
		query += " AND o.shop_name = ?"
		params = append(params, shopName)
	}

	if orderStatus != "" {
		query += " AND o.order_status = ?"
		params = append(params, orderStatus)
	}
	if talentId != "" {
		query += " AND " + orderListCountTalentCondition("o")
		params = append(params, talentId, talentId, talentId, talentId)
	}

	if startTime != "" {
		query += " AND " + orderListStatTimeExpr("o") + " >= ?"
		params = append(params, startTime)
	}
	if endTime != "" {
		query += " AND " + orderListStatTimeExpr("o") + " <= ?"
		params = append(params, endTime)
	}

	query += " ORDER BY o.id DESC"
	result, err := g.DB().GetAll(ctx, query, params...)
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "", gerror.New("没有数据可导出")
	}

	orders, err := s.enrichDouyinOrderRows(ctx, result)
	if err != nil {
		return "", err
	}
	orders = orderListFilterByTalent(orders, talentId)
	orders = orderListFilterByType(orders, orderType)
	if len(orders) == 0 {
		return "", gerror.New("没有数据可导出")
	}

	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "订单数据"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头（与前端订单列表列完全一致）
	headers := []string{
		"账号", "订单号", "商品数量", "商品名称", "商品规格", "商品单价",
		"订单应付金额", "商品成本支出", "小店服务费", "UID", "达人", "佣金率",
		"佣金", "抖音号", "团队归属", "订单归属", "匹配方式", "流量来源", "订单状态",
		"归属权重", "千川计划ID", "千川计划名称", "千川抖音号",
		"下单时间", "发货时间", "支付时间", "订单完成时间",
	}

	// 写入表头
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置表头样式
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// 写入数据
	for i, row := range orders {
		rowNum := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row["shopName"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row["mainOrderNo"])
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), row["productNum"])
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), row["productName"])
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), row["productSpec"])
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), row["productPrice"])
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), row["orderPayAmount"])
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNum), row["productCost"])
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowNum), row["shopServiceFee"])
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowNum), row["uid"])
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowNum), row["talentName"])
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowNum), row["commissionRate"])
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowNum), row["commission"])
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowNum), row["douyinAccount"])
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowNum), row["teamOwner"])
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowNum), row["orderType"])
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", rowNum), row["matchedBy"])
		f.SetCellValue(sheetName, fmt.Sprintf("R%d", rowNum), row["flowSource"])
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", rowNum), row["orderStatus"])
		f.SetCellValue(sheetName, fmt.Sprintf("T%d", rowNum), row["allocationPercent"])
		f.SetCellValue(sheetName, fmt.Sprintf("U%d", rowNum), row["qianchuanPlanId"])
		f.SetCellValue(sheetName, fmt.Sprintf("V%d", rowNum), row["qianchuanPlanName"])
		f.SetCellValue(sheetName, fmt.Sprintf("W%d", rowNum), row["qianchuanAwemeShowId"])
		f.SetCellValue(sheetName, fmt.Sprintf("X%d", rowNum), formatTime(row["orderSubmitTime"]))
		f.SetCellValue(sheetName, fmt.Sprintf("Y%d", rowNum), formatTime(row["sendTime"]))
		f.SetCellValue(sheetName, fmt.Sprintf("Z%d", rowNum), formatTime(row["payFinishTime"]))
		f.SetCellValue(sheetName, fmt.Sprintf("AA%d", rowNum), formatTime(row["orderFinishTime"]))
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 18)  // 账号
	f.SetColWidth(sheetName, "B", "B", 22)  // 订单号
	f.SetColWidth(sheetName, "C", "C", 10)  // 商品数量
	f.SetColWidth(sheetName, "D", "D", 30)  // 商品名称
	f.SetColWidth(sheetName, "E", "E", 18)  // 商品规格
	f.SetColWidth(sheetName, "F", "I", 14)  // 单价/金额
	f.SetColWidth(sheetName, "J", "J", 16)  // UID
	f.SetColWidth(sheetName, "K", "K", 14)  // 达人
	f.SetColWidth(sheetName, "L", "M", 12)  // 佣金率/佣金
	f.SetColWidth(sheetName, "N", "N", 14)  // 抖音号
	f.SetColWidth(sheetName, "O", "R", 14)  // 团队归属/订单归属/匹配方式/流量来源
	f.SetColWidth(sheetName, "S", "S", 12)  // 订单状态
	f.SetColWidth(sheetName, "T", "W", 18)  // 千川分摊信息
	f.SetColWidth(sheetName, "X", "AA", 18) // 时间列

	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, fmt.Sprintf("order_export_%s.xlsx", time.Now().Format("20060102150405")))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}

	return filename, nil
}

// formatTime 格式化时间
func formatTime(val interface{}) string {
	if val == nil {
		return ""
	}
	if t, ok := val.(time.Time); ok {
		return t.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%v", val)
}

// ImportResult 导入结果
type OrderImportResult struct {
	TotalRows    int      `json:"totalRows"`
	SuccessCount int      `json:"successCount"`
	SkipCount    int      `json:"skipCount"`
	FailCount    int      `json:"failCount"`
	Errors       []string `json:"errors"`
}

// ImportFromExcel 从Excel导入订单数据（流式处理+批量入库）
func (s *OrderService) ImportFromExcel(ctx context.Context, filePath string) (*OrderImportResult, error) {
	g.Log().Infof(ctx, "开始导入订单文件: %s", filePath)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		g.Log().Errorf(ctx, "打开Excel文件失败: %s, 错误: %v", filePath, err)
		return nil, fmt.Errorf("打开Excel文件失败(filePath=%s): %w", filePath, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		g.Log().Error(ctx, "Excel文件中没有工作表")
		return nil, gerror.New("Excel文件中没有工作表")
	}

	// 使用流式读取
	rows, err := f.Rows(sheets[0])
	if err != nil {
		g.Log().Errorf(ctx, "读取Excel数据失败: %v", err)
		return nil, fmt.Errorf("读取Excel数据失败：%w", err)
	}
	defer rows.Close()

	result := &OrderImportResult{
		Errors: make([]string, 0),
	}

	// 读取表头
	if !rows.Next() {
		g.Log().Error(ctx, "Excel文件中没有数据")
		return nil, gerror.New("Excel文件中没有数据")
	}
	headers, err := rows.Columns()
	if err != nil {
		g.Log().Errorf(ctx, "读取表头失败: %v", err)
		return nil, fmt.Errorf("读取表头失败：%w", err)
	}

	g.Log().Infof(ctx, "订单Excel表头: %v", headers)

	// 验证表头（至少包含必填字段）
	requiredHeaders := []string{"店铺", "子订单编号", "选购商品", "商品单价", "订单应付金额", "订单状态"}
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			g.Log().Errorf(ctx, "Excel表头缺少必填字段: %s, 实际表头: %v", required, headers)
			return nil, gerror.New(fmt.Sprintf("Excel表头缺少必填字段: %s\n实际读取到的表头: %v", required, headers))
		}
	}

	// 批量处理配置
	const batchSize = 1000
	batch := make([]g.Map, 0, batchSize)
	rowNum := 1

	// 预加载已存在的子订单号
	existingOrders := make(map[string]bool)
	existingRows, err := g.DB().Model("order_info").Fields("sub_order_no").All()
	if err != nil {
		g.Log().Errorf(ctx, "预加载已存在订单失败: %v", err)
	} else {
		for _, row := range existingRows {
			existingOrders[row["sub_order_no"].String()] = true
		}
		g.Log().Infof(ctx, "已存在订单数量: %d", len(existingOrders))
	}

	// 用于统计的空行、验证失败、已存在跳过数量
	var emptyRowCount, validationFailCount, existSkipCount int

	// 流式读取并批量处理
	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 读取失败 - %v", rowNum, err))
			result.FailCount++
			continue
		}

		result.TotalRows++

		// 跳过空行
		if len(cols) == 0 || (getCol(cols, headerMap, "店铺") == "" && getCol(cols, headerMap, "子订单编号") == "" && getCol(cols, headerMap, "选购商品") == "") {
			emptyRowCount++
			result.SkipCount++
			continue
		}

		// 验证必填字段
		shopName := getCol(cols, headerMap, "店铺")
		subOrderNo := getCol(cols, headerMap, "子订单编号")
		productName := getCol(cols, headerMap, "选购商品")
		orderStatus := getCol(cols, headerMap, "订单状态")

		if shopName == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 店铺不能为空 (子订单号: %s)", rowNum, subOrderNo)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}
		if subOrderNo == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 子订单编号不能为空 (店铺: %s)", rowNum, shopName)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}
		if productName == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 选购商品不能为空 (子订单号: %s)", rowNum, subOrderNo)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}
		if orderStatus == "" {
			validationFailCount++
			errMsg := fmt.Sprintf("第%d行: 订单状态不能为空 (子订单号: %s)", rowNum, subOrderNo)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Warningf(ctx, "订单导入验证失败: %s", errMsg)
			result.FailCount++
			continue
		}

		// 检查子订单号是否已存在
		if existingOrders[subOrderNo] {
			existSkipCount++
			result.SkipCount++
			continue
		}

		// 构建数据（截断可能超长的字段，避免数据库插入失败）
		data := g.Map{
			"shop_name":              truncateString(shopName, 50),
			"main_order_no":          truncateString(getCol(cols, headerMap, "主订单编号"), 50),
			"sub_order_no":           truncateString(subOrderNo, 50),
			"product_name":           truncateString(productName, 255),
			"product_spec":           truncateString(getCol(cols, headerMap, "商品规格"), 100),
			"product_num":            parseIntOrDefault(getCol(cols, headerMap, "商品数量"), 1),
			"product_id":             truncateString(getCol(cols, headerMap, "商品ID"), 50),
			"seller_code":            truncateString(getCol(cols, headerMap, "商家编码"), 50),
			"product_price":          parseFloatOrDefault(getCol(cols, headerMap, "商品单价"), 0),
			"order_pay_amount":       parseFloatOrDefault(getCol(cols, headerMap, "订单应付金额"), 0),
			"shipping_fee":           parseFloatOrDefault(getCol(cols, headerMap, "运费"), 0),
			"total_discount":         parseFloatOrDefault(getCol(cols, headerMap, "优惠总金额"), 0),
			"platform_discount":      parseFloatOrDefault(getCol(cols, headerMap, "平台优惠"), 0),
			"seller_discount":        parseFloatOrDefault(getCol(cols, headerMap, "商家优惠"), 0),
			"talent_discount":        parseFloatOrDefault(getCol(cols, headerMap, "达人优惠"), 0),
			"seller_change_price":    parseFloatOrDefault(getCol(cols, headerMap, "商家改价"), 0),
			"pay_discount":           parseFloatOrDefault(getCol(cols, headerMap, "支付优惠"), 0),
			"red_deduction":          parseFloatOrDefault(getCol(cols, headerMap, "红包抵扣"), 0),
			"pay_type":               truncateString(getCol(cols, headerMap, "支付方式"), 20),
			"service_fee":            parseFloatOrDefault(getCol(cols, headerMap, "手续费"), 0),
			"receiver_name":          truncateString(getCol(cols, headerMap, "收件人"), 50),
			"receiver_phone":         truncateString(getCol(cols, headerMap, "收件人手机号"), 20),
			"province":               truncateString(getCol(cols, headerMap, "省"), 20),
			"city":                   truncateString(getCol(cols, headerMap, "市"), 20),
			"district":               truncateString(getCol(cols, headerMap, "区"), 255),
			"street":                 truncateString(getCol(cols, headerMap, "街道"), 100),
			"address_detail":         truncateString(getCol(cols, headerMap, "详细地址"), 255),
			"is_address_changed":     truncateString(getCol(cols, headerMap, "是否修改过地址"), 10),
			"address_change_time":    truncateString(getCol(cols, headerMap, "地址修改时段"), 50),
			"buyer_note":             truncateString(getCol(cols, headerMap, "买家留言"), 255),
			"order_submit_time":      parseDateTime(getCol(cols, headerMap, "订单提交时间")),
			"flag_color":             truncateString(getCol(cols, headerMap, "旗帜颜色"), 10),
			"seller_note":            truncateString(getCol(cols, headerMap, "商家备注"), 255),
			"order_finish_time":      parseDateTime(getCol(cols, headerMap, "订单完成时间")),
			"pay_finish_time":        parseDateTime(getCol(cols, headerMap, "支付完成时间")),
			"app_channel":            truncateString(getCol(cols, headerMap, "APP渠道"), 20),
			"flow_source":            truncateString(getCol(cols, headerMap, "流量来源"), 20),
			"order_status":           truncateString(orderStatus, 20),
			"promise_send_time":      parseDateTime(getCol(cols, headerMap, "承诺发货时间")),
			"order_type":             truncateString(getCol(cols, headerMap, "订单类型"), 20),
			"luban_page_id":          truncateString(getCol(cols, headerMap, "鲁班落地页ID"), 50),
			"talent_id":              truncateString(getCol(cols, headerMap, "达人ID"), 50),
			"talent_name":            truncateString(getCol(cols, headerMap, "达人昵称"), 50),
			"store_id":               truncateString(getCol(cols, headerMap, "所属门店ID"), 50),
			"after_sale_status":      truncateString(getCol(cols, headerMap, "售后状态"), 20),
			"cancel_reason":          truncateString(getCol(cols, headerMap, "取消原因"), 100),
			"appoint_send_time":      parseDateTime(getCol(cols, headerMap, "预约发货时间")),
			"warehouse_id":           truncateString(getCol(cols, headerMap, "仓库ID"), 50),
			"warehouse_name":         truncateString(getCol(cols, headerMap, "仓库名称"), 50),
			"is_safe_buy":            truncateString(getCol(cols, headerMap, "是否安心购"), 10),
			"ad_channel":             truncateString(getCol(cols, headerMap, "广告渠道"), 20),
			"flow_type":              truncateString(getCol(cols, headerMap, "流量类型"), 20),
			"flow_format":            truncateString(getCol(cols, headerMap, "流量体裁"), 20),
			"flow_channel":           truncateString(getCol(cols, headerMap, "流量渠道"), 20),
			"send_user":              truncateString(getCol(cols, headerMap, "发货主体"), 20),
			"send_user_detail":       truncateString(getCol(cols, headerMap, "发货主体明细"), 50),
			"send_time":              parseDateTime(getCol(cols, headerMap, "发货时间")),
			"reduce_discount":        parseFloatOrDefault(getCol(cols, headerMap, "降价类优惠"), 0),
			"platform_real_discount": parseFloatOrDefault(getCol(cols, headerMap, "平台实际承担优惠金额"), 0),
			"seller_real_discount":   parseFloatOrDefault(getCol(cols, headerMap, "商家实际承担优惠金额"), 0),
			"talent_real_discount":   parseFloatOrDefault(getCol(cols, headerMap, "达人实际承担优惠金额"), 0),
			"expect_arrive_time":     parseDateTime(getCol(cols, headerMap, "预计送达时间")),
			"create_time":            time.Now().Format("2006-01-02 15:04:05"),
		}

		batch = append(batch, data)
		existingOrders[subOrderNo] = true

		// 达到批次大小，执行批量插入
		if len(batch) >= batchSize {
			g.Log().Debugf(ctx, "准备批量插入第%d批，共%d条数据", rowNum/batchSize, len(batch))
			if err := s.batchInsert(ctx, batch); err != nil {
				errMsg := fmt.Sprintf("批量插入失败（约第%d-%d行）: %v", rowNum-batchSize+1, rowNum, err)
				result.Errors = append(result.Errors, errMsg)
				g.Log().Errorf(ctx, "订单导入%s", errMsg)
				result.FailCount += len(batch)
			} else {
				g.Log().Infof(ctx, "批量插入成功：%d条（约第%d-%d行）", len(batch), rowNum-batchSize+1, rowNum)
				result.SuccessCount += len(batch)
			}
			batch = batch[:0]
		}
	}

	// 处理剩余数据
	if len(batch) > 0 {
		if err := s.batchInsert(ctx, batch); err != nil {
			errMsg := fmt.Sprintf("批量插入失败（最后%d行）: %v", len(batch), err)
			result.Errors = append(result.Errors, errMsg)
			g.Log().Errorf(ctx, "订单导入%s", errMsg)
			result.FailCount += len(batch)
		} else {
			g.Log().Infof(ctx, "批量插入成功：%d条（最后一批）", len(batch))
			result.SuccessCount += len(batch)
		}
	}

	// 记录所有错误信息到日志
	if len(result.Errors) > 0 {
		g.Log().Errorf(ctx, "订单导入失败详情（共%d条错误）:", len(result.Errors))
		for i, errMsg := range result.Errors {
			g.Log().Errorf(ctx, "  错误%d: %s", i+1, errMsg)
			// 只打印前200条到日志，避免日志过大
			if i >= 200 {
				g.Log().Errorf(ctx, "  ... 还有 %d 条错误信息请查看返回结果", len(result.Errors)-200)
				break
			}
		}
	}

	// 限制返回给前端的错误信息数量（避免响应过大）
	if len(result.Errors) > 50 {
		result.Errors = append(result.Errors[:50], fmt.Sprintf("... 还有 %d 条错误信息未显示，请查看日志获取完整信息", len(result.Errors)-50))
	}

	g.Log().Infof(ctx, "订单导入完成: 成功%d条，跳过%d条（含空行%d+已存在%d+验证失败%d），失败%d条",
		result.SuccessCount, result.SkipCount, emptyRowCount, existSkipCount, validationFailCount, result.FailCount)
	return result, nil
}

// batchInsert 批量插入订单数据
func (s *OrderService) batchInsert(ctx context.Context, batch []g.Map) error {
	if len(batch) == 0 {
		return nil
	}
	_, err := g.DB().Model("order_info").Ctx(ctx).Data(batch).Insert()
	return err
}

// 辅助函数：获取列值
func getCol(cols []string, headerMap map[string]int, header string) string {
	if idx, exists := headerMap[header]; exists && idx < len(cols) {
		return strings.TrimSpace(cols[idx])
	}
	return ""
}

// alignCols 移除数据列尾部的空列，使列数不超过目标列数
func alignCols(cols []string, targetLen int) []string {
	if len(cols) <= targetLen {
		return cols
	}
	result := make([]string, len(cols))
	copy(result, cols)
	for len(result) > targetLen && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}
	return result
}

// getColFlex 获取列值，支持模糊匹配：当精确索引为空时，搜索附近索引
func getColFlex(cols []string, headerMap map[string]int, header string) string {
	idx, exists := headerMap[header]
	if !exists {
		return ""
	}
	if idx < len(cols) {
		if v := strings.TrimSpace(cols[idx]); v != "" {
			return v
		}
	}
	// 精确索引为空或越界，搜索附近（Excel列错位通常偏移1-2列）
	for _, offset := range []int{1, -1, 2, -2} {
		adj := idx + offset
		if adj >= 0 && adj < len(cols) {
			if v := strings.TrimSpace(cols[adj]); v != "" {
				return v
			}
		}
	}
	return ""
}

// 辅助函数：截断字符串到指定字符数（不是字节数），并清理非法字符
func truncateString(s string, maxLen int) string {
	s = cleanIllegalChars(s)
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return s
}

// cleanIllegalChars 清理非法UTF-8字符和不完整序列
func cleanIllegalChars(s string) string {
	result := strings.Builder{}
	for i := 0; i < len(s); {
		// 解码一个 rune 和它的字节宽度
		r, w := utf8.DecodeRuneInString(s[i:])

		// 情况1: 空字节
		if r == 0 {
			i += w
			continue
		}

		// 情况2: C0 控制字符（保留换行和回车）
		if r < 32 && r != '\n' && r != '\r' {
			i += w
			continue
		}

		// 情况3: 替换字符（表示输入中有不完整的 UTF-8 序列）
		if r == unicode.ReplacementChar {
			// 检查是否是不完整的多字节序列
			// 通过尝试解码下一个位置来判断
			remaining := len(s) - i - w
			if remaining > 0 {
				// 跳过不完整序列的剩余字节
				// 寻找下一个有效字符的起始位置
				i += w
				// 跳过后续的 continuation byte (10xxxxxx) 或不完整序列
				for i < len(s) && (s[i]&0xC0) == 0x80 {
					i++
				}
				continue
			}
			i += w
			continue
		}

		// 情况4: 代理对（surrogate pairs）无效
		if r >= 0xD800 && r <= 0xDFFF {
			i += w
			continue
		}

		// 情况5: 非 Unicode 平面字符（大于 0x10FFFF）
		if r > 0x10FFFF {
			i += w
			continue
		}

		// 有效字符，添加到结果
		result.WriteRune(r)
		i += w
	}
	return result.String()
}

// 辅助函数：解析整数
func parseIntOrDefault(s string, defaultVal int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// 辅助函数：解析浮点数
func parseFloatOrDefault(s string, defaultVal float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// 辅助函数：解析日期时间
func parseDateTime(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// 尝试多种日期格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02",
		"2006/01/02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return nil
}

// getOrderStatsFromTable 从预聚合表读取订单统计
func (s *OrderService) getOrderStatsFromTable(ctx context.Context, shopName, startTime, endTime string) *OrderStats {
	query := g.DB().Model("daily_order_stats").
		Where("stat_date >= ?", startTime[:10]).
		Where("stat_date <= ?", endTime[:10])
	if shopName != "" {
		query = query.Where("shop_name = ?", shopName)
	}

	result, err := query.Fields(`
		COALESCE(SUM(total_order_count),0) as total_order_count,
		COALESCE(SUM(valid_order_count),0) as valid_order_count,
		COALESCE(SUM(invalid_order_count),0) as invalid_order_count,
		COALESCE(SUM(valid_pay_amount),0) as valid_pay_amount,
		COALESCE(SUM(product_cost_amount),0) as product_cost_amount,
		COALESCE(SUM(reserve_amount),0) as reserve_amount,
		COALESCE(SUM(refund_cost_amount),0) as refund_cost_amount,
		COALESCE(SUM(refund_order_count),0) as refund_order_count,
		COALESCE(SUM(shop_service_fee),0) as shop_service_fee,
		COALESCE(SUM(commission),0) as commission,
		COALESCE(SUM(valid_order_cost_match),0) as valid_order_cost_match
	`).One()

	if err != nil || result == nil {
		return nil
	}

	totalOrderCount := result["total_order_count"].Int64()
	validOrderCount := result["valid_order_count"].Int64()
	invalidOrderCount := result["invalid_order_count"].Int64()
	if totalOrderCount == 0 && validOrderCount == 0 {
		return nil
	}

	validPayAmount := result["valid_pay_amount"].Float64()
	productCostAmount := result["product_cost_amount"].Float64()
	// 历史聚合数据也按当前统一口径读取，避免旧的5%留底继续影响利润。
	reserveAmount := CalculateReserveAmount(validPayAmount)
	shopServiceFee := CalculateDouyinShopServiceFee(validPayAmount)
	commission := result["commission"].Float64()

	// 查询千川广告消耗
	adCost := s.getAdCost(ctx, shopName, startTime, endTime)

	return &OrderStats{
		TotalOrderCount:      totalOrderCount,
		ValidOrderCount:      validOrderCount,
		InvalidOrderCount:    invalidOrderCount,
		UnmatchedOrderCount:  totalOrderCount - validOrderCount - invalidOrderCount,
		ValidPayAmount:       validPayAmount,
		ProductCostAmount:    productCostAmount,
		ReserveAmount:        reserveAmount,
		RefundCostAmount:     result["refund_cost_amount"].Float64(),
		RefundOrderCount:     result["refund_order_count"].Int64(),
		ShopServiceFee:       shopServiceFee,
		AdCostAmount:         adCost,
		OnlineCommission:     commission,
		RealtimeProfit:       CalculateOrderProfit(validPayAmount, productCostAmount, reserveAmount, result["refund_cost_amount"].Float64(), shopServiceFee, commission, 0, adCost),
		ValidOrderCostMatch:  result["valid_order_cost_match"].Int64(),
		CommissionOrderCount: validOrderCount,
		SyncDate:             time.Now().Format("2006-01-02"),
	}
}
