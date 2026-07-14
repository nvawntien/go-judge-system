package grpc

import (
	"context"
	"errors"
	"testing"

	googlegrpc "google.golang.org/grpc"
)

func TestNewClientConnRejectsEmptyTarget(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"", "   \t\n"} {
		_, err := NewClientConn(target, WithInsecureTransport())
		if !errors.Is(err, ErrEmptyTarget) {
			t.Fatalf("NewClientConn(%q) error = %v, want %v", target, err, ErrEmptyTarget)
		}
	}
}

func TestNewClientConnRequiresExplicitTransportCredentials(t *testing.T) {
	t.Parallel()

	_, err := NewClientConn("localhost:50051")
	if !errors.Is(err, ErrTransportCredentialsRequired) {
		t.Fatalf("NewClientConn() error = %v, want %v", err, ErrTransportCredentialsRequired)
	}
}

func TestNewClientConnWithInsecureTransport(t *testing.T) {
	t.Parallel()

	conn, err := NewClientConn("localhost:50051", WithInsecureTransport())
	if err != nil {
		t.Fatalf("NewClientConn() error = %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close client connection: %v", err)
		}
	})
}

func TestClientOptionsAreCollected(t *testing.T) {
	t.Parallel()

	interceptor := func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *googlegrpc.ClientConn,
		invoker googlegrpc.UnaryInvoker,
		opts ...googlegrpc.CallOption,
	) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	options := clientOptions{}
	for _, option := range []ClientOption{
		WithInsecureTransport(),
		WithUnaryClientInterceptor(interceptor),
		WithDialOption(googlegrpc.WithDefaultCallOptions(googlegrpc.WaitForReady(true))),
	} {
		if err := option(&options); err != nil {
			t.Fatalf("apply client option: %v", err)
		}
	}

	if !options.transportSelected {
		t.Fatal("transport credentials were not marked as selected")
	}
	if got, want := len(options.dialOptions), 3; got != want {
		t.Fatalf("dial option count = %d, want %d", got, want)
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	server := NewServer()
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	server.Stop()
}

func TestServerOptionsAreCollected(t *testing.T) {
	t.Parallel()

	interceptor := func(
		ctx context.Context,
		req any,
		info *googlegrpc.UnaryServerInfo,
		handler googlegrpc.UnaryHandler,
	) (any, error) {
		return handler(ctx, req)
	}

	options := serverOptions{}
	for _, option := range []ServerOption{
		WithUnaryServerInterceptor(interceptor),
		WithServerOption(googlegrpc.MaxRecvMsgSize(1024)),
	} {
		option(&options)
	}

	if got, want := len(options.grpcOptions), 2; got != want {
		t.Fatalf("server option count = %d, want %d", got, want)
	}
}
