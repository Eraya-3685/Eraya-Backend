package chat

import (
	"eraya/chat"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type WebSocketHandler struct {
	svc chat.Service
}

func NewWebSocketHandler(svc chat.Service) *WebSocketHandler {
	return &WebSocketHandler{
		svc: svc,
	}
}

// HandleConnections godoc
// @Summary Connect to real-time chat
// @Description Establish a WebSocket connection for real-time messaging. Requires token in query param.
// @Tags chat
// @Param with query int true "Chat with User ID"
// @Param token query string true "JWT Token"
// @Router /ws/chat [get]
func (h *WebSocketHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(int64)

	withStr := r.URL.Query().Get("with")
	var withID int64
	if withStr != "" {
		withID, _ = strconv.ParseInt(withStr, 10, 64)
	} else {
		http.Error(w, "missing 'with' query param", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)
		return
	}
	defer ws.Close()

	slog.Info("User connected to chat", "user_id", userID, "with_id", withID)

	for {
		var msg map[string]string
		err := ws.ReadJSON(&msg)
		if err != nil {
			slog.Error("Error reading from websocket", "error", err)
			break
		}

		text := msg["text"]
		if text != "" {
			_, err = h.svc.SendMessage(r.Context(), userID, withID, text)
			if err != nil {
				slog.Error("Error sending message", "error", err)
				ws.WriteJSON(map[string]string{"error": "Failed to send message"})
			} else {
				// Echo back to sender
				ws.WriteJSON(map[string]string{
					"status": "sent",
					"text":   text,
				})
			}
		}
	}
}
