package wsrt

import (
	"strings"
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type Service struct {
	Messages  ports.MessageRepository
	Users     ports.UserRepository
	Notify    ports.NotificationService
	Realtime  ports.RealtimePublisher
	Now       func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) SendPrivateMessage(fromID int, toID int, content string) error {
	content = strings.TrimSpace(content)
	if fromID <= 0 || toID <= 0 || content == "" {
		return nil
	}
	if len(content) > 2000 {
		return nil
	}

	ts := s.now()
	if err := s.Messages.InsertPrivateMessage(fromID, toID, content, ts); err != nil {
		return err
	}

	fromNickname, _ := s.Users.GetNicknameByID(fromID)

	if s.Realtime != nil {
		s.Realtime.BroadcastToUser(toID, "new_private_message", map[string]any{
			"from_id":       fromID,
			"from_nickname": fromNickname,
			"content":       content,
			"created_at":    ts.Format("2006-01-02 15:04:05"),
		})
	}

	if s.Notify != nil {
		_ = s.Notify.Create(
			toID,
			fromID,
			"private_message",
			"New message",
			fromNickname+": "+content,
			"",
			ts,
		)
	}

	return nil
}

func (s *Service) Typing(fromID int, toID int, isTyping bool) error {
	if fromID <= 0 || toID <= 0 || toID == fromID {
		return nil
	}
	fromNickname, _ := s.Users.GetNicknameByID(fromID)
	if s.Realtime != nil {
		s.Realtime.BroadcastToUser(toID, "typing", map[string]any{
			"from_id":       fromID,
			"from_nickname": fromNickname,
			"is_typing":     isTyping,
		})
	}
	return nil
}

