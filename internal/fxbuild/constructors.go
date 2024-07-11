package fxbuild

import (
	"fmt"
	"net"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"

	"github.com/HazyCorp/checker/internal/checkerctrl"
	"github.com/HazyCorp/checker/internal/checkerserver"
	"github.com/HazyCorp/checker/internal/configuration"
	"github.com/HazyCorp/checker/internal/fxutil"
	grpcutil "github.com/HazyCorp/checker/internal/grpcutil"
	"github.com/HazyCorp/checker/internal/metricsrv"
	"github.com/HazyCorp/checker/internal/registrar"
	"github.com/HazyCorp/checker/pkg/hazycheck"
)

func NewGRPCServer(l *zap.Logger, c configuration.Serve, lc fx.Lifecycle) *grpc.Server {
	listen := fmt.Sprintf("0.0.0.0:%d", c.Port)

	lis, err := net.Listen("tcp", listen)
	if err != nil {
		l.Fatal(err.Error())
	}

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(grpcutil.LoggingUnaryInterceptor(l)))

	lc.Append(fx.StartStopHook(
		func() {
			l.Sugar().Infof("grpc server listenning on %s", listen)
			go grpcSrv.Serve(lis)
		},
		func() {
			grpcSrv.GracefulStop()
		},
	),
	)
	return grpcSrv
}

func NewLogger(lc fx.Lifecycle) *zap.Logger {
	var c *zap.Config
	if env, exists := os.LookupEnv("ENVIRONMENT"); exists {
		if env == "prod" {
			conf := zap.NewProductionConfig()
			c = &conf
		}
	}
	if c == nil {
		conf := zap.NewDevelopmentConfig()
		c = &conf
	}

	c.OutputPaths = []string{"stdout"}
	c.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l := zap.Must(c.Build())
	lc.Append(fx.StopHook(func() { l.Sync() }))

	return zap.Must(c.Build())
}

func NewSaveStrategy() (checkerctrl.SaveStrategy, error) {
	return checkerctrl.NewDummySaveStratedgy(512, 0.01)
}

func GetConstructors() []interface{} {
	return append(
		registrar.GetRegistered(),
		fxutil.AsIface[hazycheck.Provider](hazycheck.NewDummyProvider),
		configuration.Read,
		NewLogger,
		NewGRPCServer,
		NewSaveStrategy,
		checkerctrl.NewFX,
		checkerserver.NewFX,
		metricsrv.NewFX,
		fxutil.AsIface[checkerctrl.ControllerStorage](checkerctrl.NewAsyncFileStoreFX),
	)
}
