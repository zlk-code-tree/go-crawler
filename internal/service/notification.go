package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/controller/ws"
)

// NotifyLevel 消息级别
const (
	NotifyLevelNormal    = 0 // 普通
	NotifyLevelImportant = 1 // 重要
	NotifyLevelUrgent    = 2 // 紧急
)

// CreateNotification 创建一条系统消息，并可选择推送给指定用户
// userId: 接收消息的用户ID（用于WebSocket实时推送，0表示不推送）
func CreateNotification(ctx context.Context, title, content, notifyType, sourceType string, sourceID int64, level int, userId int64) (int64, error) {
	result, err := g.DB().Ctx(ctx).Model("system_notification").Data(g.Map{
		"title":       title,
		"content":     content,
		"type":        notifyType,
		"source_type": sourceType,
		"source_id":   sourceID,
		"level":       level,
		"is_read":     0,
	}).Insert()
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()

	// WebSocket 实时推送给对应用户
	if userId > 0 {
		payload, _ := json.Marshal(g.Map{
			"type":    "notification",
			"id":      id,
			"title":   title,
			"message": content,
			"level":   level,
		})
		ws.BroadcastNotification(userId, string(payload))
	}

	return id, nil
}

// GetUnreadCount 获取未读消息数
func GetUnreadCount(ctx context.Context) (int, error) {
	count, err := g.DB().Ctx(ctx).Model("system_notification").Where("is_read", 0).Count()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetNotificationList 获取消息列表（分页）
func GetNotificationList(ctx context.Context, notifyType string, page, pageSize int) ([]g.Map, int, error) {
	m := g.DB().Ctx(ctx).Model("system_notification")
	if notifyType != "" {
		m = m.Where("type", notifyType)
	}

	total, err := m.Count()
	if err != nil {
		return nil, 0, err
	}

	rows, err := m.Order("id DESC").Page(page, pageSize).All()
	if err != nil {
		return nil, 0, err
	}

	var list []g.Map
	for _, row := range rows {
		list = append(list, row.Map())
	}
	if list == nil {
		list = []g.Map{}
	}

	return list, total, nil
}

// MarkNotificationRead 标记单条消息为已读
func MarkNotificationRead(ctx context.Context, id int64) error {
	_, err := g.DB().Ctx(ctx).Model("system_notification").
		Where("id", id).
		Data(g.Map{"is_read": 1, "read_at": time.Now().Format("2006-01-02 15:04:05")}).
		Update()
	return err
}

// MarkAllRead 全部标记为已读
func MarkAllRead(ctx context.Context) error {
	_, err := g.DB().Ctx(ctx).Model("system_notification").
		Where("is_read", 0).
		Data(g.Map{"is_read": 1, "read_at": time.Now().Format("2006-01-02 15:04:05")}).
		Update()
	return err
}

// GetNotificationDetail 获取消息详情
func GetNotificationDetail(ctx context.Context, id int64) (g.Map, error) {
	row, err := g.DB().Ctx(ctx).Model("system_notification").Where("id", id).One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, nil
	}
	return row.Map(), nil
}
