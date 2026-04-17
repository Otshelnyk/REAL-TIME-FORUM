package auth

import (
	"time"

	"github.com/ndanbaev/forum/internal/models"
)

type UserStore interface {
	Create(u models.User) (models.User, error)
	GetByLogin(login string) (models.User, error) // nickname OR email
	ExistsNickname(nickname string) (bool, error)
	ExistsEmail(email string) (bool, error)
}

type SessionStore interface {
	Create(s models.Session) error
	Get(sessionUUID string) (models.Session, error)
	Delete(sessionUUID string) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) bool
}

type Service struct {
	Users    UserStore
	Sessions SessionStore
	Hasher   PasswordHasher

	// TTL сессии. Если 0 — по умолчанию 24h.
	SessionTTL time.Duration
}

type RegisterInput struct {
	Nickname  string
	Age       int
	Gender    string
	FirstName string
	LastName  string
	Email     string
	Password  string
}

func NewAuthService(users UserStore, sessions SessionStore, hasher PasswordHasher) *Service {
	return &Service{
		Users:    users,
		Sessions: sessions,
		Hasher:   hasher,
	}
}
