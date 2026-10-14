package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/model"
	"go-crawler/internal/service"
)

// getProductsFromDB 从数据库获取已抓取的商品列表
func (s *sCrawler) getProductsFromDB(ctx context.Context) ([]ProductInfo, error) {
	// 只读取最新一次榜单快照，避免把多个任务的重复商品混成“最新 200 条”。
	latest, err := g.DB().Model("tiktok_ranking_list").
		Fields("task_id").
		WhereNot("task_id", "").
		OrderDesc("created_at").
		One()
	if err != nil {
		return nil, err
	}
	if latest.IsEmpty() {
		return nil, nil
	}

	taskId := latest["task_id"].String()
	result, err := g.DB().Model("tiktok_ranking_list").
		Where("task_id", taskId).
		OrderAsc("rank").
		Limit(200).
		All()
	if err != nil {
		return nil, err
	}

	var products []ProductInfo
	for _, row := range result {
		product := ProductInfo{
			ProductID:   row["product_id"].String(),
			ProductName: row["product_name"].String(),
			ShopID:      row["shop_id"].String(),
			ShopName:    row["shop_name"].String(),
			ProductURL:  row["product_url"].String(),
		}
		if product.ProductID != "" {
			products = append(products, product)
		}
	}

	return products, nil
}

// deduplicateVideos 基于 video_id 去重（借鉴 Python 爬虫的 _clean_videos_against_previous）
func deduplicateVideos(videos []VideoAnalysis) []VideoAnalysis {
	seen := make(map[string]bool)
	var result []VideoAnalysis

	for _, v := range videos {
		if v.VideoId == "" {
			continue
		}
		if !seen[v.VideoId] {
			seen[v.VideoId] = true
			result = append(result, v)
		}
	}

	return result
}

// saveVideoAnalysis 保存视频分析数据到数据库
func (s *sCrawler) saveVideoAnalysis(ctx context.Context, taskId string, videos []VideoAnalysis) error {
	initialStatus := "discovered"
	if g.Cfg().MustGet(ctx, "crawler.video.autoDownload", true).Bool() {
		initialStatus = "pending"
	}

	videos = deduplicateVideos(videos)
	if len(videos) == 0 {
		return nil
	}
	ids := make([]string, 0, len(videos))
	for _, video := range videos {
		ids = append(ids, video.VideoId)
	}
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		existing, err := tx.Model("video_analysis_data").Ctx(ctx).Where("task_id", taskId).WhereIn("video_id", ids).Fields("video_id").All()
		if err != nil {
			return fmt.Errorf("读取已保存视频失败：%w", err)
		}
		seen := make(map[string]bool, len(existing))
		for _, row := range existing {
			seen[row["video_id"].String()] = true
		}
		rows := make([]g.Map, 0, len(videos))
		for _, video := range videos {
			if seen[video.VideoId] {
				continue
			}
			rows = append(rows, g.Map{
				"task_id":             taskId,
				"product_id":          video.ProductID,
				"product_name":        video.ProductName,
				"video_id":            video.VideoId,
				"video_desc":          video.VideoDesc,
				"author_name":         video.AuthorName,
				"cover_url":           video.CoverURL,
				"publish_time":        video.PublishTime,
				"source_url":          video.SourceURL,
				"download_status":     initialStatus,
				"download_url":        "",
				"resolve_strategy":    "",
				"payment_amount":      video.PaymentAmount,
				"expose_user_count":   video.ExposeUserCount,
				"click_user_count":    video.ClickUserCount,
				"convert_user_count":  video.ConvertUserCount,
				"click_rate":          video.ClickRate,
				"click_convert_rate":  video.ClickConvertRate,
				"expose_convert_rate": video.ExposeConvertRate,
				"order_count":         video.OrderCount,
				"sales_volume":        video.SalesVolume,
				"created_at":          time.Now(),
			})
		}
		if len(rows) == 0 {
			return nil
		}
		if _, err := tx.Model("video_analysis_data").Ctx(ctx).Data(rows).Insert(); err != nil {
			return fmt.Errorf("保存视频批次失败：%w", err)
		}
		return nil
	})
}

