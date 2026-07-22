package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"

	"go-crawler/internal/model"
	"go-crawler/internal/service/platform/wechatshop"
)

type StoreService struct{}

const wechatShopAuthTable = "wechat_shop_auth"

func NewStoreService() *StoreService {
	return &StoreService{}
}

func (s *StoreService) ensureWechatShopSchema(ctx context.Context) error {
	return wechatshop.NewService().EnsureSchema(ctx)
}

// List 查询店铺列表
func (s *StoreService) List(ctx context.Context, page, pageSize int, keyword, platform string) ([]model.Store, int64, error) {
	if err := s.ensureWechatShopSchema(ctx); err != nil {
		return nil, 0, err
	}

	m := g.DB().Model("store s").Ctx(ctx)

	if keyword != "" {
		m = m.WhereLike("s.store_name", "%"+keyword+"%")
	}
	if platform != "" {
		m = m.Where("s.platform", platform)
	}

	total, err := m.Count()
	if err != nil {
		return nil, 0, err
	}

	rows, err := m.
		LeftJoin(wechatShopAuthTable+" a", "a.store_id = s.id").
		Fields(`
			s.*,
			a.appid AS wechat_app_id,
			IF(a.app_secret IS NULL OR a.app_secret = '', 0, 1) AS has_wechat_app_secret,
			a.expires_at AS wechat_token_expires_at,
			a.last_pull_time AS wechat_last_pull_time`).
		Order("s.id DESC").
		Page(page, pageSize).
		All()
	if err != nil {
		return nil, 0, err
	}

	stores := make([]model.Store, 0, len(rows))
	for _, row := range rows {
		store := model.Store{
			Id:                    row["id"].Int(),
			StoreName:             row["store_name"].String(),
			StoreId:               row["store_id"].String(),
			Platform:              row["platform"].String(),
			Status:                row["status"].Int(),
			Author:                row["author"].Int(),
			BlankOrderEnabled:     row["blank_order_enabled"].Int(),
			BlankOrderOwnerName:   row["blank_order_owner_name"].String(),
			WechatAppId:           row["wechat_app_id"].String(),
			HasWechatAppSecret:    row["has_wechat_app_secret"].Int(),
			MaskedWechatAppSecret: maskAppSecret(row["wechat_app_id"].String(), row["has_wechat_app_secret"].Int()),
			CreatedAt:             row["created_at"].Time(),
			UpdatedAt:             row["updated_at"].Time(),
		}
		if !row["last_poll_time"].IsEmpty() {
			t := row["last_poll_time"].Time()
			store.LastPollTime = &t
		}
		if !row["wechat_token_expires_at"].IsEmpty() {
			t := row["wechat_token_expires_at"].Time()
			store.WechatTokenExpiresAt = &t
		}
		if !row["wechat_last_pull_time"].IsEmpty() {
			t := row["wechat_last_pull_time"].Time()
			store.WechatLastPullTime = &t
		}
		stores = append(stores, store)
	}

	return stores, int64(total), nil
}

// Detail 获取店铺详情
func (s *StoreService) Detail(ctx context.Context, id int) (*model.Store, error) {
	if id <= 0 {
		return nil, gerror.New("店铺ID不能为空")
	}

	if err := s.ensureWechatShopSchema(ctx); err != nil {
		return nil, err
	}

	row, err := g.DB().Model("store s").Ctx(ctx).
		LeftJoin(wechatShopAuthTable+" a", "a.store_id = s.id").
		Where("s.id", id).
		Fields(`
			s.*,
			a.appid AS wechat_app_id,
			IF(a.app_secret IS NULL OR a.app_secret = '', 0, 1) AS has_wechat_app_secret,
			a.expires_at AS wechat_token_expires_at,
			a.last_pull_time AS wechat_last_pull_time`).
		One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, nil
	}

	store := &model.Store{
		Id:                    row["id"].Int(),
		StoreName:             row["store_name"].String(),
		StoreId:               row["store_id"].String(),
		Platform:              row["platform"].String(),
		Status:                row["status"].Int(),
		Author:                row["author"].Int(),
		BlankOrderEnabled:     row["blank_order_enabled"].Int(),
		BlankOrderOwnerName:   row["blank_order_owner_name"].String(),
		WechatAppId:           row["wechat_app_id"].String(),
		HasWechatAppSecret:    row["has_wechat_app_secret"].Int(),
		MaskedWechatAppSecret: maskAppSecret(row["wechat_app_id"].String(), row["has_wechat_app_secret"].Int()),
		CreatedAt:             row["created_at"].Time(),
		UpdatedAt:             row["updated_at"].Time(),
	}
	if !row["last_poll_time"].IsEmpty() {
		t := row["last_poll_time"].Time()
		store.LastPollTime = &t
	}
	if !row["wechat_token_expires_at"].IsEmpty() {
		t := row["wechat_token_expires_at"].Time()
		store.WechatTokenExpiresAt = &t
	}
	if !row["wechat_last_pull_time"].IsEmpty() {
		t := row["wechat_last_pull_time"].Time()
		store.WechatLastPullTime = &t
	}
	return store, nil
}

