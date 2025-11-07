package check

import (
	"time"

	"github.com/HazyCorp/govnilo/internal/cmd/globflags"
	"github.com/HazyCorp/govnilo/internal/cmdutil"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/internal/util"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
		checkers, err := cmdutil.ExtractCheckers(false)
		if err != nil {
			return errors.Wrap(err, "cannot extract checkers from command context")
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
			return errors.Errorf("service with name %q not registered", service)
		}

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
