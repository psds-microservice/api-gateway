package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/psds-microservice/api-gateway/api"
	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/psds-microservice/api-gateway/internal/controller"
	"github.com/psds-microservice/api-gateway/internal/grpc_client"
	"github.com/psds-microservice/api-gateway/internal/grpc_server"
	"github.com/psds-microservice/api-gateway/internal/handler"
	"github.com/psds-microservice/api-gateway/pkg/gen"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NewRouter создаёт http.Handler с net/http + grpc-gateway (по PROJECT_PROMPT, без Gin).
func NewRouter(cfg *config.Config, logger *zap.Logger) (http.Handler, *grpc.Server, *grpc_server.VideoStreamServer, *grpc_server.ClientInfoServer, error) {
	userClient, err := grpc_client.NewUserServiceClient(cfg, logger)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("user service client: %w", err)
	}

	streamRepo := controller.NewStreamRepository()
	clientRepo := controller.NewClientRepository()
	clientInfoService := controller.NewClientInfoService(logger, clientRepo)
	videoStreamService := controller.NewVideoStreamService(logger, streamRepo, userClient)

	deps := grpc_server.Deps{
		Video:      videoStreamService,
		ClientInfo: clientInfoService,
		Logger:     logger,
	}
	servers := grpc_server.NewServersFromDeps(deps)

	grpcSrv := grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)
	gen.RegisterVideoStreamServiceServer(grpcSrv, servers.Video)
	gen.RegisterClientInfoServiceServer(grpcSrv, servers.ClientInfo)
	reflection.Register(grpcSrv)

	gatewayMux := runtime.NewServeMux()
	ctx := context.Background()
	if err := gen.RegisterVideoStreamServiceHandlerServer(ctx, gatewayMux, servers.Video); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("register video gateway: %w", err)
	}
	if err := gen.RegisterClientInfoServiceHandlerServer(ctx, gatewayMux, servers.ClientInfo); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("register client_info gateway: %w", err)
	}

	rateLimiter := handler.NewRateLimitState(5, time.Second)
	rateLimited := handler.RateLimitedLimitsHandler(rateLimiter)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok", "service": "api-gateway", "version": "1.0.0", "time": time.Now().Unix(),
		})
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/openapi.json", serveOpenAPISpec())
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/openapi.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
	))

	mux.HandleFunc("/v1/limits/rate-limited", rateLimited)
	mux.HandleFunc("/api/v1/limits/rate-limited", rateLimited)

	if targetURL := cfg.UserServiceHTTPURL(); targetURL != "" {
		if u, err := url.Parse(targetURL); err == nil {
			userProxy := httputil.NewSingleHostReverseProxy(u)
			mux.Handle("/api/v1/auth/", userProxy)
			mux.Handle("/api/v1/users/", userProxy)
			mux.Handle("/api/v1/sessions/", userProxy)
			// user-service: operators/available, operators/stats, operators/availability, operators/{id}/availability
			mux.Handle("/api/v1/operators/available", userProxy)
			mux.Handle("/api/v1/operators/available/", userProxy)
			mux.Handle("/api/v1/operators/stats", userProxy)
			mux.Handle("/api/v1/operators/stats/", userProxy)
			mux.Handle("/api/v1/operators/availability", userProxy)
			mux.Handle("/api/v1/operators/availability/", userProxy)
			// operators/{user_id}/availability — один общий прокси для /api/v1/operators/ на user-service
			// не регистрируем: operator-directory обрабатывает GET/POST/PUT /api/v1/operators и /api/v1/operators/{id}
		}
	}

	// Прокси к backend-сервисам (единая точка входа).
	// /api/v1/operators/ — operator-directory (список, по id); запросы available/stats/availability уже ушли в user-service выше.
	if targetURL := cfg.UserServiceHTTPURL(); targetURL != "" {
		if userURL, err := url.Parse(targetURL); err == nil && userURL.Host != "" {
			if dirURL, err := url.Parse(cfg.OperatorDirectoryURL); err == nil && dirURL.Host != "" {
				mux.Handle("/api/v1/operators/", operatorsRouter(userURL, dirURL))
			} else {
				mux.Handle("/api/v1/operators/", httputil.NewSingleHostReverseProxy(userURL))
			}
		}
	}
	if cfg.OperatorDirectoryURL != "" {
		if u, err := url.Parse(cfg.OperatorDirectoryURL); err == nil && u.Host != "" {
			// регистрируем только если user-service не задан (иначе уже зарегистрирован operatorsRouter)
			if cfg.UserServiceHTTPURL() == "" {
				mux.Handle("/api/v1/operators/", httputil.NewSingleHostReverseProxy(u))
				mux.Handle("/api/v1/operators", httputil.NewSingleHostReverseProxy(u))
			}
		}
	}
	if u, err := url.Parse(cfg.SessionManagerURL); err == nil && u.Host != "" {
		mux.Handle("/session/", httputil.NewSingleHostReverseProxy(u))
	}
	if u, err := url.Parse(cfg.TicketServiceURL); err == nil && u.Host != "" {
		mux.Handle("/api/v1/tickets/", httputil.NewSingleHostReverseProxy(u))
		mux.Handle("/api/v1/tickets", httputil.NewSingleHostReverseProxy(u))
	}
	if u, err := url.Parse(cfg.SearchServiceURL); err == nil && u.Host != "" {
		searchProxy := httputil.NewSingleHostReverseProxy(u)
		mux.Handle("/search/", searchProxy)
		mux.Handle("/search", searchProxy)
	}
	// /api/v1/operators (без слэша) — operator-directory, если ещё не зарегистрирован operatorsRouter
	if cfg.UserServiceHTTPURL() == "" && cfg.OperatorDirectoryURL != "" {
		if u, err := url.Parse(cfg.OperatorDirectoryURL); err == nil && u.Host != "" {
			mux.Handle("/api/v1/operators/", httputil.NewSingleHostReverseProxy(u))
			mux.Handle("/api/v1/operators", httputil.NewSingleHostReverseProxy(u))
		}
	}
	if cfg.UserServiceHTTPURL() != "" && cfg.OperatorDirectoryURL != "" {
		if u, err := url.Parse(cfg.OperatorDirectoryURL); err == nil && u.Host != "" {
			mux.Handle("/api/v1/operators", httputil.NewSingleHostReverseProxy(u))
		}
	}
	if u, err := url.Parse(cfg.OperatorPoolURL); err == nil && u.Host != "" {
		mux.Handle("/operator/", httputil.NewSingleHostReverseProxy(u))
	}
	if u, err := url.Parse(cfg.NotificationServiceURL); err == nil && u.Host != "" {
		notifyProxy := httputil.NewSingleHostReverseProxy(u)
		mux.Handle("/notify/", notifyProxy)
		mux.Handle("/ws/notify/", notifyProxy)
	}
	if u, err := url.Parse(cfg.DataChannelServiceURL); err == nil && u.Host != "" {
		dataProxy := httputil.NewSingleHostReverseProxy(u)
		mux.Handle("/data/", dataProxy)
		mux.Handle("/ws/data/", dataProxy)
	}

	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "running", "timestamp": time.Now().Unix(),
			"endpoints": []string{
				"/api/v1/video/*", "/api/v1/clients/*",
				"/api/v1/auth/*", "/api/v1/tickets", "/api/v1/operators",
				"/session/*", "/search", "/search/*", "/operator/*",
				"/notify/*", "/ws/notify/*", "/data/*", "/ws/data/*",
			},
		})
	})

	mux.HandleFunc("/api/v1/test/endpoints", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok", "message": "Available test endpoints",
			"endpoints": map[string]string{
				"health": "/health", "status": "/api/v1/status",
				"start_stream": "/api/v1/video/start", "send_frame": "/api/v1/video/frame",
				"stop_stream": "/api/v1/video/stop", "active_streams": "/api/v1/video/active",
				"stream_stats": "/api/v1/video/stats/{client_id}",
			},
		})
	})

	mux.HandleFunc("/api/v1/test/auto-stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ClientID string `json:"client_id"`
			UserID   string `json:"user_id"`
			Camera   string `json:"camera"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		if req.ClientID == "" {
			req.ClientID = fmt.Sprintf("test_client_%d", time.Now().Unix())
		}
		if req.UserID == "" {
			req.UserID = req.ClientID
		}
		if req.Camera == "" {
			req.Camera = "test_camera"
		}
		streamID := fmt.Sprintf("stream_%s_%d", req.ClientID, time.Now().UnixNano())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok", "message": "Use this stream_id for testing",
			"stream_id": streamID, "client_id": req.ClientID,
		})
	})

	mux.Handle("/", gatewayMux)

	corsOpts := cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With"},
		AllowCredentials: true,
	}
	return cors.New(corsOpts).Handler(mux), grpcSrv, servers.Video, servers.ClientInfo, nil
}

func serveOpenAPISpec() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if len(api.OpenAPISpec) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write(api.OpenAPISpec)
			return
		}
		for _, path := range []string{"api/openapi.swagger.json", "api/openapi.json", "openapi.json"} {
			data, err := os.ReadFile(path)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
				return
			}
		}
		if exe, err := os.Executable(); err == nil && exe != "" {
			dir := filepath.Dir(exe)
			for _, name := range []string{"openapi.swagger.json", "openapi.json"} {
				data, err := os.ReadFile(filepath.Join(dir, "api", name))
				if err == nil {
					w.Header().Set("Content-Type", "application/json")
					w.Write(data)
					return
				}
			}
		}
		http.Error(w, "openapi.json not found. Run: make proto-openapi", http.StatusNotFound)
	}
}

// operatorsRouter направляет запросы: .../operators/{id}/availability — в user-service, остальные /api/v1/operators/* — в operator-directory.
func operatorsRouter(userServiceURL, operatorDirectoryURL *url.URL) http.Handler {
	userProxy := httputil.NewSingleHostReverseProxy(userServiceURL)
	dirProxy := httputil.NewSingleHostReverseProxy(operatorDirectoryURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")
		if strings.HasSuffix(path, "/availability") && len(path) > len("/api/v1/operators/")+len("/availability") {
			userProxy.ServeHTTP(w, r)
			return
		}
		dirProxy.ServeHTTP(w, r)
	})
}