func (s *sCrawler) loadProductsFromDB(ctx context.Context, taskId string) ([]ProductInfo, error) {
	rows, err := g.DB().Model("tiktok_ranking_list").
		Ctx(ctx).
		Where("task_id", taskId).
		OrderAsc("rank").
		All()
	if err != nil {
		return nil, err
	}
	products := make([]ProductInfo, 0, len(rows))
	for _, row := range rows {
		products = append(products, ProductInfo{
			Rank:             row["rank"].Int(),
			ProductID:        row["product_id"].String(),
			ShopID:           row["shop_id"].String(),
			ProductURL:       row["product_url"].String(),
			ProductName:      row["product_name"].String(),
			ShopName:         row["shop_name"].String(),
			PaymentAmount:    row["payment_amount"].String(),
			ClickCount:       row["click_count"].String(),
			ClickConvertRate: row["click_convert_rate"].String(),
			SalesVolume:      row["sales_volume"].String(),
			DeliveryVideoNum: row["delivery_video_num"].String(),
			ProductQRCodeURL: row["product_qr_code_url"].String(),
			ShopLogoURL:      row["shop_logo_url"].String(),
			ShopQRCodeURL:    row["shop_qr_code_url"].String(),
			PriceRange:       row["price_range"].String(),
			FirstList:        row["first_list"].Int(),
		})
	}
	return products, nil
}

// countVideoRowsChecked must be used when persisting progress or deciding
// completion; a failed query must not become a successfully checkpointed zero.
func (s *sCrawler) countVideoRowsChecked(ctx context.Context, taskId string) (int, error) {
	count, err := g.DB().Model("video_analysis_data").Ctx(ctx).Where("task_id", taskId).Count()
	if err != nil {
		return 0, fmt.Errorf("统计任务 %s 已保存视频失败：%w", taskId, err)
	}
	return count, nil
}

func (s *sCrawler) updateCrawlTaskSummary(ctx context.Context, taskId, status string, exportFiles map[string]string) error {
	productCount, err := g.DB().Model("tiktok_ranking_list").Ctx(ctx).Where("task_id", taskId).Count()
	if err != nil {
		return fmt.Errorf("统计任务 %s 商品失败：%w", taskId, err)
	}
	videoCount, err := s.countVideoRowsChecked(ctx, taskId)
	if err != nil {
		return err
	}
	data := g.Map{
		"product_count": productCount,
		"video_count":   videoCount,
	}
	if status != "" {
		data["status"] = status
	}
	if status == "completed" || status == "paused" || status == "failed" || status == "stopped" {
		data["end_time"] = time.Now()
	}
	if len(exportFiles) > 0 {
		encoded, err := json.Marshal(exportFiles)
		if err != nil {
			return fmt.Errorf("序列化任务导出文件失败：%w", err)
		}
		data["export_files"] = string(encoded)
	}
	_, err = g.DB().Model("crawl_tasks").Ctx(ctx).Where("task_id", taskId).Update(data)
	return err
}

// saveAllData enriches the current task's saved ranking snapshot with details
// actually collected in this run; skipped or empty details leave existing data intact.
func (s *sCrawler) saveAllData(ctx context.Context, taskId string, products []ProductInfo, details []ProductDetail, videos []VideoAnalysis) error {
	taskId = strings.TrimSpace(taskId)
	if taskId == "" {
		return fmt.Errorf("保存采集结果缺少任务 ID")
	}
	for _, record := range rankingDetailUpdates(taskId, products, details) {
		if err := service.TikTokRankingList().UpdateByProductId(ctx, record); err != nil {
			return fmt.Errorf("更新商品详情失败：%w", err)
		}
	}
	// Use the same task-scoped idempotent path as the streaming video collector.
	return s.saveVideoAnalysis(ctx, taskId, videos)
}

func rankingDetailUpdates(taskId string, products []ProductInfo, details []ProductDetail) []*model.TikTokRankingListInput {
	productNames := make(map[string]string, len(products))
	for _, product := range products {
		if id := strings.TrimSpace(product.ProductID); id != "" {
			productNames[id] = product.ProductName
		}
	}
	records := make([]*model.TikTokRankingListInput, 0, len(details))
	for _, detail := range details {
		productID := strings.TrimSpace(detail.ProductID)
		productName, ok := productNames[productID]
		if !ok {
			continue
		}
		record := &model.TikTokRankingListInput{
			TaskId:             strings.TrimSpace(taskId),
			ProductId:          productID,
			ProductName:        productName,
			FirstShelfTime:     detail.FirstShelfTime,
			MinPrice:           detail.MinPrice,
			PriceRange:         detail.PriceRange,
			DetailAccessStatus: detail.DetailAccessStatus,
			DetailAccessNote:   detail.DetailAccessNote,
		}
		if detail.HasShippingInsurance != nil {
			insurance := 0
			if *detail.HasShippingInsurance {
				insurance = 1
			}
			record.HasShippingInsurance = &insurance
		}
		records = append(records, record)
	}
	return records
}

