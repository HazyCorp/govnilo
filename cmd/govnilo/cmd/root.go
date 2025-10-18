package cmd

import (
	"fmt"
	"os"

	"github.com/HazyCorp/govnilo/cmd/govnilo/cmd/do"
	"github.com/HazyCorp/govnilo/cmd/govnilo/cmd/list"
	"github.com/HazyCorp/govnilo/cmd/govnilo/cmd/run"
	"github.com/HazyCorp/govnilo/internal/util"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:              "checker",
	Short:            "tool to test checker on your service",
	Version:          "0.0.1",
	TraverseChildren: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ValidateFlagGroups(); err != nil {
			return err
		}
		if err := cmd.ValidateRequiredFlags(); err != nil {
			return err
		}

		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		return nil
	},
}

func Execute() {
	ctx, cancel := util.CtxWithShutdown()
	defer cancel()

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.SilenceErrors = false
		cmd.SilenceUsage = false

		return err
	})

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println()
		fmt.Printf("❌❌❌ Error occurred: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(do.DoCmd)
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(run.RunCmd)
}
