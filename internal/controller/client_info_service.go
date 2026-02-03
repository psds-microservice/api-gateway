package controller

import (
	"context"
	"time"

	pb "github.com/psds-microservice/api-gateway/pkg/gen"
	"github.com/psds-microservice/helpy"
	"go.uber.org/zap"
)

// ClientInfoServiceImpl реализует сервис информации о клиентах
type ClientInfoServiceImpl struct {
	logger *zap.Logger
	repo   *ClientRepository
}

// NewClientInfoService создает новый сервис
func NewClientInfoService(logger *zap.Logger) *ClientInfoServiceImpl {
	return &ClientInfoServiceImpl{
		logger: logger,
		repo:   NewClientRepository(),
	}
}

func (s *ClientInfoServiceImpl) ClientConnected(ctx context.Context, req *pb.ConnectionEvent) (*helpy.ApiResponse, error) {
	s.logger.Info("Client connected",
		zap.String("client_id", req.ClientId),
		zap.String("ip", req.IpAddress))
	s.repo.SaveClient(req.ClientInfo)
	return &helpy.ApiResponse{
		Status:    "ok",
		Message:   "Client connected successfully",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (s *ClientInfoServiceImpl) ClientDisconnected(ctx context.Context, req *pb.ConnectionEvent) (*helpy.ApiResponse, error) {
	s.logger.Info("Client disconnected", zap.String("client_id", req.ClientId))
	s.repo.RemoveClient(req.ClientId)
	return &helpy.ApiResponse{
		Status:    "ok",
		Message:   "Client disconnected",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (s *ClientInfoServiceImpl) UpdateClientInfo(ctx context.Context, req *pb.UpdateClientRequest) (*helpy.ApiResponse, error) {
	s.logger.Info("Updating client info", zap.String("client_id", req.ClientId))
	if req.ClientInfo != nil {
		s.repo.SaveClient(req.ClientInfo)
	}
	return &helpy.ApiResponse{
		Status:    "ok",
		Message:   "Client info updated",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (s *ClientInfoServiceImpl) GetClientInfo(ctx context.Context, req *pb.GetClientInfoRequest) (*pb.ClientInfo, error) {
	s.logger.Debug("Getting client info", zap.String("client_id", req.ClientId))
	return s.repo.GetClient(req.ClientId), nil
}

func (s *ClientInfoServiceImpl) ListActiveClients(ctx context.Context, req *pb.ListClientsRequest) (*pb.ListClientsResponse, error) {
	s.logger.Debug("Listing active clients")
	allClients := s.repo.GetAllClients()
	page := int(req.Page)
	limit := int(req.Limit)
	totalClients := len(allClients)
	start := (page - 1) * limit
	end := start + limit

	if start >= totalClients {
		return &pb.ListClientsResponse{
			Clients: []*pb.ClientInfo{},
			Total:   int32(totalClients),
		}, nil
	}
	if end > totalClients {
		end = totalClients
	}

	return &pb.ListClientsResponse{
		Clients: allClients[start:end],
		Total:   int32(totalClients),
	}, nil
}
