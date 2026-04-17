package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ndanbaev/forum/internal/app/forum"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/infra/notify"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
	"github.com/ndanbaev/forum/internal/middleware"
	"github.com/ndanbaev/forum/internal/ws"
)

// APICategories returns all categories as JSON
func APICategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB)}
	list, err := svc.Categories()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: list})
}

// APIPosts returns paginated posts as JSON (for SPA feed)
func APIPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := middleware.GetCurrentUser(r)
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, _ := strconv.Atoi(p); n > 0 {
			page = n
		}
	}
	const pageSize = 10
	category := r.URL.Query().Get("category")
	categoriesRaw := strings.TrimSpace(r.URL.Query().Get("categories"))
	filter := r.URL.Query().Get("filter")
	selectedCategories := make([]int, 0)
	if categoriesRaw != "" {
		seen := map[int]bool{}
		for _, part := range strings.Split(categoriesRaw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, convErr := strconv.Atoi(part)
			if convErr != nil || id <= 0 || seen[id] {
				continue
			}
			seen[id] = true
			selectedCategories = append(selectedCategories, id)
		}
	}
	if len(selectedCategories) == 0 && category != "" {
		if id, convErr := strconv.Atoi(strings.TrimSpace(category)); convErr == nil && id > 0 {
			selectedCategories = append(selectedCategories, id)
		}
	}

	userID := 0
	if user != nil {
		userID = user.ID
	}
	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB)}
	res, err := svc.ListPosts(forum.ListPostsParams{
		UserID:              userID,
		Page:                page,
		PageSize:            pageSize,
		SelectedCategories:  selectedCategories,
		Filter:              filter,
	})
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var posts []map[string]interface{}
	for _, p := range res.Posts {
		created := formatTime(p.CreatedAtRaw)
		posts = append(posts, map[string]interface{}{
			"id":            p.ID,
			"user_id":       p.UserID,
			"title":         p.Title,
			"content":       p.Content,
			"created_at":    created,
			"author":        p.Author,
			"author_avatar": p.AuthorAvatar,
			"likes":         p.Likes,
			"dislikes":      p.Dislikes,
			"comment_count": p.CommentCount,
			"categories":    p.Categories,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"posts":       posts,
			"page":        res.Page,
			"total_pages": res.TotalPages,
			"total_posts": res.TotalPosts,
		},
	})
}

// APIPostByID returns a single post with comments (JSON)
func APIPostByID(w http.ResponseWriter, r *http.Request) {
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, APIResponse{Success: false, Message: "Database error"})
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/post/") {
		http.NotFound(w, r)
		return
	}
	idStr := strings.TrimPrefix(path, "/api/post/")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		jsonResponse(w, APIResponse{Success: false, Message: "Invalid post id"})
		return
	}

	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB)}
	res, err := svc.GetPost(id)
	if err != nil {
		if err == forum.ErrNotFound || err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			jsonResponse(w, APIResponse{Success: false, Message: "Post not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, APIResponse{Success: false, Message: "Database error"})
		return
	}

	var comments []map[string]interface{}
	for _, c := range res.Comments {
		comments = append(comments, map[string]interface{}{
			"id": c.ID, "post_id": c.PostID, "user_id": c.UserID, "content": c.Content,
			"created_at": formatTime(c.CreatedAtRaw), "author": c.Author, "author_avatar": c.AuthorAvatar, "likes": c.Likes, "dislikes": c.Dislikes,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"post": map[string]interface{}{
				"id": res.Post.ID, "user_id": res.Post.UserID, "title": res.Post.Title, "content": res.Post.Content,
				"created_at": formatTime(res.Post.CreatedAtRaw), "author": res.Post.Author, "likes": res.Post.Likes, "dislikes": res.Post.Dislikes,
				"categories": res.Post.Categories, "author_avatar": res.Post.AuthorAvatar,
			},
			"comments": comments,
		},
	})
}

// APICreatePost creates a post (JSON)
func APICreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		jsonResponse(w, APIResponse{Success: false, Message: "Error creating post"})
		return
	}
	var body struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		CategoryIDs []int  `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Invalid JSON"})
		return
	}
	title := strings.TrimSpace(body.Title)
	content := strings.TrimSpace(body.Content)
	if title == "" || content == "" || len(body.CategoryIDs) == 0 {
		jsonResponse(w, APIResponse{Success: false, Message: "Title, content and at least one category required"})
		return
	}
	if len([]rune(title)) > 200 {
		jsonResponse(w, APIResponse{Success: false, Message: "Title max 200 characters"})
		return
	}
	if len([]rune(content)) > 10000 {
		jsonResponse(w, APIResponse{Success: false, Message: "Content max 10000 characters"})
		return
	}
	var hub *ws.Hub
	if h, ok := d.Hub.(*ws.Hub); ok {
		hub = h
	}
	notifier := &notify.Service{Repo: sqlite.NewNotificationRepo(d.DB), Realtime: ws.NewPublisher(hub)}
	svc := &forum.Service{Repo: sqlite.NewForumRepo(d.DB), Notify: notifier, Now: time.Now}
	postID, err := svc.CreatePost(user.ID, title, content, body.CategoryIDs)
	if err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Error creating post"})
		return
	}
	jsonResponse(w, APIResponse{Success: true, Data: map[string]interface{}{"id": postID}})
}

func formatTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", s)
	}
	if err == nil {
		return t.Format("02.01.2006 15:04")
	}
	return s
}
