package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

type AllianceService struct{}

func NewAllianceService() *AllianceService {
	return &AllianceService{}
}

// AllianceOrder 精选联盟订单
type AllianceOrder struct {
	Id                 int64   `json:"id"`
	OrderId            string  `json:"orderId"`
	ProductId          string  `json:"productId"`
	ProductName        string  `json:"productName"`
	ProductNum         int     `json:"productNum"`
	ProductPrice       float64 `json:"productPrice"`
	PayAmount          float64 `json:"payAmount"`
	CommissionRate     string  `json:"commissionRate"`
	Commission         float64 `json:"commission"`
	OrderStatus        string  `json:"orderStatus"`
	AuthorId           string  `json:"authorId"`
	AuthorBuyinId      string  `json:"authorBuyinId"`
	AuthorName         string  `json:"authorName"`
	AuthorAccount      string  `json:"authorAccount"`
	ShopName           string  `json:"shopName"`
	PayTime            string  `json:"payTime"`
	CreateTime         string  `json:"createTime"`
	EstimatedComission int     `json:"estimatedComission"`
	RealComission      int     `json:"realComission"`
	TotalPayAmount     int     `json:"totalPayAmount"`
	AllianceBizType    string  `json:"allianceBizType"`
}

// List 精选联盟订单列表
func (s *AllianceService) List(ctx context.Context, page, pageSize int, keyword, shopName, authorName, authorId, orderStatus, startTime, endTime string) ([]AllianceOrder, int, error) {
	query := g.DB().Model("alliance_orders ao").Ctx(ctx).
		LeftJoin("account a", "ao.author_account COLLATE utf8mb4_unicode_ci = a.account_name COLLATE utf8mb4_unicode_ci")
	params := []interface{}{}

	if keyword != "" {
		query = query.Where("(ao.order_id LIKE ? OR ao.product_name LIKE ? OR ao.author_account LIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if shopName != "" {
		query = query.Where("ao.shop_name LIKE ?", "%"+shopName+"%")
	}
	if authorName != "" {
		query = query.Where("ao.author_account LIKE ?", "%"+authorName+"%")
	}
	if authorId != "" {
		query = query.Where("ao.author_id LIKE ? OR a.uid LIKE ?", "%"+authorId+"%", "%"+authorId+"%")
	}
	if orderStatus != "" {
		query = query.Where("ao.order_status = ?", orderStatus)
	}
	if startTime != "" {
		query = query.Where("ao.pay_time >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("ao.pay_time <= ?", endTime)
	}

	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	result, err := query.Fields("ao.*, COALESCE(a.uid, '') as buyer_uid").
		Order("ao.pay_time DESC").Limit(pageSize).Offset(offset).All()
	if err != nil {
		return nil, 0, err
	}

	var orders []AllianceOrder
	for _, row := range result {
		orders = append(orders, rowToAllianceOrder(row))
	}
	_ = params
	return orders, total, nil
}

func rowToAllianceOrder(row gdb.Record) AllianceOrder {
	return AllianceOrder{
		Id:                 row["id"].Int64(),
		OrderId:            row["order_id"].String(),
		ProductId:          row["product_id"].String(),
		ProductName:        row["product_name"].String(),
		ProductNum:         row["product_num"].Int(),
		ProductPrice:       row["product_price"].Float64(),
		PayAmount:          row["pay_amount"].Float64(),
		CommissionRate:     row["commission_rate"].String(),
		Commission:         row["commission"].Float64(),
		OrderStatus:        row["order_status"].String(),
		AuthorId:           row["author_id"].String(),
		AuthorBuyinId:      row["buyer_uid"].String(),
		AuthorName:         row["author_name"].String(),
		AuthorAccount:      row["author_account"].String(),
		ShopName:           row["shop_name"].String(),
		PayTime:            row["pay_time"].String(),
		CreateTime:         row["create_time"].String(),
		EstimatedComission: row["estimated_comission"].Int(),
		RealComission:      row["real_comission"].Int(),
		TotalPayAmount:     row["total_pay_amount"].Int(),
		AllianceBizType:    row["alliance_biz_type"].String(),
	}
}

// Export 导出精选联盟订单
func (s *AllianceService) Export(ctx context.Context, keyword, shopName, authorName, authorId, orderStatus, startTime, endTime string) (string, error) {
	query := g.DB().Model("alliance_orders ao").Ctx(ctx).
		LeftJoin("account a", "ao.author_account COLLATE utf8mb4_unicode_ci = a.account_name COLLATE utf8mb4_unicode_ci")

	if keyword != "" {
		query = query.Where("(ao.order_id LIKE ? OR ao.product_name LIKE ? OR ao.author_account LIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if shopName != "" {
		query = query.Where("ao.shop_name LIKE ?", "%"+shopName+"%")
	}
	if authorName != "" {
		query = query.Where("ao.author_account LIKE ?", "%"+authorName+"%")
	}
	if authorId != "" {
		query = query.Where("ao.author_id LIKE ? OR a.uid LIKE ?", "%"+authorId+"%", "%"+authorId+"%")
	}
	if orderStatus != "" {
		query = query.Where("ao.order_status = ?", orderStatus)
	}
	if startTime != "" {
		query = query.Where("ao.pay_time >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("ao.pay_time <= ?", endTime)
	}

	result, err := query.Fields("ao.*, COALESCE(a.uid, '') as buyer_uid").Order("ao.pay_time DESC").All()
	if err != nil {
		return "", err
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "精选联盟订单"
	headers := []string{"订单号", "商品名称", "数量", "单价", "支付金额", "佣金率", "佣金", "订单状态", "达人ID", "达人名称", "达人账号", "店铺", "支付时间"}
	colWidths := []float64{22, 40, 8, 12, 14, 10, 14, 14, 18, 16, 16, 20, 18}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, w)
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}
	f.SetRowStyle(sheetName, 1, 1, headerStyle)

	for rowIdx, row := range result {
		dataRow := []interface{}{
			row["order_id"].String(),
			row["product_name"].String(),
			row["product_num"].Int(),
			row["product_price"].Float64(),
			row["pay_amount"].Float64(),
			row["commission_rate"].String(),
			row["commission"].Float64(),
			row["order_status"].String(),
			row["author_id"].String(),
			row["author_name"].String(),
			row["author_account"].String(),
			row["shop_name"].String(),
			row["pay_time"].String(),
		}
		for colIdx, val := range dataRow {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, fmt.Sprintf("alliance_export_%s.xlsx", time.Now().Format("20060102150405")))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel失败: %w", err)
	}
	return filename, nil
}

// FillBuyinId 遍历alliance_orders，通过author_account关联account.account_name获取uid
func (s *AllianceService) FillBuyinId(ctx context.Context) (int, error) {
	rows, err := g.DB().Model("alliance_orders").Ctx(ctx).
		Fields("id, author_account").
		All()
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, row := range rows {
		id := row["id"].Int64()
		name := row["author_account"].String()
		if name == "" {
			continue
		}
		acct, err := g.DB().Model("account").Ctx(ctx).
			Where("account_name COLLATE utf8mb4_unicode_ci = ?", name).
			Fields("uid").
			One()
		if err != nil || acct.IsEmpty() || acct["uid"].String() == "" {
			continue
		}
		uid := acct["uid"].String()
		g.DB().Model("alliance_orders").Ctx(ctx).Where("id", id).Data(g.Map{"author_buyin_id": uid}).Update()
		updated++
	}
	return updated, nil
}

// BackfillTalentInfo 回填达人信息到 order_info（优化版：使用 short_id 匹配 platform_nickname）
// 只更新 talent_id 为空或为'0'的订单
func (s *AllianceService) BackfillTalentInfo(ctx context.Context) (int, error) {
	// 使用 short_id 匹配 account.platform_nickname
	result, err := g.DB().Exec(ctx, `
		UPDATE order_info o
		JOIN alliance_orders ao ON o.sub_order_no = ao.order_id AND ao.platform = 'douyin'
		JOIN account a ON ao.short_id COLLATE utf8mb4_unicode_ci = a.platform_nickname COLLATE utf8mb4_unicode_ci AND a.platform = 'douyin'
		SET o.talent_id = a.uid,
		    o.talent_name = a.account_name
		WHERE (o.talent_id IS NULL OR o.talent_id = '' OR o.talent_id = '0')
		  AND ao.short_id IS NOT NULL
		  AND ao.short_id != ''
		  AND a.uid IS NOT NULL
		  AND a.uid != ''
	`)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// BackfillTalentInfoIncremental 增量回填达人信息（只处理指定时间范围的数据）
// 优化：分两步执行，避免大表 JOIN
func (s *AllianceService) BackfillTalentInfoIncremental(ctx context.Context, startTime, endTime string) (int, error) {
	start := time.Now()

	// Step1: 从 alliance_orders + account 构建 short_id → (uid, account_name) 映射
	// 这个映射是相对稳定的，可以快速查询
	mappingRows, err := g.DB().GetAll(ctx, `
		SELECT DISTINCT ao.short_id, a.uid, a.account_name
		FROM alliance_orders ao
		INNER JOIN account a ON ao.short_id COLLATE utf8mb4_unicode_ci = a.platform_nickname COLLATE utf8mb4_unicode_ci AND a.platform = 'douyin'
		WHERE ao.short_id IS NOT NULL AND ao.short_id != ''
		  AND a.uid IS NOT NULL AND a.uid != ''
	`)
	if err != nil {
		return 0, fmt.Errorf("查询映射关系失败: %v", err)
	}

	// 构建映射表
	shortIdMap := make(map[string]struct{ uid, name string })
	for _, row := range mappingRows {
		shortId := row["short_id"].String()
		shortIdMap[shortId] = struct{ uid, name string }{
			uid:  row["uid"].String(),
			name: row["account_name"].String(),
		}
	}

	g.Log().Infof(ctx, "[Backfill] 构建映射表: %d 条, 耗时 %dms", len(shortIdMap), time.Since(start).Milliseconds())

	if len(shortIdMap) == 0 {
		return 0, nil
	}

	// Step2: 查询需要回填的订单（只查 order_info，不 JOIN）
	needBackfill, err := g.DB().GetAll(ctx, `
		SELECT id, sub_order_no
		FROM order_info
		WHERE (talent_id IS NULL OR talent_id = '' OR talent_id = '0')
		  AND pay_finish_time >= ?
		  AND pay_finish_time < ?
	`, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("查询待回填订单失败: %v", err)
	}

	g.Log().Infof(ctx, "[Backfill] 待回填订单: %d 条, 耗时 %dms", len(needBackfill), time.Since(start).Milliseconds())

	// Step3: 查询这些订单在 alliance_orders 中的 short_id
	orderNos := make([]string, 0, len(needBackfill))
	orderIdMap := make(map[string]int64) // sub_order_no -> order_info.id
	for _, row := range needBackfill {
		subNo := row["sub_order_no"].String()
		orderNos = append(orderNos, subNo)
		orderIdMap[subNo] = row["id"].Int64()
	}

	// 分批查询 alliance_orders
	batchSize := 1000
	totalUpdated := 0

	for i := 0; i < len(orderNos); i += batchSize {
		end := i + batchSize
		if end > len(orderNos) {
			end = len(orderNos)
		}
		batch := orderNos[i:end]

		// 构建 IN 条件
		placeholders := ""
		args := make([]interface{}, 0, len(batch))
		for j, no := range batch {
			if j > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, no)
		}

		// 查询 alliance_orders 获取 short_id
		allianceSQL := fmt.Sprintf(`
			SELECT order_id, short_id
			FROM alliance_orders
			WHERE order_id IN (%s) AND platform = 'douyin'
		`, placeholders)
		allianceRows, err := g.DB().GetAll(ctx, allianceSQL, args...)
		if err != nil {
			g.Log().Warningf(ctx, "[Backfill] 查询 alliance_orders 失败: %v", err)
			continue
		}

		// 按 uid 分组需要更新的订单
		uidUpdates := make(map[string][]int64) // uid -> []order_info.id
		uidNames := make(map[string]string)

		for _, row := range allianceRows {
			orderId := row["order_id"].String()
			shortId := row["short_id"].String()

			if mapping, ok := shortIdMap[shortId]; ok {
				if infoId, exists := orderIdMap[orderId]; exists {
					uidUpdates[mapping.uid] = append(uidUpdates[mapping.uid], infoId)
					uidNames[mapping.uid] = mapping.name
				}
			}
		}

		// 批量更新 order_info
		for uid, ids := range uidUpdates {
			accountName := uidNames[uid]
			updateBatchSize := 500
			for k := 0; k < len(ids); k += updateBatchSize {
				updateEnd := k + updateBatchSize
				if updateEnd > len(ids) {
					updateEnd = len(ids)
				}
				updateBatch := ids[k:updateEnd]

				updatePlaceholders := ""
				updateArgs := []interface{}{uid, accountName}
				for j, id := range updateBatch {
					if j > 0 {
						updatePlaceholders += ","
					}
					updatePlaceholders += "?"
					updateArgs = append(updateArgs, id)
				}

				updateSQL := fmt.Sprintf(`
					UPDATE order_info
					SET talent_id = ?, talent_name = ?
					WHERE id IN (%s)
				`, updatePlaceholders)

				result, err := g.DB().Exec(ctx, updateSQL, updateArgs...)
				if err != nil {
					g.Log().Warningf(ctx, "[Backfill] 批量更新失败 uid=%s: %v", uid, err)
					continue
				}
				n, _ := result.RowsAffected()
				totalUpdated += int(n)
			}
		}
	}

	g.Log().Infof(ctx, "[Backfill] 回填完成: %d 条, 总耗时 %dms", totalUpdated, time.Since(start).Milliseconds())
	return totalUpdated, nil
}

// BackfillTalentInfoBatch 批量回填（分批处理，避免长时间锁表）
func (s *AllianceService) BackfillTalentInfoBatch(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 5000
	}

	totalBackfilled := 0

	for {
		result, err := g.DB().Exec(ctx, `
			UPDATE order_info o
			JOIN alliance_orders ao ON o.sub_order_no = ao.order_id AND ao.platform = 'douyin'
			JOIN account a ON ao.short_id COLLATE utf8mb4_unicode_ci = a.platform_nickname COLLATE utf8mb4_unicode_ci AND a.platform = 'douyin'
			SET o.talent_id = a.uid,
			    o.talent_name = a.account_name
			WHERE (o.talent_id IS NULL OR o.talent_id = '' OR o.talent_id = '0')
			  AND ao.short_id IS NOT NULL
			  AND ao.short_id != ''
			  AND a.uid IS NOT NULL
			  AND a.uid != ''
			LIMIT ?
		`, batchSize)
		if err != nil {
			return totalBackfilled, err
		}

		n, _ := result.RowsAffected()
		totalBackfilled += int(n)

		// 如果没有更多数据需要处理，退出循环
		if n == 0 {
			break
		}

		// 短暂休息，避免数据库压力过大
		time.Sleep(100 * time.Millisecond)
	}

	return totalBackfilled, nil
}

func formatAllianceTime(val interface{}) string {
	if val == nil {
		return ""
	}
	if t, ok := val.(time.Time); ok {
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02 15:04:05")
	}
	return ""
}
