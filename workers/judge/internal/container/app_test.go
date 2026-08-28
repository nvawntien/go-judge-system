package container

import (
	"testing"

	sharedgrpc "go-judge-system/pkg/grpc"

	"go.uber.org/zap"
	"google.golang.org/grpc/connectivity"
)

func TestAppCloseClosesSandboxGRPCConnection(t *testing.T) {
	conn, err := sharedgrpc.NewClientConn("passthrough:///go-judge-test", sharedgrpc.WithInsecureTransport())
	if err != nil {
		t.Fatalf("create sandbox gRPC connection: %v", err)
	}
	app := &App{
		Logger:      zap.NewNop(),
		SandboxConn: &SandboxClientConn{ClientConn: conn},
	}

	if err := app.Close(); err != nil {
		t.Fatalf("App.Close() error = %v", err)
	}
	if got := conn.GetState(); got != connectivity.Shutdown {
		t.Fatalf("sandbox gRPC connection state = %s, want %s", got, connectivity.Shutdown)
	}
}
