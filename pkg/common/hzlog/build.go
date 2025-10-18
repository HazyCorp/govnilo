package hzlog

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextHandler struct {
	slog.Handler
}

// Handle overrides the default Handle method to add context values.
func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := getAttrs(ctx)
	r.AddAttrs(attrs...)

	return h.Handler.Handle(ctx, r)
}

func MustBuild(c Config) *slog.Logger {
	logger, err := Build(c)
	if err != nil {
		panic("cannot build logger: " + err.Error())
	}

	return logger
}

func Build(c Config) (*slog.Logger, error) {
	zapC := zap.NewProductionConfig()

	lvl, err := zapcore.ParseLevel(c.Level)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse level")
	}
	zapC.Level.SetLevel(lvl)

	zapC.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	zapC.OutputPaths = []string{"stdout"}

	if c.Mode == "" {
		c.Mode = "json"
	}

	switch c.Mode {
	case "console":
		zapC.Encoding = "console"
	case "json":
		zapC.Encoding = "json"
	default:
		return nil, errors.Wrapf(err, "cannot build zap logger, unknown encoding %s, allowed options are only [console, json]", c.Mode)
	}

	zapLogger := zap.Must(zapC.Build())

	slogLvl := zapLevelToSlogLevel(lvl)
	base := slogzap.Option{Level: slogLvl, Logger: zapLogger}.NewZapHandler()
	ctxHanlder := contextHandler{Handler: base}
	l := slog.New(&ctxHanlder)
	slog.SetDefault(l)

	return l, nil
}

func zapLevelToSlogLevel(lvl zapcore.Level) slog.Level {
	zapLvlToSlogLvl := reverseMap(slogzap.LogLevels)
	if slogLvl, found := zapLvlToSlogLvl[lvl]; found {
		return slogLvl
	}

	panic(fmt.Sprintf("unknown slog level %s provided, cannot be mapped to slog level", lvl))
}

func reverseMap[TKey comparable, TValue comparable](mp map[TKey]TValue) map[TValue]TKey {
	ret := make(map[TValue]TKey, len(mp))
	for k, v := range mp {
		ret[v] = k
	}

	return ret
}
