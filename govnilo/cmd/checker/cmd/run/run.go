package run

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/HazyCorp/govnilo/govnilo/cmd/checker/globflags"
	"github.com/HazyCorp/govnilo/govnilo/internal/checkerserver"
	"github.com/HazyCorp/govnilo/govnilo/internal/fxbuild"
	"github.com/HazyCorp/govnilo/govnilo/internal/metricsrv"
)

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "runs the checker as a standalone service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		constructors := fxbuild.GetConstructors()

		var e struct {
			fx.In

			Logger *zap.Logger
		}

		app := fx.New(
			fx.Provide(constructors...),

			fx.Populate(&e),

			fx.Invoke(
				func(*checkerserver.CheckerServer, *metricsrv.Server) {},
			),

			fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
				return &fxevent.ZapLogger{Logger: l}
			}),
		)

		if err := app.Start(ctx); err != nil {
			return errors.Wrap(err, "cannot start the application")
		}

		l := e.Logger

		select {
		case <-ctx.Done():
			l.Info("got shutdown signal")
		case stopSignal := <-app.Wait():
			l.Sugar().Infof("application ended it's work with message %s", stopSignal.String())
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
	RunCmd.Flags().StringVarP(&globflags.ConfigPath, "config", "c", "", "path to config for checker run")
	RunCmd.MarkFlagRequired("config")
}
