package metricsrv

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/VictoriaMetrics/metrics"
	"go.uber.org/fx"
)

type Config struct {
	Port uint64 `json:"port" yaml:"port"`
}

type Server struct {
	srv *http.Server
	l   *slog.Logger
}

func New(l *slog.Logger, conf Config) *Server {
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

func NewFX(l *slog.Logger, conf Config, lc fx.Lifecycle) *Server {
	s := New(l, conf)
	lc.Append(fx.StartStopHook(
		func() {
			s.l.Info("metrics server is listenning", slog.String("address", s.srv.Addr))
			go s.srv.ListenAndServe()
		},
		func(ctx context.Context) error {
			return s.srv.Shutdown(ctx)
		},
	))

	return s
}
