package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/psds-microservice/api-gateway/internal/controller"
	pb "github.com/psds-microservice/api-gateway/pkg/gen"
)

// VideoStreamHandler обрабатывает HTTP запросы для видеостримов
type VideoStreamHandler struct {
	logger       *zap.Logger
	service      *controller.VideoStreamServiceImpl
	maxFrameSize int // 0 = без лимита (для обратной совместимости)
}

// NewVideoStreamHandler создает новый хендлер
func NewVideoStreamHandler(logger *zap.Logger, service *controller.VideoStreamServiceImpl, maxFrameSize int) *VideoStreamHandler {
	return &VideoStreamHandler{logger: logger, service: service, maxFrameSize: maxFrameSize}
}

// RegisterRoutes регистрирует маршруты
func (h *VideoStreamHandler) RegisterRoutes(router *gin.RouterGroup) {
	video := router.Group("/video")
	{
		video.POST("/start", h.StartStream)
		video.POST("/frame", h.SendFrame)
		video.POST("/stop", h.StopStream)
		video.GET("/active", h.GetActiveStreams)
		video.GET("/stats/:client_id", h.GetStreamStats)
		video.GET("/client/:client_id/streams", h.GetClientStreams)
		video.GET("/stream/:stream_id", h.GetStreamInfo)
		video.GET("/all-stats", h.GetAllStats)
	}
}

func (h *VideoStreamHandler) StartStream(c *gin.Context) {
	var req pb.StartStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request", zap.Error(err))
		c.JSON(400, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}
	if req.ClientId == "" {
		req.ClientId = fmt.Sprintf("client_%d", time.Now().Unix())
	}
	if req.UserId == "" {
		req.UserId = req.ClientId
	}
	if req.CameraName == "" {
		req.CameraName = "default_camera"
	}
	if req.Filename == "" {
		req.Filename = fmt.Sprintf("stream_%s_%d.mp4", req.ClientId, time.Now().Unix())
	}

	h.logger.Info("Starting stream", zap.String("client_id", req.ClientId), zap.String("camera", req.CameraName))

	response, err := h.service.StartStream(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to start stream", zap.Error(err))
		c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":    "ok",
		"stream_id": response.StreamId,
		"message":   response.Message,
		"timestamp": time.Now().Unix(),
		"details":   gin.H{"client_id": req.ClientId, "user_id": req.UserId, "camera_name": req.CameraName, "filename": req.Filename},
	})
}

func (h *VideoStreamHandler) SendFrame(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		h.handleMultipartFrame(c)
	} else {
		h.handleJSONFrame(c)
	}
}

func (h *VideoStreamHandler) handleMultipartFrame(c *gin.Context) {
	file, header, err := c.Request.FormFile("frame")
	if err != nil {
		h.logger.Error("No frame file in multipart", zap.Error(err))
		c.JSON(400, gin.H{"error": "No frame file", "message": "Please include 'frame' file in multipart form"})
		return
	}
	defer file.Close()

	frameData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read frame data", zap.Error(err))
		c.JSON(500, gin.H{"error": "Failed to read frame", "message": err.Error()})
		return
	}
	if h.maxFrameSize > 0 && len(frameData) > h.maxFrameSize {
		h.logger.Warn("Frame too large", zap.Int("size", len(frameData)), zap.Int("max", h.maxFrameSize))
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":     "Frame too large",
			"message":   "Frame size exceeds maximum allowed",
			"max_bytes": h.maxFrameSize,
		})
		return
	}

	metadataStr := c.PostForm("metadata")
	var metadata map[string]interface{}
	if metadataStr != "" {
		_ = json.Unmarshal([]byte(metadataStr), &metadata)
	}

	streamID := getStringFromMap(metadata, "stream_id", "")
	clientID := getStringFromMap(metadata, "client_id", "")
	userName := getStringFromMap(metadata, "user_name", "multipart_client")
	width := getIntFromMap(metadata, "width", 1920)
	height := getIntFromMap(metadata, "height", 1080)

	if streamID == "" {
		if clientID == "" {
			clientID = fmt.Sprintf("multipart_%d", time.Now().Unix())
		}
		streamID = fmt.Sprintf("stream_%s_%d", clientID, time.Now().UnixNano())
	}

	frame := &pb.VideoFrame{
		FrameId:   fmt.Sprintf("frame_%d", time.Now().UnixNano()),
		FrameData: frameData,
		Timestamp: time.Now().Unix(),
		ClientId:  clientID,
		CameraId:  "multipart_stream",
		Width:     int32(width),
		Height:    int32(height),
		Format:    header.Header.Get("Content-Type"),
	}

	response, err := h.service.SendFrameInternal(streamID, clientID, userName, frame)
	if err != nil {
		h.logger.Error("Failed to process frame", zap.Error(err))
		c.JSON(500, gin.H{"error": "Failed to process frame", "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":     response.Status,
		"message":    response.Message,
		"timestamp":  response.Timestamp,
		"metadata":   response.Metadata,
		"format":     "multipart",
		"frame_size": len(frameData),
		"stream_id":  streamID,
	})
}

