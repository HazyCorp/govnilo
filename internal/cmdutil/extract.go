package cmdutil

import (
	"context"

	"github.com/HazyCorp/govnilo/internal/fxbuild"
	"github.com/HazyCorp/govnilo/internal/hazycheck"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type fxOptsKey struct{}

func InjectFxOpts(ctx context.Context, opts []fx.Option) context.Context {
	return context.WithValue(ctx, fxOptsKey{}, opts)
}

func ExtractFxOpts(ctx context.Context) []fx.Option {
	return ctx.Value(fxOptsKey{}).([]fx.Option)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func buildFXLogger(enabled bool) fx.Option {
	if !enabled {
		return fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger })
	}

	return fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
		return &fxevent.ZapLogger{Logger: l}
	})
}

func ExtractCheckers(enableFXLogs bool) ([]hazycheck.Checker, error) {
	fxOpts := []fx.Option{
		fx.Provide(
			fxbuild.GetConstructors()...,
		),
		buildFXLogger(enableFXLogs),
	}

	type checkersIn struct {
		fx.In

		Checkers []hazycheck.Checker `group:"checkers"`
	}

	var registeredCheckers []hazycheck.Checker
	fxOpts = append(fxOpts, fx.Invoke(func(in checkersIn) {
		registeredCheckers = in.Checkers
	}))

	app := fx.New(fxOpts...)
	if err := app.Start(context.Background()); err != nil {
		return nil, errors.Wrap(err, "cannot build the app to extract the checkers")
	}
	_ = app.Stop(context.Background())

	return registeredCheckers, nil
}
