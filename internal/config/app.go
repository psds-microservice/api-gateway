package config

// LoadConfig загружает конфигурацию только из .env (12-factor, PROJECT_PROMPT).
// Вызывает godotenv.Load() перед Load() — вызывающий код должен сделать _ = godotenv.Load().
func LoadConfig(_ string) (*Config, error) {
	return Load(), nil
}
