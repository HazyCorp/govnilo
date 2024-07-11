package run

import "github.com/spf13/cobra"

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "runs the checker as a standalone service",
	RunE: func(cmd *cobra.Command, args []string) error {
		panic("not implemented")
	},
}

func init() {
	// TODO: add flags
}