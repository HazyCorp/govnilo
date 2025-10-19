package hzlog

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/go-slog/otelslog"
	"github.com/pkg/errors"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type hzlogHandler struct {
	c                 Config
	hasInfraComponent bool
	slog.Handler
}

func newHzlogHandler(c Config, h slog.Handler) *hzlogHandler {
	return &hzlogHandler{c: c, Handler: h, hasInfraComponent: false}
}

func getComponent(attrs []slog.Attr) (string, bool) {
	for _, attr := range attrs {
		if attr.Key == "component" {
			if attr.Value.Kind() != slog.KindString {
				return "", true
			}

			return attr.Value.String(), true
		}
	}

	return "", false
}

func isInfraComponentAttr(attr slog.Attr) bool {
	if attr.Value.Kind() != slog.KindString {
		return false
	}

	if attr.Key == "component" {
		val := attr.Value.String()
		if strings.HasPrefix(val, "infra:") {
			return true
		}
	}

	return false
}

// Handle overrides the default Handle method to add context values and filter infra logs.
func (h *hzlogHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := getAttrs(ctx)

	isInfraLog := h.hasInfraComponent
	r.Attrs(func(attr slog.Attr) bool {
		if isInfraComponentAttr(attr) {
			// if infra log, then skip
			isInfraLog = true
			return false
		}

		return true
	})

	if slices.ContainsFunc(attrs, isInfraComponentAttr) {
		isInfraLog = true
	}

	if isInfraLog {
		if !h.c.Filter.Infra.Enabled {
			// log belongs to infra, but infra logging is disabled, skip
			return nil
		}

		// log belongs to infra, and infra logging is enabled, check the level
		// skip infra log, but we need to check the level first
		if r.Level < h.c.Filter.Infra.level {
			// too low level to log infra log, skip
			return nil
		}
	}

	// not infra log at all, or level of the log is enough to log infra log
	r.AddAttrs(attrs...)
	return h.Handler.Handle(ctx, r)
}

func (h *hzlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nextHandler := h.Handler.WithAttrs(attrs)

	component, componentFound := getComponent(attrs)
	if !componentFound {
		// component not found, keep the same state
		return &hzlogHandler{Handler: nextHandler, c: h.c, hasInfraComponent: h.hasInfraComponent}
	}

	// new component is set, override the old state
	newIsInfra := strings.HasPrefix(component, "infra:")
	return &hzlogHandler{Handler: nextHandler, c: h.c, hasInfraComponent: newIsInfra}
}

func (h *hzlogHandler) WithGroup(name string) slog.Handler {
	return &hzlogHandler{Handler: h.Handler.WithGroup(name), c: h.c, hasInfraComponent: h.hasInfraComponent}
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

	infraLogLevel, err := zapcore.ParseLevel(c.Filter.Infra.Level)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse infra.level %s", c.Filter.Infra.Level)
	}
	c.Filter.Infra.level = zapLevelToSlogLevel(infraLogLevel)

	ctxHandler := newHzlogHandler(c, base)
	otelHandler := otelslog.NewHandler(ctxHandler)

	l := slog.New(otelHandler)
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
