package models

import "time"

type User struct {
	ID        int64
	Nickname  string
	Age       int
	Gender    string
	FirstName string
	LastName  string
	Email     string
	Password  string // hashed
}

type Category struct {
	ID   int64
	Name string
}

type Post struct {
	ID        int64
	UserID    int64
	Title     string
	Content   string
	CreatedAt time.Time
}

type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Content   string
	CreatedAt time.Time
}

type Session struct {
	UserID    int64
	UUID      string
	ExpiresAt time.Time
}

type PrivateMessage struct {
	ID        int64
	FromID    int64
	ToID      int64
	Content   string
	CreatedAt time.Time
}

type Notification struct {
	ID        int64
	UserID    int64
	ActorID   int64
	Type      string
	Title     string
	Message   string
	Link      string
	IsRead    bool
	CreatedAt time.Time
}