// Save 保存店铺
func (s *StoreService) Save(ctx context.Context, req *model.StoreSaveReq) (int64, error) {
	if err := s.ensureWechatShopSchema(ctx); err != nil {
		return 0, err
	}

	// 检查同一平台下 store_id 是否已存在
	count, err := g.DB().Model("store").Ctx(ctx).
		Where("platform", req.Platform).
		Where("store_id", req.StoreId).
		WhereNot("id", req.Id).
		Count()
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, gerror.New("该平台下店铺ID已存在")
	}

	data := g.Map{
		"store_name":             req.StoreName,
		"store_id":               req.StoreId,
		"platform":               req.Platform,
		"status":                 req.Status,
		"author":                 req.Author,
		"blank_order_enabled":    req.BlankOrderEnabled,
		"blank_order_owner_name": strings.TrimSpace(req.BlankOrderOwnerName),
	}

	now := time.Now()
	var storeID int64
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if req.Id > 0 {
			storeID = int64(req.Id)
			data["updated_at"] = now
			if _, err := tx.Model("store").Ctx(ctx).Where("id", req.Id).Data(data).Update(); err != nil {
				return err
			}
		} else {
			data["created_at"] = now
			data["updated_at"] = now
			id, err := tx.Model("store").Ctx(ctx).Data(data).InsertAndGetId()
			if err != nil {
				return err
			}
			storeID = id
		}

		return s.syncWechatShopAuth(ctx, tx, storeID, req)
	})
	if err != nil {
		return 0, err
	}
	return storeID, nil
}

// Delete 删除店铺
func (s *StoreService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return gerror.New("店铺ID不能为空")
	}
	if err := s.ensureWechatShopSchema(ctx); err != nil {
		return err
	}
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(wechatShopAuthTable).Ctx(ctx).Where("store_id", id).Delete(); err != nil {
			return err
		}
		_, err := tx.Model("store").Ctx(ctx).Where("id", id).Delete()
		return err
	})
}

// Stats 店铺平台统计
func (s *StoreService) Stats(ctx context.Context) (g.Map, error) {
	rows, err := g.DB().GetAll(ctx, "SELECT platform, COUNT(*) as cnt FROM store GROUP BY platform")
	if err != nil {
		return nil, err
	}

	stats := g.Map{"total": 0, "douyin": 0, "kuaishou": 0, "shipinhao": 0}
	for _, row := range rows {
		platform := row["platform"].String()
		cnt := row["cnt"].Int()
		stats[platform] = cnt
		stats["total"] = stats["total"].(int) + cnt
	}

	return stats, nil
}

func (s *StoreService) syncWechatShopAuth(ctx context.Context, tx gdb.TX, storeID int64, req *model.StoreSaveReq) error {
	if req.Platform != "shipinhao" {
		_, err := tx.Model(wechatShopAuthTable).Ctx(ctx).Where("store_id", storeID).Delete()
		return err
	}

	existing, err := tx.Model(wechatShopAuthTable).Ctx(ctx).Where("store_id", storeID).One()
	if err != nil {
		return err
	}

	appID := strings.TrimSpace(req.WechatAppId)
	appSecret := strings.TrimSpace(req.WechatAppSecret)
	if appID == "" && !existing.IsEmpty() {
		appID = existing["appid"].String()
	}
	if appSecret == "" && !existing.IsEmpty() {
		appSecret = existing["app_secret"].String()
	}

	if req.Author == 1 && (appID == "" || appSecret == "") {
		return gerror.New("视频号已授权店铺需配置微信小店 appid/appsecret")
	}
	if appID == "" && appSecret == "" && existing.IsEmpty() {
		return nil
	}
	if appID == "" || appSecret == "" {
		return gerror.New("视频号店铺的微信小店 appid/appsecret 需要同时填写")
	}

	data := g.Map{
		"store_id":   storeID,
		"appid":      appID,
		"app_secret": appSecret,
		"updated_at": time.Now(),
	}
	if existing.IsEmpty() {
		data["created_at"] = time.Now()
		_, err = tx.Model(wechatShopAuthTable).Ctx(ctx).Data(data).Insert()
		return err
	}
	_, err = tx.Model(wechatShopAuthTable).Ctx(ctx).Where("store_id", storeID).Data(data).Update()
	return err
}

