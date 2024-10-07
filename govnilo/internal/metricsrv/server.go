package metricsrv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/VictoriaMetrics/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Port uint64 `json:"port" yaml:"port"`
}

type Server struct {
	srv *http.Server
	l   *zap.Logger
}

func New(l *zap.Logger, conf Config) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	})

	return &Server{
		l: l,
		srv: &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", conf.Port),
			Handler: mux,
		},
	}
}

func NewFX(l *zap.Logger, conf Config, lc fx.Lifecycle) *Server {
	s := New(l, conf)
	lc.Append(fx.StartStopHook(
		func() {
			s.l.Sugar().Infof("metrics server is listenning on %s/metrics", s.srv.Addr)
			go s.srv.ListenAndServe()
		},
		func(ctx context.Context) error {
			return s.srv.Shutdown(ctx)
		},
	))

	return s
}
