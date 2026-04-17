package models

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNicknameTaken      = errors.New("nickname already taken")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidSession     = errors.New("invalid session")
)

