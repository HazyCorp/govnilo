package cmdutil

import (
	"context"

	"github.com/HazyCorp/checker/internal/fxbuild"
	"github.com/HazyCorp/checker/pkg/checker"
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

func ExtractCheckers(enableLogs bool) ([]checker.Checker, error) {
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

		Checkers []checker.Checker `group:"checkers"`
	}

	var registeredCheckers []checker.Checker
	fxOpts = append(fxOpts, fx.Invoke(func(in checkersIn) {
		registeredCheckers = in.Checkers
	}))

	app := fx.New(fxOpts...)
	if err := app.Start(context.Background()); err != nil {
		return nil, errors.Wrap(err, "cannot build the app to extract the checkers")
	}

	return registeredCheckers, nil
}

func ExtractSploits(enableLogs bool) ([]checker.Sploit, error) {
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

		Sploits []checker.Sploit `group:"sploits"`
	}

	var registeredSploits []checker.Sploit
	fxOpts = append(fxOpts, fx.Invoke(func(in sploitsIn) {
		registeredSploits = in.Sploits
	}))

	app := fx.New(fxOpts...)
	if err := app.Start(context.Background()); err != nil {
		return nil, errors.Wrap(err, "cannot build the app to extract the sploits")
	}

	return registeredSploits, nil
}
