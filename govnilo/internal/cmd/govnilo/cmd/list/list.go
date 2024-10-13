package list

import (
	"github.com/spf13/cobra"

	"github.com/HazyCorp/govnilo/govnilo/internal/cmd/govnilo/cmd/list/checkers"
	"github.com/HazyCorp/govnilo/govnilo/internal/cmd/govnilo/cmd/list/sploits"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "lists registered entities (sploits and checkers at this moment)",
}

func init() {
	ListCmd.AddCommand(checkers.ListCheckersCmd)
	ListCmd.AddCommand(sploits.ListSploitsCmd)
}
