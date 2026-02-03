package command

import (
	"database/sql"
)

// Seed выполняет сиды (заглушка — при наличии database/seeds/*.sql можно реализовать)
func Seed(db *sql.DB) error {
	_ = db
	// Заглушка: database/seeds/ пуст — нечего выполнять
	return nil
}
