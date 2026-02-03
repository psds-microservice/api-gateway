package grpc_server

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/psds-microservice/api-gateway/internal/controller"
	pb "github.com/psds-microservice/api-gateway/pkg/gen"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VideoStreamServer реализует gRPC сервер для видеостримов
type VideoStreamServer struct {
	pb.UnimplementedVideoStreamServiceServer
	service *controller.VideoStreamServiceImpl
	logger  *zap.Logger
	streams map[string]*StreamSession
	mu      sync.RWMutex
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

// NewVideoStreamServer создает gRPC сервер
func NewVideoStreamServer(service *controller.VideoStreamServiceImpl, logger *zap.Logger) *VideoStreamServer {
	return &VideoStreamServer{
		service: service,
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

		s.service.SendFrameInternal(chunk.StreamId, chunk.ClientId, "gRPC Client", frame)

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
	return s.service.SendFrame(ctx, req)
}

// StartStream старт стрима
func (s *VideoStreamServer) StartStream(ctx context.Context, req *pb.StartStreamRequest) (*pb.StartStreamResponse, error) {
	return s.service.StartStream(ctx, req)
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
	return s.service.GetStreamStats(ctx, req)
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
		return nil, status.Error(codes.NotFound, fmt.Sprintf("stream %s not found", req.StreamId))
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

// RunGRPC запускает gRPC сервер с Video и ClientInfo сервисами
func RunGRPC(port string, videoServer *VideoStreamServer, clientInfoServer *ClientInfoServer, logger *zap.Logger) error {
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
