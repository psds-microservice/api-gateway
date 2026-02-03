package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// Open открывает соединение с PostgreSQL
func Open(dsn string) (*sql.DB, error) {
	return sql.Open("postgres", dsn)
}
