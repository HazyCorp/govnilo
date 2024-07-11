package bench

import "github.com/spf13/cobra"

var BenchCmd = &cobra.Command{
	Use:   "bench",
	Short: "allows to bench your service.",
	Long: "allows to bench your service. \n" +
		"after run will show some stats about running checks on your service.\n" +
		"At the end of the benchmark max checks per second will be shown",
	RunE: func(cmd *cobra.Command, args []string) error {
		panic("not implemented")
	},
}
