package grpc_client

import (
	"context"

	"github.com/psds-microservice/api-gateway/internal/config"
	"go.uber.org/zap"
)

// UserServiceClient — интерфейс клиента user-service
type UserServiceClient interface {
	Close() error
	GetUserByClientID(ctx context.Context, clientID string) (*UserInfo, error)
	GetStreamingConfig(ctx context.Context, userID string) (*StreamingConfig, error)
	HealthCheck(ctx context.Context) error
}

// NewUserServiceClient создаёт клиента (stub, если реальный сервис недоступен)
func NewUserServiceClient(cfg *config.Config, logger *zap.Logger) (UserServiceClient, error) {
	// Пока всегда возвращаем stub; реальный клиент можно добавить с зависимостью user-service
	logger.Info("Using stub user-service client",
		zap.String("host", cfg.UserService.Host),
		zap.Int("port", cfg.UserService.Port))
	return NewStubUserServiceClient(cfg, logger), nil
}
