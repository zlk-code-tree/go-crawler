package data

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	v1 "go-crawler/api/hello/v1"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

func recordInt(row gdb.Record, key string) (int, bool) {
	value, ok := row[key]
	if !ok || value == nil || value.IsNil() {
		return 0, false
	}
	return value.Int(), true
}

// Overview GET /api/data/overview
func (c *Controller) Overview(ctx context.Context, req *v1.DataOverviewReq) (res *v1.DataOverviewRes, err error) {
	r := g.RequestFromCtx(ctx)
	taskTotal, _ := g.DB().Model("crawl_tasks").Count()
	taskRunning, _ := g.DB().Model("crawl_tasks").Where("status", "running").Count()
	taskCompleted, _ := g.DB().Model("crawl_tasks").Where("status", "completed").Count()
	rankingTotal, _ := g.DB().Model("tiktok_ranking_list").Count()
	videoTotal, _ := g.DB().Model("video_analysis").Count()
	trendRows, _ := g.DB().GetAll(ctx, `SELECT DATE(created_at) as date, COUNT(*) as count FROM crawl_tasks WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) GROUP BY DATE(created_at) ORDER BY date ASC`)
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"task_total":     taskTotal,
			"task_running":   taskRunning,
			"task_completed": taskCompleted,
			"ranking_total":  rankingTotal,
			"video_total":    videoTotal,
			"task_trend":     trendRows,
		},
	})
	return
}

// RankingList GET /api/data/ranking
func (c *Controller) RankingList(ctx context.Context, req *v1.DataRankingListReq) (res *v1.DataRankingListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	effectiveTaskId, taskErr := resolveRankingTaskId(ctx, req.TaskId)
	if taskErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": taskErr.Error(), "data": nil})
		return
	}
	previousTaskId := ""
	if effectiveTaskId != "" {
		previousTaskId, _ = resolvePreviousRankingTaskId(ctx, effectiveTaskId)
	}

	m := g.DB().Model("tiktok_ranking_list")
	if effectiveTaskId != "" {
		m = m.Where("task_id", effectiveTaskId)
	}
	if req.Keyword != "" {
		m = m.WhereLike("product_name", "%"+req.Keyword+"%")
	}
	if req.FirstList == "1" {
		m = m.Where("first_list", 1)
	}

	total, e := m.Count()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	list, e := m.OrderAsc("rank").OrderAsc("id").Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	enriched := make([]g.Map, 0, len(list))
	for _, row := range list {
		item := g.Map{}
		for key, value := range row {
			item[key] = value.Val()
		}

		productId := row["product_id"].String()
		productName := row["product_name"].String()
		currentRank := row["rank"].Int()
		storedPreviousRank, hasStoredPreviousRank := recordInt(row, "previous_rank")
		storedRankDelta, hasStoredRankDelta := recordInt(row, "rank_delta")
		onListCount := countProductRankingCycles(ctx, productId, productName)

		if hasStoredRankDelta {
			if hasStoredPreviousRank {
				item["previous_rank"] = storedPreviousRank
			} else {
				item["previous_rank"] = nil
			}
			item["rank_delta"] = storedRankDelta
			item["is_new_list"] = false
		} else if previousRank, hasPreviousRank := getPreviousProductRank(ctx, previousTaskId, productId, productName); hasPreviousRank {
			item["previous_rank"] = previousRank
			item["rank_delta"] = previousRank - currentRank
			item["is_new_list"] = false
		} else {
			item["previous_rank"] = nil
			item["rank_delta"] = nil
			item["is_new_list"] = true
		}
		item["on_list_count"] = onListCount
		enriched = append(enriched, item)
	}
	r.Response.WriteJsonExit(g.Map{
		"code":             0,
		"msg":              "ok",
		"data":             enriched,
		"total":            total,
		"page":             page,
		"pageSize":         pageSize,
		"current_task_id":  effectiveTaskId,
		"previous_task_id": previousTaskId,
	})
	return
}

func resolveRankingTaskId(ctx context.Context, requestedTaskId string) (string, error) {
	if requestedTaskId != "" {
		return requestedTaskId, nil
	}
	value, err := g.DB().GetValue(ctx, `
		SELECT task_id
		FROM (
			SELECT
				r.task_id,
				MAX(r.created_at) AS latest_created_at,
				MAX(t.status) AS status,
				COUNT(*) AS product_count
			FROM tiktok_ranking_list r
			LEFT JOIN crawl_tasks t
				ON CONVERT(r.task_id USING utf8mb4) COLLATE utf8mb4_unicode_ci =
				   CONVERT(t.task_id USING utf8mb4) COLLATE utf8mb4_unicode_ci
			WHERE r.task_id <> ''
			GROUP BY r.task_id
		) cycles
		ORDER BY
			CASE WHEN status = 'completed' AND product_count >= 200 THEN 0 ELSE 1 END,
			latest_created_at DESC
		LIMIT 1
	`)
	if err != nil || value.IsNil() {
		return "", err
	}
	return value.String(), nil
}

