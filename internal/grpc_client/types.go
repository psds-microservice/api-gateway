package grpc_client

// UserInfo — данные пользователя (совместимость с user-service)
type UserInfo struct {
	ID              string
	Username        string
	Email           string
	Status          string
	StreamingConfig *StreamingConfig
}

// StreamingConfig — конфигурация стриминга пользователя
type StreamingConfig struct {
	ServerURL      string
	ServerPort     int
	APIKey         string
	StreamEndpoint string
	MaxBitrate     int
	MaxResolution  int
	Codec          string
	UseSSL         bool
}
