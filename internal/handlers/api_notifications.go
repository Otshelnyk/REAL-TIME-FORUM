package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ndanbaev/forum/internal/middleware"
	"github.com/ndanbaev/forum/internal/app/notifications"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
)

// APINotifications returns latest notifications and unread count.
func APINotifications(w http.ResponseWriter, r *http.Request) {
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

	svc := &notifications.Service{Repo: sqlite.NewNotificationRepo(d.DB)}
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	res, err := svc.List(user.ID, limit)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var items []map[string]interface{}
	for _, it := range res.Items {
		createdAtRaw := ""
		ts := notifications.FormatCreatedAt(it.CreatedAt, createdAtRaw)
		items = append(items, map[string]interface{}{
			"id":         it.ID,
			"type":       it.Type,
			"title":      it.Title,
			"message":    it.Message,
			"link":       it.Link,
			"is_read":    it.IsRead,
			"created_at": ts,
		})
	}

	jsonResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"items":        items,
			"unread_count": res.UnreadCount,
		},
	})
}

// APIReadNotification marks one notification as read.
func APIReadNotification(w http.ResponseWriter, r *http.Request) {
	method := requestMethod(r)
	if method != http.MethodPost && method != http.MethodPatch {
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
	svc := &notifications.Service{Repo: sqlite.NewNotificationRepo(d.DB)}
	var body struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID <= 0 {
		jsonResponse(w, APIResponse{Success: false, Message: "Invalid notification id"})
		return
	}
	_ = svc.MarkRead(user.ID, body.ID)
	jsonResponse(w, APIResponse{Success: true})
}

// APIReadAllNotifications marks all notifications as read.
func APIReadAllNotifications(w http.ResponseWriter, r *http.Request) {
	method := requestMethod(r)
	if method != http.MethodPost && method != http.MethodPatch {
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
	svc := &notifications.Service{Repo: sqlite.NewNotificationRepo(d.DB)}
	_ = svc.MarkAllRead(user.ID)
	jsonResponse(w, APIResponse{Success: true})
}
