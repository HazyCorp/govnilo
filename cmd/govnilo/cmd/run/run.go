package run

import (
	"context"
	"github.com/HazyCorp/govnilo/checkerctrl"
	"github.com/HazyCorp/govnilo/cmd/govnilo/globflags"
	"github.com/HazyCorp/govnilo/fxbuild"
	"github.com/HazyCorp/govnilo/metricsrv"
	"log/slog"
	"runtime"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"net/http"
	_ "net/http/pprof"
)

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "runs the checker as a standalone service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		constructors := fxbuild.GetConstructors()

		var e struct {
			fx.In
			Logger *slog.Logger
		}

		app := fx.New(
			fx.Provide(constructors...),

			fx.Populate(&e),

			fx.Invoke(
				func(*metricsrv.Server, *checkerctrl.Controller) {},
			),

			fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
				return &fxevent.ZapLogger{Logger: l}
			}),
		)

		if err := app.Start(ctx); err != nil {
			return errors.Wrap(err, "cannot start the application")
		}

		l := e.Logger

		// TODO: remove, used for debug
		runtime.SetMutexProfileFraction(1)
		runtime.SetBlockProfileRate(1)
		go http.ListenAndServe(":8080", nil)

		select {
		case <-ctx.Done():
			l.Info("got shutdown signal")
		case stopSignal := <-app.Wait():
			l.Info("application ended it's work", slog.String("message", stopSignal.String()))
		}

		tCtx, tCancel := context.WithTimeout(context.Background(), app.StopTimeout())
		defer tCancel()

		if err := app.Stop(tCtx); err != nil {
			return errors.Wrap(err, "cannot gracefully stop the application")
		}

		l.Info("application shuted down successfully!")
		return nil
	},
}

func init() {
	RunCmd.
		Flags().
		StringVarP(
			&globflags.ConfigPath,
			"config",
			"c",
			"",
			"path to config for checker run. If config is not specifed, the default one will be used.",
		)
}
