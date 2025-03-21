package list

import (
	"github.com/HazyCorp/govnilo/cmd/govnilo/cmd/list/checkers"
	"github.com/HazyCorp/govnilo/cmd/govnilo/cmd/list/sploits"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "lists registered entities (sploits and checkers at this moment)",
}

func init() {
	ListCmd.AddCommand(checkers.ListCheckersCmd)
	ListCmd.AddCommand(sploits.ListSploitsCmd)
}
