package handlers

import (
	"database/sql"
	"net/http"
)

type Handlers struct {
	db *sql.DB
}

func New(db *sql.DB) *Handlers {
	return &Handlers{db: db}
}

func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

