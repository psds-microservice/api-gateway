package grpc_server

import (
	"context"

	"github.com/psds-microservice/api-gateway/internal/controller"
	pb "github.com/psds-microservice/api-gateway/pkg/gen"
)

// ClientInfoServer реализует gRPC сервер для ClientInfoService (принимает controller.ClientInfoService).
type ClientInfoServer struct {
	pb.UnimplementedClientInfoServiceServer
	service controller.ClientInfoService
}

// NewClientInfoServer создаёт gRPC сервер
func NewClientInfoServer(svc controller.ClientInfoService) *ClientInfoServer {
	return &ClientInfoServer{service: svc}
}

func (s *ClientInfoServer) ClientConnected(ctx context.Context, req *pb.ConnectionEvent) (*pb.ApiResponse, error) {
	resp, err := s.service.ClientConnected(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

func (s *ClientInfoServer) ClientDisconnected(ctx context.Context, req *pb.ConnectionEvent) (*pb.ApiResponse, error) {
	resp, err := s.service.ClientDisconnected(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

func (s *ClientInfoServer) UpdateClientInfo(ctx context.Context, req *pb.UpdateClientRequest) (*pb.ApiResponse, error) {
	resp, err := s.service.UpdateClientInfo(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

func (s *ClientInfoServer) GetClientInfo(ctx context.Context, req *pb.GetClientInfoRequest) (*pb.ClientInfo, error) {
	info, err := s.service.GetClientInfo(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}
	return info, nil
}

func (s *ClientInfoServer) ListActiveClients(ctx context.Context, req *pb.ListClientsRequest) (*pb.ListClientsResponse, error) {
	return s.service.ListActiveClients(ctx, req)
}
