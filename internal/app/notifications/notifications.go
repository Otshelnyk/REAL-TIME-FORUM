package notifications

import (
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type Service struct {
	Repo ports.NotificationRepository
}

type ListResult struct {
	Items       []ports.NotificationItem
	UnreadCount int
}

func (s *Service) List(userID int, limit int) (ListResult, error) {
	items, err := s.Repo.ListByUser(userID, limit)
	if err != nil {
		return ListResult{}, err
	}
	unread, err := s.Repo.CountUnread(userID)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, UnreadCount: unread}, nil
}

func (s *Service) MarkRead(userID int, notificationID int) error {
	return s.Repo.MarkRead(userID, notificationID)
}

func (s *Service) MarkAllRead(userID int) error {
	return s.Repo.MarkAllRead(userID)
}

func FormatCreatedAt(t time.Time, fallback string) string {
	if t.IsZero() {
		return fallback
	}
	return t.Format("02.01.2006 15:04")
}

