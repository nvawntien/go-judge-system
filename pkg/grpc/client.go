package grpc

import (
	"fmt"
	"strings"

	googlegrpc "google.golang.org/grpc"
)

// NewClientConn creates a lazy grpc-go client connection for target. The
// caller owns the returned connection and must close it when it is no longer
// needed.
//
// The caller must explicitly select transport security with
// WithTransportCredentials or WithInsecureTransport.
func NewClientConn(target string, opts ...ClientOption) (*googlegrpc.ClientConn, error) {
	if strings.TrimSpace(target) == "" {
		return nil, ErrEmptyTarget
	}

	options := clientOptions{}
	for index, option := range opts {
		if option == nil {
			return nil, fmt.Errorf("apply gRPC client option %d: %w", index, errNilClientOption)
		}
		if err := option(&options); err != nil {
			return nil, fmt.Errorf("apply gRPC client option %d: %w", index, err)
		}
	}

	if !options.transportSelected {
		return nil, ErrTransportCredentialsRequired
	}

	conn, err := googlegrpc.NewClient(target, options.dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("create gRPC client connection for target %q: %w", target, err)
	}

	return conn, nil
}