func (h *VideoStreamHandler) handleJSONFrame(c *gin.Context) {
	var req struct {
		StreamID string                 `json:"stream_id"`
		ClientID string                 `json:"client_id"`
		UserName string                 `json:"user_name"`
		Frame    map[string]interface{} `json:"frame"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid JSON request", zap.Error(err))
		c.JSON(400, gin.H{"error": "Invalid JSON", "message": err.Error()})
		return
	}

	if req.StreamID == "" {
		if req.ClientID == "" {
			req.ClientID = fmt.Sprintf("json_%d", time.Now().Unix())
		}
		req.StreamID = fmt.Sprintf("stream_%s_%d", req.ClientID, time.Now().UnixNano())
	}
	if req.UserName == "" {
		req.UserName = req.ClientID
	}

	frameDataVal, ok := req.Frame["frame_data"]
	if !ok {
		c.JSON(400, gin.H{"error": "Invalid frame data", "message": "frame.frame_data is required"})
		return
	}
	var frameData []byte
	switch v := frameDataVal.(type) {
	case string:
		frameData = []byte(v)
	default:
		c.JSON(400, gin.H{"error": "Invalid frame data", "message": "frame.frame_data must be string or base64"})
		return
	}
	if h.maxFrameSize > 0 && len(frameData) > h.maxFrameSize {
		h.logger.Warn("Frame too large", zap.Int("size", len(frameData)), zap.Int("max", h.maxFrameSize))
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":     "Frame too large",
			"message":   "Frame size exceeds maximum allowed",
			"max_bytes": h.maxFrameSize,
		})
		return
	}

	frame := &pb.VideoFrame{
		FrameId:   fmt.Sprintf("frame_%d", time.Now().UnixNano()),
		FrameData: frameData,
		Timestamp: getInt64FromMap(req.Frame, "timestamp", time.Now().Unix()),
		ClientId:  req.ClientID,
		CameraId:  getStringFromMapInterface(req.Frame, "camera_id", "json_camera"),
		Width:     int32(getIntFromMap(req.Frame, "width", 1920)),
		Height:    int32(getIntFromMap(req.Frame, "height", 1080)),
		Format:    getStringFromMapInterface(req.Frame, "format", "jpeg"),
	}

	response, err := h.service.SendFrameInternal(req.StreamID, req.ClientID, req.UserName, frame)
	if err != nil {
		h.logger.Error("Failed to process frame", zap.Error(err))
		c.JSON(500, gin.H{"error": "Failed to process frame", "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":     response.Status,
		"message":    response.Message,
		"timestamp":  response.Timestamp,
		"metadata":   response.Metadata,
		"format":     "json_base64",
		"frame_size": len(frameData),
		"stream_id":  req.StreamID,
	})
}

func (h *VideoStreamHandler) StopStream(c *gin.Context) {
	var req struct {
		StreamID string `json:"stream_id"`
		ClientID string `json:"client_id"`
		Filename string `json:"filename"`
		EndTime  int64  `json:"end_time"`
		FileSize int64  `json:"file_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}
	if req.EndTime == 0 {
		req.EndTime = time.Now().Unix()
	}

	stopReq := &pb.StopStreamRequest{
		StreamId: req.StreamID, ClientId: req.ClientID,
		Filename: req.Filename, EndTime: req.EndTime, FileSize: req.FileSize,
	}
	response, err := h.service.StopStream(c.Request.Context(), stopReq)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": response.Status, "message": response.Message, "timestamp": response.Timestamp, "metadata": response.Metadata})
}

func (h *VideoStreamHandler) GetActiveStreams(c *gin.Context) {
	activeStreams := h.service.GetAllActiveStreams()
	streams := make([]gin.H, 0, len(activeStreams))
	for _, stream := range activeStreams {
		streams = append(streams, gin.H{
			"stream_id":    stream.StreamId,
			"client_id":    stream.ClientId,
			"user_name":    stream.UserName,
			"camera_name":  stream.CameraName,
			"is_recording": stream.IsRecording,
			"is_streaming": stream.IsStreaming,
		})
	}
	c.JSON(200, gin.H{"status": "ok", "active_streams": len(streams), "streams": streams, "timestamp": time.Now().Unix()})
}

func (h *VideoStreamHandler) GetStreamStats(c *gin.Context) {
	clientID := c.Param("client_id")
	clientStreams := h.service.GetStreamsByClient(clientID)
	stats := make([]gin.H, 0)
	for _, stream := range clientStreams {
		streamStats, err := h.service.GetStreamStats(c.Request.Context(), &pb.GetStreamStatsRequest{StreamId: stream.StreamId, ClientId: clientID})
		if err == nil {
			stats = append(stats, gin.H{
				"stream_id":       streamStats.StreamId,
				"client_id":       streamStats.ClientId,
				"start_time":      streamStats.StartTime,
				"duration":        streamStats.Duration,
				"frames_received": streamStats.FramesReceived,
				"bytes_received":  streamStats.BytesReceived,
				"average_fps":     streamStats.AverageFps,
				"current_fps":     streamStats.CurrentFps,
				"width":           streamStats.Width,
				"height":          streamStats.Height,
				"codec":           streamStats.Codec,
				"is_recording":    streamStats.IsRecording,
				"is_streaming":    streamStats.IsStreaming,
			})
		}
	}
	c.JSON(200, gin.H{"status": "ok", "client_id": clientID, "stats": stats, "timestamp": time.Now().Unix()})
}

func (h *VideoStreamHandler) GetClientStreams(c *gin.Context) {
	clientID := c.Param("client_id")
	streams := h.service.GetStreamsByClient(clientID)
	result := make([]gin.H, 0)
	for _, stream := range streams {
		result = append(result, gin.H{
			"stream_id": stream.StreamId, "client_id": stream.ClientId, "user_name": stream.UserName,
			"camera_name": stream.CameraName, "is_recording": stream.IsRecording, "is_streaming": stream.IsStreaming,
		})
	}
	c.JSON(200, gin.H{"status": "ok", "client_id": clientID, "count": len(result), "streams": result, "timestamp": time.Now().Unix()})
}

func (h *VideoStreamHandler) GetStreamInfo(c *gin.Context) {
	streamID := c.Param("stream_id")
	c.JSON(200, gin.H{
		"status": "ok", "stream_id": streamID, "message": "Stream info endpoint",
		"timestamp": time.Now().Unix(),
		"endpoints": []string{
			"/api/v1/video/start", "/api/v1/video/frame", "/api/v1/video/stop",
			"/api/v1/video/active", "/api/v1/video/stats/{client_id}", "/api/v1/video/client/{client_id}/streams",
		},
	})
}

func (h *VideoStreamHandler) GetAllStats(c *gin.Context) {
	allStats := h.service.GetAllStats()
	stats := make([]gin.H, 0)
	var totalFrames, totalBytes int64
	for _, stat := range allStats {
		totalFrames += stat.FramesReceived
		totalBytes += stat.BytesReceived
		stats = append(stats, gin.H{
			"stream_id": stat.StreamId, "client_id": stat.ClientId,
			"start_time": stat.StartTime, "duration": stat.Duration,
			"frames_received": stat.FramesReceived, "bytes_received": stat.BytesReceived,
			"average_fps": stat.AverageFps, "current_fps": stat.CurrentFps,
		})
	}
	c.JSON(200, gin.H{"status": "ok", "total_streams": len(stats), "total_frames": totalFrames, "total_bytes": totalBytes, "stats": stats, "timestamp": time.Now().Unix()})
}

func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if m == nil {
		return defaultValue
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

func getStringFromMapInterface(m map[string]interface{}, key, defaultValue string) string {
	return getStringFromMap(m, key, defaultValue)
}

func getIntFromMap(m map[string]interface{}, key string, defaultValue int) int {
	if m == nil {
		return defaultValue
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return defaultValue
}

func getInt64FromMap(m map[string]interface{}, key string, defaultValue int64) int64 {
	if m == nil {
		return defaultValue
	}
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	if v, ok := m[key].(int64); ok {
		return v
	}
	if v, ok := m[key].(int); ok {
		return int64(v)
	}
	return defaultValue
}
