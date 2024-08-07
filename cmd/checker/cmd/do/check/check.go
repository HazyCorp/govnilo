package check

import (
	"context"
	"fmt"
	"time"

	"github.com/HazyCorp/checker/cmd/checker/globflags"
	"github.com/HazyCorp/checker/internal/cmdutil"
	"github.com/HazyCorp/checker/pkg/hazycheck"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "runs Checker.Check on your service",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkers, err := cmdutil.ExtractCheckers(false)
		if err != nil {
			return errors.Wrap(err, "cannot extract checkers from command context")
		}

		service := globflags.Service
		target := globflags.Target

		checker, exists := lo.Find(checkers, func(item hazycheck.Checker) bool {
			return item.CheckerID() == service
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
				"Target:       %s\n"+
				"Method:       Checker.Check\n"+
				"Duration:     %s\n"+
				"Output:       %s\n",
			service, target, duration, string(data),
		)
		return nil
	},
}

func init() {
	CheckCmd.MarkPersistentFlagRequired("service")
	CheckCmd.MarkPersistentFlagRequired("target")
}