func maskAppSecret(appID string, hasSecret int) string {
	if hasSecret == 0 {
		return ""
	}
	if len(appID) >= 6 {
		return appID[:3] + strings.Repeat("*", len(appID)-6) + appID[len(appID)-3:]
	}
	return "******"
}

// ExportToExcel 导出店铺数据到Excel
func (s *StoreService) ExportToExcel(ctx context.Context, keyword, platform string) (string, error) {
	m := g.DB().Model("store").Ctx(ctx)
	if keyword != "" {
		m = m.WhereLike("store_name", "%"+keyword+"%")
	}
	if platform != "" {
		m = m.Where("platform", platform)
	}

	result, err := m.Order("id DESC").All()
	if err != nil {
		return "", err
	}
	if len(result) == 0 {
		return "", gerror.New("没有数据可导出")
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "店铺信息"
	f.NewSheet(sheetName)

	headers := []string{"店铺名称", "店铺ID", "平台", "状态", "授权", "是否参与空白单归属", "空白单归属人"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	platformMap := map[string]string{"douyin": "抖音", "kuaishou": "快手", "shipinhao": "视频号"}
	statusMap := map[int64]string{0: "禁用", 1: "启用"}
	authorMap := map[int64]string{0: "未授权", 1: "已授权"}
	for i, row := range result {
		rowNum := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row["store_name"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row["store_id"])
		p := row["platform"].String()
		if v, ok := platformMap[p]; ok {
			p = v
		}
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), p)
		s := row["status"].Int64()
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), statusMap[s])
		a := row["author"].Int64()
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), authorMap[a])
		enabled := "是"
		if row["blank_order_enabled"].Int() == 0 {
			enabled = "否"
		}
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), enabled)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), row["blank_order_owner_name"])
	}

	f.SetColWidth(sheetName, "A", "A", 30)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 15)
	f.SetColWidth(sheetName, "D", "D", 10)
	f.SetColWidth(sheetName, "E", "E", 10)
	f.SetColWidth(sheetName, "F", "G", 20)
	f.DeleteSheet("Sheet1")

	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, fmt.Sprintf("store_export_%s.xlsx", time.Now().Format("20060102150405")))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败：%w", err)
	}

	return filename, nil
}

// ExportTemplate 导出店铺导入模板
func (s *StoreService) ExportTemplate(ctx context.Context) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "店铺信息"
	f.NewSheet(sheetName)

	headers := []string{"店铺名称", "店铺ID", "平台", "状态", "授权", "是否参与空白单归属", "空白单归属人"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// 示例数据
	samples := [][]string{
		{"利云文化官方旗舰店", "21739359", "douyin", "启用", "已授权", "是", "王梅雪"},
		{"示例店铺", "12345678", "kuaishou", "启用", "未授权", "否", ""},
	}
	for i, sample := range samples {
		for j, val := range sample {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	f.SetColWidth(sheetName, "A", "A", 30)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 15)
	f.SetColWidth(sheetName, "D", "D", 10)
	f.SetColWidth(sheetName, "E", "E", 10)
	f.SetColWidth(sheetName, "F", "G", 20)

	helpSheet := "填写说明"
	f.NewSheet(helpSheet)
	f.SetCellValue(helpSheet, "A1", "字段说明")
	f.SetCellValue(helpSheet, "A2", "【店铺名称】必填，店铺的显示名称")
	f.SetCellValue(helpSheet, "A3", "【店铺ID】必填，外部平台的店铺ID（如抖店shop_id）")
	f.SetCellValue(helpSheet, "A4", "【平台】必填，支持：douyin/抖音、kuaishou/快手、shipinhao/视频号")
	f.SetCellValue(helpSheet, "A5", "【状态】选填，启用/禁用，默认启用")
	f.SetCellValue(helpSheet, "A6", "【授权】选填，已授权/未授权，默认未授权")
	f.SetCellValue(helpSheet, "A7", "【是否参与空白单归属】选填，是/否，默认是")
	f.SetCellValue(helpSheet, "A8", "【空白单归属人】选填，参与空白单归属时建议填写最终运营者/投手名称")
	f.SetCellValue(helpSheet, "A10", "注意：同一平台下店铺ID不能重复")
	helpStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
	f.SetCellStyle(helpSheet, "A1", "A1", helpStyle)
	f.SetColWidth(helpSheet, "A", "A", 60)

	f.DeleteSheet("Sheet1")

	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	filename := filepath.Join(exportDir, "store_import_template.xlsx")
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存模版文件失败：%w", err)
	}
	return filename, nil
}

