package grpc_server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/psds-microservice/api-gateway/internal/controller"
	apperrors "github.com/psds-microservice/api-gateway/internal/errors"
	pb "github.com/psds-microservice/api-gateway/pkg/gen"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Logger — минимальный интерфейс логгера для Deps (D: зависимость от абстракции).
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

// Deps — зависимости gRPC-серверов (D: все зависимости — интерфейсы).
type Deps struct {
	Video      controller.VideoStreamService
	ClientInfo controller.ClientInfoService
	Logger     Logger
}

// Servers — пара gRPC-серверов (VideoStream + ClientInfo), создаётся из Deps.
type Servers struct {
	Video      *VideoStreamServer
	ClientInfo *ClientInfoServer
}

// NewServersFromDeps создаёт оба gRPC-сервера из Deps (как NewServer(deps) в user-service).
func NewServersFromDeps(deps Deps) *Servers {
	return &Servers{
		Video:      NewVideoStreamServer(deps.Video, deps.Logger),
		ClientInfo: NewClientInfoServer(deps.ClientInfo),
	}
}

// VideoStreamServer реализует gRPC сервер для видеостримов
type VideoStreamServer struct {
	pb.UnimplementedVideoStreamServiceServer
	service controller.VideoStreamService
	logger  Logger
	streams map[string]*StreamSession
	mu      sync.RWMutex
}

// mapError маппит доменные ошибки в gRPC status (как в user-service grpc/server.go).
func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, apperrors.ErrStreamNotFound), errors.Is(err, apperrors.ErrClientNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, apperrors.ErrInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// StreamSession управляет сессией стрима
type StreamSession struct {
	StreamID   string
	ClientID   string
	StartTime  time.Time
	LastFrame  time.Time
	FrameCount int64
	BytesCount int64
	mu         sync.RWMutex
}

// NewVideoStreamServer создает gRPC сервер (принимает интерфейсы controller.VideoStreamService и Logger).
func NewVideoStreamServer(svc controller.VideoStreamService, logger Logger) *VideoStreamServer {
	return &VideoStreamServer{
		service: svc,
		logger:  logger,
		streams: make(map[string]*StreamSession),
	}
}

// StreamVideo потоковая передача видео
func (s *VideoStreamServer) StreamVideo(stream pb.VideoStreamService_StreamVideoServer) error {
	s.logger.Info("Starting gRPC video stream")

	var session *StreamSession
	var totalBytes, totalFrames int64
	startTime := time.Now()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			if session != nil {
				s.logger.Info("Stream completed",
					zap.String("stream_id", session.StreamID),
					zap.Int64("frames", totalFrames),
					zap.Int64("bytes", totalBytes),
					zap.Duration("duration", time.Since(startTime)))
			}
			return nil
		}
		if err != nil {
			s.logger.Error("Stream receive error", zap.Error(err))
			return status.Error(codes.Internal, err.Error())
		}

		if session == nil {
			session = &StreamSession{
				StreamID:  chunk.StreamId,
				ClientID:  chunk.ClientId,
				StartTime: time.Now(),
				LastFrame: time.Now(),
			}
			s.mu.Lock()
			s.streams[chunk.StreamId] = session
			s.mu.Unlock()

			s.logger.Info("New gRPC stream session",
				zap.String("stream_id", chunk.StreamId),
				zap.String("client_id", chunk.ClientId))
		}

		session.mu.Lock()
		session.FrameCount++
		session.BytesCount += int64(len(chunk.Data))
		session.LastFrame = time.Now()
		session.mu.Unlock()

		totalFrames++
		totalBytes += int64(len(chunk.Data))

		if totalFrames%100 == 0 {
			s.logger.Debug("Stream progress",
				zap.String("stream_id", chunk.StreamId),
				zap.Int64("frames", totalFrames),
				zap.Int64("bytes", totalBytes),
				zap.Float64("fps", float64(totalFrames)/time.Since(startTime).Seconds()))
		}

		frame := &pb.VideoFrame{
			FrameId:   fmt.Sprintf("grpc_%d", totalFrames),
			FrameData: chunk.Data,
			Timestamp: time.Now().Unix(),
			ClientId:  chunk.ClientId,
			CameraId:  "grpc_stream",
			Width:     1920,
			Height:    1080,
			Format:    "jpeg",
			Metadata:  chunk.Metadata,
		}

		_, _ = s.service.SendFrameInternal(stream.Context(), chunk.StreamId, chunk.ClientId, "gRPC Client", frame)

		ack := &pb.ChunkAck{
			Status:           "ok",
			Message:          "Frame received",
			ReceivedAt:       time.Now().Unix(),
			NextExpected:     int32(totalFrames + 1),
			ProcessingTimeMs: float32(time.Since(startTime).Seconds() * 1000),
		}
		if err := stream.Send(ack); err != nil {
			s.logger.Error("Failed to send ack", zap.Error(err))
			return err
		}
	}
}

