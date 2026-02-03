package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/psds-microservice/api-gateway/internal/app"
	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/psds-microservice/api-gateway/internal/grpc_server"
	"go.uber.org/zap"
)

func runDualServer(
	application *app.Application,
	grpcServer *grpc_server.VideoStreamServer,
	grpcPort string,
	logger *zap.Logger,
	cfg *config.Config,
) error {
	httpErrChan := make(chan error, 1)
	grpcErrChan := make(chan error, 1)

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer stop()

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		logger.Info("Запуск HTTP сервера", zap.String("address", fmt.Sprintf("http://%s", addr)))
		if err := application.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP сервер завершился с ошибкой", zap.Error(err))
			httpErrChan <- err
		}
	}()

	go func() {
		logger.Info("Запуск gRPC сервера", zap.String("address", fmt.Sprintf(":%s", grpcPort)))
		if err := grpcServer.Run(grpcPort); err != nil {
			logger.Error("gRPC сервер завершился с ошибкой", zap.Error(err))
			grpcErrChan <- err
		}
	}()

	logger.Info("Сервис запущен в dual режиме (HTTP + gRPC)")
	logger.Info(fmt.Sprintf("  HTTP REST API: http://%s:%d", cfg.Host, cfg.Port))
	logger.Info("  gRPC endpoint: localhost:" + grpcPort)
	logger.Info(fmt.Sprintf("  Health check: http://%s:%d/health", cfg.Host, cfg.Port))

	select {
	case <-ctx.Done():
		logger.Info("Получен сигнал завершения...")
	case <-httpErrChan:
	case <-grpcErrChan:
	}

	logger.Info("Остановка серверов...")
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := application.Stop(); err != nil {
		logger.Error("Ошибка при остановке HTTP сервера", zap.Error(err))
	}
	logger.Info("Сервис остановлен корректно")
	return nil
}
