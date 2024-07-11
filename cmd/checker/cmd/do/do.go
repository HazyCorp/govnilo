package do

import (
	"github.com/HazyCorp/checker/cmd/checker/cmd/do/check"
	"github.com/HazyCorp/checker/cmd/checker/cmd/do/get"
	"github.com/HazyCorp/checker/cmd/checker/globflags"
	"github.com/spf13/cobra"
)

var DoCmd = &cobra.Command{
	Use:   "do",
	Short: "runs single check or get on the provided service",
	RunE: func(cmd *cobra.Command, args []string) error {
		panic("not implemented")
	},
}

func init() {
	DoCmd.PersistentFlags().
		StringVarP(&globflags.Service, "service", "s", "", "specifies service name to run checks on")
	DoCmd.MarkFlagRequired("service")

	DoCmd.PersistentFlags().
		StringVarP(&globflags.Target, "target", "t", "", "specifies target, which will be provided to Checker methods")
	DoCmd.MarkFlagRequired("target")

	DoCmd.AddCommand(check.CheckCmd)
	DoCmd.AddCommand(get.GetCmd)
}
