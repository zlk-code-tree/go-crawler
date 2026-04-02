package ws

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

type taskLogHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

var (
	hubsMu sync.Mutex
	hubs   = make(map[string]*taskLogHub)
)

func getHub(taskId string) *taskLogHub {
	hubsMu.Lock()
	defer hubsMu.Unlock()
	if h, ok := hubs[taskId]; ok {
		return h
	}
	h := &taskLogHub{clients: make(map[*websocket.Conn]bool)}
	hubs[taskId] = h
	return h
}

// BroadcastLog broadcasts a log message to all WebSocket clients of a task.
func BroadcastLog(taskId, level, message string) {
	h := getHub(taskId)
	h.mu.Lock()
	defer h.mu.Unlock()
	payload := fmt.Sprintf(`{"time":"%s","level":"%s","msg":"%s"}`,
		time.Now().Format("15:04:05"), level, message)
	for conn := range h.clients {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(payload))
	}
}

// Controller is the WebSocket controller.
type Controller struct{}

// New returns a Controller.
func New() *Controller { return &Controller{} }

// Log handles WS /api/ws/log/{taskId} - realtime log stream.
func (c *Controller) Log(r *ghttp.Request) {
	taskId := r.GetRouter("taskId").String()
	conn, err := upgrader.Upgrade(r.Response.Writer, r.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h := getHub(taskId)
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
	}()

	_ = conn.WriteMessage(websocket.TextMessage,
		[]byte(fmt.Sprintf(`{"time":"%s","level":"INFO","msg":"connected to task %s log stream"}`,
			time.Now().Format("15:04:05"), taskId)))

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// CrawlAssist handles WS /api/ws/crawl-assist/{taskId} - manual login assist.
func (c *Controller) CrawlAssist(r *ghttp.Request) {
	taskId := r.GetRouter("taskId").String()
	conn, err := upgrader.Upgrade(r.Response.Writer, r.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	row, dbErr := g.DB().Model("crawl_tasks").Where("task_id", taskId).One()
	if dbErr == nil && !row.IsEmpty() {
		status := row["status"].String()
		_ = conn.WriteMessage(websocket.TextMessage,
			[]byte(fmt.Sprintf(`{"type":"status","task_id":"%s","status":"%s"}`, taskId, status)))
	}

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
