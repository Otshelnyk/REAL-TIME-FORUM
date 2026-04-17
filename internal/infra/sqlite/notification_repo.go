package sqlite

import (
	"database/sql"
	"time"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type NotificationRepo struct {
	db *sql.DB
}

func NewNotificationRepo(db *sql.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Insert(userID, actorID int, notifType, title, message, link string, createdAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO notifications (user_id, actor_id, type, title, message, link, is_read, created_at) VALUES (?, ?, ?, ?, ?, ?, 0, ?)",
		userID, actorID, notifType, title, message, link, createdAt,
	)
	return err
}

func (r *NotificationRepo) ListByUser(userID int, limit int) ([]ports.NotificationItem, error) {
	rows, err := r.db.Query(
		"SELECT id, type, title, message, link, is_read, created_at FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ports.NotificationItem
	for rows.Next() {
		var it ports.NotificationItem
		var createdAt string
		if err := rows.Scan(&it.ID, &it.Type, &it.Title, &it.Message, &it.Link, &it.IsRead, &createdAt); err != nil {
			continue
		}
		t, perr := time.Parse(time.RFC3339, createdAt)
		if perr != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		it.CreatedAt = t
		items = append(items, it)
	}
	return items, nil
}

func (r *NotificationRepo) CountUnread(userID int) (int, error) {
	var unread int
	err := r.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = 0", userID).Scan(&unread)
	return unread, err
}

func (r *NotificationRepo) MarkRead(userID int, notificationID int) error {
	_, err := r.db.Exec("UPDATE notifications SET is_read = 1 WHERE id = ? AND user_id = ?", notificationID, userID)
	return err
}

func (r *NotificationRepo) MarkAllRead(userID int) error {
	_, err := r.db.Exec("UPDATE notifications SET is_read = 1 WHERE user_id = ?", userID)
	return err
}