// convertToModelProducts 转换 ProductInfo 为 model.TikTokRankingListInput（按 product_id 关联详情）
func (s *sCrawler) convertToModelProducts(products []ProductInfo, details []ProductDetail) []model.TikTokRankingListInput {
	// 构建详情数据映射：product_id -> detail
	detailMap := make(map[string]*ProductDetail)
	for i := range details {
		if details[i].ProductID != "" {
			d := details[i]
			detailMap[details[i].ProductID] = &d
		}
	}

	var result []model.TikTokRankingListInput
	for _, p := range products {
		record := model.TikTokRankingListInput{
			ProductId:        p.ProductID,
			ShopId:           p.ShopID,
			ProductUrl:       p.ProductURL,
			Rank:             p.Rank,
			ProductName:      p.ProductName,
			ImageUrl:         p.ImageURL,
			ShopName:         p.ShopName,
			PaymentAmount:    p.PaymentAmount,
			ClickCount:       p.ClickCount,
			ClickConvertRate: p.ClickConvertRate,
			SalesVolume:      p.SalesVolume,
			DeliveryVideoNum: p.DeliveryVideoNum,
			ProductQRCodeUrl: p.ProductQRCodeURL,
			ShopLogoUrl:      p.ShopLogoURL,
			ShopQRCodeUrl:    p.ShopQRCodeURL,
			PreviousRank:     p.PreviousRank,
			RankDelta:        p.RankDelta,
			RankTrendSource:  p.RankTrendSource,
			FirstList:        p.FirstList,
			CreatedAt:        time.Now(),
		}
		// 按 product_id 匹配详情数据
		if detail, ok := detailMap[p.ProductID]; ok {
			record.FirstShelfTime = detail.FirstShelfTime
			record.MinPrice = detail.MinPrice
			record.DetailAccessStatus = detail.DetailAccessStatus
			record.DetailAccessNote = detail.DetailAccessNote
			if detail.HasShippingInsurance != nil {
				val := 0
				if *detail.HasShippingInsurance {
					val = 1
				}
				record.HasShippingInsurance = &val
			}
		}
		result = append(result, record)
	}
	return result
}

func (s *sCrawler) convertToModelVideos(taskId string, videos []VideoAnalysis) []model.VideoAnalysisData {
	result := make([]model.VideoAnalysisData, 0, len(videos))
	for _, v := range videos {
		if v.VideoId == "" && v.VideoDesc == "" {
			continue
		}
		result = append(result, model.VideoAnalysisData{
			TaskId: taskId, ProductId: v.ProductID, ProductName: v.ProductName,
			VideoId: v.VideoId, VideoDesc: v.VideoDesc, AuthorName: v.AuthorName,
			CoverURL: v.CoverURL, PublishTime: v.PublishTime, SourceURL: v.SourceURL,
			DownloadStatus: "pending", PaymentAmount: v.PaymentAmount,
			ExposeUserCount: v.ExposeUserCount, ClickUserCount: v.ClickUserCount,
			ConvertUserCount: v.ConvertUserCount, ClickRate: v.ClickRate,
			ClickConvertRate: v.ClickConvertRate, ExposeConvertRate: v.ExposeConvertRate,
			OrderCount: v.OrderCount, SalesVolume: v.SalesVolume, CreatedAt: time.Now(),
		})
	}
	return result
}

