package deps

import (
	"context"
	"database/sql"
	"net/http"
)

type Deps struct {
	DB  *sql.DB
	Hub any
}

type ctxKey struct{}

func WithDeps(next http.Handler, d *Deps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKey{}, d)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func FromRequest(r *http.Request) *Deps {
	if r == nil {
		return nil
	}
	if d, ok := r.Context().Value(ctxKey{}).(*Deps); ok {
		return d
	}
	return nil
}

