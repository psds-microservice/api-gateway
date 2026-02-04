package application

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/psds-microservice/api-gateway/internal/controller"
	"github.com/psds-microservice/api-gateway/internal/grpc_client"
	"github.com/psds-microservice/api-gateway/internal/handler"
)

// Application основное приложение (dual HTTP + gRPC)
type Application struct {
	config             *config.Config
	logger             *zap.Logger
	router             http.Handler
	server             *http.Server
	clientInfoService  *controller.ClientInfoServiceImpl
	videoStreamService *controller.VideoStreamServiceImpl
	clientInfoHandler  *handler.ClientInfoHandler
	videoStreamHandler *handler.VideoStreamHandler
	userClient         grpc_client.UserServiceClient
}

// NewApplicationWithConfig создает приложение с конфигурацией
func NewApplicationWithConfig(cfg *config.Config, logger *zap.Logger) (*Application, error) {
	userClient, err := grpc_client.NewUserServiceClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("user service client: %w", err)
	}

	// Репозитории и сервисы (DIP: сервисы зависят от интерфейсов Store)
	streamRepo := controller.NewStreamRepository()
	clientRepo := controller.NewClientRepository()
	clientInfoService := controller.NewClientInfoService(logger, clientRepo)
	videoStreamService := controller.NewVideoStreamService(logger, streamRepo, userClient)

	clientInfoHandler := handler.NewClientInfoHandler(logger, clientInfoService)
	videoStreamHandler := handler.NewVideoStreamHandler(logger, videoStreamService, cfg.Video.MaxFrameSize)

	router := NewRouter(clientInfoHandler, videoStreamHandler, logger, cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return &Application{
		config:             cfg,
		logger:             logger,
		router:             router,
		server:             server,
		clientInfoService:  clientInfoService,
		videoStreamService: videoStreamService,
		clientInfoHandler:  clientInfoHandler,
		videoStreamHandler: videoStreamHandler,
		userClient:         userClient,
	}, nil
}

// GetUserClient возвращает клиент user-service
func GetUserClient(a *Application) grpc_client.UserServiceClient {
	return a.userClient
}

// GetVideoStreamService возвращает видеосервис
func GetVideoStreamService(a *Application) *controller.VideoStreamServiceImpl {
	return a.videoStreamService
}

// GetClientInfoService возвращает клиентский сервис
func GetClientInfoService(a *Application) *controller.ClientInfoServiceImpl {
	return a.clientInfoService
}

// GetConfig возвращает конфигурацию
func GetConfig(a *Application) *config.Config {
	return a.config
}

// Start запускает приложение
func (a *Application) Start() error {
	a.logger.Info("Starting application",
		zap.String("address", a.server.Addr),
		zap.String("user_service", fmt.Sprintf("%s:%d", a.config.UserService.Host, a.config.UserService.Port)))
	return a.server.ListenAndServe()
}

// Stop останавливает приложение
func (a *Application) Stop() error {
	a.logger.Info("Stopping application")
	if a.userClient != nil {
		if err := a.userClient.Close(); err != nil {
			a.logger.Error("Failed to close user service client", zap.Error(err))
		}
	}
	return a.server.Close()
}

// GetRouter возвращает роутер
func (a *Application) GetRouter() http.Handler {
	return a.router
}
