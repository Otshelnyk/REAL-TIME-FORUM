package profile

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/ndanbaev/forum/internal/app/ports"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrInvalidFile      = errors.New("invalid file")
	ErrStorage          = errors.New("storage error")
	ErrDatabase         = errors.New("database error")
)

type AvatarService struct {
	Users   ports.UserRepository
	Storage ports.AvatarStorage
}

func (s *AvatarService) UpdateAvatar(userID int, data []byte, contentType string) (string, error) {
	if userID <= 0 {
		return "", ErrNotAuthenticated
	}
	if len(data) == 0 {
		return "", ErrInvalidFile
	}
	if s.Users == nil || s.Storage == nil {
		return "", ErrStorage
	}

	oldAvatar, _ := s.Users.GetAvatarURLByID(userID)
	relativeURL, fullPath, err := s.Storage.SaveAvatar(userID, data, contentType)
	if err != nil {
		return "", ErrStorage
	}

	if err := s.Users.UpdateAvatarURLByID(userID, relativeURL); err != nil {
		_ = s.Storage.DeleteFile(fullPath)
		return "", ErrDatabase
	}

	if strings.HasPrefix(oldAvatar, "/static/uploads/avatars/") {
		oldPath := filepath.Join("web", "static", strings.TrimPrefix(oldAvatar, "/static/"))
		if oldPath != fullPath {
			_ = s.Storage.DeleteFile(oldPath)
		}
	}

	return relativeURL, nil
}

