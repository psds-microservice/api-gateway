package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	// Загружаем .env из текущей директории (если файл есть)
	_ = godotenv.Load()

	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		if err := handleCommand(os.Args[1:]); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}
		return
	}

	mode := flag.String("mode", "server", "Режим: server, worker, version")
	debug := flag.Bool("debug", false, "Debug режим")
	configPath := flag.String("config", "./config/config.yaml", "Путь к config.yaml")
	grpcPort := flag.String("grpc-port", "9090", "Порт gRPC")

	flag.Parse()

	switch *mode {
	case "server":
		if err := runServerCommand(*debug, *configPath, *grpcPort); err != nil {
			fmt.Printf("Ошибка запуска сервера: %v\n", err)
			os.Exit(1)
		}
	case "worker":
		runWorkerCommand()
	case "version":
		fmt.Printf("API Gateway\nVersion: %s\nCommit: %s\nBuild Date: %s\n", Version, Commit, BuildDate)
	default:
		fmt.Printf("Неизвестный режим: %s\n", *mode)
		os.Exit(1)
	}
}
