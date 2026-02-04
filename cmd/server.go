package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/api-gateway/internal/application"
	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	serverDebug    bool
	serverConfig   string
	serverGrpcPort string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run HTTP + gRPC API server (dual mode, net/http + grpc-gateway)",
	RunE:  runServer,
}

func init() {
	serverCmd.Flags().BoolVar(&serverDebug, "debug", false, "Debug logging")
	serverCmd.Flags().StringVar(&serverConfig, "config", "", "Config path (ignored, config from .env only)")
	serverCmd.Flags().StringVar(&serverGrpcPort, "grpc-port", "9090", "gRPC port")
}

func runServer(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	var logger *zap.Logger
	var err error
	if serverDebug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer logger.Sync()

	cfg := config.Load()
	if serverGrpcPort != "" {
		cfg.GRPCPort = serverGrpcPort
	}

	app, err := application.NewAPI(cfg, logger)
	if err != nil {
		return fmt.Errorf("application: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return app.Run(ctx)
}
