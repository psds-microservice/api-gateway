package config

// Config — алиас для YamlConfig (dual HTTP+gRPC режим)
type Config = YamlConfig

// LoadConfig загружает конфигурацию: сначала YAML (если файл есть), затем применяет переопределения из .env
// Если YAML недоступен — конфиг собирается целиком из env (LoadConfigFromEnv)
func LoadConfig(path string) (*Config, error) {
	cfg, err := LoadYamlConfig(path)
	if err != nil {
		// Файла нет или ошибка — используем только env + дефолты
		return LoadConfigFromEnv(), nil
	}
	ApplyEnvOverrides(cfg)
	return cfg, nil
}

// GetDefaultConfig возвращает конфигурацию по умолчанию
func GetDefaultConfig() *Config {
	return GetDefaultYamlConfig()
}
