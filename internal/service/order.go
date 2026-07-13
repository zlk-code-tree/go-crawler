package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	done   chan struct{}
	orders []g.Map
	err    error
}

var enrichedDouyinOrderCache sync.Map
var enrichedDouyinOrderLoadMu sync.Mutex
var enrichedDouyinOrderLoads = make(map[string]*enrichedDouyinOrderLoad)

const (
	enrichedDouyinOrderCacheTTL     = time.Minute
	enrichedDouyinOrderBuildTimeout = 2 * time.Minute
)

const orderListInternalCommissionRate = 0.05

func NewOrderService() *OrderService {
	return &OrderService{}
}

func (s *OrderService) loadEnrichedDouyinOrders(ctx context.Context, startDate, endDate string) ([]g.Map, error) {
	cacheKey := startDate + "\x00" + endDate
	if orders, ok := getCachedEnrichedDouyinOrders(cacheKey); ok {
		return orders, nil
	}

	enrichedDouyinOrderLoadMu.Lock()
	if orders, ok := getCachedEnrichedDouyinOrders(cacheKey); ok {
		enrichedDouyinOrderLoadMu.Unlock()
		return orders, nil
	}
	load, loading := enrichedDouyinOrderLoads[cacheKey]
	if !loading {
		load = &enrichedDouyinOrderLoad{done: make(chan struct{})}
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

func (s *OrderService) buildEnrichedDouyinOrderCache(parent context.Context, cacheKey, startDate, endDate string, load *enrichedDouyinOrderLoad) {
	buildCtx, cancel := context.WithTimeout(parent, enrichedDouyinOrderBuildTimeout)
	defer cancel()

	load.orders, load.err = s.queryEnrichedDouyinOrders(buildCtx, startDate, endDate)
	if load.err == nil {
		enrichedDouyinOrderCache.Store(cacheKey, enrichedDouyinOrderCacheEntry{
			expiresAt: time.Now().Add(enrichedDouyinOrderCacheTTL),
			orders:    load.orders,
		})
	}

	enrichedDouyinOrderLoadMu.Lock()
	delete(enrichedDouyinOrderLoads, cacheKey)
	close(load.done)
	enrichedDouyinOrderLoadMu.Unlock()
}

func (s *OrderService) queryEnrichedDouyinOrders(ctx context.Context, startDate, endDate string) ([]g.Map, error) {
	rows, err := g.DB().GetAll(ctx, `
		SELECT o.*, COALESCE(g.base_price, 0) AS base_price
		FROM order_info o
		LEFT JOIN (SELECT goods_code, MAX(base_price) AS base_price FROM goods GROUP BY goods_code) g ON o.seller_code = g.goods_code
		WHERE o.pay_finish_time >= ?
		  AND o.pay_finish_time < ?
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
	return fmt.Sprintf(`COALESCE(
		(SELECT saa.target_value
		 FROM shop_account_attribution saa
		 WHERE saa.shop_name COLLATE utf8mb4_unicode_ci = %s.shop_name COLLATE utf8mb4_unicode_ci
		   AND saa.source_account_uid = %s
		   AND saa.status = 1
		 LIMIT 1),
		NULLIF(au.operator, ''),
		NULLIF(an.operator, ''),
		''
	)`, alias, orderListTalentUIDExpr(alias))
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

func orderListStatTimeExpr(alias string) string {
	return alias + ".pay_finish_time"
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
	for _, field := range []string{"pay_finish_time", "order_submit_time"} {
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
		payTime := formatTime(order["payFinishTime"])
		if startTime != "" && payTime < startTime {
			continue
		}
		if endTime != "" && payTime > endTime {
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

func (s *OrderService) enrichDouyinOrderRows(ctx context.Context, result gdb.Result) ([]g.Map, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	type accountInfo struct {
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
		const allianceBatchSize = 500
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

		sql := "SELECT a.id, a.uid, a.account_name, a.platform_nickname, a.operator, c.name AS company_name FROM account a LEFT JOIN sys_company c ON a.company_id = c.id WHERE a.platform = 'douyin' AND a.status = 1"
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

	type qianchuanMatchRequest struct {
		statDate  string
		shopName  string
		productID string
	}
	type qianchuanMatchInfo struct {
		accountInfo
		planID          int64
		planName        string
		planCount       int
		awemeShowID     string
		statCost        float64
		payOrderCount   int64
		allocationRatio float64
		allocationBasis string
	}
	mergeQianchuanMatchesByOwner := func(matches []qianchuanMatchInfo) []qianchuanMatchInfo {
		merged := make([]qianchuanMatchInfo, 0, len(matches))
		indexByOwner := make(map[string]int, len(matches))
		for _, match := range matches {
			ownerKey := orderListFirstNonEmpty(match.uid, match.awemeShowID)
			if ownerKey == "" {
				continue
			}
			if index, ok := indexByOwner[ownerKey]; ok {
				merged[index].statCost += match.statCost
				merged[index].payOrderCount += match.payOrderCount
				merged[index].planCount++
				merged[index].planID = 0
				merged[index].planName = fmt.Sprintf("多个计划（%d个）", merged[index].planCount)
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
			if totalPayOrders > 0 {
				matches[i].allocationRatio = orderListWeightedRatio(float64(totalPayOrders), float64(matches[i].payOrderCount))
				matches[i].allocationBasis = "pay_order_count"
			} else {
				matches[i].allocationRatio = orderListWeightedRatio(totalCost, matches[i].statCost)
				matches[i].allocationBasis = "stat_cost"
			}
		}
		return matches
	}
	selectQianchuanMatch := func(orderKey string, matches []qianchuanMatchInfo) (qianchuanMatchInfo, bool) {
		if len(matches) == 0 {
			return qianchuanMatchInfo{}, false
		}
		ratios := make([]float64, len(matches))
		for i, match := range matches {
			ratios[i] = match.allocationRatio
		}
		if index := orderListWeightedIndex(orderKey, ratios); index >= 0 {
			return matches[index], true
		}
		return qianchuanMatchInfo{}, false
	}

	qianchuanRequests := make([]qianchuanMatchRequest, 0, len(result))
	qianchuanRequestSet := make(map[string]struct{}, len(result))
	for _, row := range result {
		alliance := allianceByOrder[row["sub_order_no"].String()]
		if orderListNormalizedTalentID(row["talent_id"].String()) != "" || strings.TrimSpace(alliance["short_id"].String()) != "" {
			continue
		}

		statDate := orderListStatDateFromRow(row)
		shopName := strings.TrimSpace(row["shop_name"].String())
		productID := strings.TrimSpace(row["product_id"].String())
		if statDate == "" || shopName == "" || productID == "" {
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

	qianchuanByKey := make(map[string][]qianchuanMatchInfo, len(qianchuanRequests))
	if len(qianchuanRequests) > 0 {
		const qianchuanBatchSize = 200
		for start := 0; start < len(qianchuanRequests); start += qianchuanBatchSize {
			end := start + qianchuanBatchSize
			if end > len(qianchuanRequests) {
				end = len(qianchuanRequests)
			}

			batch := qianchuanRequests[start:end]
			conds := make([]string, 0, len(batch))
			params := make([]interface{}, 0, len(batch)*3)
			for _, req := range batch {
				conds = append(conds, "(LEFT(qar.stat_datetime, 10) = ? AND COALESCE(mapped_shop.shop_name, legacy_shop.account_name) COLLATE utf8mb4_unicode_ci = CAST(? AS CHAR) COLLATE utf8mb4_unicode_ci AND CAST(qar.product_id AS CHAR) COLLATE utf8mb4_unicode_ci = CAST(? AS CHAR) COLLATE utf8mb4_unicode_ci)")
				params = append(params, req.statDate, req.shopName, req.productID)
			}

			sql := fmt.Sprintf(`
					SELECT
						q.stat_date,
						q.shop_name,
						q.product_id,
						q.ad_id,
						q.ad_name,
						q.aweme_show_id,
						q.pay_order_count,
						q.stat_cost,
						a.uid,
						a.account_name,
						a.platform_nickname,
					a.operator,
					c.name AS company_name
				FROM (
					SELECT
							LEFT(qar.stat_datetime, 10) AS stat_date,
							COALESCE(mapped_shop.shop_name, legacy_shop.account_name) AS shop_name,
							CAST(qar.product_id AS CHAR) AS product_id,
							qar.ad_id AS ad_id,
							MAX(qar.ad_name) AS ad_name,
							qar.aweme_show_id AS aweme_show_id,
							COALESCE(SUM(qar.pay_order_count), 0) AS pay_order_count,
							COALESCE(SUM(qar.stat_cost), 0) AS stat_cost
					FROM qianchuan_ad_report qar
					LEFT JOIN account legacy_shop ON legacy_shop.id = qar.account_id
					LEFT JOIN (
						SELECT sqm.qianchuan_advertiser_id, MAX(s.store_name) AS shop_name
						FROM store_qianchuan_mapping sqm
						INNER JOIN store s ON s.id = sqm.store_id
						GROUP BY sqm.qianchuan_advertiser_id
					) mapped_shop ON mapped_shop.qianchuan_advertiser_id = qar.advertiser_id
					WHERE qar.product_id IS NOT NULL
					  AND qar.product_id != ''
					  AND qar.aweme_show_id IS NOT NULL
						  AND qar.aweme_show_id != ''
						  AND qar.stat_cost > 0
						  AND COALESCE(mapped_shop.shop_name, legacy_shop.account_name) IS NOT NULL
						  AND (%s)
					GROUP BY LEFT(qar.stat_datetime, 10), COALESCE(mapped_shop.shop_name, legacy_shop.account_name), CAST(qar.product_id AS CHAR), qar.ad_id, qar.aweme_show_id
				) q
				LEFT JOIN (
					SELECT
						platform_nickname,
						MAX(id) AS id,
						MAX(uid) AS uid,
						MAX(account_name) AS account_name,
						MAX(operator) AS operator,
						MAX(company_id) AS company_id
					FROM account
					WHERE platform = 'douyin'
					  AND status = 1
					  AND platform_nickname IS NOT NULL
					  AND platform_nickname != ''
					GROUP BY platform_nickname
				) a ON a.platform_nickname COLLATE utf8mb4_unicode_ci = q.aweme_show_id COLLATE utf8mb4_unicode_ci
				LEFT JOIN sys_company c ON a.company_id = c.id
					ORDER BY q.stat_date, q.shop_name, q.product_id, q.pay_order_count DESC, q.stat_cost DESC, a.id DESC, q.ad_id ASC`,
				strings.Join(conds, " OR "),
			)
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			rows, err := g.DB().GetAll(ctx, sql, params...)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				key := orderListQianchuanMatchKey(row["stat_date"].String(), row["shop_name"].String(), row["product_id"].String())
				if key == "\x00\x00" {
					continue
				}
				qianchuanByKey[key] = append(qianchuanByKey[key], qianchuanMatchInfo{
					accountInfo: accountInfo{
						uid:              row["uid"].String(),
						accountName:      row["account_name"].String(),
						platformNickname: row["platform_nickname"].String(),
						operator:         row["operator"].String(),
						companyName:      row["company_name"].String(),
					},
					planID:        row["ad_id"].Int64(),
					planName:      row["ad_name"].String(),
					awemeShowID:   row["aweme_show_id"].String(),
					statCost:      row["stat_cost"].Float64(),
					payOrderCount: row["pay_order_count"].Int64(),
				})
			}
		}

		for key, matches := range qianchuanByKey {
			qianchuanByKey[key] = applyQianchuanAllocationRatio(mergeQianchuanMatchesByOwner(matches))
		}
	}

	type resolvedMeta struct {
		talentUID           string
		talentName          string
		commissionRate      string
		douyinAccount       string
		operator            string
		companyName         string
		externalTeamName    string
		commission          float64
		talentOwnership     string
		talentOwnershipTag  string
		orderOwnership      string
		orderOwnershipTag   string
		matchedBy           string
		isShopSelfSale      bool
		allocationRatio     float64
		qianchuanPlanID     int64
		qianchuanPlanName   string
		qianchuanPlanCount  int
		qianchuanAwemeID    string
		qianchuanStatCost   float64
		qianchuanPayOrders  int64
		allocationBasis     string
		externalOnlineRate  float64
		externalOfflineRate float64
	}

	resolved := make([][]resolvedMeta, len(result))
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
			if internalField == "shop_account" || internalField == "qianchuan_plan" {
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
				talentUID:           talentUID,
				talentName:          talentName,
				commissionRate:      orderListActualCommissionRate(row, commission),
				douyinAccount:       douyinAccount,
				operator:            operator,
				companyName:         internalInfo.companyName,
				externalTeamName:    externalInfo.teamName,
				commission:          commission,
				talentOwnership:     talentOwnership,
				talentOwnershipTag:  talentOwnershipCode,
				orderOwnership:      orderOwnership,
				orderOwnershipTag:   orderOwnershipCode,
				matchedBy:           matchedBy,
				isShopSelfSale:      isShopSelfSale,
				allocationRatio:     allocationRatio,
				qianchuanPlanID:     qianchuanMatch.planID,
				qianchuanPlanName:   qianchuanMatch.planName,
				qianchuanPlanCount:  qianchuanMatch.planCount,
				qianchuanAwemeID:    qianchuanMatch.awemeShowID,
				qianchuanStatCost:   qianchuanMatch.statCost,
				qianchuanPayOrders:  qianchuanMatch.payOrderCount,
				allocationBasis:     qianchuanMatch.allocationBasis,
				externalOnlineRate:  externalInfo.onlineCommissionRate,
				externalOfflineRate: externalInfo.offlineCommissionRate,
			})
		}

		if !hasInternal && !hasExternal && originTalentID == "" && shortID == "" {
			key := orderListQianchuanMatchKey(orderListStatDateFromRow(row), row["shop_name"].String(), row["product_id"].String())
			if matches, ok := qianchuanByKey[key]; ok && len(matches) > 0 {
				orderKey := orderListFirstNonEmpty(row["sub_order_no"].String(), row["main_order_no"].String(), row["id"].String())
				if match, matched := selectQianchuanMatch(orderKey, matches); matched {
					appendResolved(match.accountInfo, "qianchuan_plan", true, match)
				}
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

	attributionMap := make(map[string]string)
	if len(shopNames) > 0 && len(resolvedUIDs) > 0 {
		params := make([]interface{}, 0, len(shopNames)+len(resolvedUIDs))
		for _, shopName := range shopNames {
			params = append(params, shopName)
		}
		for _, talentUID := range resolvedUIDs {
			params = append(params, talentUID)
		}
		sql := fmt.Sprintf(
			"SELECT shop_name, source_account_uid, target_value FROM shop_account_attribution WHERE status = 1 AND shop_name IN (%s) AND source_account_uid IN (%s)",
			orderListPlaceholders(len(shopNames)),
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
			key := orderListCompositeKey(row["shop_name"].String(), row["source_account_uid"].String())
			if _, exists := attributionMap[key]; !exists {
				attributionMap[key] = row["target_value"].String()
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
			teamOwner := orderListFirstNonEmpty(
				attributionMap[orderListCompositeKey(row["shop_name"].String(), item.talentUID)],
				item.operator,
				item.externalTeamName,
			)
			allocationRowID := row["id"].String()
			if len(resolved[i]) > 1 || item.qianchuanPlanID != 0 {
				allocationRowID = fmt.Sprintf("%s_%d_%d", row["id"].String(), item.qianchuanPlanID, allocationIndex+1)
			}

			orders = append(orders, g.Map{
				"id":                            row["id"],
				"allocationRowId":               allocationRowID,
				"shopName":                      row["shop_name"],
				"mainOrderNo":                   row["main_order_no"],
				"subOrderNo":                    row["sub_order_no"],
				"productName":                   row["product_name"],
				"productSpec":                   row["product_spec"],
				"productNum":                    row["product_num"],
				"originalProductNum":            row["product_num"],
				"productId":                     row["product_id"],
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
				"productCost":                   productCost,
				"originalProductCost":           roundFloatOrder(row["base_price"].Float64()*float64(row["product_num"].Int()), 2),
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
				"qianchuanStatCost":             roundFloatOrder(item.qianchuanStatCost, 2),
				"qianchuanPayOrderCount":        item.qianchuanPayOrders,
				"qianchuanAllocationBasis":      item.allocationBasis,
				"qianchuanAllocationMatched":    item.qianchuanAwemeID != "",
			})
		}
	}

	return orders, nil
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
			if IsRefundCostOrderForStats(afterSaleStatus, gconv.String(order["sendTime"])) {
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

// statsDouyinOrdersByOwner 在汇总投手抖音订单前复用订单列表的联盟/千川归属规则。
func (s *OrderService) statsDouyinOrdersByOwner(ctx context.Context, accountIDs []int, traderName, startDate, endDate string) (*model.FinanceStatsSummary, error) {
	if len(accountIDs) == 0 || strings.TrimSpace(traderName) == "" {
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
		talentUID := orderListNormalizedTalentID(gconv.String(order["talentId"]))
		owner := strings.TrimSpace(gconv.String(order["teamOwner"]))
		_, selected := selectedUIDs[talentUID]
		if owner != traderName && !(owner == "" && selected) {
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
			if IsRefundCostOrderForStats(afterSaleStatus, gconv.String(order["sendTime"])) {
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

	// 时间范围与千川归属统一使用支付完成时间。
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
