package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	appauth "github.com/ndanbaev/forum/internal/app/auth"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
	"github.com/ndanbaev/forum/internal/utils"
)

func authService(r *http.Request) *appauth.Service {
	d := deps.FromRequest(r)
	return &appauth.Service{
		Users:        sqlite.NewUserRepo(d.DB),
		Sessions:     sqlite.NewSessionRepo(d.DB),
		IsValidEmail: utils.IsValidEmail,
		Now:          time.Now,
	}
}

func sessionCookieValue(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}

// APIMe returns current user or success: false (200) when not logged in — avoids 401 in console on page load
func APIMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _ := authService(r).Me(sessionCookieValue(r))
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Пользователь не авторизован"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":         user.ID,
			"nickname":   user.Nickname,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"age":        user.Age,
			"gender":     user.Gender,
			"avatar_url": user.AvatarURL,
		},
	})
}

// APICheckAvailability returns whether nickname and/or email are available (for registration form validation)
func APICheckAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	res, err := authService(r).CheckAvailability(nickname, email)
	if err != nil {
		// keep prior behavior: treat as available on internal error
		res.NicknameAvailable = true
		res.EmailAvailable = true
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"nickname_available": res.NicknameAvailable, "email_available": res.EmailAvailable})
}

// APIRegister handles JSON registration
func APIRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if u, _ := authService(r).Me(sessionCookieValue(r)); u != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Вы уже вошли в аккаунт"})
		return
	}
	var body struct {
		Nickname  string `json:"nickname"`
		Age       int    `json:"age"`
		Gender    string `json:"gender"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Некорректный формат запроса"})
		return
	}
	msg, err := authService(r).Register(appauth.RegisterInput{
		Nickname:  body.Nickname,
		Age:       body.Age,
		Gender:    body.Gender,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		Password:  body.Password,
	})
	if err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: msg})
		return
	}
	jsonResponse(w, APIResponse{Success: true, Message: msg})
}

// APILogin handles JSON login (nickname or email + password)
func APILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if u, _ := authService(r).Me(sessionCookieValue(r)); u != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Вы уже вошли в аккаунт"})
		return
	}
	var body struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Некорректный формат запроса"})
		return
	}
	res, msg, err := authService(r).Login(appauth.LoginInput{Login: body.Login, Password: body.Password})
	if err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: msg})
		return
	}
	cookie := &http.Cookie{Name: "session", Value: res.SessionID, Path: "/", Expires: res.Expires, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	http.SetCookie(w, cookie)
	jsonResponse(w, APIResponse{Success: true, Message: msg})
}

// APILogout clears session
func APILogout(w http.ResponseWriter, r *http.Request) {
	method := requestMethod(r)
	if method != http.MethodPost && method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = authService(r).Logout(sessionCookieValue(r))
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	jsonResponse(w, APIResponse{Success: true})
}

func jsonResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
