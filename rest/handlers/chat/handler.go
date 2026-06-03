package chat

import (
	"context"
	"encoding/json"
	"eraya/chat"
	"eraya/domain"
	"eraya/util"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type Handler struct {
	svc chat.Service
}

func NewHandler(svc chat.Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// GetConversation godoc
// @Summary Get messages in a conversation
// @Description Fetch all messages between the current user and another user.
// @Tags chat
// @Param withID path int true "Other User ID"
// @Success 200 {array} domain.Message
// @Router /chat/conversation/{withID} [get]
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)
	withIDStr := chi.URLParam(r, "withID")
	withID, _ := strconv.ParseInt(withIDStr, 10, 64)

	messages, err := h.svc.GetConversation(r.Context(), userID, withID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}

// GetConversations godoc
// @Summary List all conversations
// @Description Fetch all active conversations for the current user.
// @Tags chat
// @Success 200 {array} domain.Conversation
// @Router /chat/conversations [get]
func (h *Handler) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)
	convs, err := h.svc.GetConversations(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(convs)
}

// HandleConnections godoc
// @Summary Connect to real-time chat
// @Description Establish a WebSocket connection for real-time messaging. Requires token in query param.
// @Tags chat
// @Param with query int true "Chat with User ID"
// @Param token query string true "JWT Token"
// @Router /chat/ws [get]
func (h *Handler) HandleConnections(w http.ResponseWriter, r *http.Request) {
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

	h.svc.RegisterPresence(r.Context(), userID)
	defer h.svc.UnregisterPresence(context.Background(), userID)

	// Thread-safe write loop
	writeQueue := make(chan any, 32)
	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case msg := <-writeQueue:
				if err := ws.WriteJSON(msg); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Send initial status
	if withID != 0 {
		writeQueue <- &domain.Message{
			Type:     "presence",
			SenderID: withID,
			Online:   h.svc.IsUserOnline(withID),
		}
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Fetch role ONCE at connection time, not on every message
	userRole, _ := h.svc.IsAdmin(ctx, userID)
	isAdminUser := userRole == "admin" || userRole == "moderator"

	// Subscribe to incoming messages
	go func() {
		err := h.svc.Subscribe(ctx, userID, withID, func(msg *domain.Message) {
			// Filter for current view
			show := false

			switch msg.Type {
			case "presence":
				if withID != 0 && msg.SenderID == withID {
					show = true
				}
			case "delete_conversation":
				show = true
			default:
				// 1. Is this message from ME? (To confirm "Sent" status)
				// 2. Is this message for ME? (To receive real-time reply)
				isMe := (msg.SenderID == userID)
				isForMe := (msg.ReceiverID == userID)

				if isMe || isForMe {
					show = true
				} else if isAdminUser && withID != 0 && (msg.SenderID == withID || msg.ReceiverID == withID) {
					// Admin viewing a specific conversation
					show = true
				} else if isAdminUser && withID == 0 {
					// Admin in general view list
					show = true
				}
			}

			if show {
				broadcastMsg := map[string]any{
					"type":            msg.Type,
					"id":              msg.ID,
					"sender_id":       msg.SenderID,
					"sender_name":     msg.SenderName,
					"sender_role":     msg.SenderRole,
					"receiver_id":     msg.ReceiverID,
					"conversation_id": msg.ConversationID,
					"message_text":    msg.MessageText,
					"created_at":      msg.CreatedAt,
					"temp_id":         msg.TempID,
				}

				select {
				case writeQueue <- broadcastMsg:
				default:
				}
			}
		})
		if err != nil && err != context.Canceled {
			slog.Error("Failed to subscribe to chat", "error", err)
			// Send error message to client via writeQueue
			writeQueue <- map[string]string{
				"type":  "error",
				"error": "You don't have permission to access the chat system.",
			}
		}
	}()

	for {
		var msg map[string]string
		if err := ws.ReadJSON(&msg); err != nil {
			// slog.Error("WS: ReadJSON error", "error", err, "userID", userID)
			break
		}

		switch msg["type"] {
		case "ping":
			writeQueue <- map[string]string{"type": "pong"}
		case "delete":
			msgID, _ := strconv.ParseInt(msg["id"], 10, 64)
			_ = h.svc.DeleteMessage(ctx, userID, msgID)
		case "update":
			msgID, _ := strconv.ParseInt(msg["id"], 10, 64)
			newText := msg["text"]
			_, _ = h.svc.UpdateMessage(ctx, userID, msgID, newText)
		case "typing":
			// Ignore typing events since they are informational
		case "message", "":
			text := msg["text"]
			if text != "" {
				var replyToID *int64
				if idStr, ok := msg["replyToID"]; ok && idStr != "" {
					if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
						replyToID = &id
					}
				}

				// Allow overriding receiverID from body (more reliable for admins)
				targetID := withID
				if ridStr, ok := msg["receiverID"]; ok && ridStr != "" {
					if rid, err := strconv.ParseInt(ridStr, 10, 64); err == nil && rid != 0 {
						targetID = rid
					}
				}

				sentMsg, err := h.svc.SendMessage(ctx, userID, targetID, text, replyToID, msg["temp_id"])
				if err != nil {
					writeQueue <- map[string]string{"error": "Failed to send: " + err.Error()}
				} else {
					// Directly confirm to the sender so the UI updates to 'sent' immediately
					writeQueue <- map[string]any{
						"type":            "message",
						"id":              sentMsg.ID,
						"temp_id":         sentMsg.TempID,
						"sender_id":       sentMsg.SenderID,
						"sender_name":     sentMsg.SenderName,
						"sender_role":     sentMsg.SenderRole,
						"receiver_id":     sentMsg.ReceiverID,
						"conversation_id": sentMsg.ConversationID,
						"message_text":    sentMsg.MessageText,
						"created_at":      sentMsg.CreatedAt,
						"status":          "sent",
					}
				}
			}
		}
	}
}

