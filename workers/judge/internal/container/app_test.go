package container

import (
	"fmt"
	"testing"

	sharedgrpc "go-judge-system/pkg/grpc"

	"go.uber.org/zap"
	"google.golang.org/grpc/connectivity"
)

func TestAppCloseOrdersAllLifecycleDependencies(t *testing.T) {
	var events []string
	hook := func(name string) func() error { return func() error { events = append(events, name); return nil } }
	app := &App{Logger: zap.NewNop(), closeHooks: &appCloseHooks{
		grpcStop: hook("grpc.stop"), consumerClose: hook("consumer.close"), cacheClose: hook("testcase_cache.close"),
		sandboxClose: hook("sandbox_conn.close"), problemClose: hook("problem_conn.close"), producerClose: hook("producer.close"),
	}}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(events), "[grpc.stop consumer.close testcase_cache.close sandbox_conn.close problem_conn.close producer.close]"; got != want {
		t.Fatalf("close order=%s want=%s", got, want)
	}
}

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
