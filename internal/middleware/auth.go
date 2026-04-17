package middleware

import (
	"net/http"
	"time"

	appauth "github.com/ndanbaev/forum/internal/app/auth"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
	"github.com/ndanbaev/forum/internal/models"
	"github.com/ndanbaev/forum/internal/utils"
)

// GetCurrentUser retrieves the current user from session
func GetCurrentUser(r *http.Request) *models.User {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value == "" {
		return nil
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		return nil
	}
	u, _ := (&appauth.Service{
		Users:        sqlite.NewUserRepo(d.DB),
		Sessions:     sqlite.NewSessionRepo(d.DB),
		IsValidEmail: utils.IsValidEmail,
		Now:          time.Now,
	}).Me(cookie.Value)
	if u == nil {
		return nil
	}
	return &models.User{
		ID:        u.ID,
		Nickname:  u.Nickname,
		Age:       u.Age,
		Gender:    u.Gender,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		Password:  u.Password,
		AvatarURL: u.AvatarURL,
	}
}

// GetUserBySession retrieves user by session UUID
func GetUserBySession(session string) *models.User {
	// Legacy helper kept for compatibility; delegates to Me() logic.
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: session})
	return GetCurrentUser(r)
}

// RequireAuth middleware that redirects to login if user is not authenticated
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetCurrentUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
