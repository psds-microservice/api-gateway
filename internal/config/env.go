package config

import (
	"os"
	"strconv"
)

func getEnvInt(key string, def int) int {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// ApplyEnvOverrides применяет переменные окружения поверх конфига (env переопределяет YAML)
func ApplyEnvOverrides(cfg *YamlConfig) {
	if v := getEnv("HOST", ""); v != "" {
		cfg.Host = v
	}
	if v := getEnv("HTTP_PORT", ""); v != "" {
		cfg.Port = getEnvInt("HTTP_PORT", cfg.Port)
	} else if v := getEnv("PORT", ""); v != "" {
		cfg.Port = getEnvInt("PORT", cfg.Port)
	}
	if v := getEnv("GRPC_PORT", ""); v != "" {
		cfg.GRPCPort = v
	}

	if v := getEnv("USER_SERVICE_HOST", ""); v != "" {
		cfg.UserService.Host = v
	}
	if p := getEnvInt("USER_SERVICE_PORT", 0); p != 0 {
		cfg.UserService.Port = p
	}

	if v := getEnv("DB_HOST", ""); v != "" {
		cfg.Database.Host = v
	}
	if p := getEnvInt("DB_PORT", 0); p != 0 {
		cfg.Database.Port = p
	}
	if v := getEnv("DB_USER", ""); v != "" {
		cfg.Database.User = v
	}
	if v := getEnv("DB_PASSWORD", ""); v != "" {
		cfg.Database.Password = v
	}
	if v := getEnv("DB_DATABASE", ""); v != "" {
		cfg.Database.Name = v
	}
	if v := getEnv("DB_NAME", ""); v != "" {
		cfg.Database.Name = v
	}
	if v := getEnv("DB_SSLMODE", ""); v != "" {
		cfg.Database.SSLMode = v
	}

	if v := getEnv("REDIS_HOST", ""); v != "" {
		cfg.Redis.Host = v
	}
	if p := getEnvInt("REDIS_PORT", 0); p != 0 {
		cfg.Redis.Port = p
	}
	if v := getEnv("REDIS_PASSWORD", ""); v != "" {
		cfg.Redis.Password = v
	}
	if p := getEnvInt("REDIS_DB", -1); p >= 0 {
		cfg.Redis.DB = p
	}

	if v := getEnv("JWT_SECRET", ""); v != "" {
		cfg.JWT.Secret = v
	}
	if p := getEnvInt("JWT_EXPIRATION", 0); p > 0 {
		cfg.JWT.Expiration = p
	}

	if v := getEnv("LOG_LEVEL", ""); v != "" {
		cfg.Logging.Level = v
	}
	if v := getEnv("LOG_FORMAT", ""); v != "" {
		cfg.Logging.Format = v
	}
}

// LoadConfigFromEnv собирает конфиг только из переменных окружения (для работы без YAML)
func LoadConfigFromEnv() *YamlConfig {
	cfg := GetDefaultYamlConfig()
	ApplyEnvOverrides(cfg)
	return cfg
}
