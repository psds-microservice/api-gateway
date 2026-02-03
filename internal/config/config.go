package config

import (
	"fmt"
	"os"
)

// EnvConfig — конфигурация из переменных окружения (простой HTTP-режим)
type EnvConfig struct {
	HTTPPort string
	DB       struct {
		Host     string
		Port     string
		User     string
		Password string
		Database string
		SSLMode  string
	}
}

// LoadEnv загружает конфигурацию из env
func LoadEnv() (*EnvConfig, error) {
	c := &EnvConfig{
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		DB: struct {
			Host     string
			Port     string
			User     string
			Password string
			Database string
			SSLMode  string
		}{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_DATABASE", "app"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
	return c, nil
}

func (c *EnvConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Database, c.DB.SSLMode)
}

func (c *EnvConfig) GetHTTPPort() string {
	if c.HTTPPort == "" {
		return "8080"
	}
	return c.HTTPPort
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
