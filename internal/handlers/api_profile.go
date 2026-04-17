package handlers

import (
	"io"
	"net/http"

	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/app/profile"
	"github.com/ndanbaev/forum/internal/infra/fs"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
	"github.com/ndanbaev/forum/internal/middleware"
)

// APIUpdateAvatar uploads and updates current user's avatar image.
func APIUpdateAvatar(w http.ResponseWriter, r *http.Request) {
	method := requestMethod(r)
	if method != http.MethodPost && method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := middleware.GetCurrentUser(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		jsonResponse(w, APIResponse{Success: false, Message: "Not authenticated"})
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Could not update avatar"})
		return
	}

	const maxAvatarBytes = 2 << 20 // 2 MB
	if err := r.ParseMultipartForm(maxAvatarBytes); err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Could not parse file"})
		return
	}
	file, _, err := r.FormFile("avatar")
	if err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Avatar file is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Could not read file"})
		return
	}
	if len(data) == 0 {
		jsonResponse(w, APIResponse{Success: false, Message: "Empty file"})
		return
	}
	if len(data) > maxAvatarBytes {
		jsonResponse(w, APIResponse{Success: false, Message: "Avatar is too large (max 2 MB)"})
		return
	}

	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		// ok
	default:
		jsonResponse(w, APIResponse{Success: false, Message: "Only jpg, png, gif, webp are allowed"})
		return
	}

	svc := &profile.AvatarService{
		Users:   sqlite.NewUserRepo(d.DB),
		Storage: fs.NewAvatarStorage("web/static/uploads/avatars"),
	}
	relativeURL, err := svc.UpdateAvatar(user.ID, data, contentType)
	if err != nil {
		jsonResponse(w, APIResponse{Success: false, Message: "Could not update avatar"})
		return
	}

	jsonResponse(w, APIResponse{
		Success: true,
		Data: map[string]string{
			"avatar_url": relativeURL,
		},
	})
}
