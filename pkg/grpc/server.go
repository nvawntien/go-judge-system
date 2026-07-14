package grpc

import googlegrpc "google.golang.org/grpc"

// NewServer creates a grpc-go server. The caller remains responsible for
// registering services, serving a listener, and stopping the server.
func NewServer(opts ...ServerOption) *googlegrpc.Server {
	options := serverOptions{}
	for _, option := range opts {
		if option != nil {
			option(&options)
		}
	}

	return googlegrpc.NewServer(options.grpcOptions...)
}
