package cmdutil

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/HazyCorp/govnilo/internal/fxbuild"
	"github.com/HazyCorp/govnilo/pkg/hazycheck"
)

type fxOptsKey struct{}

func InjectFxOpts(ctx context.Context, opts []fx.Option) context.Context {
	return context.WithValue(ctx, fxOptsKey{}, opts)
}

func ExtractFxOpts(ctx context.Context) []fx.Option {
	return ctx.Value(fxOptsKey{}).([]fx.Option)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func ExtractCheckers(enableLogs bool) ([]hazycheck.Checker, error) {
	var withLogger fx.Option
	if !enableLogs {
		withLogger = fx.WithLogger(func() fxevent.Logger {
			return &fxevent.NopLogger
		})
	} else {
		withLogger = fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: l}
		})
	}

	fxOpts := []fx.Option{
		fx.Provide(
			fxbuild.GetConstructors()...,
		),
		withLogger,
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

func ExtractSploits(enableLogs bool) ([]hazycheck.Sploit, error) {
	// TODO: remove boilerplate, maybe use global flag
	var withLogger fx.Option
	if !enableLogs {
		withLogger = fx.WithLogger(func() fxevent.Logger {
			return &fxevent.NopLogger
		})
	} else {
		withLogger = fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: l}
		})
	}

	fxOpts := []fx.Option{
		fx.Provide(
			fxbuild.GetConstructors()...,
		),
		withLogger,
	}

	type sploitsIn struct {
		fx.In

		Sploits []hazycheck.Sploit `group:"sploits"`
	}

	var registeredSploits []hazycheck.Sploit
	fxOpts = append(fxOpts, fx.Invoke(func(in sploitsIn) {
		registeredSploits = in.Sploits
	}))

	app := fx.New(fxOpts...)
	if err := app.Start(context.Background()); err != nil {
		return nil, errors.Wrap(err, "cannot build the app to extract the sploits")
	}
	_ = app.Stop(context.Background())

	return registeredSploits, nil
}