// SendFrame единичный кадр (обратная совместимость)
func (s *VideoStreamServer) SendFrame(ctx context.Context, req *pb.SendFrameRequest) (*pb.ApiResponse, error) {
	s.logger.Info("gRPC SendFrame called",
		zap.String("stream_id", req.StreamId),
		zap.String("client_id", req.ClientId))
	resp, err := s.service.SendFrame(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

// StartStream старт стрима
func (s *VideoStreamServer) StartStream(ctx context.Context, req *pb.StartStreamRequest) (*pb.StartStreamResponse, error) {
	resp, err := s.service.StartStream(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

// StopStream остановка стрима
func (s *VideoStreamServer) StopStream(ctx context.Context, req *pb.StopStreamRequest) (*pb.ApiResponse, error) {
	return s.service.StopStream(ctx, req)
}

// GetActiveStreams получение активных стримов
func (s *VideoStreamServer) GetActiveStreams(req *pb.EmptyRequest, stream pb.VideoStreamService_GetActiveStreamsServer) error {
	activeStreams := s.service.GetAllActiveStreams()
	for _, as := range activeStreams {
		if err := stream.Send(as); err != nil {
			return err
		}
	}
	return nil
}

// GetStreamStats получение статистики
func (s *VideoStreamServer) GetStreamStats(ctx context.Context, req *pb.GetStreamStatsRequest) (*pb.StreamStats, error) {
	stats, err := s.service.GetStreamStats(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}
	return stats, nil
}

// GetStreamsByClient стримы клиента
func (s *VideoStreamServer) GetStreamsByClient(ctx context.Context, req *pb.GetStreamsByClientRequest) (*pb.GetStreamsByClientResponse, error) {
	streams := s.service.GetStreamsByClient(req.ClientId)
	return &pb.GetStreamsByClientResponse{Streams: streams}, nil
}

// GetStream информация о стриме
func (s *VideoStreamServer) GetStream(ctx context.Context, req *pb.GetStreamRequest) (*pb.ActiveStream, error) {
	stream := s.service.GetStream(req.StreamId)
	if stream == nil {
		return nil, mapError(apperrors.ErrStreamNotFound)
	}
	return stream, nil
}

// GetAllStats общая статистика
func (s *VideoStreamServer) GetAllStats(ctx context.Context, req *pb.EmptyRequest) (*pb.GetAllStatsResponse, error) {
	stats := s.service.GetAllStats()
	total := s.service.GetTotalStats()
	var totalFrames, totalBytes int64
	if f, ok := total["total_frames"].(int64); ok {
		totalFrames = f
	}
	if b, ok := total["total_bytes"].(int64); ok {
		totalBytes = b
	}
	var avgFPS float32
	if a, ok := total["average_fps"].(float32); ok {
		avgFPS = a
	}
	return &pb.GetAllStatsResponse{
		Stats:       stats,
		TotalFrames: totalFrames,
		TotalBytes:  totalBytes,
		AverageFps:  avgFPS,
	}, nil
}

// Run запускает gRPC сервер (только VideoStreamService)
func (s *VideoStreamServer) Run(port string) error {
	return RunGRPC(port, s, nil, s.logger)
}

// RunGRPC запускает gRPC сервер с Video и ClientInfo сервисами (принимает Deps или отдельные серверы).
func RunGRPC(port string, videoServer *VideoStreamServer, clientInfoServer *ClientInfoServer, logger Logger) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024), // 50MB для видео
		grpc.MaxSendMsgSize(10*1024*1024), // 10MB
	)
	pb.RegisterVideoStreamServiceServer(grpcServer, videoServer)
	if clientInfoServer != nil {
		pb.RegisterClientInfoServiceServer(grpcServer, clientInfoServer)
	}

	if logger != nil {
		logger.Info("Starting gRPC server", zap.String("port", port))
	}
	return grpcServer.Serve(lis)
}
