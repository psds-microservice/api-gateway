package grpc_client

import (
	"context"
	"fmt"
	"time"

	"github.com/psds-microservice/api-gateway/internal/config"
	uspb "github.com/psds-microservice/user-service/pkg/gen/user_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserServiceClient — интерфейс клиента user-service
type UserServiceClient interface {
	Close() error
	GetUserByClientID(ctx context.Context, clientID string) (*UserInfo, error)
	GetStreamingConfig(ctx context.Context, userID string) (*StreamingConfig, error)
	HealthCheck(ctx context.Context) error
}

// grpcUserServiceClient — реальный gRPC‑клиент user-service
type grpcUserServiceClient struct {
	conn   *grpc.ClientConn
	client uspb.UserServiceClient
	logger *zap.Logger
	cfg    *config.Config
}

// NewUserServiceClient создаёт gRPC‑клиент (при ошибке — stub)
func NewUserServiceClient(cfg *config.Config, logger *zap.Logger) (UserServiceClient, error) {
	addr := fmt.Sprintf("%s:%d", cfg.UserService.Host, cfg.UserService.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.Warn("Failed to connect user-service via gRPC, using stub client",
			zap.String("addr", addr),
			zap.Error(err))
		return NewStubUserServiceClient(cfg, logger), nil
	}

	logger.Info("Connected to user-service via gRPC", zap.String("addr", addr))
	client := uspb.NewUserServiceClient(conn)

	return &grpcUserServiceClient{
		conn:   conn,
		client: client,
		logger: logger,
		cfg:    cfg,
	}, nil
}

func (c *grpcUserServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetUserByClientID — для простоты считаем, что clientID == user_id (UUID)
func (c *grpcUserServiceClient) GetUserByClientID(ctx context.Context, clientID string) (*UserInfo, error) {
	resp, err := c.client.GetUser(ctx, &uspb.GetUserRequest{Id: clientID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("user %s not found", clientID)
	}

	return &UserInfo{
		ID:       resp.Id,
		Username: resp.Username,
		Email:    resp.Email,
		Status:   resp.Status,
		// StreamingConfig сейчас не хранится в user-service — заполняем nil.
		StreamingConfig: nil,
	}, nil
}

// GetStreamingConfig — пока user-service не даёт отдельный стриминг‑конфиг,
// возвращаем базовый статический конфиг (как раньше в stub).
func (c *grpcUserServiceClient) GetStreamingConfig(ctx context.Context, userID string) (*StreamingConfig, error) {
	_ = ctx
	return &StreamingConfig{
		ServerURL:      "video-service-1",
		ServerPort:     8080,
		APIKey:         "user-" + userID,
		StreamEndpoint: "/api/v1/video/frame",
		MaxBitrate:     5000,
		MaxResolution:  1080,
		Codec:          "h264",
		UseSSL:         false,
	}, nil
}

// HealthCheck — пробуем простой gRPC‑вызов (GetAvailableOperators limit=1)
func (c *grpcUserServiceClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.client.GetAvailableOperators(ctx, &uspb.GetAvailableOperatorsRequest{
		Limit:  1,
		Offset: 0,
	})
	return err
}
