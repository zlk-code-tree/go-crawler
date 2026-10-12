package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/logger"
	"go-crawler/internal/model"
)

// iTikTokRankingList 定义了抖音排行榜列表服务的接口。
type iTikTokRankingList interface {
	CreateBatch(ctx context.Context, data []*model.TikTokRankingListInput) error
	UpdateByProductId(ctx context.Context, data *model.TikTokRankingListInput) error
}

// sTikTokRankingList 抖音排行榜列表服务的具体实现结构体。
type sTikTokRankingList struct{}

// 初始化时创建并注册一个全局的 sTikTokRankingList 实例。
var localService = sTikTokRankingList{}

// TikTokRankingList 返回一个抖音排行榜列表服务的实例。
func TikTokRankingList() iTikTokRankingList {
	return &localService
}

// rankingRecordKey identifies a product within a single task snapshot. DOM-only
// results may not have a product ID, so their stable rank and name form the key.
type rankingRecordKey struct {
	productID   string
	rank        int
	productName string
}

func rankingKey(productID string, rank int, productName string) rankingRecordKey {
	if productID = strings.TrimSpace(productID); productID != "" {
		return rankingRecordKey{productID: productID}
	}
	return rankingRecordKey{rank: rank, productName: strings.TrimSpace(productName)}
}

func prepareRankingBatches(data []*model.TikTokRankingListInput) (map[string][]*model.TikTokRankingListInput, error) {
	batches := make(map[string][]*model.TikTokRankingListInput)
	seen := make(map[string]map[rankingRecordKey]bool)
	for index, input := range data {
		if input == nil {
			return nil, fmt.Errorf("第 %d 条榜单记录为空", index+1)
		}
		record := *input
		record.TaskId = strings.TrimSpace(record.TaskId)
		record.ProductId = strings.TrimSpace(record.ProductId)
		if record.TaskId == "" {
			return nil, fmt.Errorf("第 %d 条榜单记录缺少任务 ID", index+1)
		}
		key := rankingKey(record.ProductId, record.Rank, record.ProductName)
		if key.productID == "" && (key.rank <= 0 || key.productName == "") {
			return nil, fmt.Errorf("第 %d 条榜单记录缺少可用于重放去重的商品 ID 或排名与名称", index+1)
		}
		if seen[record.TaskId] == nil {
			seen[record.TaskId] = make(map[rankingRecordKey]bool)
		}
		if seen[record.TaskId][key] {
			continue
		}
		seen[record.TaskId][key] = true
		batches[record.TaskId] = append(batches[record.TaskId], &record)
	}
	return batches, nil
}

// missingRankingRecords is shared by each task-scoped replay transaction.
func missingRankingRecords(records []*model.TikTokRankingListInput, existing gdb.Result) []*model.TikTokRankingListInput {
	seen := make(map[rankingRecordKey]bool, len(existing))
	for _, row := range existing {
		seen[rankingKey(row["product_id"].String(), row["rank"].Int(), row["product_name"].String())] = true
	}
	pending := make([]*model.TikTokRankingListInput, 0, len(records))
	for _, record := range records {
		key := rankingKey(record.ProductId, record.Rank, record.ProductName)
		if !seen[key] {
			pending = append(pending, record)
			seen[key] = true
		}
	}
	return pending
}

// CreateBatch saves only products not already present in the task snapshot.
// Replaying a committed page must preserve its original rows and detail fields.
func (s *sTikTokRankingList) CreateBatch(ctx context.Context, data []*model.TikTokRankingListInput) error {
	if len(data) == 0 {
		return nil
	}
	batches, err := prepareRankingBatches(data)
	if err != nil {
		return err
	}
	taskIDs := make([]string, 0, len(batches))
	for taskID := range batches {
		taskIDs = append(taskIDs, taskID)
	}
	sort.Strings(taskIDs)

	// Existing installations have no task/product unique key. Explicitly use
	// SERIALIZABLE and a locking range read so concurrent replay cannot both
	// observe missing rows and insert duplicates, even under READ COMMITTED defaults.
	db := g.DB()
	return db.TransactionWithOptions(ctx, gdb.TxOptions{Isolation: sql.LevelSerializable}, func(ctx context.Context, tx gdb.TX) error {
		for _, taskID := range taskIDs {
			existing, err := tx.Model("tiktok_ranking_list").Ctx(ctx).
				Where("task_id", taskID).Fields("product_id", "rank", "product_name").LockUpdate().All()
			if err != nil {
				return fmt.Errorf("读取任务 %s 已保存榜单失败：%w", taskID, err)
			}
			pending := missingRankingRecords(batches[taskID], existing)
			if len(pending) == 0 {
				continue
			}
			if _, err := tx.Model("tiktok_ranking_list").Ctx(ctx).Insert(pending); err != nil {
				return fmt.Errorf("保存任务 %s 榜单失败：%w", taskID, err)
			}
		}
		return nil
	})
}

func rankingDetailUpdate(data *model.TikTokRankingListInput) (g.Map, g.Map, error) {
	if data == nil || strings.TrimSpace(data.TaskId) == "" || strings.TrimSpace(data.ProductId) == "" {
		return nil, nil, fmt.Errorf("更新商品详情必须同时提供任务 ID 和商品 ID")
	}
	fields := make(g.Map)
	for field, value := range map[string]string{
		"first_shelf_time":     data.FirstShelfTime,
		"min_price":            data.MinPrice,
		"price_range":          data.PriceRange,
		"detail_access_status": data.DetailAccessStatus,
		"detail_access_note":   data.DetailAccessNote,
	} {
		if value = strings.TrimSpace(value); value != "" {
			fields[field] = value
		}
	}
	if data.HasShippingInsurance != nil {
		fields["has_shipping_insurance"] = *data.HasShippingInsurance
	}
	return g.Map{"task_id": strings.TrimSpace(data.TaskId), "product_id": strings.TrimSpace(data.ProductId)}, fields, nil
}

// UpdateByProductId enriches only the specified task's product. Missing detail
// values are unknown, so they must not clear values captured earlier.
func (s *sTikTokRankingList) UpdateByProductId(ctx context.Context, data *model.TikTokRankingListInput) error {
	where, fields, err := rankingDetailUpdate(data)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return nil
	}
	result, err := g.DB().Model("tiktok_ranking_list").Ctx(ctx).Where(where).Update(fields)
	if err != nil {
		return fmt.Errorf("更新任务 %s 商品 %s 详情失败：%w", data.TaskId, data.ProductId, err)
	}
	if affected, err := result.RowsAffected(); err == nil {
		logger.GetSQLLogger().Info("UpdateByProductId - affected rows: %d, TaskId: %s, ProductId: %s", affected, data.TaskId, data.ProductId)
	}
	return nil
}
