package notify

import (
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type Service struct {
	Repo      ports.NotificationRepository
	Realtime  ports.RealtimePublisher // optional
}

func (s *Service) Create(userID, actorID int, notifType, title, message, link string, createdAt time.Time) error {
	if userID <= 0 {
		return nil
	}
	_ = s.Repo.Insert(userID, actorID, notifType, title, message, link, createdAt)
	if s.Realtime != nil {
		s.Realtime.BroadcastToUser(userID, "notification_created", map[string]any{"type": notifType})
	}
	return nil
}

