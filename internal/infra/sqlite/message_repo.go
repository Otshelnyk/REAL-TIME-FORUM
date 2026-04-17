package sqlite

import (
	"database/sql"
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type MessageRepo struct {
	db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) InsertPrivateMessage(fromID, toID int, content string, createdAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO private_messages (from_id, to_id, content, created_at) VALUES (?, ?, ?, ?)",
		fromID, toID, content, createdAt,
	)
	return err
}

func (r *MessageRepo) ListConversationUsersWithPreview(userID int) ([]ports.ConversationRow, error) {
	query := `
		SELECT u.id, u.nickname, u.avatar_url,
			(SELECT MAX(pm.created_at) FROM private_messages pm
			 WHERE (pm.from_id = ? AND pm.to_id = u.id) OR (pm.from_id = u.id AND pm.to_id = ?)) as last_at,
			(SELECT pm.content FROM private_messages pm
			 WHERE (pm.from_id = ? AND pm.to_id = u.id) OR (pm.from_id = u.id AND pm.to_id = ?)
			 ORDER BY pm.created_at DESC LIMIT 1) as last_preview
		FROM users u
		WHERE u.id != ?
		ORDER BY CASE WHEN last_at IS NULL THEN 0 ELSE 1 END DESC, last_at DESC, u.nickname ASC
	`
	rows, err := r.db.Query(query, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ports.ConversationRow
	for rows.Next() {
		var row ports.ConversationRow
		var lastAt, lastPreview interface{}
		if err := rows.Scan(&row.UserID, &row.Nickname, &row.AvatarURL, &lastAt, &lastPreview); err != nil {
			continue
		}
		if lastAt != nil {
			if s, ok := lastAt.(string); ok {
				t, perr := time.Parse(time.RFC3339, s)
				if perr != nil {
					t, _ = time.Parse("2006-01-02 15:04:05", s)
				}
				row.LastAt = &t
			}
		}
		if lastPreview != nil {
			if s, ok := lastPreview.(string); ok {
				row.LastMsgPreview = &s
			}
		}
		list = append(list, row)
	}
	return list, nil
}

func (r *MessageRepo) ListMessagesBetween(userID, otherID int, limit int, beforeID int) ([]ports.PrivateMessageRow, bool, error) {
	// Keep query semantics identical to handlers/api_messages.go: order DESC, fetch limit+1 then reverse at adapter layer.
	var rows *sql.Rows
	var err error
	if beforeID > 0 {
		rows, err = r.db.Query(`
			SELECT pm.id, pm.from_id, pm.to_id, pm.content, pm.created_at, u.nickname
			FROM private_messages pm
			JOIN users u ON u.id = pm.from_id
			WHERE ((pm.from_id = ? AND pm.to_id = ?) OR (pm.from_id = ? AND pm.to_id = ?)) AND pm.id < ?
			ORDER BY pm.id DESC LIMIT ?
		`, userID, otherID, otherID, userID, beforeID, limit+1)
	} else {
		rows, err = r.db.Query(`
			SELECT pm.id, pm.from_id, pm.to_id, pm.content, pm.created_at, u.nickname
			FROM private_messages pm
			JOIN users u ON u.id = pm.from_id
			WHERE (pm.from_id = ? AND pm.to_id = ?) OR (pm.from_id = ? AND pm.to_id = ?)
			ORDER BY pm.id DESC LIMIT ?
		`, userID, otherID, otherID, userID, limit+1)
	}
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var messages []ports.PrivateMessageRow
	for rows.Next() {
		var m ports.PrivateMessageRow
		var createdAt string
		if err := rows.Scan(&m.ID, &m.FromID, &m.ToID, &m.Content, &createdAt, &m.FromName); err != nil {
			continue
		}
		t, perr := time.Parse(time.RFC3339, createdAt)
		if perr != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		m.CreatedAt = t
		messages = append(messages, m)
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

