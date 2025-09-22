package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"mailman/internal/services"

	"github.com/gorilla/websocket"
)

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	notificationService *services.EmailNotificationService
	upgrader            websocket.Upgrader
}

// NewWebSocketHandler 创建WebSocket处理器
func NewWebSocketHandler(notificationService *services.EmailNotificationService) *WebSocketHandler {
	return &WebSocketHandler{
		notificationService: notificationService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// 在生产环境中需要更严格的检查
				return true
			},
		},
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级为WebSocket连接
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocketHandler] Failed to upgrade connection: %v", err)
		http.Error(w, "Failed to upgrade to websocket", http.StatusBadRequest)
		return
	}

	// 生成客户端ID
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())

	// 注册客户端
	h.notificationService.RegisterClient(clientID, conn)

	log.Printf("[WebSocketHandler] WebSocket connection established for client %s", clientID)
}

// HandleNotificationStats 获取通知统计
func (h *WebSocketHandler) HandleNotificationStats(w http.ResponseWriter, r *http.Request) {
	stats := h.notificationService.GetStats()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"data":    stats,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleRecentNotifications 获取最近通知
func (h *WebSocketHandler) HandleRecentNotifications(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	notifications := h.notificationService.GetRecentNotifications(limit)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"data":    notifications,
	}

	json.NewEncoder(w).Encode(response)
}
