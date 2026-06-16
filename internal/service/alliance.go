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
func (s *AllianceService) Export(ctx context.Context, keyword, shopName, authorName, orderStatus, startTime, endTime string) (string, error) {
	query := g.DB().Model("alliance_orders").Ctx(ctx)

	if keyword != "" {
		query = query.Where("(order_id LIKE ? OR product_name LIKE ? OR author_name LIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if shopName != "" {
		query = query.Where("shop_name LIKE ?", "%"+shopName+"%")
	}
	if authorName != "" {
		query = query.Where("author_name LIKE ?", "%"+authorName+"%")
	}
	if orderStatus != "" {
		query = query.Where("order_status = ?", orderStatus)
	}
	if startTime != "" {
		query = query.Where("pay_time >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("pay_time <= ?", endTime)
	}

	result, err := query.Order("pay_time DESC").All()
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

// BackfillTalentInfo 回填达人信息到 order_info
// 从 alliance_orders.author_account 关联 account 表获取 uid/talent_name
// 只更新 talent_id 为空或为'0'的订单
func (s *AllianceService) BackfillTalentInfo(ctx context.Context) (int, error) {
	result, err := g.DB().Exec(ctx, `
		UPDATE order_info o
		JOIN alliance_orders ao ON o.sub_order_no = ao.order_id AND ao.platform = 'douyin'
		JOIN account a ON ao.author_account COLLATE utf8mb4_unicode_ci = a.account_name COLLATE utf8mb4_unicode_ci
		SET o.talent_id = a.uid,
		    o.talent_name = a.account_name
		WHERE (o.talent_id IS NULL OR o.talent_id = '' OR o.talent_id = '0')
		  AND ao.author_account IS NOT NULL
		  AND ao.author_account != ''
		  AND a.uid IS NOT NULL
		  AND a.uid != ''
	`)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
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
