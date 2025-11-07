package list

import (
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/internal/util"
	"github.com/spf13/cobra"
)

var ListCheckersCmd = &cobra.Command{
	Use:   "list-checkers",
	Short: "lists all registered checkers",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkerIDs := hazycheck.GetRegistered()

		serviceCheckers := make(map[string][]string)
		for _, checkerID := range checkerIDs {

			service := checkerID.Service
			checkerName := checkerID.Name

			serviceCheckers[service] = append(serviceCheckers[service], checkerName)
		}

		util.PrintJson(serviceCheckers)
		return nil
	},
}
