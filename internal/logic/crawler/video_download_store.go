package crawler

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

type videoTaskRecord struct {
	Status   string
	URL      string
	Strategy string
}

// Keep scheduling independent of SQL so cancellation and recovery can be tested
// without a production database. video_id is the unit of work in one process.
type videoTaskStore interface {
	lookup(context.Context, string) (videoTaskRecord, bool, error)
	mark(context.Context, string, string, int, string) error
	complete(context.Context, string, string, string) error
	pending(context.Context, string, int) ([]string, error)
}

type sqlVideoTaskStore struct{}

func (sqlVideoTaskStore) lookup(ctx context.Context, id string) (videoTaskRecord, bool, error) {
	row, err := g.DB().Model("video_analysis_data").Ctx(ctx).Fields("download_status, download_url, resolve_strategy").Where("video_id", id).
		OrderDesc("created_at").OrderDesc("id").One()
	if err != nil || row.IsEmpty() {
		return videoTaskRecord{}, false, err
	}
	return videoTaskRecord{Status: row["download_status"].String(), URL: row["download_url"].String(), Strategy: row["resolve_strategy"].String()}, true, nil
}

func (sqlVideoTaskStore) mark(ctx context.Context, id, status string, retries int, message string) error {
	updates := g.Map{"download_status": status, "last_error": message}
	if retries >= 0 {
		updates["retry_count"] = retries
	}
	// A delayed cancellation or failure must not overwrite a successful result.
	_, err := g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", id).
		WhereNot("download_status", "resolved").Update(updates)
	return err
}

func (sqlVideoTaskStore) complete(ctx context.Context, id, url, strategy string) error {
	_, err := g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", id).Update(g.Map{
		"download_status": "resolved", "download_url": url, "resolve_strategy": strategy,
		"storage_path": "", "file_size": 0, "last_error": "",
	})
	return err
}

func (sqlVideoTaskStore) pending(ctx context.Context, cursor string, limit int) ([]string, error) {
	query := g.DB().Model("video_analysis_data").Ctx(ctx).Fields("video_id").
		WhereIn("download_status", []string{"pending", "queued", "resolving", "downloading"}).
		WhereNot("video_id", "")
	if cursor != "" {
		query = query.WhereGT("video_id", cursor)
	}
	rows, err := query.Group("video_id").OrderAsc("video_id").Limit(limit).All()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row["video_id"].String())
	}
	return ids, nil
}
