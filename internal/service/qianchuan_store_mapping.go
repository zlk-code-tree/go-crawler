package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

type QianchuanStoreImportRow struct {
	RowNumber         int    `json:"rowNumber"`
	AdvertiserName    string `json:"advertiserName"`
	AdvertiserID      int64  `json:"advertiserId"`
	SourceStoreName   string `json:"sourceStoreName"`
	StoreID           int    `json:"storeId"`
	StoreName         string `json:"storeName"`
	ExistingStoreID   int    `json:"existingStoreId"`
	ExistingStoreName string `json:"existingStoreName"`
	Status            string `json:"status"`
	Reason            string `json:"reason"`
}

type QianchuanStoreImportResult struct {
	TotalCount     int                       `json:"totalCount"`
	ReadyCount     int                       `json:"readyCount"`
	ImportedCount  int                       `json:"importedCount"`
	UnchangedCount int                       `json:"unchangedCount"`
	UnmatchedCount int                       `json:"unmatchedCount"`
	ConflictCount  int                       `json:"conflictCount"`
	InvalidCount   int                       `json:"invalidCount"`
	DryRun         bool                      `json:"dryRun"`
	Rows           []QianchuanStoreImportRow `json:"rows"`
}

type QianchuanOfficialMappingRow struct {
	AdvertiserID    int64  `json:"advertiserId"`
	AdvertiserName  string `json:"advertiserName"`
	PlatformShopID  int64  `json:"platformShopId"`
	StoreID         int    `json:"storeId"`
	StoreName       string `json:"storeName"`
	StoreStatus     int    `json:"storeStatus"`
	ExistingStoreID int    `json:"existingStoreId"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
}

type QianchuanOfficialMappingResult struct {
	TotalCount     int                           `json:"totalCount"`
	ReadyCount     int                           `json:"readyCount"`
	ImportedCount  int                           `json:"importedCount"`
	UnchangedCount int                           `json:"unchangedCount"`
	UnmatchedCount int                           `json:"unmatchedCount"`
	ConflictCount  int                           `json:"conflictCount"`
	DryRun         bool                          `json:"dryRun"`
	Rows           []QianchuanOfficialMappingRow `json:"rows"`
}

type QianchuanStoreMappingService struct{}

var (
	qianchuanStoreSchemaMu    sync.Mutex
	qianchuanStoreSchemaReady bool
)

func NewQianchuanStoreMappingService() *QianchuanStoreMappingService {
	return &QianchuanStoreMappingService{}
}

func (s *QianchuanStoreMappingService) EnsureSchema(ctx context.Context) error {
	qianchuanStoreSchemaMu.Lock()
	defer qianchuanStoreSchemaMu.Unlock()
	if qianchuanStoreSchemaReady {
		return nil
	}
	_, err := g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS store_qianchuan_mapping (
		id BIGINT NOT NULL AUTO_INCREMENT,
		store_id INT NOT NULL,
		qianchuan_advertiser_id BIGINT NOT NULL,
		qianchuan_advertiser_name VARCHAR(200) NOT NULL DEFAULT '',
		source_store_name VARCHAR(255) NOT NULL DEFAULT '',
		source VARCHAR(30) NOT NULL DEFAULT 'manual',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		UNIQUE KEY uk_qianchuan_advertiser (qianchuan_advertiser_id),
		KEY idx_store_id (store_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='店铺与千川账户归属关系表'`)
	if err == nil {
		qianchuanStoreSchemaReady = true
	}
	return err
}

func normalizeQianchuanStoreName(value string) string {
	value = strings.TrimSpace(value)
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, value)
}

func normalizeQianchuanHeader(value string) string {
	return normalizeQianchuanStoreName(strings.TrimPrefix(value, "\ufeff"))
}

