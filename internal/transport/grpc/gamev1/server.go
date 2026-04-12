package gamev1

import (
	"fmt"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/service"
)

const subjectIDHeader = "x-fs-subject-id"

// Server exposes the gRPC campaign transport.
type Server struct {
	gamev1.UnimplementedGameServiceServer

	service service.API
}

// NewServer constructs one gRPC campaign transport adapter.
func NewServer(svc service.API) (*Server, error) {
	if svc == nil {
		return nil, fmt.Errorf("campaign service is required")
	}
	return &Server{service: svc}, nil
}

func (s *Server) requireService() (service.API, error) {
	if s == nil || s.service == nil {
		return nil, internalStatus(fmt.Errorf("campaign server is required"))
	}
	return s.service, nil
}

func requireRequest[T any](req *T) error {
	if req == nil {
		return invalidArgument(fmt.Errorf("request is required"))
	}
	return nil
}

func requireStream(stream gamev1.GameService_StreamCampaignEventsServer) error {
	if stream == nil {
		return internalStatus(fmt.Errorf("stream is required"))
	}
	return nil
}
