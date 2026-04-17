package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func ConnectDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "forum.db")
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect db: %w", err)
	}
	return db, nil
}

func RunMigrationDB(db *sql.DB) error {
	pathToFile := filepath.Join("migration", "001_init.sql")

	file, err := os.ReadFile(pathToFile)
	if err != nil {
		return fmt.Errorf("run migration: %w", err)
	}

	_, err = db.Exec(string(file))
	if err != nil {
		return fmt.Errorf("run migration: %w", err)
	}
	return nil
}
