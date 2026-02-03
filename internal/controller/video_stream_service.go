package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/psds-microservice/api-gateway/internal/grpc_client"
	pb "github.com/psds-microservice/api-gateway/pkg/gen"
	"github.com/psds-microservice/helpy"
	"go.uber.org/zap"
)

// VideoStreamServiceImpl сервис управления видеостримами
type VideoStreamServiceImpl struct {
	repo       *StreamRepository
	logger     *zap.Logger
	userClient grpc_client.UserServiceClient
	mu         sync.RWMutex
}

// NewVideoStreamService создает новый сервис
func NewVideoStreamService(logger *zap.Logger, userClient grpc_client.UserServiceClient) *VideoStreamServiceImpl {
	return &VideoStreamServiceImpl{
		repo:       NewStreamRepository(),
		logger:     logger,
		userClient: userClient,
	}
}

func (s *VideoStreamServiceImpl) StartStream(ctx context.Context, req *pb.StartStreamRequest) (*pb.StartStreamResponse, error) {
	s.logger.Info("Starting stream",
		zap.String("client_id", req.ClientId),
		zap.String("camera", req.CameraName))

	userName := req.UserId
	if s.userClient != nil {
		user, err := s.userClient.GetUserByClientID(ctx, req.ClientId)
		if err != nil {
			return nil, fmt.Errorf("user not found or unauthorized: %w", err)
		}
		userName = user.Username
		_, _ = s.userClient.GetStreamingConfig(ctx, req.ClientId)
	}
	if userName == "" {
		userName = req.ClientId
	}

	streamID := fmt.Sprintf("stream_%s_%d", req.ClientId, time.Now().UnixNano())

	activeStream := &pb.ActiveStream{
		StreamId:    streamID,
		ClientId:    req.ClientId,
		UserName:    userName,
		CameraName:  req.CameraName,
		IsRecording: true,
		IsStreaming: true,
	}
	s.repo.SaveStream(streamID, activeStream)

	return &pb.StartStreamResponse{
		StreamId: streamID,
		Status:   "started",
		Message:  fmt.Sprintf("Stream %s started", streamID),
	}, nil
}

func (s *VideoStreamServiceImpl) SendFrame(ctx context.Context, req *pb.SendFrameRequest) (*helpy.ApiResponse, error) {
	streamID := req.StreamId
	clientID := req.ClientId
	userName := req.UserName
	if userName == "" {
		userName = clientID
	}
	return s.SendFrameInternal(streamID, clientID, userName, req.Frame)
}

// SendFrameInternal внутренний метод обработки кадра
func (s *VideoStreamServiceImpl) SendFrameInternal(streamID, clientID, userName string, frame *pb.VideoFrame) (*helpy.ApiResponse, error) {
	if frame == nil {
		return &helpy.ApiResponse{Status: "error", Message: "Frame is nil"}, nil
	}

	s.mu.RLock()
	stream := s.repo.GetStream(streamID)
	s.mu.RUnlock()

	if stream == nil {
		s.logger.Info("Auto-creating stream",
			zap.String("stream_id", streamID),
			zap.String("client_id", clientID))

		userNameToUse := userName
		if s.userClient != nil {
			user, err := s.userClient.GetUserByClientID(context.Background(), clientID)
			if err != nil {
				return &helpy.ApiResponse{Status: "error", Message: fmt.Sprintf("User validation failed: %v", err)}, nil
			}
			userNameToUse = user.Username
		}
		if userNameToUse == "" {
			userNameToUse = clientID
		}

		activeStream := &pb.ActiveStream{
			StreamId:    streamID,
			ClientId:    clientID,
			UserName:    userNameToUse,
			CameraName:  "auto_created",
			IsRecording: true,
			IsStreaming: true,
		}
		s.mu.Lock()
		s.repo.SaveStream(streamID, activeStream)
		s.mu.Unlock()
	}

	stats := s.repo.UpdateStats(streamID, frame)
	s.logger.Debug("Frame received",
		zap.String("stream_id", streamID),
		zap.String("client_id", clientID),
		zap.Int64("frame_size", int64(len(frame.FrameData))),
		zap.Int64("total_frames", stats.FramesReceived),
		zap.Int64("total_bytes", stats.BytesReceived))

	return &helpy.ApiResponse{
		Status:    "ok",
		Message:   "Frame received",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"stream_id":       streamID,
			"client_id":       clientID,
			"frame_id":        frame.FrameId,
			"frames_received": fmt.Sprintf("%d", stats.FramesReceived),
			"bytes_received":  fmt.Sprintf("%d", stats.BytesReceived),
			"source":          "video_service",
		},
	}, nil
}

func (s *VideoStreamServiceImpl) StopStream(ctx context.Context, req *pb.StopStreamRequest) (*helpy.ApiResponse, error) {
	s.logger.Info("Stopping stream",
		zap.String("stream_id", req.StreamId),
		zap.String("client_id", req.ClientId))
	s.repo.RemoveStream(req.StreamId)
	return &helpy.ApiResponse{
		Status:    "ok",
		Message:   fmt.Sprintf("Stream %s stopped", req.StreamId),
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"stream_id": req.StreamId,
			"client_id": req.ClientId,
			"end_time":  fmt.Sprintf("%d", req.EndTime),
			"file_size": fmt.Sprintf("%d", req.FileSize),
			"filename":  req.Filename,
		},
	}, nil
}

func (s *VideoStreamServiceImpl) GetStreamStats(ctx context.Context, req *pb.GetStreamStatsRequest) (*pb.StreamStats, error) {
	stats := s.repo.GetStats(req.StreamId)
	if stats == nil {
		return nil, fmt.Errorf("stream %s not found", req.StreamId)
	}
	return stats, nil
}

func (s *VideoStreamServiceImpl) GetAllActiveStreams() []*pb.ActiveStream {
	return s.repo.GetAllActiveStreams()
}

func (s *VideoStreamServiceImpl) GetAllStats() []*pb.StreamStats {
	return s.repo.GetAllStats()
}

func (s *VideoStreamServiceImpl) GetStreamsByClient(clientID string) []*pb.ActiveStream {
	allStreams := s.repo.GetAllStreams()
	var clientStreams []*pb.ActiveStream
	for _, stream := range allStreams {
		if stream.ClientId == clientID {
			clientStreams = append(clientStreams, stream)
		}
	}
	return clientStreams
}

func (s *VideoStreamServiceImpl) GetActiveStreamsCount() int {
	return len(s.repo.GetAllActiveStreams())
}

func (s *VideoStreamServiceImpl) GetTotalStats() map[string]interface{} {
	allStats := s.repo.GetAllStats()
	var totalFrames, totalBytes int64
	for _, stats := range allStats {
		totalFrames += stats.FramesReceived
		totalBytes += stats.BytesReceived
	}
	return map[string]interface{}{
		"active_streams": len(allStats),
		"total_frames":   totalFrames,
		"total_bytes":    totalBytes,
		"average_fps":    calculateAverageFPS(allStats),
		"timestamp":      time.Now().Unix(),
	}
}

func calculateAverageFPS(stats []*pb.StreamStats) float32 {
	if len(stats) == 0 {
		return 0
	}
	var totalFPS float32
	for _, s := range stats {
		totalFPS += s.AverageFps
	}
	return totalFPS / float32(len(stats))
}
