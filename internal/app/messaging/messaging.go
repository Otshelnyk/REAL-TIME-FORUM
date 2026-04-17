package messaging

import (
	"strings"
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type Service struct {
	Messages ports.MessageRepository
	Presence ports.RealtimePublisher // optional
}

type ConversationItem struct {
	UserID         int
	Nickname       string
	AvatarURL      string
	Online         bool
	LastMsgAt      *time.Time
	LastMsgPreview string
}

func (s *Service) Conversations(userID int) ([]ConversationItem, error) {
	rows, err := s.Messages.ListConversationUsersWithPreview(userID)
	if err != nil {
		return nil, err
	}
	online := map[int]bool{}
	if s.Presence != nil {
		for _, id := range s.Presence.OnlineUserIDs() {
			online[id] = true
		}
	}
	out := make([]ConversationItem, 0, len(rows))
	for _, r := range rows {
		item := ConversationItem{
			UserID:    r.UserID,
			Nickname:  r.Nickname,
			AvatarURL: r.AvatarURL,
			Online:    online[r.UserID],
		}
		if r.LastAt != nil && !r.LastAt.IsZero() {
			item.LastMsgAt = r.LastAt
		}
		if r.LastMsgPreview != nil {
			s := *r.LastMsgPreview
			s = strings.ReplaceAll(s, "\n", " ")
			s = strings.TrimSpace(s)
			if len(s) > 50 {
				s = s[:47] + "..."
			}
			item.LastMsgPreview = s
		}
		out = append(out, item)
	}
	return out, nil
}

type MessagesResult struct {
	Messages []ports.PrivateMessageRow
	HasMore  bool
}

func (s *Service) MessagesWith(userID, otherID int, limit int, beforeID int) (MessagesResult, error) {
	msgs, hasMore, err := s.Messages.ListMessagesBetween(userID, otherID, limit, beforeID)
	if err != nil {
		return MessagesResult{}, err
	}
	return MessagesResult{Messages: msgs, HasMore: hasMore}, nil
}

