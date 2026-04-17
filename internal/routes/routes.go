package routes

import (
	"database/sql"
	"net/http"

	"github.com/ndanbaev/forum/internal/handlers"
)

func New(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	// API
	h := handlers.New(db)
	mux.HandleFunc("GET /healthz", h.Healthz)

	// Web (SPA shell + static assets)
	mux.Handle("/", handlers.NewWeb())

	return mux
}

