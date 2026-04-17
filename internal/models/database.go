package models

import (
	"database/sql"
	"log"
)

// InitSchema creates tables and default data on the provided db connection.
func InitSchema(db *sql.DB) {
	createTables(db)
	insertDefaultCategories(db)
}

// createTables creates all necessary database tables
func createTables(db *sql.DB) {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nickname TEXT UNIQUE NOT NULL,
		age INTEGER NOT NULL DEFAULT 0,
		gender TEXT NOT NULL DEFAULT '',
		first_name TEXT NOT NULL DEFAULT '',
		last_name TEXT NOT NULL DEFAULT '',
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);
	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		title TEXT,
		content TEXT,
		created_at DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS post_categories (
		post_id INTEGER,
		category_id INTEGER,
		FOREIGN KEY(post_id) REFERENCES posts(id),
		FOREIGN KEY(category_id) REFERENCES categories(id)
	);
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER,
		user_id INTEGER,
		content TEXT,
		created_at DATETIME,
		FOREIGN KEY(post_id) REFERENCES posts(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS post_likes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER,
		user_id INTEGER,
		is_like BOOLEAN,
		FOREIGN KEY(post_id) REFERENCES posts(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS comment_likes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		comment_id INTEGER,
		user_id INTEGER,
		is_like BOOLEAN,
		FOREIGN KEY(comment_id) REFERENCES comments(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		uuid TEXT UNIQUE,
		expires DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS private_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_id INTEGER NOT NULL,
		to_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(from_id) REFERENCES users(id),
		FOREIGN KEY(to_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		actor_id INTEGER DEFAULT 0,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		link TEXT DEFAULT '',
		is_read BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read);
	`)
	if err != nil {
		log.Fatal(err)
	}
	migrateUsersTable(db)
}

// migrateUsersTable adds new user columns if old schema exists (username -> nickname, etc.)
func migrateUsersTable(db *sql.DB) {
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return
	}
	hasNickname := false
	hasAvatarURL := false
	for rows.Next() {
		var cid interface{}
		var cname string
		var ctype string
		var notnull, pk interface{}
		_ = rows.Scan(&cid, &cname, &ctype, &notnull, &pk, &pk)
		if cname == "nickname" {
			hasNickname = true
		}
		if cname == "avatar_url" {
			hasAvatarURL = true
		}
	}
	rows.Close()
	if !hasNickname {
		// Old schema: add new columns
		db.Exec("ALTER TABLE users ADD COLUMN nickname TEXT DEFAULT ''")
		db.Exec("ALTER TABLE users ADD COLUMN age INTEGER DEFAULT 0")
		db.Exec("ALTER TABLE users ADD COLUMN gender TEXT DEFAULT ''")
		db.Exec("ALTER TABLE users ADD COLUMN first_name TEXT DEFAULT ''")
		db.Exec("ALTER TABLE users ADD COLUMN last_name TEXT DEFAULT ''")
		db.Exec("UPDATE users SET nickname = username WHERE nickname = '' OR nickname IS NULL")
	}
	if !hasAvatarURL {
		db.Exec("ALTER TABLE users ADD COLUMN avatar_url TEXT DEFAULT ''")
	}
}

// insertDefaultCategories inserts default categories if they don't exist
func insertDefaultCategories(db *sql.DB) {
	categories := []string{
		"Animals & Pets", "Arts", "Business", "Education & Career", "Fashion & Beauty",
		"Food & Drinks", "Funny", "Games", "Home & Garden", "Humanities & Law",
		"Interesting", "Memes", "Movies & TV", "Music", "Nature & Outdoors",
		"News & Politics", "Places & Travel", "Pop Culture", "Programming", "Q&As",
		"Science", "Spooky", "Sports", "Technology", "Vehicles", "Wellness",
	}

	// Check if categories already exist
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count); err != nil {
		log.Fatal("Error checking categories count:", err)
	}

	// Only insert if no categories exist
	if count == 0 {
		for _, category := range categories {
			if _, err := db.Exec("INSERT INTO categories (name) VALUES (?)", category); err != nil {
				log.Fatal("Error inserting category:", category, err)
			}
		}
	}
}
