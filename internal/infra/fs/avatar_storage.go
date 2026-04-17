package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type AvatarStorage struct {
	Dir string // filesystem directory, e.g. static/uploads/avatars
}

func NewAvatarStorage(dir string) *AvatarStorage {
	return &AvatarStorage{Dir: dir}
}

func (s *AvatarStorage) SaveAvatar(userID int, data []byte, contentType string) (relativeURL string, fullPath string, err error) {
	ext := ""
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		return "", "", fmt.Errorf("unsupported content type")
	}

	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return "", "", err
	}

	filename := fmt.Sprintf("u%d_%d%s", userID, time.Now().UnixNano(), ext)
	fullPath = filepath.Join(s.Dir, filename)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", "", err
	}

	// Current app serves uploads from /static/uploads/avatars/.
	// We keep the URL stable regardless of OS path separators.
	relativeURL = "/static/uploads/avatars/" + filename
	return relativeURL, fullPath, nil
}

func (s *AvatarStorage) DeleteFile(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

