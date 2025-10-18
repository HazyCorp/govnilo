package sploit

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/HazyCorp/govnilo/internal/cmd/globflags"
	"github.com/HazyCorp/govnilo/internal/cmdutil"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
)

var sploitName string

var SploitCmd = &cobra.Command{
	Use:   "sploit",
	Short: "runs specified Sploit.RunAttack on your service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		checkers, err := cmdutil.ExtractSploits(false)
		if err != nil {
			return errors.Wrap(err, "cannot build registered sploits")
		}

		target := globflags.Target
		service := globflags.Service
		sploitName := sploitName

		sploitID := hazycheck.SploitID{
			Service: service,
			Name:    sploitName,
		}

		sploit, exists := lo.Find(checkers, func(c hazycheck.Sploit) bool {
			return c.SploitID() == sploitID
		})
		if !exists {
			return errors.Errorf("sploit with id %+v not registered", sploitID)
		}

		start := time.Now()
		err = sploit.RunAttack(ctx, target)
		if err != nil {
			return err
		}
		duration := time.Since(start)

		fmt.Printf(
			"Service name: %s\n"+
				"Sploit name:  %s\n"+
				"Target:       %s\n"+
				"Method:       Sploit.RunAttack\n"+
				"Duration:     %s\n",
			service, sploitName, target, duration,
		)
		return nil
	},
}

func init() {
	SploitCmd.Flags().
		StringVar(&sploitName, "sploit", "sp", "specifies the sploit name to be run")
	SploitCmd.MarkFlagRequired("sploit")
}
