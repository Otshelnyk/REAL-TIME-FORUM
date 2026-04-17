package models

import "time"

// User represents a user in the system
type User struct {
	ID        int
	Nickname  string
	Age       int
	Gender    string
	FirstName string
	LastName  string
	Email     string
	Password  string
	AvatarURL string
}

// Category represents a post category
type Category struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	PostCount int    `json:"post_count"` // number of posts in this category
}

// Post represents a forum post
type Post struct {
	ID           int
	UserID       int
	Title        string
	Content      string
	CreatedAt    string
	Author       string
	Likes        int
	Dislikes     int
	Categories   []Category // categories for this post
	CommentCount int        // number of comments
	Comments     []Comment  // recent comments for this post
}

// Comment represents a comment on a post
type Comment struct {
	ID        int
	PostID    int
	UserID    int
	Content   string
	CreatedAt string
	Author    string
	Likes     int
	Dislikes  int
}

// Session represents a user session
type Session struct {
	ID      int
	UserID  int
	UUID    string
	Expires time.Time
}
