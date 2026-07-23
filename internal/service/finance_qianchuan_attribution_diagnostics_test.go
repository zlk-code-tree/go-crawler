package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-crawler/internal/model"

	_ "go-crawler/internal/packed"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
)

func TestQianchuanAttributionDatabaseDiagnostics(t *testing.T) {
	if os.Getenv("FINANCE_QIANCHUAN_ATTRIBUTION_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_QIANCHUAN_ATTRIBUTION_DB_INTEGRATION=1 to inspect qianchuan attribution")
	}
	statDate := os.Getenv("FINANCE_QIANCHUAN_ATTRIBUTION_TEST_DATE")
	traderID, err := strconv.ParseInt(os.Getenv("FINANCE_QIANCHUAN_ATTRIBUTION_TEST_TRADER_ID"), 10, 64)
	if statDate == "" || err != nil || traderID <= 0 {
		t.Fatal("FINANCE_QIANCHUAN_ATTRIBUTION_TEST_DATE and FINANCE_QIANCHUAN_ATTRIBUTION_TEST_TRADER_ID are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ownerOnly, err := g.DB().GetOne(ctx, effectiveQianchuanCostCTE+`
		SELECT COALESCE(SUM(effective.stat_cost), 0) AS ad_cost
		FROM effective_qianchuan_cost effective
		INNER JOIN qianchuan_account_owner_mapping owner
			ON owner.advertiser_id = effective.advertiser_id
			AND owner.status = 1
			AND owner.trader_id = ?`, statDate, statDate, statDate, statDate, traderID)
	if err != nil {
		t.Fatalf("query owner-only cost failed: %v", err)
	}

	attributedArgs := append(qianchuanAttributionDateArgs(statDate, statDate), traderID)
	attributed, err := g.DB().GetOne(ctx, attributedQianchuanCostCTE+`
		SELECT
			COALESCE(SUM(stat_cost), 0) AS ad_cost,
			COALESCE(SUM(CASE WHEN cost_type = 'normal' THEN stat_cost ELSE 0 END), 0) AS normal_cost,
			COALESCE(SUM(CASE WHEN cost_type = 'gap' THEN stat_cost ELSE 0 END), 0) AS gap_cost
		FROM attributed_qianchuan_cost
		WHERE resolved_trader_id = ?`, attributedArgs...)
	if err != nil {
		t.Fatalf("query attributed cost failed: %v", err)
	}

	rows, err := g.DB().GetAll(ctx, attributedQianchuanCostCTE+`
		SELECT
			cost.advertiser_id,
			cost.account_id,
			MAX(account.account_name) AS account_name,
			MAX(account.platform_nickname) AS platform_nickname,
			COALESCE(SUM(cost.stat_cost), 0) AS ad_cost,
			MAX(CASE WHEN EXISTS (
				SELECT 1
				FROM qianchuan_advertiser_attribution_rule rule
				WHERE rule.advertiser_id = cost.advertiser_id
				  AND rule.status = 1
				  AND cost.stat_date >= rule.effective_start_date
				  AND (rule.effective_end_date IS NULL OR cost.stat_date <= rule.effective_end_date)
				  AND rule.target_trader_id = cost.resolved_trader_id
				  AND FIND_IN_SET(
					LOWER(TRIM(COALESCE(NULLIF(account.account_name, ''), NULLIF(cost.aweme_name, ''), ''))) COLLATE utf8mb4_unicode_ci,
					LOWER(REPLACE(rule.source_account_names, '，', ',')) COLLATE utf8mb4_unicode_ci
				  ) > 0
			) THEN 1 ELSE 0 END) AS special_rule_hit,
			MAX(CASE WHEN EXISTS (
				SELECT 1
				FROM trader_account_relation account_owner
				WHERE account_owner.account_id = cost.account_id
				  AND account_owner.trader_id = cost.resolved_trader_id
			) THEN 1 ELSE 0 END) AS account_owner_hit
		FROM attributed_qianchuan_cost cost
		INNER JOIN account account
			ON account.id = cost.account_id
		WHERE cost.resolved_trader_id = ?
		GROUP BY cost.advertiser_id, cost.account_id
		ORDER BY special_rule_hit DESC, ad_cost DESC`, attributedArgs...)
	if err != nil {
		t.Fatalf("query attribution detail failed: %v", err)
	}

	t.Logf("date=%s trader=%d owner_only_ad=%.2f attributed_ad=%.2f normal_ad=%.2f gap_ad=%.2f rows=%d",
		statDate, traderID, ownerOnly["ad_cost"].Float64(), attributed["ad_cost"].Float64(),
		attributed["normal_cost"].Float64(), attributed["gap_cost"].Float64(), len(rows))
	for _, row := range rows {
		t.Logf("advertiser=%d account=%d accountName=%s nickname=%s ad=%.2f specialRule=%d accountOwner=%d",
			row["advertiser_id"].Int64(), row["account_id"].Int(), row["account_name"].String(),
			row["platform_nickname"].String(), row["ad_cost"].Float64(),
			row["special_rule_hit"].Int(), row["account_owner_hit"].Int())
	}
}

func TestQianchuanPlanTalentDatabaseDiagnostics(t *testing.T) {
	if os.Getenv("QIANCHUAN_PLAN_TALENT_DB_INTEGRATION") != "1" {
		t.Skip("set QIANCHUAN_PLAN_TALENT_DB_INTEGRATION=1 to inspect qianchuan plan talent data")
	}
	statDate := os.Getenv("QIANCHUAN_PLAN_TALENT_TEST_DATE")
	talentKey := os.Getenv("QIANCHUAN_PLAN_TALENT_TEST_KEY")
	if statDate == "" || talentKey == "" {
		t.Fatal("QIANCHUAN_PLAN_TALENT_TEST_DATE and QIANCHUAN_PLAN_TALENT_TEST_KEY are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
		Level:     QianchuanHierarchyLevelTalent,
		StartDate: statDate,
		EndDate:   statDate,
	})
	if err != nil {
		t.Fatalf("query qianchuan plan talent hierarchy failed: %v", err)
	}

	var matched *QianchuanPlanHierarchyNode
	for i := range result.List {
		node := &result.List[i]
		if node.TalentKey == talentKey || strings.Contains(node.Name, talentKey) {
			matched = node
			break
		}
	}
	if matched == nil {
		for _, node := range result.List {
			t.Logf("talent key=%s name=%s cost=%.2f profit=%.2f", node.TalentKey, node.Name, node.StatCost, node.Profit)
		}
		t.Fatalf("talent %s not found", talentKey)
	}

	t.Logf("date=%s talentKey=%s name=%s advertisers=%d stores=%d plans=%d pay=%.2f cost=%.2f commission=%.2f deduction=%.2f profit=%.2f validOrders=%.2f orderCost=%.2f",
		statDate, matched.TalentKey, matched.Name, matched.AdvertiserCount, matched.StoreCount, matched.PlanCount,
		matched.PayAmount, matched.StatCost, matched.Commission, matched.PayAmount*0.005, matched.Profit,
		matched.ValidOrderCount, matched.CostAmount)

	advertisers, err := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
		Level:     QianchuanHierarchyLevelAdvertiser,
		StartDate: statDate,
		EndDate:   statDate,
		TalentKey: talentKey,
		AllStores: true,
	})
	if err != nil {
		t.Fatalf("query attributed advertisers failed: %v", err)
	}
	if len(advertisers.List) == 0 {
		t.Fatalf("expected attributed advertisers for %s", talentKey)
	}
	advertiser := advertisers.List[0]
	plans, err := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
		Level:        QianchuanHierarchyLevelPlan,
		StartDate:    statDate,
		EndDate:      statDate,
		AdvertiserID: advertiser.AdvertiserID,
		StoreID:      advertiser.StoreID,
		TalentKey:    talentKey,
	})
	if err != nil {
		t.Fatalf("query attributed plans failed: %v", err)
	}
	planCost := 0.0
	planOrders := 0.0
	for _, plan := range plans.List {
		planCost += plan.StatCost
		planOrders += plan.ValidOrderCount
	}
	if math.Abs(planCost-advertiser.StatCost) > 0.02 {
		t.Fatalf("plan cost does not reconcile advertiser: advertiser=%.2f plans=%.2f", advertiser.StatCost, planCost)
	}
	t.Logf("drilldown advertiser=%d account=%d cost=%.2f plans=%d auditedOrders=%.0f",
		advertiser.AdvertiserID, advertiser.AccountID, advertiser.StatCost, len(plans.List), planOrders)
}

func TestFinanceVsQianchuanPlanOrderPoolDiagnostics(t *testing.T) {
	if os.Getenv("FINANCE_ORDER_POOL_COMPARE_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_ORDER_POOL_COMPARE_DB_INTEGRATION=1 to compare finance and qianchuan order pools")
	}
	statDate := os.Getenv("FINANCE_ORDER_POOL_COMPARE_TEST_DATE")
	traderID, err := strconv.ParseInt(os.Getenv("FINANCE_ORDER_POOL_COMPARE_TEST_TRADER_ID"), 10, 64)
	if statDate == "" || err != nil || traderID <= 0 {
		t.Fatal("FINANCE_ORDER_POOL_COMPARE_TEST_DATE and FINANCE_ORDER_POOL_COMPARE_TEST_TRADER_ID are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	finance := NewFinanceCommissionService()
	accountIDs, err := finance.getAccountIdsByDimension(ctx, &RealtimeProfitReq{TraderId: traderID})
	if err != nil {
		t.Fatalf("query trader account ids failed: %v", err)
	}
	children, err := finance.GetChildren(ctx, &model.FinanceChildrenReq{
		NodeId:    model.EncodeTraderID(traderID),
		NodeType:  "trader",
		TimeType:  "custom",
		StartDate: statDate,
		EndDate:   statDate,
		Platform:  "douyin",
	})
	if err != nil {
		t.Fatalf("query finance account children failed: %v", err)
	}
	financePay := 0.0
	financeOrders := 0
	for _, child := range children.Children {
		if child.PayAmount == 0 {
			continue
		}
		financePay += child.PayAmount
		financeOrders += child.OrderCount
		t.Logf("finance accountNode=%d name=%s orders=%d pay=%.2f commission=%.2f cost=%.2f",
			child.Id, child.Name, child.OrderCount, child.PayAmount, child.Commission, child.CostAmount)
	}
	liveLeafStats, err := NewOrderService().statsDouyinOrdersByAccounts(ctx, accountIDs, statDate, statDate)
	if err != nil {
		t.Fatalf("query live finance leaf order pool failed: %v", err)
	}
	liveFinancePay := 0.0
	liveFinanceOrders := 0
	for accountID, stats := range liveLeafStats {
		if stats == nil || stats.PayAmount == 0 {
			continue
		}
		liveFinancePay += stats.PayAmount
		liveFinanceOrders += stats.OrderCount
		t.Logf("live finance account=%d orders=%d pay=%.2f commission=%.2f cost=%.2f",
			accountID, stats.OrderCount, stats.PayAmount, stats.Commission-stats.ShopServiceFee, stats.CostAmount)
	}

	planResult, err := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
		Level:     QianchuanHierarchyLevelAdvertiser,
		StartDate: statDate,
		EndDate:   statDate,
		TalentKey: "trader:" + strconv.FormatInt(traderID, 10),
		AllStores: true,
	})
	if err != nil {
		t.Fatalf("query qianchuan plan advertiser pool failed: %v", err)
	}
	storeRows, err := g.DB().Model("store").Ctx(ctx).Fields("id, store_name").All()
	if err != nil {
		t.Fatalf("query stores failed: %v", err)
	}
	storeNames := make(map[int64]string, len(storeRows))
	for _, row := range storeRows {
		storeNames[row["id"].Int64()] = row["store_name"].String()
	}
	planPay := 0.0
	planOrders := 0.0
	for _, node := range planResult.List {
		planPay += node.PayAmount
		planOrders += node.ValidOrderCount
		t.Logf("plan advertiser=%d store=%d storeName=%s account=%d orders=%.2f pay=%.2f commission=%.2f statCost=%.2f",
			node.AdvertiserID, node.StoreID, storeNames[node.StoreID], node.AccountID,
			node.ValidOrderCount, node.PayAmount, node.Commission, node.StatCost)
	}
	t.Logf("compare date=%s trader=%d financeOrders=%d financePay=%.2f planOrders=%.2f planPay=%.2f payDiff=%.2f",
		statDate, traderID, financeOrders, financePay, planOrders, planPay, planPay-financePay)
	t.Logf("live compare date=%s trader=%d financeOrders=%d financePay=%.2f planPay=%.2f payDiff=%.2f",
		statDate, traderID, liveFinanceOrders, liveFinancePay, planPay, planPay-liveFinancePay)
	t.Logf("finance bound account count=%d", len(accountIDs))
}

func TestBorrowedAccountAttributionDiagnostics(t *testing.T) {
	if os.Getenv("BORROWED_ACCOUNT_ATTRIBUTION_DB_INTEGRATION") != "1" {
		t.Skip("set BORROWED_ACCOUNT_ATTRIBUTION_DB_INTEGRATION=1 to inspect borrowed account attribution")
	}
	statDate := os.Getenv("BORROWED_ACCOUNT_ATTRIBUTION_TEST_DATE")
	shopName := os.Getenv("BORROWED_ACCOUNT_ATTRIBUTION_TEST_SHOP")
	if statDate == "" || shopName == "" {
		t.Fatal("BORROWED_ACCOUNT_ATTRIBUTION_TEST_DATE and BORROWED_ACCOUNT_ATTRIBUTION_TEST_SHOP are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	storeRows, err := g.DB().GetAll(ctx, `
		SELECT id, store_id, store_name, blank_order_enabled, blank_order_owner_name
		FROM store
		WHERE platform = 'douyin'
		  AND store_name LIKE ?
		ORDER BY id`, "%"+shopName+"%")
	if err != nil {
		t.Fatalf("query stores failed: %v", err)
	}
	for _, row := range storeRows {
		t.Logf("store id=%d storeId=%s name=%s blankEnabled=%d blankOwner=%s",
			row["id"].Int64(), row["store_id"].String(), row["store_name"].String(),
			row["blank_order_enabled"].Int(), row["blank_order_owner_name"].String())
	}

	ruleRows, err := g.DB().GetAll(ctx, `
		SELECT id, shop_id, shop_name, source_account_uid, source_account_name, target_type, target_value, status
		FROM shop_account_attribution
		WHERE shop_name LIKE ?
		   OR source_account_name LIKE ?
		   OR target_value LIKE ?
		ORDER BY id`, "%"+shopName+"%", "%天书%", "%王梅雪%")
	if err != nil {
		t.Fatalf("query borrowed account rules failed: %v", err)
	}
	for _, row := range ruleRows {
		t.Logf("rule id=%d shopId=%s shop=%s sourceUid=%s sourceName=%s target=%s/%s status=%d",
			row["id"].Int(), row["shop_id"].String(), row["shop_name"].String(),
			row["source_account_uid"].String(), row["source_account_name"].String(),
			row["target_type"].String(), row["target_value"].String(), row["status"].Int())
	}

	validCondition := FinanceValidOrderCondition("o")
	rows, err := g.DB().GetAll(ctx, fmt.Sprintf(`
		SELECT
			o.talent_id,
			MAX(o.talent_name) AS talent_name,
			COALESCE(MAX(saa.target_value), '') AS matched_target,
			COUNT(DISTINCT o.main_order_no) AS total_orders,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS valid_orders,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount
		FROM order_info o
		LEFT JOIN shop_account_attribution saa
			ON %s
		WHERE o.shop_name = ?
		  AND o.pay_finish_time >= ?
		  AND o.pay_finish_time < ?
		GROUP BY o.talent_id
		ORDER BY pay_amount DESC, valid_orders DESC`, validCondition, validCondition, shopAccountAttributionRuleSQL("o", "o.talent_id")),
		shopName, statDate, CalcEndDateNext(statDate))
	if err != nil {
		t.Fatalf("query borrowed account order distribution failed: %v", err)
	}
	for _, row := range rows {
		t.Logf("orders talentUid=%s talentName=%s matchedTarget=%s total=%d valid=%d pay=%.2f",
			row["talent_id"].String(), row["talent_name"].String(), row["matched_target"].String(),
			row["total_orders"].Int(), row["valid_orders"].Int(), row["pay_amount"].Float64())
	}

	blankTargetRows, err := g.DB().GetAll(ctx, `
		SELECT
			COALESCE(t.id, 0) AS trader_id,
			saa.target_value,
			MAX(CASE WHEN saa.source_account_uid = '__blank__' THEN 1 ELSE 0 END) AS explicit_blank,
			COUNT(DISTINCT saa.source_account_uid) AS source_account_count
		FROM shop_account_attribution saa
		LEFT JOIN trader t
		  ON t.status = 1
		 AND (
			saa.target_value COLLATE utf8mb4_unicode_ci = t.name COLLATE utf8mb4_unicode_ci
			OR saa.target_value COLLATE utf8mb4_unicode_ci = CONCAT(t.name, '组') COLLATE utf8mb4_unicode_ci
			OR (saa.target_value LIKE '%组' AND TRIM(TRAILING '组' FROM saa.target_value) COLLATE utf8mb4_unicode_ci = t.name COLLATE utf8mb4_unicode_ci)
		 )
		WHERE saa.status = 1
		  AND saa.shop_name = ?
		GROUP BY trader_id, saa.target_value
		ORDER BY source_account_count DESC`, shopName)
	if err != nil {
		t.Fatalf("query borrowed account blank target failed: %v", err)
	}
	for _, row := range blankTargetRows {
		t.Logf("blank shop target trader=%d target=%s explicitBlank=%d sourceAccounts=%d",
			row["trader_id"].Int64(), row["target_value"].String(), row["explicit_blank"].Int(), row["source_account_count"].Int())
	}
	inheritedTarget := ""
	for _, row := range blankTargetRows {
		if row["explicit_blank"].Int() == 1 {
			inheritedTarget = row["target_value"].String()
			break
		}
	}
	if inheritedTarget == "" && len(blankTargetRows) == 1 {
		inheritedTarget = blankTargetRows[0]["target_value"].String()
	}

	blankRows, err := g.DB().GetAll(ctx, fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT o.main_order_no) AS total_orders,
			COUNT(DISTINCT CASE WHEN %s THEN o.main_order_no END) AS valid_orders,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount
		FROM order_info o
		LEFT JOIN alliance_orders ao ON ao.platform = 'douyin' AND ao.order_id = o.sub_order_no
		WHERE o.shop_name = ?
		  AND o.pay_finish_time >= ?
		  AND o.pay_finish_time < ?
		  AND COALESCE(NULLIF(NULLIF(o.talent_id, ''), '0'), NULLIF(ao.short_id, ''), NULLIF(o.talent_name, ''), NULLIF(ao.author_account, '')) IS NULL`, validCondition, validCondition),
		shopName, statDate, CalcEndDateNext(statDate))
	if err != nil {
		t.Fatalf("query borrowed account blank orders failed: %v", err)
	}
	for _, row := range blankRows {
		t.Logf("blank orders inheritedTarget=%s total=%d valid=%d pay=%.2f",
			inheritedTarget, row["total_orders"].Int(), row["valid_orders"].Int(), row["pay_amount"].Float64())
	}
}
