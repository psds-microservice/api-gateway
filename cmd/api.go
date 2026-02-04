package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/api-gateway/internal/application"
	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/psds-microservice/api-gateway/internal/grpc_server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	apiDebug    bool
	apiConfig   string
	apiGrpcPort string
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Run HTTP + gRPC API server (dual mode)",
	RunE:  runAPI,
}

func init() {
	apiCmd.Flags().BoolVar(&apiDebug, "debug", false, "Debug logging")
	apiCmd.Flags().StringVar(&apiConfig, "config", "./config/config.yaml", "Path to config.yaml")
	apiCmd.Flags().StringVar(&apiGrpcPort, "grpc-port", "9090", "gRPC port")
}

func runAPI(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	var logger *zap.Logger
	var err error
	if apiDebug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig(apiConfig)
	if err != nil {
		logger.Warn("Failed to load config", zap.Error(err))
		cfg = config.GetDefaultConfig()
	}

	app, err := application.NewApplicationWithConfig(cfg, logger)
	if err != nil {
		return fmt.Errorf("application: %w", err)
	}

	// gRPC-серверы из единого Deps (как в user-service NewServer(deps))
	deps := grpc_server.Deps{
		Video:      application.GetVideoStreamService(app),
		ClientInfo: application.GetClientInfoService(app),
		Logger:     logger,
	}
	servers := grpc_server.NewServersFromDeps(deps)

	grpcPort := apiGrpcPort
	if cfg.GRPCPort != "" {
		grpcPort = cfg.GRPCPort
	}

	return runDualServer(app, servers.Video, servers.ClientInfo, grpcPort, logger, cfg)
}

func runDualServer(
	app *application.Application,
	videoGrpcServer *grpc_server.VideoStreamServer,
	clientInfoGrpcServer *grpc_server.ClientInfoServer,
	grpcPort string,
	logger *zap.Logger,
	cfg *config.Config,
) error {
	httpErrChan := make(chan error, 1)
	grpcErrChan := make(chan error, 1)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// HTTP server
	go func() {
		if err := app.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
			select {
			case httpErrChan <- err:
			default:
			}
		}
	}()

	// gRPC server
	go func() {
		if err := grpc_server.RunGRPC(grpcPort, videoGrpcServer, clientInfoGrpcServer, logger); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
			grpcErrChan <- err
		}
	}()

	host := cfg.Host
	if host == "0.0.0.0" {
		host = "localhost"
	}
	httpBase := fmt.Sprintf("http://%s:%d", host, cfg.Port)
	httpAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	grpcAddr := ":" + grpcPort

	log.Printf("HTTP server listening on %s", httpAddr)
	log.Printf("  Swagger UI:    %s/swagger/index.html", httpBase)
	log.Printf("  Swagger spec:  %s/openapi.json", httpBase)
	log.Printf("  Health:        %s/health", httpBase)
	log.Printf("  API v1:        %s/api/v1/", httpBase)
	log.Printf("gRPC server listening on %s", grpcAddr)
	log.Printf("  gRPC endpoint: %s", grpcAddr)

	select {
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
	case err := <-httpErrChan:
		return err
	case err := <-grpcErrChan:
		return err
	}

	logger.Info("Stopping servers...")
	if err := app.Stop(); err != nil {
		logger.Error("HTTP stop error", zap.Error(err))
	}
	logger.Info("Server stopped")
	return nil
}