// ImportFromExcel 从Excel导入店铺数据
func (s *StoreService) ImportFromExcel(ctx context.Context, filePath string) (*ImportResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开Excel文件失败：%w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, gerror.New("Excel文件中没有工作表")
	}

	rows, err := f.Rows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读取Excel数据失败：%w", err)
	}
	defer rows.Close()

	result := &ImportResult{Errors: make([]string, 0)}

	if !rows.Next() {
		return nil, gerror.New("Excel文件中没有数据")
	}
	headers, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("读取表头失败：%w", err)
	}

	// 表头映射：支持中英文
	headerMapping := map[string]string{
		"store_name":             "store_name",
		"店铺名称":                   "store_name",
		"store_id":               "store_id",
		"店铺ID":                   "store_id",
		"平台":                     "platform",
		"platform":               "platform",
		"状态":                     "status",
		"授权":                     "author",
		"author":                 "author",
		"是否参与空白单归属":              "blank_order_enabled",
		"blank_order_enabled":    "blank_order_enabled",
		"空白单归属人":                 "blank_order_owner_name",
		"blank_order_owner_name": "blank_order_owner_name",
	}
	headerIndex := make(map[string]int)
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if mapped, ok := headerMapping[h]; ok {
			headerIndex[mapped] = i
		}
	}
	for _, required := range []string{"store_name", "store_id", "platform"} {
		if _, ok := headerIndex[required]; !ok {
			return nil, gerror.New(fmt.Sprintf("缺少必要的列: %s\n实际读取到的表头: %v\n支持的中文列名：店铺名称、店铺ID、平台", required, headers))
		}
	}

	rowNum := 1
	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 读取失败 - %v", rowNum, err))
			result.FailCount++
			continue
		}
		result.TotalRows++

		getCol := func(key string) string {
			if idx, ok := headerIndex[key]; ok && idx < len(cols) {
				return strings.TrimSpace(cols[idx])
			}
			return ""
		}

		storeName := getCol("store_name")
		storeId := getCol("store_id")
		platform := normalizePlatform(getCol("platform"))
		statusStr := getCol("status")
		blankOrderEnabledStr := getCol("blank_order_enabled")
		blankOrderOwnerName := getCol("blank_order_owner_name")

		if storeName == "" || storeId == "" || platform == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: 店铺名称/店铺ID/平台不能为空", rowNum))
			result.FailCount++
			continue
		}

		status := 1
		if statusStr != "" && (strings.Contains(statusStr, "禁用") || statusStr == "0") {
			status = 0
		}

		authorStr := getCol("author")
		author := 0
		if authorStr != "" && (strings.Contains(authorStr, "已授权") || authorStr == "1") {
			author = 1
		}
		blankOrderEnabled := 1
		if blankOrderEnabledStr != "" && (strings.Contains(blankOrderEnabledStr, "否") || blankOrderEnabledStr == "0") {
			blankOrderEnabled = 0
		}

		// 检查是否已存在
		count, _ := g.DB().Model("store").Ctx(ctx).
			Where("platform", platform).
			Where("store_id", storeId).
			Count()
		if count > 0 {
			result.SkipCount++
			continue
		}

		_, err = g.DB().Model("store").Ctx(ctx).Data(g.Map{
			"store_name":             storeName,
			"store_id":               storeId,
			"platform":               platform,
			"status":                 status,
			"author":                 author,
			"blank_order_enabled":    blankOrderEnabled,
			"blank_order_owner_name": blankOrderOwnerName,
			"created_at":             time.Now(),
			"updated_at":             time.Now(),
		}).Insert()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行: %v", rowNum, err))
			result.FailCount++
			continue
		}
		result.SuccessCount++
	}

	return result, nil
}
