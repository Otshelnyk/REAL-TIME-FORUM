package sqlite

import (
	"database/sql"
)

type DB struct {
	Conn *sql.DB
}

func New(conn *sql.DB) *DB {
	return &DB{Conn: conn}
}

