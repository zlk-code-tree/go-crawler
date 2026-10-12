package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"
)

const (
	DetailCacheTTL = 7 * 24 * time.Hour
	CacheFileName  = "product_detail_cache.json"
)

type ProductDetailCache struct {
	records     map[string]*ProductDetailRecord
	lock        sync.RWMutex
	persistLock sync.Mutex // Serialize snapshots and file replacement, separately from cache reads.
	cachePath   string
}

type ProductDetailRecord struct {
	ProductID            string    `json:"product_id"`
	FirstShelfTime       string    `json:"first_shelf_time"`
	MinPrice             string    `json:"min_price"`
	HasShippingInsurance *bool     `json:"has_shipping_insurance"`
	DetailAccessStatus   string    `json:"detail_access_status"` // available, removed, category_mismatch
	DetailAccessNote     string    `json:"detail_access_note"`
	UpdatedAt            time.Time `json:"updated_at"`
	ProductName          string    `json:"product_name"`
	ShopName             string    `json:"shop_name"`
	DetailURL            string    `json:"detail_url"`
}

type ProductDetailSyncResult struct {
	CachedCount  int `json:"cached_count"`
	CheckedCount int `json:"checked_count"`
	UpdatedCount int `json:"updated_count"`
	MissingCount int `json:"missing_count"`
	ErrorCount   int `json:"error_count"`
}

var (
	productDetailCache *ProductDetailCache
	cacheOnce          sync.Once
)

func GetProductDetailCache() *ProductDetailCache {
	cacheOnce.Do(func() {
		productDetailCache = NewProductDetailCache()
	})
	return productDetailCache
}

func NewProductDetailCache() *ProductDetailCache {
	cacheDir := "cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("创建缓存目录失败: %v\n", err)
	}

	cachePath := filepath.Join(cacheDir, CacheFileName)
	cache := &ProductDetailCache{
		records:   make(map[string]*ProductDetailRecord),
		cachePath: cachePath,
	}

	cache.loadRecords()
	return cache
}

func (c *ProductDetailCache) loadRecords() {
	c.persistLock.Lock()
	defer c.persistLock.Unlock()
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, err := os.Stat(c.cachePath); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		fmt.Printf("读取缓存文件失败: %v\n", err)
		return
	}

	var records map[string]*ProductDetailRecord
	if err := json.Unmarshal(data, &records); err != nil {
		fmt.Printf("解析缓存文件失败: %v\n", err)
		return
	}

	if records == nil {
		records = make(map[string]*ProductDetailRecord)
	}
	c.records = records
}

func (c *ProductDetailCache) saveRecords() error {
	// Take the snapshot only after acquiring the persistence lock. Otherwise a
	// delayed writer could overwrite a newer on-disk snapshot with older data.
	c.persistLock.Lock()
	defer c.persistLock.Unlock()
	c.lock.RLock()
	records := make(map[string]*ProductDetailRecord, len(c.records))
	for id, record := range c.records {
		records[id] = record
	}
	c.lock.RUnlock()

	// Stored records are immutable; readers and writers receive their own copies.
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存数据失败: %w", err)
	}

	// Write alongside the destination and atomically replace it only when the
	// complete JSON is on disk. Failed writes leave the previous cache intact.
	file, err := os.CreateTemp(filepath.Dir(c.cachePath), ".product_detail_cache-*")
	if err != nil {
		return fmt.Errorf("创建缓存临时文件失败: %w", err)
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)
	defer file.Close()
	if err := file.Chmod(0644); err != nil {
		return fmt.Errorf("设置缓存文件权限失败: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("写入缓存文件失败: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("同步缓存文件失败: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭缓存文件失败: %w", err)
	}
	if err := os.Rename(temporaryPath, c.cachePath); err != nil {
		return fmt.Errorf("替换缓存文件失败: %w", err)
	}
	return nil
}

