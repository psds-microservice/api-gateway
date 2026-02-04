package application

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/psds-microservice/api-gateway/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// API — приложение dual HTTP + gRPC (по PROJECT_PROMPT: net/http + grpc-gateway).
type API struct {
	cfg     *config.Config
	httpSrv *http.Server
	grpcSrv *grpc.Server
	lis     net.Listener
}

// NewAPI создаёт приложение. Конфиг только из .env (Load).
func NewAPI(cfg *config.Config, logger *zap.Logger) (*API, error) {
	handler, grpcSrv, _, _, err := NewRouter(cfg, logger)
	if err != nil {
		return nil, err
	}

	httpAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpSrv := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	grpcPort := cfg.GRPCPort
	if grpcPort == "" {
		grpcPort = "9090"
	}
	grpcAddr := ":" + grpcPort
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("grpc listen %s: %w", grpcAddr, err)
	}

	return &API{
		cfg:     cfg,
		httpSrv: httpSrv,
		grpcSrv: grpcSrv,
		lis:     lis,
	}, nil
}

// Run запускает HTTP и gRPC серверы, блокируется до отмены ctx.
func (a *API) Run(ctx context.Context) error {
	httpAddr := a.httpSrv.Addr
	grpcAddr := a.lis.Addr().String()
	host := a.cfg.Host
	if host == "0.0.0.0" {
		host = "localhost"
	}
	httpBase := fmt.Sprintf("http://%s:%d", host, a.cfg.Port)
	log.Printf("HTTP server listening on %s", httpAddr)
	log.Printf("  Swagger UI:    %s/swagger/index.html", httpBase)
	log.Printf("  Swagger spec:  %s/openapi.json", httpBase)
	log.Printf("  Health:        %s/health", httpBase)
	log.Printf("  Ready:         %s/ready", httpBase)
	log.Printf("  API v1:        %s/api/v1/", httpBase)
	log.Printf("gRPC server listening on %s", grpcAddr)

	go func() {
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http: %v", err)
		}
	}()
	go func() {
		if err := a.grpcSrv.Serve(a.lis); err != nil {
			log.Printf("grpc: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	a.grpcSrv.GracefulStop()
	return nil
}
