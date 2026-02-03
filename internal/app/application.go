package app

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

	clientInfoService := controller.NewClientInfoService(logger)
	videoStreamService := controller.NewVideoStreamService(logger, userClient)

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
func GetUserClient(app *Application) grpc_client.UserServiceClient {
	return app.userClient
}

// GetVideoStreamService возвращает видеосервис
func GetVideoStreamService(app *Application) *controller.VideoStreamServiceImpl {
	return app.videoStreamService
}

// GetClientInfoService возвращает клиентский сервис
func GetClientInfoService(app *Application) *controller.ClientInfoServiceImpl {
	return app.clientInfoService
}

// GetConfig возвращает конфигурацию
func GetConfig(app *Application) *config.Config {
	return app.config
}

// Start запускает приложение
func (app *Application) Start() error {
	app.logger.Info("Starting application",
		zap.String("address", app.server.Addr),
		zap.String("user_service", fmt.Sprintf("%s:%d", app.config.UserService.Host, app.config.UserService.Port)))
	return app.server.ListenAndServe()
}

// Stop останавливает приложение
func (app *Application) Stop() error {
	app.logger.Info("Stopping application")
	if app.userClient != nil {
		if err := app.userClient.Close(); err != nil {
			app.logger.Error("Failed to close user service client", zap.Error(err))
		}
	}
	return app.server.Close()
}

// GetRouter возвращает роутер
func (app *Application) GetRouter() http.Handler {
	return app.router
}
