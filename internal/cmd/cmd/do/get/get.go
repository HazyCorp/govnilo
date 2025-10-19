package get

import (
	"time"

	"github.com/HazyCorp/govnilo/internal/cmd/globflags"
	"github.com/HazyCorp/govnilo/internal/cmdutil"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/internal/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var (
	data        string
	checkerName string
)

var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "runs specified Checker.Get on your service",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkers, err := cmdutil.ExtractCheckers(false)
		if err != nil {
			return errors.Wrap(err, "cannot build registered checkers")
		}

		target := globflags.Target
		service := globflags.Service
		checkerName := checkerName

		ctx := cmd.Context()
		tracer := otel.Tracer("govnilo/checker")
		ctx, span := tracer.Start(ctx, "checker.Check")
		defer span.End()

		span.SetAttributes(attribute.String("service", service))
		span.SetAttributes(attribute.String("target", target))

		checkerID := hazycheck.CheckerID{
			Service: service,
			Name:    checkerName,
		}

		checker, exists := lo.Find(checkers, func(c hazycheck.Checker) bool {
			return c.CheckerID() == checkerID
		})
		if !exists {
			return errors.Errorf("service with name %q not registered", service)
		}

		start := time.Now()
		err = checker.Get(ctx, target, []byte(data))
		if err != nil {
			return err
		}
		duration := time.Since(start)

		output := map[string]any{
			"service":  service,
			"checker":  checkerName,
			"target":   target,
			"method":   "Checker.Get",
			"duration": duration.String(),
		}
		util.PrintJson(output)
		return nil
	},
}

func init() {
	GetCmd.Flags().
		StringVarP(&data, "data", "d", "", "use this flag to provide data, returned from Checker.Check method")
	GetCmd.MarkFlagRequired("data")

	GetCmd.Flags().StringVarP(&checkerName, "checker", "c", "", "specifies the checker name to be run")
	GetCmd.MarkFlagRequired("checker")
}
