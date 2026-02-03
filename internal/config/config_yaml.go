package config

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

// YamlConfig представляет конфигурацию приложения из YAML (dual HTTP+gRPC режим)
type YamlConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	GRPCPort string `yaml:"grpc_port"`

	UserService struct {
		Host              string `yaml:"host"`
		Port              int    `yaml:"port"`
		DialTimeoutSec    int    `yaml:"dial_timeout_sec"`
		RequestTimeoutSec int    `yaml:"request_timeout_sec"`
		MaxRetries        int    `yaml:"max_retries"`
		RetryDelaySec     int    `yaml:"retry_delay_sec"`
	} `yaml:"user_service"`

	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
		SSLMode  string `yaml:"ssl_mode"`
	} `yaml:"database"`

	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	JWT struct {
		Secret     string `yaml:"secret"`
		Expiration int    `yaml:"expiration"`
	} `yaml:"jwt"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`

	Video struct {
		MaxFrameSize int    `yaml:"max_frame_size"`
		MaxFPS       int    `yaml:"max_fps"`
		Codec        string `yaml:"codec"`
	} `yaml:"video"`
}

// LoadYamlConfig загружает конфигурацию из YAML файла
func LoadYamlConfig(path string) (*YamlConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg YamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetDefaultYamlConfig возвращает конфигурацию по умолчанию
func GetDefaultYamlConfig() *YamlConfig {
	cfg := &YamlConfig{
		Host:     "localhost",
		Port:     8080,
		GRPCPort: "9090",
	}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.Name = "api_gateway"
	cfg.Database.SSLMode = "disable"
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.Password = ""
	cfg.Redis.DB = 0
	cfg.JWT.Secret = "your-secret-key-change-in-production"
	cfg.JWT.Expiration = 24
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Video.MaxFrameSize = 10 * 1024 * 1024
	cfg.Video.MaxFPS = 30
	cfg.Video.Codec = "h264"
	cfg.UserService.Host = "localhost"
	cfg.UserService.Port = 9091
	cfg.UserService.DialTimeoutSec = 10
	cfg.UserService.RequestTimeoutSec = 5
	cfg.UserService.MaxRetries = 3
	cfg.UserService.RetryDelaySec = 1
	return cfg
}

// DSN возвращает connection string для lib/pq
func (c *YamlConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password, c.Database.Name, c.Database.SSLMode)
}

// DatabaseURL возвращает postgres URL для golang-migrate
func (c *YamlConfig) DatabaseURL() string {
	pass := url.QueryEscape(c.Database.Password)
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User, pass, c.Database.Host, c.Database.Port, c.Database.Name, c.Database.SSLMode)
}
