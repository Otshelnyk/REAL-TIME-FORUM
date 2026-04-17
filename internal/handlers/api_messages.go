package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/ndanbaev/forum/internal/app/messaging"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
	"github.com/ndanbaev/forum/internal/middleware"
	"github.com/ndanbaev/forum/internal/ws"
)

// ConversationItem for sidebar: user + last message time
type ConversationItem struct {
	UserID         int    `json:"user_id"`
	Nickname       string `json:"nickname"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	Online         bool   `json:"online"`
	LastMsgAt      string `json:"last_msg_at,omitempty"`
	LastMsgPreview string `json:"last_msg_preview,omitempty"`
}

// APIConversations returns list of users to show in PM sidebar (users we chatted with + all users for new chats), ordered by last message
func APIConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := middleware.GetCurrentUser(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Not authenticated"})
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	var hub *ws.Hub
	if h, ok := d.Hub.(*ws.Hub); ok {
		hub = h
	}

	svc := &messaging.Service{
		Messages: sqlite.NewMessageRepo(d.DB),
		Presence: ws.NewPublisher(hub),
	}
	list, err := svc.Conversations(user.ID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	out := make([]ConversationItem, 0, len(list))
	for _, it := range list {
		item := ConversationItem{
			UserID:         it.UserID,
			Nickname:       it.Nickname,
			AvatarURL:      it.AvatarURL,
			Online:         it.Online,
			LastMsgPreview: it.LastMsgPreview,
		}
		if it.LastMsgAt != nil {
			item.LastMsgAt = it.LastMsgAt.Format("2006-01-02 15:04:05")
		}
		out = append(out, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: out})
}

// APIMessagesWith returns private messages between current user and :id, paginated (limit 10, before=message_id for older)
func APIMessagesWith(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := middleware.GetCurrentUser(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Not authenticated"})
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/messages/with/") {
		http.NotFound(w, r)
		return
	}
	otherIDStr := strings.TrimPrefix(path, "/api/messages/with/")
	otherID, err := strconv.Atoi(otherIDStr)
	if err != nil || otherID <= 0 {
		jsonResponse(w, APIResponse{Success: false, Message: "Invalid user id"})
		return
	}
	if otherID == user.ID {
		jsonResponse(w, APIResponse{Success: false, Message: "Cannot load chat with yourself"})
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, _ := strconv.Atoi(l); n > 0 && n <= 50 {
			limit = n
		}
	}
	beforeID := 0
	if b := r.URL.Query().Get("before"); b != "" {
		beforeID, _ = strconv.Atoi(b)
	}

	// Messages where (from_id=me and to_id=other) or (from_id=other and to_id=me), order by id DESC so newest last when we reverse
	svc := &messaging.Service{Messages: sqlite.NewMessageRepo(d.DB)}
	res, err := svc.MessagesWith(user.ID, otherID, limit, beforeID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	type MsgRow struct {
		ID        int    `json:"id"`
		FromID    int    `json:"from_id"`
		ToID      int    `json:"to_id"`
		Content   string `json:"content"`
		CreatedAt string `json:"created_at"`
		FromName  string `json:"from_nickname"`
	}
	messages := make([]MsgRow, 0, len(res.Messages))
	for _, row := range res.Messages {
		m := MsgRow{
			ID:       row.ID,
			FromID:   row.FromID,
			ToID:     row.ToID,
			Content:  row.Content,
			FromName: row.FromName,
		}
		if !row.CreatedAt.IsZero() {
			m.CreatedAt = row.CreatedAt.Format("02.01.2006 15:04")
		} else {
			m.CreatedAt = ""
		}
		messages = append(messages, m)
	}
	// Reverse so oldest first (for display)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"messages": messages,
			"has_more": res.HasMore,
			"other_id": otherID,
		},
	})
}
