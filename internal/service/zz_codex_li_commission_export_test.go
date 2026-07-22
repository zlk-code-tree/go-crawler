package service

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

func TestCodexExportLiCommissionCurrent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	trader, err := g.DB().Model("trader").Ctx(ctx).
		WhereIn("name", []string{"李百川", "李百川组"}).
		OrderDesc("name = '李百川组'").
		OrderAsc("id").
		One()
	if err != nil {
		t.Fatal(err)
	}
	if trader.IsEmpty() {
		t.Fatal("trader 李百川/李百川组 not found")
	}

	finance := NewFinanceCommissionService()
	accountIDs, err := finance.getAccountIdsByNode(ctx, model.EncodeTraderID(trader["id"].Int64()), "trader", "douyin")
	if err != nil {
		t.Fatal(err)
	}

	rows, err := g.DB().Model("account").Ctx(ctx).
		LeftJoin("trader_account_relation tar", "tar.account_id = account.id").
		WhereIn("id", accountIDs).
		Where("platform", "douyin").
		Where("status", 1).
		Fields("account.id, account.uid, account.account_name, MIN(tar.trader_id) AS trader_id").
		Group("account.id, account.uid, account.account_name").
		All()
	if err != nil {
		t.Fatal(err)
	}
	accountByUID := make(map[string]int, len(rows))
	sourceTraderByAccountID := make(map[int]int64, len(rows))
	firstAccountByTraderID := make(map[int64]int, len(rows))
	accountNameByID := make(map[int]string, len(rows))
	for _, row := range rows {
		accountID := row["id"].Int()
		traderID := row["trader_id"].Int64()
		sourceTraderByAccountID[accountID] = traderID
		accountNameByID[accountID] = row["account_name"].String()
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

	ClearEnrichedDouyinOrderCache()
	service := NewOrderService()
	enriched, err := service.loadEnrichedDouyinOrders(ctx, "2026-07-19", "2026-07-19")
	if err != nil {
		t.Fatal(err)
	}

	qianchuanBlankOrderInferenceEnabled := IsQianchuanPlanBlankOrderInferenceEnabled(ctx)
	type orderRow struct {
		MainOrderNo             string  `json:"mainOrderNo"`
		SubOrderNo              string  `json:"subOrderNo"`
		AccountID               int     `json:"accountId"`
		AccountName             string  `json:"accountName"`
		ShopName                string  `json:"shopName"`
		ProductName             string  `json:"productName"`
		ProductNum              int     `json:"productNum"`
		ProductPrice            float64 `json:"productPrice"`
		ProductCost             float64 `json:"productCost"`
		PayAmount               float64 `json:"payAmount"`
		ReserveAmount           float64 `json:"reserveAmount"`
		ShopServiceFee          float64 `json:"shopServiceFee"`
		SystemCommission        float64 `json:"systemCommission"`
		OrderStatus             string  `json:"orderStatus"`
		AfterSaleStatus         string  `json:"afterSaleStatus"`
		ProfitAttributionSource string  `json:"profitAttributionSource"`
		TalentName              string  `json:"talentName"`
		DouyinAccount           string  `json:"douyinAccount"`
		OrderSubmitTime         string  `json:"orderSubmitTime"`
	}

	var orders []orderRow
	seen := make(map[string]struct{})
	var payTotal, costTotal, orderCommissionTotal float64
	for _, order := range enriched {
		if !qianchuanBlankOrderInferenceEnabled && gconv.String(order["profitAttributionSource"]) == "qianchuan_plan" {
			continue
		}
		uid := orderListNormalizedTalentID(gconv.String(order["originalTalentId"]))
		if uid == "" && gconv.String(order["profitAttributionSource"]) == "qianchuan_plan" {
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
		orderKey := orderListFirstNonEmpty(gconv.String(order["mainOrderNo"]), gconv.String(order["subOrderNo"]), gconv.String(order["allocationRowId"]))
		if orderKey == "" {
			continue
		}
		if _, ok := seen[orderKey]; ok {
			continue
		}
		seen[orderKey] = struct{}{}
		orderStatus := gconv.String(order["rawOrderStatus"])
		afterSaleStatus := gconv.String(order["afterSaleStatus"])
		if !IsValidOrderForStats(orderStatus, afterSaleStatus) {
			continue
		}
		productNum := gconv.Int(order["productNum"])
		pay := gconv.Float64(order["productPrice"]) * float64(productNum)
		cost := gconv.Float64(order["productCost"])
		reserve := roundFloatOrder(pay*FinanceReserveRate, 2)
		shopFee := roundFloatOrder(pay*DouyinShopServiceFeeRate, 2)
		payTotal += pay
		costTotal += cost
		systemCommission := roundFloatOrder(pay-cost-reserve-shopFee, 2)
		orderCommissionTotal += systemCommission
		orders = append(orders, orderRow{
			MainOrderNo:             gconv.String(order["mainOrderNo"]),
			SubOrderNo:              gconv.String(order["subOrderNo"]),
			AccountID:               accountID,
			AccountName:             accountNameByID[accountID],
			ShopName:                gconv.String(order["shopName"]),
			ProductName:             gconv.String(order["productName"]),
			ProductNum:              productNum,
			ProductPrice:            gconv.Float64(order["productPrice"]),
			ProductCost:             cost,
			PayAmount:               roundFloatOrder(pay, 2),
			ReserveAmount:           reserve,
			ShopServiceFee:          shopFee,
			SystemCommission:        systemCommission,
			OrderStatus:             orderStatus,
			AfterSaleStatus:         afterSaleStatus,
			ProfitAttributionSource: gconv.String(order["profitAttributionSource"]),
			TalentName:              gconv.String(order["talentName"]),
			DouyinAccount:           gconv.String(order["douyinAccount"]),
			OrderSubmitTime:         gconv.String(order["orderSubmitTime"]),
		})
	}

	reserveTotal := CalculateReserveAmount(payTotal)
	shopFeeTotal := CalculateDouyinShopServiceFee(payTotal)
	out := map[string]interface{}{
		"traderId":           trader["id"].Int64(),
		"accountIds":         accountIDs,
		"count":              len(orders),
		"payAmount":          roundFloatOrder(payTotal, 2),
		"costAmount":         roundFloatOrder(costTotal, 2),
		"reserveAmount":      reserveTotal,
		"shopServiceFee":     shopFeeTotal,
		"systemCommission":   roundFloatOrder(payTotal-costTotal-reserveTotal-shopFeeTotal, 2),
		"sumOrderCommission": roundFloatOrder(orderCommissionTotal, 2),
		"orders":             orders,
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/codex_li_commission_current.json", b, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("exported %d orders commission %.2f", len(orders), out["systemCommission"])
}
