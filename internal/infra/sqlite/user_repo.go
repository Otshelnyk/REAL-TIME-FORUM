package sqlite

import (
	"database/sql"
	"strings"

	"github.com/ndanbaev/forum/internal/app/ports"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CountByEmailLower(email string) (int, error) {
	email = strings.TrimSpace(email)
	var n int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER(?)", email).Scan(&n)
	return n, err
}

func (r *UserRepo) CountByNicknameLower(nickname string) (int, error) {
	nickname = strings.TrimSpace(nickname)
	var n int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE LOWER(nickname) = LOWER(?)", nickname).Scan(&n)
	return n, err
}

func (r *UserRepo) GetByID(id int) (*ports.User, error) {
	var u ports.User
	err := r.db.QueryRow(
		"SELECT id, nickname, age, gender, first_name, last_name, email, password, avatar_url FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Nickname, &u.Age, &u.Gender, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.AvatarURL)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetAuthByEmailLower(email string) (id int, passwordHash string, err error) {
	email = strings.TrimSpace(email)
	err = r.db.QueryRow("SELECT id, password FROM users WHERE LOWER(email) = LOWER(?)", email).Scan(&id, &passwordHash)
	return id, passwordHash, err
}

func (r *UserRepo) GetAuthByNicknameLower(nickname string) (id int, passwordHash string, err error) {
	nickname = strings.TrimSpace(nickname)
	err = r.db.QueryRow("SELECT id, password FROM users WHERE LOWER(nickname) = LOWER(?)", nickname).Scan(&id, &passwordHash)
	return id, passwordHash, err
}

func (r *UserRepo) GetAuthByUsernameLower(username string) (id int, passwordHash string, err error) {
	username = strings.TrimSpace(username)
	err = r.db.QueryRow("SELECT id, password FROM users WHERE LOWER(username) = LOWER(?)", username).Scan(&id, &passwordHash)
	return id, passwordHash, err
}

func (r *UserRepo) GetNicknameByID(id int) (string, error) {
	var nickname string
	err := r.db.QueryRow("SELECT nickname FROM users WHERE id = ?", id).Scan(&nickname)
	return nickname, err
}

func (r *UserRepo) CreateUser(nickname string, age int, gender, firstName, lastName, email, passwordHash string) error {
	_, err := r.db.Exec(
		"INSERT INTO users (nickname, age, gender, first_name, last_name, email, password) VALUES (?, ?, ?, ?, ?, ?, ?)",
		nickname, age, gender, firstName, lastName, email, passwordHash,
	)
	return err
}

func (r *UserRepo) CreateUserLegacyUsername(nickname string, age int, gender, firstName, lastName, email, passwordHash string) error {
	_, err := r.db.Exec(
		"INSERT INTO users (nickname, username, age, gender, first_name, last_name, email, password) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		nickname, nickname, age, gender, firstName, lastName, email, passwordHash,
	)
	return err
}

func (r *UserRepo) UpdatePasswordHashByID(id int, passwordHash string) error {
	_, err := r.db.Exec("UPDATE users SET password = ? WHERE id = ?", passwordHash, id)
	return err
}

func (r *UserRepo) GetAvatarURLByID(id int) (string, error) {
	var url string
	err := r.db.QueryRow("SELECT avatar_url FROM users WHERE id = ?", id).Scan(&url)
	return url, err
}

func (r *UserRepo) UpdateAvatarURLByID(id int, avatarURL string) error {
	_, err := r.db.Exec("UPDATE users SET avatar_url = ? WHERE id = ?", avatarURL, id)
	return err
}

