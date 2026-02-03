package grpc_client

import (
	"context"

	"github.com/psds-microservice/api-gateway/internal/config"
	"go.uber.org/zap"
)

// StubUserServiceClient — заглушка для тестов и работы без user-service
type StubUserServiceClient struct {
	logger *zap.Logger
	config *config.Config
}

// NewStubUserServiceClient создаёт stub-клиент
func NewStubUserServiceClient(cfg *config.Config, logger *zap.Logger) *StubUserServiceClient {
	return &StubUserServiceClient{logger: logger, config: cfg}
}

func (c *StubUserServiceClient) Close() error {
	c.logger.Debug("Stub user-service client closed")
	return nil
}

func (c *StubUserServiceClient) GetUserByClientID(ctx context.Context, clientID string) (*UserInfo, error) {
	c.logger.Debug("Stub: Getting user by client ID", zap.String("client_id", clientID))
	return &UserInfo{
		ID:       clientID,
		Username: "user_" + clientID,
		Email:    "user_" + clientID + "@example.com",
		Status:   "active",
		StreamingConfig: &StreamingConfig{
			ServerURL:      "video-service-1.example.com",
			ServerPort:     8082,
			APIKey:         "video_api_key_" + clientID,
			StreamEndpoint: "/api/v1/video/stream",
			MaxBitrate:     5000,
			MaxResolution:  1080,
			Codec:          "h264",
			UseSSL:         false,
		},
	}, nil
}

func (c *StubUserServiceClient) GetStreamingConfig(ctx context.Context, userID string) (*StreamingConfig, error) {
	c.logger.Debug("Stub: Getting streaming config", zap.String("user_id", userID))
	return &StreamingConfig{
		ServerURL:      "video-service-1.example.com",
		ServerPort:     8082,
		APIKey:         "video_api_key_" + userID,
		StreamEndpoint: "/api/v1/video/stream",
		MaxBitrate:     5000,
		MaxResolution:  1080,
		Codec:          "h264",
		UseSSL:         false,
	}, nil
}

func (c *StubUserServiceClient) HealthCheck(ctx context.Context) error {
	c.logger.Debug("Stub: Health check")
	return nil
}
