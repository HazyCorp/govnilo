package adminklient

import (
	"context"
	"log/slog"

	"github.com/HazyCorp/govnilo/proto"

	"github.com/pkg/errors"
)

type clientOptions struct {
	Logger *slog.Logger
}

type ClientOpt interface {
	apply(*clientOptions)
}

type ClientOptFunc func(o *clientOptions)

func (f ClientOptFunc) apply(o *clientOptions) {
	f(o)
}

func WithLogger(l *slog.Logger) ClientOpt {
	return ClientOptFunc(func(o *clientOptions) { o.Logger = l })
}

type Client interface {
	GetConfig(ctx context.Context) (*proto.Settings, error)
}

type TransportConfig struct {
	GRPC *GRPCClientConfig
	// TODO: HTTP config
}

type ClientConfig struct {
	Async     *AsyncClientConfig `json:"async" yaml:"async"`
	Transport *TransportConfig   `json:"transport" yaml:"transport"`
}

func New(c ClientConfig, opts ...ClientOpt) (Client, error) {
	if c.Transport == nil {
		return nil, errors.New("transport settings are required")
	}

	if c.Transport.GRPC == nil {
		return nil, errors.New("only one transport setting must be provided, allowed options are [grpc]")
	}

	var client Client

	grpcClient, err := NewGRPC(*c.Transport.GRPC, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create transport for client")
	}
	client = grpcClient

	if c.Async != nil {
		client = NewAsync(grpcClient, *c.Async, opts...)
	}

	return client, nil
}
