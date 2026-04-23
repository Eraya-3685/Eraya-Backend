package chat

import (
	"eraya/chat"
	middleware "eraya/rest/middlewares"
	"log"
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
	middlewares *middleware.Middlewares
	svc         chat.Service
}

func NewWebSocketHandler(middlewares *middleware.Middlewares, svc chat.Service) *WebSocketHandler {
	return &WebSocketHandler{
		middlewares: middlewares,
		svc:         svc,
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

	// In a real app, you might want to specify who you are chatting with via URL params
	// e.g. /chat?with=2
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
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()

	log.Printf("User %d connected to chat with %d", userID, withID)

	for {
		var msg map[string]string
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error reading from websocket: %v", err)
			break
		}

		text := msg["text"]
		if text != "" {
			_, err = h.svc.SendMessage(userID, withID, text)
			if err != nil {
				log.Printf("error sending message: %v", err)
				ws.WriteJSON(map[string]string{"error": "Failed to send message"})
			} else {
				// Echo back to sender for immediate feedback
				ws.WriteJSON(map[string]string{
					"status": "sent",
					"text":   text,
				})
			}
		}
	}
}