func resolvePreviousRankingTaskId(ctx context.Context, currentTaskId string) (string, error) {
	currentLatest, err := g.DB().GetValue(ctx, `
		SELECT MAX(created_at)
		FROM tiktok_ranking_list
		WHERE CONVERT(task_id USING utf8mb4) COLLATE utf8mb4_unicode_ci =
		      CONVERT(? USING utf8mb4) COLLATE utf8mb4_unicode_ci
	`, currentTaskId)
	if err != nil || currentLatest.IsNil() {
		return "", err
	}
	value, err := g.DB().GetValue(ctx, `
		SELECT task_id
		FROM (
			SELECT task_id, MAX(created_at) AS latest_created_at
			FROM tiktok_ranking_list
			WHERE task_id <> ''
			GROUP BY task_id
		) cycles
		WHERE latest_created_at < ?
		ORDER BY latest_created_at DESC
		LIMIT 1
	`, currentLatest.Val())
	if err != nil || value.IsNil() {
		return "", err
	}
	return value.String(), nil
}

func getPreviousProductRank(ctx context.Context, previousTaskId, productId, productName string) (int, bool) {
	if previousTaskId == "" || (productId == "" && productName == "") {
		return 0, false
	}
	value, err := g.DB().GetValue(ctx, `
		SELECT rank
		FROM tiktok_ranking_list
		WHERE task_id = ?
		  AND (
			(? <> '' AND product_id = ?)
			OR (? <> '' AND product_name = ?)
		  )
		ORDER BY
			CASE WHEN ? <> '' AND product_id = ? THEN 0 ELSE 1 END,
			rank ASC,
			id ASC
		LIMIT 1
	`, previousTaskId, productId, productId, productName, productName, productId, productId)
	if err != nil || value.IsNil() {
		return 0, false
	}
	return value.Int(), true
}

func countProductRankingCycles(ctx context.Context, productId, productName string) int {
	if productId == "" && productName == "" {
		return 0
	}
	value, err := g.DB().GetValue(ctx, `
		SELECT COUNT(DISTINCT task_id)
		FROM tiktok_ranking_list
		WHERE (? <> '' AND product_id = ?)
		   OR (? <> '' AND product_name = ?)
	`, productId, productId, productName, productName)
	if err != nil || value.IsNil() {
		return 0
	}
	return value.Int()
}

// RankingStats GET /api/data/ranking/stats
func (c *Controller) RankingStats(ctx context.Context, req *v1.DataRankingStatsReq) (res *v1.DataRankingStatsRes, err error) {
	r := g.RequestFromCtx(ctx)
	m := g.DB().Model("tiktok_ranking_list")
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	total, _ := m.Count()
	firstListCount, _ := m.Clone().Where("first_list", 1).Count()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"total":            total,
			"first_list_count": firstListCount,
		},
	})
	return
}

// VideoList GET /api/data/video
func (c *Controller) VideoList(ctx context.Context, req *v1.DataVideoListReq) (res *v1.DataVideoListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	m := g.DB().Model("video_analysis").OrderDesc("payment_amount")
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	if req.ProductId != "" {
		m = m.Where("product_id", req.ProductId)
	}
	if req.Keyword != "" {
		m = m.WhereLike("product_name", "%"+req.Keyword+"%")
	}

	total, e := m.Count()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	list, e := m.Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
	return
}

// VideoStats GET /api/data/video/stats
func (c *Controller) VideoStats(ctx context.Context, req *v1.DataVideoStatsReq) (res *v1.DataVideoStatsRes, err error) {
	r := g.RequestFromCtx(ctx)
	m := g.DB().Model("video_analysis")
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	total, _ := m.Count()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"total": total,
		},
	})
	return
}

// Export GET /api/data/export/{taskId}
func (c *Controller) Export(ctx context.Context, req *v1.DataExportReq) (res *v1.DataExportRes, err error) {
	r := g.RequestFromCtx(ctx)
	matches, e := filepath.Glob(fmt.Sprintf("exports/*%s*.csv", req.TaskId))
	if e != nil || len(matches) == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": "export file not found", "data": nil})
		return
	}
	filePath := matches[0]
	fileData, e := os.ReadFile(filePath)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "read file failed", "data": nil})
		return
	}
	r.Response.Header().Set("Content-Type", "text/csv; charset=utf-8")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(filePath)))
	r.Response.WriteHeader(http.StatusOK)
	r.Response.Write(fileData)
	return
}

// TaskIds GET /api/data/task_ids
func (c *Controller) TaskIds(ctx context.Context, req *v1.DataTaskIdsReq) (res *v1.DataTaskIdsRes, err error) {
	r := g.RequestFromCtx(ctx)
	rows, e := g.DB().GetAll(ctx, `
		SELECT
			r.task_id,
			MAX(r.created_at) AS latest_created_at,
			MAX(t.start_time) AS start_time,
			MAX(t.status) AS status,
			COUNT(*) AS product_count
		FROM tiktok_ranking_list r
		LEFT JOIN crawl_tasks t
			ON CONVERT(r.task_id USING utf8mb4) COLLATE utf8mb4_unicode_ci =
			   CONVERT(t.task_id USING utf8mb4) COLLATE utf8mb4_unicode_ci
		WHERE r.task_id <> ''
		GROUP BY r.task_id
		ORDER BY latest_created_at DESC
		LIMIT 50
	`)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": rows,
	})
	return
}
