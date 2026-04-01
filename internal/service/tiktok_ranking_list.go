package service

import (
	"context"
	"fmt"
	"strings"

	"go-crawler/internal/logger"
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// iTikTokRankingList 定义了抖音排行榜列表服务的接口。
type iTikTokRankingList interface {
	CreateBatch(ctx context.Context, data []*model.TikTokRankingListInput) error
}

// sTikTokRankingList 抖音排行榜列表服务的具体实现结构体。
type sTikTokRankingList struct{}

// 初始化时创建并注册一个全局的 sTikTokRankingList 实例。
var localService = sTikTokRankingList{}

// TikTokRankingList 返回一个抖音排行榜列表服务的实例。
func TikTokRankingList() iTikTokRankingList {
	return &localService
}

// CreateBatch 执行批量插入操作到 tiktok_ranking_list 表。
func (s *sTikTokRankingList) CreateBatch(ctx context.Context, data []*model.TikTokRankingListInput) error {
	sqlLogger := logger.GetSQLLogger()

	// 如果没有数据，直接返回
	if len(data) == 0 {
		sqlLogger.Error(fmt.Errorf("没有数据"), "INSERT INTO tiktok_ranking_list")
		fmt.Printf("❌ [数据库] 没有数据需要插入\n")
		return nil
	}

	fmt.Printf("📊 [数据库] 开始批量插入 %d 条数据...\n", len(data))
	sqlLogger.Info("BEGIN TRANSACTION - 准备插入 %d 条数据到 tiktok_ranking_list 表", len(data))

	// 获取默认数据库连接
	db := g.DB()
	if db == nil {
		sqlLogger.Error(fmt.Errorf("数据库连接为空"), "g.DB()")
		return fmt.Errorf("数据库连接为空")
	}
	fmt.Printf("✅ [数据库] 获取数据库连接成功\n")
	sqlLogger.Info("Database connection: SUCCESS")

	// 指定要操作的实体表
	entity := (*model.TiktokRankingList)(nil)

	// 使用事务来保证数据一致性
	return db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		fmt.Printf("📝 [数据库] 开始执行批量插入...\n")
		sqlLogger.Info("EXECUTE INSERT - 表：tiktok_ranking_list, 数据量：%d", len(data))

		// 打印第一条数据用于调试
		if len(data) > 0 {
			firstRecord := data[0]
			sqlLogger.Info("Sample data: TaskId=%s, Rank=%d, ProductName=%s",
				firstRecord.TaskId, firstRecord.Rank, firstRecord.ProductName)
		}

		// 执行批量插入
		result, err := tx.Model(entity).Ctx(ctx).Insert(data)
		if err != nil {
			sqlLogger.Error(err, "tx.Model(entity).Insert(data) - 插入失败")
			fmt.Printf("❌ [数据库] 批量插入失败：%v\n", err)

			// 记录详细的错误信息
			errMsg := err.Error()
			if strings.Contains(errMsg, "Error 1064") {
				sqlLogger.Error(err, "SQL 语法错误 - 可能是表结构或字段名问题")
			} else if strings.Contains(errMsg, "Error 1054") {
				sqlLogger.Error(err, "未知列名 - 检查字段名是否匹配")
			} else if strings.Contains(errMsg, "Error 1146") {
				sqlLogger.Error(err, "表不存在 - 检查表名是否正确")
			}

			g.Log().Error(ctx, "批量插入失败:", err)
			return err
		}

		// 获取插入的行数
		affected, _ := result.RowsAffected()
		sqlLogger.Info("INSERT SUCCESS - affected rows: %d", affected)
		fmt.Printf("✅ [数据库] 成功插入 %d 条记录 (affected: %d)\n", len(data), affected)
		g.Log().Info(ctx, fmt.Sprintf("成功插入 %d 条记录。", len(data)))
		sqlLogger.Info("COMMIT TRANSACTION - 成功提交")
		return nil
	})
}
