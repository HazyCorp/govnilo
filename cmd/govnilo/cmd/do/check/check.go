package check

import (
	"time"

	"github.com/HazyCorp/govnilo/cmd/govnilo/globflags"
	"github.com/HazyCorp/govnilo/internal/cmdutil"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/internal/util"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var checkerName string

var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "runs specified Checker.Check on your service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := hzlog.WithNewTraceID(cmd.Context())
		// l := hzlog.MustBuild(hzlog.DefaultConfig())

		checkers, err := cmdutil.ExtractCheckers(false)
		if err != nil {
			return errors.Wrap(err, "cannot extract checkers from command context")
		}

		service := globflags.Service
		target := globflags.Target

		checkerID := hazycheck.CheckerID{
			Service: service,
			Name:    checkerName,
		}

		checker, exists := lo.Find(checkers, func(item hazycheck.Checker) bool {
			return item.CheckerID() == checkerID
		})
		if !exists {
			return errors.Errorf("service with name %q not registered", service)
		}

		start := time.Now()
		data, err := checker.Check(ctx, target)
		if err != nil {
			return err
		}
		duration := time.Since(start)

		output := map[string]any{
			"service":  service,
			"checker":  checkerName,
			"target":   target,
			"method":   "Checker.Check",
			"duration": duration.String(),
			"output":   string(data),
		}
		util.PrintJson(output)
		return nil
	},
}

func init() {
	CheckCmd.Flags().StringVarP(&checkerName, "checker", "c", "", "specifies the checker name to be run")
	CheckCmd.MarkFlagRequired("checker")
}
