package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ndanbaev/forum/internal/app/forum"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/infra/notify"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
	"github.com/ndanbaev/forum/internal/middleware"
	"github.com/ndanbaev/forum/internal/ws"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// CommentData represents comment data for API responses
type CommentData struct {
	ID           int    `json:"id"`
	PostID       int    `json:"post_id"`
	Author       string `json:"author"`
	AuthorAvatar string `json:"author_avatar"`
	Content      string `json:"content"`
	CreatedAt    string `json:"created_at"`
	Likes        int    `json:"likes"`
	Dislikes     int    `json:"dislikes"`
}

// LikeData represents like/dislike data for API responses
type LikeData struct {
	Likes      int  `json:"likes"`
	Dislikes   int  `json:"dislikes"`
	IsLiked    bool `json:"is_liked"`
	IsDisliked bool `json:"is_disliked"`
}

// APICommentHandler handles AJAX comment submission
func APICommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	postID := r.FormValue("post_id")
	content := strings.TrimSpace(r.FormValue("comment"))

	if postID == "" || content == "" {
		response := APIResponse{
			Success: false,
			Message: "Post ID and comment content are required",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(content) > 500 {
		response := APIResponse{
			Success: false,
			Message: "Comment too long (max 500 characters)",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var hub *ws.Hub
	if h, ok := d.Hub.(*ws.Hub); ok {
		hub = h
	}
	notifier := &notify.Service{Repo: sqlite.NewNotificationRepo(d.DB), Realtime: ws.NewPublisher(hub)}
	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB), Notify: notifier, Now: time.Now}
	commentID, err := svc.AddComment(user.ID, postID, content, user.Nickname)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "Error saving comment",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the created comment with author info
	row, err := svc.GetComment(commentID)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "Error retrieving comment",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var comment CommentData
	comment.ID = row.ID
	comment.PostID = row.PostID
	comment.Author = row.Author
	comment.AuthorAvatar = row.AuthorAvatar
	comment.Content = row.Content
	comment.Likes = row.Likes
	comment.Dislikes = row.Dislikes
	comment.CreatedAt = formatTime(row.CreatedAtRaw)

	response := APIResponse{
		Success: true,
		Message: "Comment added successfully",
		Data:    comment,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// APILikePostHandler handles AJAX post likes/dislikes
func APILikePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	postID := r.FormValue("post_id")
	likeStr := r.FormValue("like")

	if postID == "" || likeStr == "" {
		response := APIResponse{
			Success: false,
			Message: "Post ID and like value are required",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	isLike := likeStr == "1"
	var hub *ws.Hub
	if h, ok := d.Hub.(*ws.Hub); ok {
		hub = h
	}
	notifier := &notify.Service{Repo: sqlite.NewNotificationRepo(d.DB), Realtime: ws.NewPublisher(hub)}
	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB), Notify: notifier, Now: time.Now}
	rr, err := svc.TogglePostReaction(user.ID, postID, isLike, user.Nickname)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "Error updating like/dislike",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	likeData := LikeData{Likes: rr.Likes, Dislikes: rr.Dislikes, IsLiked: rr.IsLiked, IsDisliked: rr.IsDisliked}

	response := APIResponse{
		Success: true,
		Message: "Like/dislike updated successfully",
		Data:    likeData,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// APILikeCommentHandler handles AJAX comment likes/dislikes
func APILikeCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	commentID := r.FormValue("comment_id")
	likeStr := r.FormValue("like")

	if commentID == "" || likeStr == "" {
		response := APIResponse{
			Success: false,
			Message: "Comment ID and like value are required",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	isLike := likeStr == "1"
	var hub *ws.Hub
	if h, ok := d.Hub.(*ws.Hub); ok {
		hub = h
	}
	notifier := &notify.Service{Repo: sqlite.NewNotificationRepo(d.DB), Realtime: ws.NewPublisher(hub)}
	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB), Notify: notifier, Now: time.Now}
	rr, err := svc.ToggleCommentReaction(user.ID, commentID, isLike, user.Nickname)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "Error updating like/dislike",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	likeData := LikeData{Likes: rr.Likes, Dislikes: rr.Dislikes, IsLiked: rr.IsLiked, IsDisliked: rr.IsDisliked}

	response := APIResponse{
		Success: true,
		Message: "Like/dislike updated successfully",
		Data:    likeData,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// APIDeleteCommentHandler handles AJAX comment deletion
func APIDeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	method := requestMethod(r)
	if method != http.MethodPost && method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	commentID := strings.TrimSpace(r.FormValue("comment_id"))
	if commentID == "" {
		commentID = strings.TrimSpace(r.URL.Query().Get("comment_id"))
	}
	if commentID == "" {
		var body struct {
			CommentID string `json:"comment_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		commentID = strings.TrimSpace(body.CommentID)
	}
	if commentID == "" {
		response := APIResponse{
			Success: false,
			Message: "Comment ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB), Now: time.Now}
	err := svc.DeleteComment(user.ID, commentID)
	if err != nil {
		msg := "Error deleting comment"
		switch err {
		case forum.ErrNotFound:
			msg = "Comment not found"
		case forum.ErrForbidden:
			msg = "You can only delete your own comments"
		}
		response := APIResponse{
			Success: false,
			Message: msg,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := APIResponse{
		Success: true,
		Message: "Comment deleted successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
