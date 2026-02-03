package model

import "gorm.io/gorm"

// Base — общие поля для сущностей GORM (ID, timestamps, soft delete).
// Встраивайте в свои сущности: type MyEntity struct { model.Base; ... }
type Base struct {
	gorm.Model
}

// User — пример сущности.
type User struct {
	Base
	Email string `gorm:"size:255;not null;uniqueIndex"`
	Name  string `gorm:"size:255"`
}