// savePageData 保存单页数据到数据库
func (s *sCrawler) savePageData(ctx context.Context, taskId string, products []ProductInfo, pageNum int) error {
	if len(products) == 0 {
		return nil
	}

	// 统计价格带信息
	priceRangeCount := 0
	for _, p := range products {
		if p.PriceRange != "" {
			priceRangeCount++
		}
	}

	// 从缓存获取详情数据
	cache := service.GetProductDetailCache()
	previousTaskId := resolvePreviousRankingTaskIdForInsert(ctx, taskId)
	previousRankCache := make(map[string]*int)

	// 转换为数据库模型
	var rankingList []*model.TikTokRankingListInput
	for _, p := range products {
		enrichProductRankTrend(ctx, previousTaskId, previousRankCache, &p)
		record := &model.TikTokRankingListInput{
			TaskId:           taskId,
			ProductId:        p.ProductID,
			ShopId:           p.ShopID,
			ProductUrl:       p.ProductURL,
			Rank:             p.Rank,
			ProductName:      p.ProductName,
			ImageUrl:         p.ImageURL,
			ShopName:         p.ShopName,
			PaymentAmount:    p.PaymentAmount,
			ClickCount:       p.ClickCount,
			ClickConvertRate: p.ClickConvertRate,
			SalesVolume:      p.SalesVolume,
			DeliveryVideoNum: p.DeliveryVideoNum,
			ProductQRCodeUrl: p.ProductQRCodeURL,
			ShopLogoUrl:      p.ShopLogoURL,
			ShopQRCodeUrl:    p.ShopQRCodeURL,
			PreviousRank:     p.PreviousRank,
			RankDelta:        p.RankDelta,
			RankTrendSource:  p.RankTrendSource,
			FirstList:        p.FirstList,
			PriceRange:       p.PriceRange,
			CreatedAt:        time.Now(),
		}

		// 从缓存获取详情数据
		if cachedRecord := cache.GetRecord(p.ProductID); cachedRecord != nil && !cache.IsRecordStale(cachedRecord) {
			record.FirstShelfTime = cachedRecord.FirstShelfTime
			record.MinPrice = cachedRecord.MinPrice
			logInfo(ctx, taskId, fmt.Sprintf("✅ 从缓存获取商品详情数据：%s (ProductID: %s)", p.ProductName, p.ProductID))
		} else {
			record.FirstShelfTime = ""
			record.MinPrice = ""
		}

		rankingList = append(rankingList, record)
	}

	// 批量插入
	if err := service.TikTokRankingList().CreateBatch(ctx, rankingList); err != nil {
		return fmt.Errorf("保存第 %d 页榜单失败：%w", pageNum, err)
	} else {
		logInfo(ctx, taskId, fmt.Sprintf("💾 第 %d 页数据已保存：%d 条 (其中 %d 条包含价格带)", pageNum, len(rankingList), priceRangeCount))
	}
	return nil
}

func resolvePreviousRankingTaskIdForInsert(ctx context.Context, currentTaskId string) string {
	value, err := g.DB().GetValue(ctx, `
		SELECT task_id
		FROM (
			SELECT task_id, MAX(created_at) AS latest_created_at, COUNT(*) AS product_count
			FROM tiktok_ranking_list
			WHERE task_id <> ''
			  AND CONVERT(task_id USING utf8mb4) COLLATE utf8mb4_unicode_ci <>
			      CONVERT(? USING utf8mb4) COLLATE utf8mb4_unicode_ci
			GROUP BY task_id
		) cycles
		ORDER BY
			CASE WHEN product_count >= 200 THEN 0 ELSE 1 END,
			latest_created_at DESC
		LIMIT 1
	`, currentTaskId)
	if err != nil || value.IsNil() {
		return ""
	}
	return value.String()
}

func enrichProductRankTrend(ctx context.Context, previousTaskId string, cache map[string]*int, product *ProductInfo) {
	if product == nil {
		return
	}
	if product.RankDelta != nil {
		if product.PreviousRank == nil && product.Rank > 0 {
			previous := product.Rank + *product.RankDelta
			if previous > 0 {
				product.PreviousRank = &previous
			}
		}
		if product.RankTrendSource == "" {
			product.RankTrendSource = "official"
		}
		return
	}
	if previousTaskId == "" || product.Rank <= 0 {
		product.RankTrendSource = "unknown"
		return
	}
	cacheKey := firstNonEmptyString(product.ProductID, product.ProductName)
	var previousRank *int
	if cached, ok := cache[cacheKey]; ok {
		previousRank = cached
	} else {
		previousRank = getPreviousProductRankForInsert(ctx, previousTaskId, product.ProductID, product.ProductName)
		cache[cacheKey] = previousRank
	}
	if previousRank == nil {
		product.RankTrendSource = "unknown"
		return
	}
	delta := *previousRank - product.Rank
	product.PreviousRank = previousRank
	product.RankDelta = &delta
	product.RankTrendSource = "fallback"
}

func getPreviousProductRankForInsert(ctx context.Context, previousTaskId, productId, productName string) *int {
	if previousTaskId == "" || (productId == "" && productName == "") {
		return nil
	}
	value, err := g.DB().GetValue(ctx, `
		SELECT `+"`rank`"+`
		FROM tiktok_ranking_list
		WHERE task_id = ?
		  AND (
			(? <> '' AND product_id = ?)
			OR (? <> '' AND product_name = ?)
		  )
		ORDER BY
			CASE WHEN ? <> '' AND product_id = ? THEN 0 ELSE 1 END,
			`+"`rank`"+` ASC,
			id ASC
		LIMIT 1
	`, previousTaskId, productId, productId, productName, productName, productId, productId)
	if err != nil || value.IsNil() {
		return nil
	}
	rank := value.Int()
	return &rank
}
