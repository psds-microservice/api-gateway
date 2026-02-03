package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open создаёт подключение к PostgreSQL.
func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
