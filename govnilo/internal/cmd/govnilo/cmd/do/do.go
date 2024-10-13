package do

import (
	"github.com/spf13/cobra"

	"github.com/HazyCorp/govnilo/govnilo/internal/cmd/govnilo/cmd/do/check"
	"github.com/HazyCorp/govnilo/govnilo/internal/cmd/govnilo/cmd/do/get"
	"github.com/HazyCorp/govnilo/govnilo/internal/cmd/govnilo/cmd/do/sploit"
	"github.com/HazyCorp/govnilo/govnilo/internal/cmd/govnilo/globflags"
)

var DoCmd = &cobra.Command{
	Use:   "do",
	Short: "runs single check, get or the runAttack methods on the provided service and target",
}

func init() {
	DoCmd.PersistentFlags().
		StringVarP(&globflags.Service, "service", "s", "", "specifies service name to run checks on")
	DoCmd.MarkPersistentFlagRequired("service")

	DoCmd.PersistentFlags().
		StringVarP(&globflags.Target, "target", "t", "", "specifies target, which will be provided to Checker methods")
	DoCmd.MarkPersistentFlagRequired("target")

	DoCmd.AddCommand(check.CheckCmd)
	DoCmd.AddCommand(get.GetCmd)
	DoCmd.AddCommand(sploit.SploitCmd)
}
