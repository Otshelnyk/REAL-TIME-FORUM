package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ndanbaev/forum/internal/handlers"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/models"
	"github.com/ndanbaev/forum/internal/ws"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Addr       string
	SQLitePath string
}

func (c Config) withDefaults() Config {
	out := c
	if strings.TrimSpace(out.Addr) == "" {
		out.Addr = ":8080"
	}
	if strings.TrimSpace(out.SQLitePath) == "" {
		out.SQLitePath = "forum.db"
	}
	return out
}

func Run(cfg Config) error {
	cfg = cfg.withDefaults()

	db, err := sql.Open("sqlite3", cfg.SQLitePath)
	if err != nil {
		return err
	}
	defer db.Close()

	models.InitSchema(db)

	handlers.InitTemplates()

	hub := ws.NewHub()
	go hub.Run()

	mux := buildMux(db, hub)

	fmt.Println("Server started at http://localhost:8080/")
	return http.ListenAndServe(cfg.Addr, deps.WithDeps(mux, &deps.Deps{DB: db, Hub: hub}))
}

func buildMux(db *sql.DB, hub *ws.Hub) *http.ServeMux {
	mux := http.NewServeMux()

	const indexPath = "web/templates/index.html"
	indexExists := func() bool {
		if _, err := os.Stat(indexPath); err != nil {
			if os.IsNotExist(err) {
				return false
			}
			return false
		}
		return true
	}
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		if !indexExists() {
			handlers.Render500(w, r, "Missing SPA template: templates/index.html")
			return
		}
		http.ServeFile(w, r, indexPath)
	}
	registerRoute := func(path string, handler http.HandlerFunc) {
		mux.HandleFunc(path, handler)
		if path != "/" && !strings.HasSuffix(path, "/") {
			mux.HandleFunc(path+"/", handler)
		}
	}

	// Static
	mux.HandleFunc("/static/css/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/static/css/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/static/css"+strings.TrimPrefix(r.URL.Path, "/static/css"))
	})
	mux.HandleFunc("/static/js/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/static/js/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/static/js"+strings.TrimPrefix(r.URL.Path, "/static/js"))
	})
	mux.HandleFunc("/static/uploads/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/static/uploads/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/static/uploads"+strings.TrimPrefix(r.URL.Path, "/static/uploads"))
	})
	registerRoute("/static/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/static/favicon.ico")
	})
	registerRoute("/static/favicon.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/static/favicon.png")
	})

	// WebSocket
	registerRoute("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.HandleWebSocket(hub, w, r)
	})

	// API
	registerRoute("/api/me", handlers.APIMe)
	registerRoute("/api/check-availability", handlers.APICheckAvailability)
	registerRoute("/api/register", handlers.APIRegister)
	registerRoute("/api/login", handlers.APILogin)
	registerRoute("/api/logout", handlers.APILogout)
	registerRoute("/api/categories", handlers.APICategories)
	registerRoute("/api/posts", handlers.APIPosts)
	mux.HandleFunc("/api/post/", handlers.APIPostByID)
	mux.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.APICreatePost(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	registerRoute("/api/comment", handlers.APICommentHandler)
	registerRoute("/api/like_post", handlers.APILikePostHandler)
	registerRoute("/api/like_comment", handlers.APILikeCommentHandler)
	registerRoute("/api/delete_comment", handlers.APIDeleteCommentHandler)
	registerRoute("/api/messages/conversations", handlers.APIConversations)
	mux.HandleFunc("/api/messages/with/", handlers.APIMessagesWith)
	registerRoute("/api/profile/avatar", handlers.APIUpdateAvatar)
	registerRoute("/api/notifications", handlers.APINotifications)
	registerRoute("/api/notifications/read", handlers.APIReadNotification)
	registerRoute("/api/notifications/read-all", handlers.APIReadAllNotifications)
	registerRoute("/api/debug/method", handlers.APIDebugMethod)
	mux.HandleFunc("/api/debug/status/", handlers.APIDebugStatus)

	// SPA: serve templates/index.html for any other GET
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && (strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/static") || strings.HasPrefix(r.URL.Path, "/ws")) {
			handlers.Render404(w, r)
			return
		}
		if r.Method == http.MethodGet {
			if !indexExists() {
				handlers.Render500(w, r, "Missing SPA template: templates/index.html")
				return
			}
			path := strings.TrimSuffix(r.URL.Path, "/")
			if path == "" {
				path = "/"
			}
			if strings.HasPrefix(path, "/post/") {
				idPart := strings.TrimSpace(strings.TrimPrefix(path, "/post/"))
				postID, err := strconv.Atoi(idPart)
				if err != nil || postID <= 0 {
					handlers.Render400(w, r, "Post ID must be a positive number.")
					return
				}
				var postExists int
				if err := db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", postID).Scan(&postExists); err != nil {
					handlers.Render500(w, r, "Error checking post.")
					return
				}
				if postExists == 0 {
					handlers.Render404(w, r)
					return
				}
				serveIndex(w, r)
				return
			}
			allowed := path == "/" || path == "/login" || path == "/register" || path == "/create" || path == "/error"
			if !allowed {
				if path == "/feed" {
					allowed = true
				} else if strings.HasPrefix(path, "/feed/") {
					_, err := strconv.Atoi(strings.TrimPrefix(path, "/feed/"))
					allowed = err == nil
				}
			}
			if !allowed {
				handlers.Render404(w, r)
				return
			}
			serveIndex(w, r)
			return
		}
		handlers.Render405(w, r)
	})

	_ = log.Default()
	return mux
}

