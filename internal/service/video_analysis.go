package service

import (
	"context"
	"fmt"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// IVideoAnalysis 视频分析服务接口
type IVideoAnalysis interface {
	CreateBatch(ctx context.Context, data []*model.VideoAnalysisData) error
}

// sVideoAnalysis 视频分析服务实现
type sVideoAnalysis struct{}

var localVideoAnalysisService = sVideoAnalysis{}

// VideoAnalysis 返回视频分析服务实例
func VideoAnalysis() IVideoAnalysis {
	return &localVideoAnalysisService
}

// CreateBatch 批量创建视频分析数据
func (s *sVideoAnalysis) CreateBatch(ctx context.Context, data []*model.VideoAnalysisData) error {
	if len(data) == 0 {
		return nil
	}

	db := g.DB()
	return db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 表名是 video_analysis_data，不能依赖 VideoAnalysis 结构体自动推导，
		// 否则会误写到不存在的 video_analysis 表。
		_, err := tx.Model("video_analysis_data").Ctx(ctx).Insert(data)
		if err != nil {
			g.Log().Error(ctx, "批量插入视频分析数据失败:", err)
			return err
		}
		g.Log().Info(ctx, fmt.Sprintf("成功插入 %d 条视频分析记录。", len(data)))
		return nil
	})
}
