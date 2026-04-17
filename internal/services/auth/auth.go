package auth

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ndanbaev/forum/internal/models"
)

type RegisterResult struct {
	ID        int64
	FirstName string
	LastName  string
	UUID      string
	ExpiresAt time.Time
}

func (s Service) Register(user models.User) (RegisterResult, error) {
	nickname := strings.TrimSpace(user.Nickname)
	email := strings.TrimSpace(strings.ToLower(user.Email))
	password := user.Password

	if nickname == "" || email == "" || password == "" {
		return RegisterResult{}, models.ErrInvalidCredentials
	}

	taken, err := s.Users.ExistsNickname(nickname)
	if err != nil {
		return RegisterResult{}, err
	}
	if taken {
		return RegisterResult{}, models.ErrNicknameTaken
	}

	taken, err = s.Users.ExistsEmail(email)
	if err != nil {
		return RegisterResult{}, err
	}
	if taken {
		return RegisterResult{}, models.ErrEmailTaken
	}

	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return RegisterResult{}, err
	}

	created, err := s.Users.Create(models.User{
		Nickname:  nickname,
		Age:       user.Age,
		Gender:    user.Gender,
		FirstName: strings.TrimSpace(user.FirstName),
		LastName:  strings.TrimSpace(user.LastName),
		Email:     email,
		Password:  hash,
	})
	if err != nil {
		return RegisterResult{}, err
	}

	now := s.now()
	expiresAt := now.Add(s.ttl())
	sid := uuid.NewString()

	if err := s.Sessions.Create(models.Session{
		UserID:    created.ID,
		UUID:      sid,
		ExpiresAt: expiresAt,
	}); err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{
		ID:        created.ID,
		FirstName: created.FirstName,
		LastName:  created.LastName,
		UUID:      sid,
		ExpiresAt: expiresAt,
	}, nil
}

type LoginInput struct {
	Login    string // nickname OR email
	Password string
}

type LoginResult struct {
	User        models.User
	SessionUUID string
	ExpiresAt   time.Time
}

func (s Service) Login(in LoginInput) (LoginResult, error) {
	login := strings.TrimSpace(in.Login)
	if login == "" || in.Password == "" {
		return LoginResult{}, models.ErrInvalidCredentials
	}

	u, err := s.Users.GetByLogin(login)
	if err != nil {
		return LoginResult{}, models.ErrInvalidCredentials
	}
	if !s.Hasher.Compare(u.Password, in.Password) {
		return LoginResult{}, models.ErrInvalidCredentials
	}

	now := s.now()
	expiresAt := now.Add(s.ttl())
	sid := uuid.NewString()

	if err := s.Sessions.Create(models.Session{
		UserID:    u.ID,
		UUID:      sid,
		ExpiresAt: expiresAt,
	}); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{User: u, SessionUUID: sid, ExpiresAt: expiresAt}, nil
}

func (s Service) Logout(sessionUUID string) error {
	sessionUUID = strings.TrimSpace(sessionUUID)
	if sessionUUID == "" {
		return nil
	}
	return s.Sessions.Delete(sessionUUID)
}

func (s Service) ValidateSession(sessionUUID string) (models.Session, error) {
	sessionUUID = strings.TrimSpace(sessionUUID)
	if sessionUUID == "" {
		return models.Session{}, models.ErrInvalidSession
	}

	sess, err := s.Sessions.Get(sessionUUID)
	if err != nil {
		return models.Session{}, models.ErrInvalidSession
	}

	if s.now().After(sess.ExpiresAt) {
		_ = s.Sessions.Delete(sessionUUID)
		return models.Session{}, models.ErrInvalidSession
	}
	return sess, nil
}

func (s Service) ttl() time.Duration {
	if s.SessionTTL > 0 {
		return s.SessionTTL
	}
	return 24 * time.Hour
}

func (s Service) now() time.Time {
	return time.Now()
}
