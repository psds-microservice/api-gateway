package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

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

// Config — конфигурация из .env (12-factor).
type Config struct {
	Host     string
	Port     int
	GRPCPort string

	UserService struct {
		Host              string
		Port              int
		HTTPPort          int
		DialTimeoutSec    int
		RequestTimeoutSec int
		MaxRetries        int
		RetryDelaySec     int
	}

	// Backend HTTP base URLs for reverse proxy (optional; empty = proxy not registered).
	SessionManagerURL      string // e.g. http://localhost:8091
	TicketServiceURL       string // e.g. http://localhost:8093
	SearchServiceURL       string // e.g. http://localhost:8099
	OperatorDirectoryURL   string // e.g. http://localhost:8098
	OperatorPoolURL        string // e.g. http://localhost:8094
	NotificationServiceURL string // e.g. http://localhost:8092
	DataChannelServiceURL  string // e.g. http://localhost:8097

	Database struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
		SSLMode  string
	}

	Redis struct {
		Host     string
		Port     int
		Password string
		DB       int
	}

	JWT struct {
		Secret     string
		Expiration int
	}

	Logging struct {
		Level  string
		Format string
	}

	Video struct {
		MaxFrameSize int
		MaxFPS       int
		Codec        string
	}
}

// Load загружает конфигурацию из переменных окружения (.env через godotenv).
func Load() *Config {
	cfg := &Config{
		Host:     getEnv("APP_HOST", getEnv("HOST", "0.0.0.0")),
		Port:     getEnvInt("SERVER_PORT", 8080),
		GRPCPort: getEnv("GRPC_PORT", "9090"),
	}
	cfg.UserService.Host = getEnv("USER_SERVICE_HOST", "localhost")
	cfg.UserService.Port = getEnvInt("USER_SERVICE_PORT", 9090)
	cfg.UserService.HTTPPort = getEnvInt("USER_SERVICE_HTTP_PORT", 8080)
	cfg.UserService.DialTimeoutSec = getEnvInt("USER_SERVICE_DIAL_TIMEOUT_SEC", 10)
	cfg.UserService.RequestTimeoutSec = getEnvInt("USER_SERVICE_REQUEST_TIMEOUT_SEC", 5)
	cfg.UserService.MaxRetries = getEnvInt("USER_SERVICE_MAX_RETRIES", 3)
	cfg.UserService.RetryDelaySec = getEnvInt("USER_SERVICE_RETRY_DELAY_SEC", 1)

	cfg.SessionManagerURL = getEnv("SESSION_MANAGER_URL", "")
	cfg.TicketServiceURL = getEnv("TICKET_SERVICE_URL", "")
	cfg.SearchServiceURL = getEnv("SEARCH_SERVICE_URL", "")
	cfg.OperatorDirectoryURL = getEnv("OPERATOR_DIRECTORY_URL", "")
	cfg.OperatorPoolURL = getEnv("OPERATOR_POOL_URL", "")
	cfg.NotificationServiceURL = getEnv("NOTIFICATION_SERVICE_URL", "")
	cfg.DataChannelServiceURL = getEnv("DATA_CHANNEL_SERVICE_URL", "")

	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnvInt("DB_PORT", 5432)
	cfg.Database.User = getEnv("DB_USER", "postgres")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	cfg.Database.Name = getEnv("DB_DATABASE", getEnv("DB_NAME", "api_gateway"))
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", "disable")

	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getEnvInt("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvInt("REDIS_DB", 0)

	cfg.JWT.Secret = getEnv("JWT_SECRET", "change-me-in-production")
	cfg.JWT.Expiration = getEnvInt("JWT_EXPIRATION", 24)

	cfg.Logging.Level = getEnv("LOG_LEVEL", "info")
	cfg.Logging.Format = getEnv("LOG_FORMAT", "json")

	cfg.Video.MaxFrameSize = getEnvInt("VIDEO_MAX_FRAME_SIZE", 10*1024*1024)
	cfg.Video.MaxFPS = getEnvInt("VIDEO_MAX_FPS", 30)
	cfg.Video.Codec = getEnv("VIDEO_CODEC", "h264")
	return cfg
}

// DSN возвращает connection string для lib/pq
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password, c.Database.Name, c.Database.SSLMode)
}

// DatabaseURL возвращает postgres URL для golang-migrate
func (c *Config) DatabaseURL() string {
	pass := url.QueryEscape(c.Database.Password)
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User, pass, c.Database.Host, c.Database.Port, c.Database.Name, c.Database.SSLMode)
}

// UserServiceHTTPURL возвращает базовый URL HTTP API user-service (для прокси auth).
func (c *Config) UserServiceHTTPURL() string {
	port := c.UserService.HTTPPort
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("http://%s:%d", c.UserService.Host, port)
}
