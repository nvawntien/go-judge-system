package grpc

import (
	"errors"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// ErrEmptyTarget indicates that a client target was not provided.
	ErrEmptyTarget = errors.New("gRPC client target is empty")
	// ErrTransportCredentialsRequired indicates that the caller did not choose
	// secure or explicitly insecure transport credentials.
	ErrTransportCredentialsRequired = errors.New("gRPC client transport credentials are required")

	errNilClientOption          = errors.New("gRPC client option is nil")
	errNilDialOption            = errors.New("gRPC dial option is nil")
	errNilTransportCredentials  = errors.New("gRPC transport credentials are nil")
	errTransportAlreadySelected = errors.New("gRPC client transport credentials are already configured")
)

type clientOptions struct {
	dialOptions       []googlegrpc.DialOption
	transportSelected bool
}

// ClientOption configures a client connection created by NewClientConn.
type ClientOption func(*clientOptions) error

// WithInsecureTransport explicitly configures plaintext transport. It should
// only be used on trusted internal networks or in development environments.
func WithInsecureTransport() ClientOption {
	return WithTransportCredentials(insecure.NewCredentials())
}

// WithTransportCredentials configures client transport security, including
// TLS credentials supplied by the caller.
func WithTransportCredentials(creds credentials.TransportCredentials) ClientOption {
	return func(options *clientOptions) error {
		if creds == nil {
			return errNilTransportCredentials
		}
		if options.transportSelected {
			return errTransportAlreadySelected
		}

		options.transportSelected = true
		options.dialOptions = append(options.dialOptions, googlegrpc.WithTransportCredentials(creds))
		return nil
	}
}

// WithDialOption appends an official grpc-go dial option. Transport credentials
// must still be selected with WithTransportCredentials or WithInsecureTransport.
func WithDialOption(option googlegrpc.DialOption) ClientOption {
	return func(options *clientOptions) error {
		if option == nil {
			return errNilDialOption
		}

		options.dialOptions = append(options.dialOptions, option)
		return nil
	}
}

// WithDefaultCallOptions applies bounded grpc-go call options to every RPC
// made through the connection. Callers should use this only when a protocol's
// documented response envelope exceeds grpc-go's conservative defaults.
func WithDefaultCallOptions(options ...googlegrpc.CallOption) ClientOption {
	return WithDialOption(googlegrpc.WithDefaultCallOptions(options...))
}

// WithUnaryClientInterceptor appends unary client interceptors in call order.
func WithUnaryClientInterceptor(interceptors ...googlegrpc.UnaryClientInterceptor) ClientOption {
	return WithDialOption(googlegrpc.WithChainUnaryInterceptor(interceptors...))
}

// WithStreamClientInterceptor appends stream client interceptors in call order.
func WithStreamClientInterceptor(interceptors ...googlegrpc.StreamClientInterceptor) ClientOption {
	return WithDialOption(googlegrpc.WithChainStreamInterceptor(interceptors...))
}

type serverOptions struct {
	grpcOptions []googlegrpc.ServerOption
}

// ServerOption configures a server created by NewServer.
type ServerOption func(*serverOptions)

// WithServerOption appends an official grpc-go server option.
func WithServerOption(option googlegrpc.ServerOption) ServerOption {
	return func(options *serverOptions) {
		options.grpcOptions = append(options.grpcOptions, option)
	}
}

// WithServerTransportCredentials configures server transport credentials,
// including TLS credentials supplied by the caller.
func WithServerTransportCredentials(creds credentials.TransportCredentials) ServerOption {
	return WithServerOption(googlegrpc.Creds(creds))
}

// WithUnaryServerInterceptor appends unary server interceptors in call order.
func WithUnaryServerInterceptor(interceptors ...googlegrpc.UnaryServerInterceptor) ServerOption {
	return WithServerOption(googlegrpc.ChainUnaryInterceptor(interceptors...))
}

// WithStreamServerInterceptor appends stream server interceptors in call order.
func WithStreamServerInterceptor(interceptors ...googlegrpc.StreamServerInterceptor) ServerOption {
	return WithServerOption(googlegrpc.ChainStreamInterceptor(interceptors...))
}
