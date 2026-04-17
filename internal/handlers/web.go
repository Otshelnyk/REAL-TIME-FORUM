package handlers

import (
	"net/http"
	"os"
	"path/filepath"
)

func NewWeb() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join("web", "static")))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA: always serve index.html for non-static routes
		b, err := os.ReadFile(filepath.Join("web", "templates", "index.html"))
		if err != nil {
			http.Error(w, "index not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})

	return mux
}