// DeleteConversation godoc
// @Summary Delete a conversation
// @Description Permantently delete a conversation and all its messages. Admin only.
// @Tags chat
// @Param id path int true "Conversation ID"
// @Success 204 "No Content"
// @Router /chat/conversation/{id} [delete]
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	userID := r.Context().Value("user_id").(int64)

	err := h.svc.DeleteConversation(r.Context(), userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkAsRead godoc
// @Summary Mark messages as read
// @Description Mark all messages in a conversation as read.
// @Tags chat
// @Param id path int true "Conversation ID"
// @Success 204 "No Content"
// @Router /chat/conversation/{id}/read [post]
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	userID := r.Context().Value("user_id").(int64)

	err := h.svc.MarkAsRead(r.Context(), id, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkDeleteMessages godoc
// @Summary Delete multiple messages
// @Description Permantently delete multiple messages. Admin or sender only.
// @Tags chat
// @Param ids body []int true "Message IDs"
// @Success 204 "No Content"
// @Router /chat/messages/bulk-delete [post]
func (h *Handler) BulkDeleteMessages(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(int64)

	err := h.svc.DeleteMessages(r.Context(), userID, req.IDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchUsers godoc
// @Summary Search users for chat
// @Description Find any user by name or ID to start a conversation.
// @Tags chat
// @Param q query string true "Search Query"
// @Success 200 {array} domain.User
// @Router /chat/users/search [get]
func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing 'q' query param", http.StatusBadRequest)
		return
	}

	users, err := h.svc.SearchUsers(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(users)
}

// GetUnreadCount godoc
// @Summary Get total unread messages count
// @Description Get the total number of unread messages for the authenticated user across all conversations.
// @Tags chat
// @Success 200 {object} map[string]int
// @Router /chat/unread-count [get]
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		util.SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID := userIDVal.(int64)

	count, err := h.svc.GetTotalUnreadCount(r.Context(), userID)
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, "failed to get unread count")
		return
	}

	util.SendData(w, http.StatusOK, map[string]int{"unread_count": count})
}
