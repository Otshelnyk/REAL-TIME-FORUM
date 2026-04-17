package sqlite

import (
	"database/sql"
	"time"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) GetUserIDByUUID(uuid string) (userID int, expires time.Time, err error) {
	err = r.db.QueryRow("SELECT user_id, expires FROM sessions WHERE uuid = ?", uuid).Scan(&userID, &expires)
	return userID, expires, err
}

func (r *SessionRepo) DeleteByUserID(userID int) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

func (r *SessionRepo) DeleteByUUID(uuid string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE uuid = ?", uuid)
	return err
}

func (r *SessionRepo) Insert(userID int, uuid string, expires time.Time) error {
	_, err := r.db.Exec("INSERT INTO sessions (user_id, uuid, expires) VALUES (?, ?, ?)", userID, uuid, expires)
	return err
}

