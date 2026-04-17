package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ndanbaev/forum/internal/app/ports"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAlreadyLoggedIn   = errors.New("already logged in")
	ErrInvalidJSON       = errors.New("invalid json")
	ErrValidation        = errors.New("validation error")
	ErrNotFound          = errors.New("not found")
	ErrWrongPassword     = errors.New("wrong password")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrPasswordHashing   = errors.New("password hashing failed")
	ErrCreateUser        = errors.New("create user failed")
	ErrCreateSession     = errors.New("create session failed")
	ErrDeleteSession     = errors.New("delete session failed")
	ErrLegacyCompatRetry = errors.New("legacy compat retry")
)

type AvailabilityResult struct {
	NicknameAvailable bool
	EmailAvailable    bool
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

type LoginInput struct {
	Login    string
	Password string
}

type LoginResult struct {
	SessionID string
	Expires   time.Time
}

type Service struct {
	Users    ports.UserRepository
	Sessions ports.SessionRepository
	// EmailValidator stays out of the domain for pragmatism; caller can pass utils.IsValidEmail.
	IsValidEmail func(string) bool
	Now          func() time.Time
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) CheckAvailability(nickname, email string) (AvailabilityResult, error) {
	res := AvailabilityResult{NicknameAvailable: true, EmailAvailable: true}

	nickname = strings.TrimSpace(nickname)
	email = strings.TrimSpace(email)

	if nickname != "" {
		n, err := s.Users.CountByNicknameLower(nickname)
		if err != nil {
			return res, err
		}
		res.NicknameAvailable = n == 0
	}
	if email != "" {
		n, err := s.Users.CountByEmailLower(email)
		if err != nil {
			return res, err
		}
		res.EmailAvailable = n == 0
	}
	return res, nil
}

func (s *Service) Register(in RegisterInput) (string, error) {
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	nickname := strings.TrimSpace(in.Nickname)
	email := strings.TrimSpace(in.Email)
	password := in.Password

	if firstName == "" {
		return "Введите имя", ErrValidation
	}
	if lastName == "" {
		return "Введите фамилию", ErrValidation
	}
	if nickname == "" {
		return "Введите никнейм", ErrValidation
	}
	if email == "" {
		return "Введите email", ErrValidation
	}
	if password == "" {
		return "Введите пароль", ErrValidation
	}
	if rn := len([]rune(nickname)); rn < 2 || rn > 30 {
		return "Никнейм должен быть от 2 до 30 символов", ErrValidation
	}
	if in.Age < 1 || in.Age > 150 {
		return "Возраст должен быть от 1 до 150", ErrValidation
	}
	if in.Gender != "male" && in.Gender != "female" {
		return "Выберите пол", ErrValidation
	}
	if re := len([]rune(email)); re < 5 || re > 100 || s.IsValidEmail == nil || !s.IsValidEmail(email) {
		return "Введите корректный email", ErrValidation
	}
	if rp := len([]rune(password)); rp < 6 || rp > 72 {
		return "Пароль должен быть от 6 до 72 символов", ErrValidation
	}
	hasLetter := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := strings.ContainsAny(password, "0123456789")
	if !hasLetter || !hasDigit {
		return "Пароль должен содержать минимум одну букву и одну цифру", ErrValidation
	}

	emailExists, err := s.Users.CountByEmailLower(email)
	if err != nil {
		return "Не удалось создать аккаунт. Попробуйте еще раз", err
	}
	nickExists, err := s.Users.CountByNicknameLower(nickname)
	if err != nil {
		return "Не удалось создать аккаунт. Попробуйте еще раз", err
	}
	if nickExists > 0 {
		return "Этот никнейм уже занят", ErrValidation
	}
	if emailExists > 0 {
		return "Этот email уже используется", ErrValidation
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "Не удалось обработать пароль. Попробуйте еще раз", ErrPasswordHashing
	}

	if err := s.Users.CreateUser(nickname, in.Age, in.Gender, firstName, lastName, email, string(hash)); err != nil {
		// Legacy schema compatibility: old DB may still require users.username NOT NULL.
		if strings.Contains(strings.ToLower(err.Error()), "users.username") {
			if err2 := s.Users.CreateUserLegacyUsername(nickname, in.Age, in.Gender, firstName, lastName, email, string(hash)); err2 == nil {
				return "Регистрация прошла успешно. Теперь войдите в аккаунт", nil
			}
		}
		// Preserve prior UX message for unique constraint errors.
		errMsg := "Не удалось создать аккаунт. Попробуйте еще раз"
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "constraint") {
			errMsg = "Никнейм или email уже используются"
		}
		return errMsg, ErrCreateUser
	}

	return "Регистрация прошла успешно. Теперь войдите в аккаунт", nil
}

func (s *Service) Login(in LoginInput) (LoginResult, string, error) {
	login := strings.TrimSpace(in.Login)
	if login == "" {
		return LoginResult{}, "Введите никнейм или email", ErrValidation
	}
	if in.Password == "" {
		return LoginResult{}, "Введите пароль", ErrValidation
	}

	var id int
	var stored string
	var err error

	if strings.Contains(login, "@") {
		id, stored, err = s.Users.GetAuthByEmailLower(login)
	} else {
		id, stored, err = s.Users.GetAuthByNicknameLower(login)
		// Legacy fallback: some older DBs still use username as main login.
		if (err != nil || id == 0) && s.Users != nil {
			id2, stored2, err2 := s.Users.GetAuthByUsernameLower(login)
			if err2 == nil && id2 > 0 {
				id, stored, err = id2, stored2, nil
			}
		}
	}
	if err != nil || id == 0 {
		return LoginResult{}, "Пользователь с таким логином не найден", ErrNotFound
	}

	// bcrypt compare
	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(in.Password)); err != nil {
		// Legacy fallback: plaintext passwords from old version.
		if stored != in.Password {
			return LoginResult{}, "Неверный пароль", ErrWrongPassword
		}
		// Auto-migrate legacy plaintext password to bcrypt on successful login.
		newHash, hErr := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if hErr == nil {
			_ = s.Users.UpdatePasswordHashByID(id, string(newHash))
		}
	}

	sid := uuid.New().String()
	expires := s.now().Add(24 * time.Hour)

	_ = s.Sessions.DeleteByUserID(id)
	if err := s.Sessions.Insert(id, sid, expires); err != nil {
		return LoginResult{}, "Ошибка входа", ErrCreateSession
	}

	return LoginResult{SessionID: sid, Expires: expires}, "Вход выполнен", nil
}

func (s *Service) Logout(sessionUUID string) error {
	sessionUUID = strings.TrimSpace(sessionUUID)
	if sessionUUID == "" {
		return nil
	}
	return s.Sessions.DeleteByUUID(sessionUUID)
}

func (s *Service) Me(sessionUUID string) (*ports.User, error) {
	sessionUUID = strings.TrimSpace(sessionUUID)
	if sessionUUID == "" {
		return nil, nil
	}
	userID, expires, err := s.Sessions.GetUserIDByUUID(sessionUUID)
	if err != nil || userID <= 0 {
		return nil, nil
	}
	if s.now().After(expires) {
		return nil, nil
	}
	u, err := s.Users.GetByID(userID)
	if err != nil {
		return nil, nil
	}
	return u, nil
}