func cloneProductDetailRecord(record *ProductDetailRecord) *ProductDetailRecord {
	if record == nil {
		return nil
	}
	cloned := *record
	if record.HasShippingInsurance != nil {
		insurance := *record.HasShippingInsurance
		cloned.HasShippingInsurance = &insurance
	}
	return &cloned
}

func (c *ProductDetailCache) GetRecord(productID string) *ProductDetailRecord {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return cloneProductDetailRecord(c.records[strings.TrimSpace(productID)])
}

func (c *ProductDetailCache) GetRecords(productIDs []string) map[string]*ProductDetailRecord {
	c.lock.RLock()
	defer c.lock.RUnlock()

	result := make(map[string]*ProductDetailRecord)
	for _, id := range productIDs {
		if record := c.records[strings.TrimSpace(id)]; record != nil {
			result[id] = cloneProductDetailRecord(record)
		}
	}
	return result
}

func (c *ProductDetailCache) UpsertRecord(record *ProductDetailRecord) *ProductDetailRecord {
	stored, err := c.UpsertRecordWithError(record)
	if err != nil {
		fmt.Printf("更新商品详情缓存失败: %v\n", err)
	}
	return stored
}

// UpsertRecordWithError updates the in-memory cache and persists a snapshot.
// On a persistence error the in-memory entry is retained and returned with the error.
func (c *ProductDetailCache) UpsertRecordWithError(record *ProductDetailRecord) (*ProductDetailRecord, error) {
	if record == nil {
		return nil, fmt.Errorf("商品详情缓存记录不能为空")
	}
	key := strings.TrimSpace(record.ProductID)
	if key == "" {
		return nil, fmt.Errorf("商品详情缓存商品 ID 不能为空")
	}

	stored := cloneProductDetailRecord(record)
	stored.ProductID = key
	stored.UpdatedAt = time.Now()
	c.lock.Lock()
	if c.records == nil {
		c.records = make(map[string]*ProductDetailRecord)
	}
	c.records[key] = stored
	c.lock.Unlock()

	err := c.saveRecords()
	return cloneProductDetailRecord(stored), err
}

func (c *ProductDetailCache) IsRecordStale(record *ProductDetailRecord) bool {
	return record == nil || time.Since(record.UpdatedAt) >= DetailCacheTTL
}

func (c *ProductDetailCache) ShouldRefreshRecord(record *ProductDetailRecord, item *model.TikTokRankingListInput) bool {
	if record == nil {
		return true
	}

	if !c.IsRecordComplete(record) {
		return true
	}

	if c.IsRecordStale(record) {
		return true
	}

	return false
}

func (c *ProductDetailCache) IsRecordComplete(record *ProductDetailRecord) bool {
	if record == nil {
		return false
	}
	if record.DetailAccessStatus == "removed" || record.DetailAccessStatus == "category_mismatch" {
		return true
	}

	return record.FirstShelfTime != "" && record.MinPrice != ""
}

func (c *ProductDetailCache) DetailURLMatchesProduct(detailURL, productID string) bool {
	cleanedProductID := strings.TrimSpace(productID)
	if cleanedProductID == "" {
		return false
	}

	// 简单实现：检查URL中是否包含product_id参数
	return strings.Contains(detailURL, "product_id="+cleanedProductID) || strings.Contains(detailURL, "id="+cleanedProductID)
}

func (c *ProductDetailCache) ApplyCachedData(items []*model.TikTokRankingListInput) {
	productIDs := make([]string, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductId
	}

	records := c.GetRecords(productIDs)
	for _, item := range items {
		if record, ok := records[item.ProductId]; ok {
			if item.FirstShelfTime == "" && record.FirstShelfTime != "" {
				item.FirstShelfTime = record.FirstShelfTime
			}
			if item.MinPrice == "" && record.MinPrice != "" {
				item.MinPrice = record.MinPrice
			}
			if item.ShopName == "" && record.ShopName != "" {
				item.ShopName = record.ShopName
			}
		}
	}
}
