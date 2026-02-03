package main

import (
	"fmt"
	"os"

	"github.com/psds-microservice/api-gateway/internal/app"
	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/psds-microservice/api-gateway/internal/grpc_server"
	"go.uber.org/zap"
)

func runServerCommand(debug bool, configPath string, grpcPort string) error {
	var logger *zap.Logger
	var err error
	if debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		return fmt.Errorf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Warn("Failed to load config", zap.Error(err))
		cfg = config.GetDefaultConfig()
	}

	application, err := app.NewApplicationWithConfig(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create application: %v", err)
	}
	videoService := app.GetVideoStreamService(application)
	grpcServer := grpc_server.NewVideoStreamServer(videoService, logger)

	return runDualServer(application, grpcServer, grpcPort, logger, cfg)
}

func runWorkerCommand() error {
	fmt.Println("Workers not implemented yet")
	return nil
}

func runHealthCheckCommand() error {
	fmt.Println("Health check not implemented yet (use curl http://localhost:8080/health)")
	return nil
}

func runGenerateDocsCommand() error {
	fmt.Println("Documentation generation not implemented yet")
	return nil
}

func handleCommand(args []string) error {
	if len(args) == 0 {
		return runServerCommand(false, "./config/config.yaml", "9090")
	}

	command := args[0]
	switch command {
	case "server":
		debug := false
		configPath := "./config/config.yaml"
		grpcPort := "9090"

		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--debug":
				debug = true
			case "--config":
				if i+1 < len(args) {
					configPath = args[i+1]
					i++
				}
			case "--grpc-port":
				if i+1 < len(args) {
					grpcPort = args[i+1]
					i++
				}
			}
		}
		return runServerCommand(debug, configPath, grpcPort)

	case "worker":
		return runWorkerCommand()
	case "version":
		fmt.Printf("API Gateway - Version %s\nBuild: dual-api\nCommit: %s\n", Version, Commit)
		return nil
	case "health-check":
		return runHealthCheckCommand()
	case "generate-docs":
		return runGenerateDocsCommand()
	case "help":
		printHelp()
		return nil
	default:
		fmt.Printf("Неизвестная команда: %s\n", command)
		printHelp()
		os.Exit(1)
		return nil
	}
}

func printHelp() {
	fmt.Println(`
API Gateway - Командная строка

Использование:
  api-gateway [команда] [флаги]

Команды:
  server         Запустить сервер (по умолчанию)
  worker         Запустить фоновых воркеров
  version        Показать версию
  health-check   Проверить здоровье сервиса
  generate-docs  Сгенерировать документацию
  help           Показать эту справку

Флаги для server:
  --debug       Включить debug режим
  --config      Путь к config.yaml (по умолчанию: ./config/config.yaml)
  --grpc-port   Порт gRPC (по умолчанию: 9090)

Примеры:
  api-gateway server --debug
  api-gateway server --grpc-port=9091
  api-gateway version
`)
}
