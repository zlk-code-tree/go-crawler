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
	records   map[string]*ProductDetailRecord
	lock      sync.RWMutex
	cachePath string
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

	c.records = records
}

func (c *ProductDetailCache) saveRecords() {
	c.lock.RLock()
	data, err := json.MarshalIndent(c.records, "", "  ")
	c.lock.RUnlock()

	if err != nil {
		fmt.Printf("序列化缓存数据失败: %v\n", err)
		return
	}

	if err := os.WriteFile(c.cachePath, data, 0644); err != nil {
		fmt.Printf("写入缓存文件失败: %v\n", err)
	}
}

func (c *ProductDetailCache) GetRecord(productID string) *ProductDetailRecord {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.records[strings.TrimSpace(productID)]
}

func (c *ProductDetailCache) GetRecords(productIDs []string) map[string]*ProductDetailRecord {
	c.lock.RLock()
	defer c.lock.RUnlock()

	result := make(map[string]*ProductDetailRecord)
	for _, id := range productIDs {
		if record, ok := c.records[strings.TrimSpace(id)]; ok {
			result[id] = record
		}
	}
	return result
}

func (c *ProductDetailCache) UpsertRecord(record *ProductDetailRecord) *ProductDetailRecord {
	c.lock.Lock()
	defer c.lock.Unlock()

	key := strings.TrimSpace(record.ProductID)
	if key == "" {
		return nil
	}

	record.UpdatedAt = time.Now()
	c.records[key] = record
	c.saveRecords()
	return record
}

func (c *ProductDetailCache) IsRecordStale(record *ProductDetailRecord) bool {
	return time.Since(record.UpdatedAt) >= DetailCacheTTL
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
