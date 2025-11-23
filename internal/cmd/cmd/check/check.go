package check

import (
	"context"
	"log/slog"
	"time"

	"github.com/HazyCorp/govnilo/internal/cmd/globflags"
	"github.com/HazyCorp/govnilo/internal/fxbuild"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/internal/util"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func init() {
	CheckCmd.PersistentFlags().
		StringVarP(&globflags.Service, "service", "s", "", "specifies service name to run checks on")
	CheckCmd.MarkPersistentFlagRequired("service")

	CheckCmd.PersistentFlags().
		StringVarP(&globflags.CheckerName, "checker", "c", "", "specifies checker name to run")
	CheckCmd.MarkPersistentFlagRequired("checker")

	CheckCmd.PersistentFlags().
		StringVarP(&globflags.Target, "target", "t", "", "specifies target, which will be provided to Checker methods")
	CheckCmd.MarkPersistentFlagRequired("target")
}

var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "runs single check using the provided target",
	RunE: func(cmd *cobra.Command, args []string) error {
		fxOpts := []fx.Option{
			fx.Provide(
				fxbuild.GetConstructors()...,
			),
			fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		}

		type checkersIn struct {
			fx.In

			Checkers []hazycheck.Checker `group:"checkers"`
		}

		var checkers []hazycheck.Checker
		fxOpts = append(fxOpts, fx.Invoke(func(in checkersIn) {
			checkers = in.Checkers
		}))

		app := fx.New(fxOpts...)
		if err := app.Start(context.Background()); err != nil {
			return errors.Wrap(err, "cannot build the app to extract the checkers")
		}

		ctx := cmd.Context()
		tracer := otel.Tracer("govnilo/checker")
		ctx, span := tracer.Start(ctx, "checker.Check")
		defer span.End()

		service := globflags.Service
		target := globflags.Target

		span.SetAttributes(attribute.String("service", service))
		span.SetAttributes(attribute.String("target", target))

		checkerID := hazycheck.CheckerID{
			Service: service,
			Name:    globflags.CheckerName,
		}

		checker, exists := lo.Find(checkers, func(item hazycheck.Checker) bool {
			return item.CheckerID() == checkerID
		})
		if !exists {
			return errors.Errorf("either checker with name %q or service with name %q is not registered. Try to run list-checkers to see all the registered checkers.", globflags.CheckerName, service)
		}

		ctx = hzlog.ContextWith(ctx, slog.Any("checker_id", checkerID))

		start := time.Now()
		if err := checker.Check(ctx, target); err != nil {
			return err
		}
		duration := time.Since(start)

		output := map[string]any{
			"service":  service,
			"checker":  globflags.CheckerName,
			"target":   target,
			"duration": duration.String(),
		}
		util.PrintJson(output)

		return nil
	},
}
