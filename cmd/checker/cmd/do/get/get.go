package get

import (
	"context"
	"fmt"
	"time"

	checkflags "github.com/HazyCorp/checker/cmd/checker/globflags"
	"github.com/HazyCorp/checker/internal/cmdutil"
	"github.com/HazyCorp/checker/pkg/hazycheck"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var data string

var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "runs Checker.Get on your service",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkers, err := cmdutil.ExtractCheckers(false)
		if err != nil {
			return errors.Wrap(err, "cannot build registered checkers")
		}

		service := checkflags.Service
		target := checkflags.Target

		checker, exists := lo.Find(checkers, func(c hazycheck.Checker) bool {
			return c.ServiceName() == service
		})
		if !exists {
			return errors.Errorf("service with name %q not registered", service)
		}

		start := time.Now()
		err = checker.Get(context.Background(), target, []byte(data))
		if err != nil {
			return err
		}
		duration := time.Since(start)

		fmt.Printf(
			"Service name: %s\n"+
				"Target:       %s\n"+
				"Method:       Checker.Get\n"+
				"Duration:     %s\n",
			service, target, duration,
		)
		return nil
	},
}

func init() {
	GetCmd.Flags().
		StringVarP(&data, "data", "d", "", "use this flag to provide data, returned from Checker.Check method")
	GetCmd.MarkFlagRequired("data")

	GetCmd.MarkPersistentFlagRequired("service")
	GetCmd.MarkPersistentFlagRequired("target")
}
