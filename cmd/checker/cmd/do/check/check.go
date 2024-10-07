package check

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/HazyCorp/govnilo/cmd/checker/globflags"
	"github.com/HazyCorp/govnilo/internal/cmdutil"
	"github.com/HazyCorp/govnilo/pkg/hazycheck"
)

var checkerName string

var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "runs specified Checker.Check on your service",
	RunE: func(cmd *cobra.Command, args []string) error {
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
		data, err := checker.Check(context.Background(), target)
		if err != nil {
			return err
		}
		duration := time.Since(start)

		fmt.Printf(
			"Service name: %s\n"+
				"Checker name: %s\n"+
				"Target:       %s\n"+
				"Method:       Checker.Check\n"+
				"Duration:     %s\n"+
				"Output:       %s\n",
			service, checkerName, target, duration, string(data),
		)
		return nil
	},
}

func init() {
	CheckCmd.Flags().StringVarP(&checkerName, "checker", "c", "", "specifies the checker name to be run")
	CheckCmd.MarkFlagRequired("checker")
}