func (s *QianchuanStoreMappingService) ImportExcel(ctx context.Context, filePath string, dryRun bool) (*QianchuanStoreImportResult, error) {
	if err := s.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("初始化千川店铺归属表失败: %w", err)
	}

	rows, err := readQianchuanStoreExcel(filePath)
	if err != nil {
		return nil, err
	}
	stores, err := g.DB().Model("store").Ctx(ctx).
		Where("platform", "douyin").
		Fields("id, store_name, status").All()
	if err != nil {
		return nil, fmt.Errorf("查询抖音店铺失败: %w", err)
	}
	existingRows, err := g.DB().Model("store_qianchuan_mapping sqm").Ctx(ctx).
		LeftJoin("store s", "s.id = sqm.store_id").
		Fields("sqm.qianchuan_advertiser_id, sqm.store_id, s.store_name").All()
	if err != nil {
		return nil, fmt.Errorf("查询现有千川店铺归属失败: %w", err)
	}

	storeByName := make(map[string][]gdb.Record)
	for _, store := range stores {
		key := normalizeQianchuanStoreName(store["store_name"].String())
		storeByName[key] = append(storeByName[key], store)
	}
	existingByAdvertiser := make(map[int64]gdb.Record, len(existingRows))
	for _, existing := range existingRows {
		existingByAdvertiser[existing["qianchuan_advertiser_id"].Int64()] = existing
	}

	result := &QianchuanStoreImportResult{DryRun: dryRun, Rows: rows}
	seenAdvertisers := make(map[int64]int)
	for i := range result.Rows {
		row := &result.Rows[i]
		result.TotalCount++
		if row.AdvertiserID <= 0 || row.AdvertiserName == "" || row.SourceStoreName == "" {
			row.Status = "invalid"
			row.Reason = "账户名称、账户ID和账户店铺均不能为空"
			result.InvalidCount++
			continue
		}
		if firstRow, duplicated := seenAdvertisers[row.AdvertiserID]; duplicated {
			row.Status = "invalid"
			row.Reason = fmt.Sprintf("账户ID与第%d行重复", firstRow)
			result.InvalidCount++
			continue
		}
		seenAdvertisers[row.AdvertiserID] = row.RowNumber

		allMatchedStores := storeByName[normalizeQianchuanStoreName(row.SourceStoreName)]
		matchedStores := make([]gdb.Record, 0, len(allMatchedStores))
		for _, store := range allMatchedStores {
			if store["status"].Int() == 1 {
				matchedStores = append(matchedStores, store)
			}
		}
		if len(allMatchedStores) > 0 && len(matchedStores) == 0 {
			row.Status = "unmatched"
			row.Reason = "系统中同名抖音店铺已禁用，请先启用店铺"
			result.UnmatchedCount++
			continue
		}
		if len(matchedStores) == 0 {
			row.Status = "unmatched"
			row.Reason = "系统中未找到同名的启用中抖音店铺"
			result.UnmatchedCount++
			continue
		}
		if len(matchedStores) > 1 {
			row.Status = "conflict"
			row.Reason = "系统中存在多个同名店铺，请先合并店铺数据"
			result.ConflictCount++
			continue
		}
		row.StoreID = matchedStores[0]["id"].Int()
		row.StoreName = matchedStores[0]["store_name"].String()

		if existing, ok := existingByAdvertiser[row.AdvertiserID]; ok {
			row.ExistingStoreID = existing["store_id"].Int()
			row.ExistingStoreName = existing["store_name"].String()
			if row.ExistingStoreID != row.StoreID {
				row.Status = "conflict"
				row.Reason = "该千川账户已归属其他店铺，本次不会覆盖"
				result.ConflictCount++
				continue
			}
			row.Status = "unchanged"
			row.Reason = "店铺归属已存在"
			result.UnchangedCount++
			continue
		}

		row.Status = "ready"
		row.Reason = "可绑定"
		result.ReadyCount++
	}

	if dryRun || result.ReadyCount == 0 {
		return result, nil
	}

	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		for i := range result.Rows {
			row := &result.Rows[i]
			if row.Status != "ready" {
				continue
			}
			_, insertErr := tx.Model("store_qianchuan_mapping").Ctx(ctx).Data(g.Map{
				"store_id":                  row.StoreID,
				"qianchuan_advertiser_id":   row.AdvertiserID,
				"qianchuan_advertiser_name": row.AdvertiserName,
				"source_store_name":         row.SourceStoreName,
				"source":                    "excel",
			}).Insert()
			if insertErr != nil {
				return fmt.Errorf("第%d行绑定失败: %w", row.RowNumber, insertErr)
			}
			row.Status = "imported"
			row.Reason = "绑定成功"
			result.ImportedCount++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// SyncOfficialRelations 按千川接口返回的平台店铺关系精确同步账户归属。
func (s *QianchuanStoreMappingService) SyncOfficialRelations(ctx context.Context, relations map[int64]int64, dryRun bool) (*QianchuanOfficialMappingResult, error) {
	if err := s.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("初始化千川店铺归属表失败: %w", err)
	}

	stores, err := g.DB().Model("store").Ctx(ctx).
		Where("platform", "douyin").Fields("id, store_id, store_name, status").All()
	if err != nil {
		return nil, fmt.Errorf("查询抖音店铺失败: %w", err)
	}
	storeByPlatformID := make(map[int64]gdb.Record, len(stores))
	for _, store := range stores {
		platformID, parseErr := strconv.ParseInt(strings.TrimSpace(store["store_id"].String()), 10, 64)
		if parseErr == nil && platformID > 0 {
			storeByPlatformID[platformID] = store
		}
	}

	existingRows, err := g.DB().Model("store_qianchuan_mapping sqm").Ctx(ctx).
		Fields("sqm.qianchuan_advertiser_id, sqm.store_id").All()
	if err != nil {
		return nil, fmt.Errorf("查询现有千川店铺归属失败: %w", err)
	}
	existingByAdvertiser := make(map[int64]int, len(existingRows))
	for _, row := range existingRows {
		existingByAdvertiser[row["qianchuan_advertiser_id"].Int64()] = row["store_id"].Int()
	}

	advertiserIDs := make([]int64, 0, len(relations))
	for advertiserID := range relations {
		advertiserIDs = append(advertiserIDs, advertiserID)
	}
	sort.Slice(advertiserIDs, func(i, j int) bool { return advertiserIDs[i] < advertiserIDs[j] })

	nameByAdvertiser := make(map[int64]string, len(advertiserIDs))
	if len(advertiserIDs) > 0 {
		nameRows, queryErr := g.DB().Model("qianchuan_ad_report").Ctx(ctx).
			WhereIn("advertiser_id", advertiserIDs).
			Where("advertiser_name != ''").
			Fields("advertiser_id, MAX(advertiser_name) AS advertiser_name").
			Group("advertiser_id").All()
		if queryErr == nil {
			for _, row := range nameRows {
				nameByAdvertiser[row["advertiser_id"].Int64()] = row["advertiser_name"].String()
			}
		}
	}

	result := &QianchuanOfficialMappingResult{DryRun: dryRun, Rows: make([]QianchuanOfficialMappingRow, 0, len(relations))}
	for _, advertiserID := range advertiserIDs {
		row := QianchuanOfficialMappingRow{
			AdvertiserID:   advertiserID,
			AdvertiserName: nameByAdvertiser[advertiserID],
			PlatformShopID: relations[advertiserID],
		}
		result.TotalCount++
		store, matched := storeByPlatformID[row.PlatformShopID]
		if !matched {
			row.Status = "unmatched"
			row.Reason = "系统中没有对应平台店铺ID"
			result.UnmatchedCount++
			result.Rows = append(result.Rows, row)
			continue
		}
		row.StoreID = store["id"].Int()
		row.StoreName = store["store_name"].String()
		row.StoreStatus = store["status"].Int()
		if existingStoreID, exists := existingByAdvertiser[advertiserID]; exists {
			row.ExistingStoreID = existingStoreID
			if existingStoreID != row.StoreID {
				row.Status = "conflict"
				row.Reason = "现有归属与千川官方店铺关系不一致，本次未覆盖"
				result.ConflictCount++
			} else {
				row.Status = "unchanged"
				row.Reason = "店铺归属已存在"
				result.UnchangedCount++
			}
			result.Rows = append(result.Rows, row)
			continue
		}
		row.Status = "ready"
		row.Reason = "可按千川官方关系绑定"
		result.ReadyCount++
		result.Rows = append(result.Rows, row)
	}

	if dryRun || result.ReadyCount == 0 {
		return result, nil
	}
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		for index := range result.Rows {
			row := &result.Rows[index]
			if row.Status != "ready" {
				continue
			}
			_, insertErr := tx.Model("store_qianchuan_mapping").Ctx(ctx).Data(g.Map{
				"store_id":                  row.StoreID,
				"qianchuan_advertiser_id":   row.AdvertiserID,
				"qianchuan_advertiser_name": row.AdvertiserName,
				"source_store_name":         row.StoreName,
				"source":                    "platform",
			}).Insert()
			if insertErr != nil {
				return fmt.Errorf("千川账户%d绑定失败: %w", row.AdvertiserID, insertErr)
			}
			row.Status = "imported"
			row.Reason = "已按千川官方关系绑定"
			result.ImportedCount++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func readQianchuanStoreExcel(filePath string) ([]QianchuanStoreImportRow, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开Excel失败: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel中没有工作表")
	}
	allRows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读取Excel失败: %w", err)
	}
	if len(allRows) < 2 {
		return nil, fmt.Errorf("Excel中没有可导入的数据")
	}

	headerIndexes := make(map[string]int)
	for index, header := range allRows[0] {
		headerIndexes[normalizeQianchuanHeader(header)] = index
	}
	required := []string{"账户名称", "账户id", "账户店铺"}
	for _, header := range required {
		if _, ok := headerIndexes[header]; !ok {
			return nil, fmt.Errorf("Excel缺少必填列：%s", header)
		}
	}
	valueAt := func(row []string, index int) string {
		if index < 0 || index >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[index])
	}

	result := make([]QianchuanStoreImportRow, 0, len(allRows)-1)
	for rowIndex, values := range allRows[1:] {
		name := valueAt(values, headerIndexes["账户名称"])
		idText := valueAt(values, headerIndexes["账户id"])
		storeName := valueAt(values, headerIndexes["账户店铺"])
		if name == "" && idText == "" && storeName == "" {
			continue
		}
		advertiserID, _ := strconv.ParseInt(idText, 10, 64)
		result = append(result, QianchuanStoreImportRow{
			RowNumber:       rowIndex + 2,
			AdvertiserName:  name,
			AdvertiserID:    advertiserID,
			SourceStoreName: storeName,
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("Excel中没有有效数据行")
	}
	return result, nil
}
