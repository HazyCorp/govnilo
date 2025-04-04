package checkers

import (
	"github.com/HazyCorp/govnilo/cmdutil"
	"github.com/HazyCorp/govnilo/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var ListCheckersCmd = &cobra.Command{
	Use:   "checkers",
	Short: "lists all registered checkers",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkers, err := cmdutil.ExtractCheckers(false)
		if err != nil {
			return errors.Wrap(err, "cannot build checkers")
		}

		serviceCheckers := make(map[string][]string)
		for _, s := range checkers {
			checkerID := s.CheckerID()
			service := checkerID.Service
			checker := checkerID.Name

			serviceCheckers[service] = append(serviceCheckers[service], checker)
		}

		util.PrintJson(serviceCheckers)
		return nil
	},
}
