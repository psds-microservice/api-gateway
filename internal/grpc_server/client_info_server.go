package grpc_server

import (
	"context"

	"github.com/psds-microservice/api-gateway/internal/controller"
	pb "github.com/psds-microservice/api-gateway/pkg/gen"
)

// ClientInfoServer реализует gRPC сервер для ClientInfoService
type ClientInfoServer struct {
	pb.UnimplementedClientInfoServiceServer
	service *controller.ClientInfoServiceImpl
}

// NewClientInfoServer создаёт gRPC сервер
func NewClientInfoServer(service *controller.ClientInfoServiceImpl) *ClientInfoServer {
	return &ClientInfoServer{service: service}
}

func (s *ClientInfoServer) ClientConnected(ctx context.Context, req *pb.ConnectionEvent) (*pb.ApiResponse, error) {
	return s.service.ClientConnected(ctx, req)
}

func (s *ClientInfoServer) ClientDisconnected(ctx context.Context, req *pb.ConnectionEvent) (*pb.ApiResponse, error) {
	return s.service.ClientDisconnected(ctx, req)
}

func (s *ClientInfoServer) UpdateClientInfo(ctx context.Context, req *pb.UpdateClientRequest) (*pb.ApiResponse, error) {
	return s.service.UpdateClientInfo(ctx, req)
}

func (s *ClientInfoServer) GetClientInfo(ctx context.Context, req *pb.GetClientInfoRequest) (*pb.ClientInfo, error) {
	return s.service.GetClientInfo(ctx, req)
}

func (s *ClientInfoServer) ListActiveClients(ctx context.Context, req *pb.ListClientsRequest) (*pb.ListClientsResponse, error) {
	return s.service.ListActiveClients(ctx, req)
}
