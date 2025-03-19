package adminklient

import (
	"context"
	"log/slog"

	"github.com/HazyCorp/govnilo/common/checkersettings"
	"github.com/HazyCorp/govnilo/common/hzlog"
	"github.com/HazyCorp/govnilo/proto"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClient struct {
	client proto.AdminServiceClient
	l      *slog.Logger
}

type GRPCClientConfig struct {
	Target string `json:"target" yaml:"target"`
}

func NewGRPC(c GRPCClientConfig, opts ...ClientOpt) (Client, error) {
	var o clientOptions
	for _, opt := range opts {
		opt.apply(&o)
	}

	if o.Logger == nil {
		o.Logger = hzlog.NopLogger()
	}
	o.Logger = o.Logger.With(slog.String("component", "grpc_adminka_client"))

	conn, err := grpc.NewClient(c.Target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create grpc client with config %+v", c)
	}

	client := proto.NewAdminServiceClient(conn)
	return &GRPCClient{client: client, l: o.Logger}, nil
}

func (c *GRPCClient) GetConfig(ctx context.Context) (*checkersettings.Settings, error) {
	rsp, err := c.client.GetSettings(ctx, &proto.GetSettingsReq{})
	if err != nil {
		return nil, errors.Wrap(err, "cannot get settings from grpc client")
	}

	settings := checkersettings.FromPB(rsp.GetSettings())
	return settings, nil
}
