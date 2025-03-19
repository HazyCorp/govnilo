package fxbuild

import (
	"fmt"
	checkerctrl2 "github.com/HazyCorp/govnilo/checkerctrl"
	"github.com/HazyCorp/govnilo/common/hzlog"
	configuration2 "github.com/HazyCorp/govnilo/configuration"
	"github.com/HazyCorp/govnilo/fxutil"
	"github.com/HazyCorp/govnilo/grpcutil"
	"github.com/HazyCorp/govnilo/metricsrv"
	"github.com/HazyCorp/govnilo/registrar"
	"log/slog"
	"net"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

func NewGRPCServer(l *slog.Logger, c configuration2.Serve, lc fx.Lifecycle) *grpc.Server {
	listen := fmt.Sprintf("0.0.0.0:%d", c.Port)

	lis, err := net.Listen("tcp", listen)
	if err != nil {
		l.Error("cannot start lietenner for grpc", hzlog.Error(err))
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(grpcutil.LoggingUnaryInterceptor(l)))

	lc.Append(fx.StartStopHook(
		func() {
			l.Info("grpc server is listenning", slog.String("addr", listen))
			go grpcSrv.Serve(lis)
		},
		func() {
			grpcSrv.GracefulStop()
		},
	))
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

	level := "info"
	if envLevel, exists := os.LookupEnv("LOGLEVEL"); exists {
		level = envLevel
	}
	zapLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		panic("invalid logger level" + level)
	}

	c.Level.SetLevel(zapLevel)

	c.OutputPaths = []string{"stdout"}
	c.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l := zap.Must(c.Build())
	lc.Append(fx.StopHook(func() { l.Sync() }))

	zap.ReplaceGlobals(l)

	return l
}

func NewSaveStrategy() (checkerctrl2.SaveStrategy, error) {
	return checkerctrl2.NewDummySaveStratedgy(512, 0.01)
}

func GetConstructors() []interface{} {
	config, err := configuration2.Read()
	if err != nil {
		slog.Error("cannot read config", hzlog.Error(err))
		os.Exit(1)
	}

	logger, err := hzlog.Build(config.Logging)
	if err != nil {
		slog.Error("cannot build logger", hzlog.Error(err))
		os.Exit(1)
	}

	logger.Info("starting with config", slog.Any("config", config))

	knownConstructors := append(
		registrar.GetRegistered(),
		// fxutil.AsIface[hazycheck.Provider](hazycheck.NewDummyProvider),
		// checkerserver.NewFX,
		func() *slog.Logger { return logger },
		func() configuration2.Config { return config },
		NewLogger,
		NewGRPCServer,
		NewSaveStrategy,
		checkerctrl2.NewFX,
		metricsrv.NewFX,
		fxutil.AsIface[checkerctrl2.ControllerStorage](checkerctrl2.NewAsyncFileStoreFX),
	)

	if config.SettingsProvider.FromFile != nil {
		knownConstructors = append(knownConstructors, fxutil.AsIface[checkerctrl2.SettingsProvider](checkerctrl2.NewFileSettingsProvider))
	} else {
		knownConstructors = append(knownConstructors, fxutil.AsIface[checkerctrl2.SettingsProvider](checkerctrl2.NewAdminSettingsProvider))
	}

	return knownConstructors
}
