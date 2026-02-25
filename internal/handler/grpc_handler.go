package handler

import (
	"Go-Microservice-Template/internal/service"

	"google.golang.org/grpc"
)

// GRPCHandler handles gRPC requests.
type GRPCHandler struct {
	userService service.UserService
}

// NewGRPCHandler creates a new gRPC handler.
func NewGRPCHandler(us service.UserService) *GRPCHandler {
	return &GRPCHandler{userService: us}
}

// Register registers gRPC services with the server.
// NOTE: After generating protobuf code with `make proto`,
// uncomment the service registration below.
func (h *GRPCHandler) Register(server *grpc.Server) {
	// pb.RegisterUserServiceServer(server, h)
	// Add more service registrations here
}
