package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"go.uber.org/zap"

	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/psds-microservice/api-gateway/internal/handler"
)

// NewRouter создает роутер с маршрутами (dual HTTP + gRPC режим)
func NewRouter(
	clientInfoHandler *handler.ClientInfoHandler,
	videoStreamHandler *handler.VideoStreamHandler,
	logger *zap.Logger,
	cfg *config.Config,
) http.Handler {

	if gin.Mode() == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Лимит размера multipart для загрузки кадров (из конфига)
	maxMultipart := int64(cfg.Video.MaxFrameSize)
	if maxMultipart <= 0 {
		maxMultipart = 10 << 20 // 10 MB по умолчанию
	}
	router.MaxMultipartMemory = maxMultipart

	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			logger.Info("HTTP Request",
				zap.String("method", param.Method),
				zap.String("path", param.Path),
				zap.Int("status", param.StatusCode),
				zap.Duration("latency", param.Latency),
				zap.String("client_ip", param.ClientIP))
			return ""
		},
	}))
	router.Use(gin.Recovery())

	corsOpts := cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With"},
		AllowCredentials: true,
	}

	router.Static("/static", "./static")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "api-gateway",
			"version": "1.0.0",
			"time":    time.Now().Unix(),
		})
	})

	apiV1 := router.Group("/api/v1")
	{
		// Лимит тела запроса для JSON-кадров (base64 ~1.33x от размера кадра)
		maxBody := int64(cfg.Video.MaxFrameSize) * 2
		if maxBody <= 0 {
			maxBody = 20 << 20 // 20 MB по умолчанию
		}
		apiV1.Use(bodyLimitMiddleware(maxBody))

		clientInfoHandler.RegisterRoutes(apiV1)
		videoStreamHandler.RegisterRoutes(apiV1)

		apiV1.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "running",
				"timestamp": time.Now().Unix(),
				"endpoints": []string{
					"/api/v1/video/start", "/api/v1/video/frame", "/api/v1/video/stop",
					"/api/v1/video/active", "/api/v1/video/stats/{client_id}", "/api/v1/video/client/{client_id}/streams",
					"/api/v1/video/stream/{stream_id}",
				},
			})
		})

		apiV1.GET("/test/endpoints", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"message": "Available test endpoints",
				"endpoints": map[string]string{
					"health":         "/health",
					"status":         "/api/v1/status",
					"start_stream":   "/api/v1/video/start",
					"send_frame":     "/api/v1/video/frame",
					"stop_stream":    "/api/v1/video/stop",
					"active_streams": "/api/v1/video/active",
					"stream_stats":   "/api/v1/video/stats/{client_id}",
				},
				"example_request": map[string]interface{}{
					"send_frame": map[string]interface{}{
						"method": "POST",
						"url":    "/api/v1/video/frame",
						"body": map[string]interface{}{
							"stream_id": "stream_user_001_123456789",
							"client_id": "user_001",
							"user_name": "Test User",
							"frame": map[string]interface{}{
								"frame_data": "base64_encoded_image_data",
								"timestamp":  time.Now().Unix(),
								"width":      1920,
								"height":     1080,
								"format":     "jpeg",
							},
						},
					},
				},
			})
		})

		apiV1.POST("/test/auto-stream", func(c *gin.Context) {
			var req struct {
				ClientID string `json:"client_id"`
				UserID   string `json:"user_id"`
				Camera   string `json:"camera"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
				return
			}
			if req.ClientID == "" {
				req.ClientID = "test_client_" + fmt.Sprintf("%d", time.Now().Unix())
			}
			if req.UserID == "" {
				req.UserID = req.ClientID
			}
			if req.Camera == "" {
				req.Camera = "test_camera"
			}
			streamID := fmt.Sprintf("stream_%s_%d", req.ClientID, time.Now().UnixNano())

			c.JSON(http.StatusOK, gin.H{
				"status":    "ok",
				"message":   "Use this stream_id for testing",
				"stream_id": streamID,
				"client_id": req.ClientID,
				"endpoints": map[string]string{
					"send_frame":      "/api/v1/video/frame",
					"stop_stream":     "/api/v1/video/stop",
					"get_stats":       fmt.Sprintf("/api/v1/video/stats/%s", req.ClientID),
					"get_stream_info": fmt.Sprintf("/api/v1/video/stream/%s", streamID),
				},
			})
		})
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "The requested resource was not found",
			"path":    c.Request.URL.Path,
			"suggestions": []string{
				"Check /health for service status",
				"Check /api/v1/status for API status",
				"Check /api/v1/test/endpoints for available endpoints",
			},
		})
	})

	return cors.New(corsOpts).Handler(router)
}

// bodyLimitMiddleware ограничивает размер тела запроса (защита от OOM)
func bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "Request entity too large",
				"message": fmt.Sprintf("Request body must not exceed %d bytes", maxBytes),
			})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
