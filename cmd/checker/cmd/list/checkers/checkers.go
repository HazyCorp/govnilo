package checkers

import (
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/HazyCorp/govnilo/internal/cmdutil"
	"github.com/HazyCorp/govnilo/internal/util"
	"github.com/HazyCorp/govnilo/pkg/hazycheck"
)

var ListCheckersCmd = &cobra.Command{
	Use:   "checkers",
	Short: "lists all registered checkers",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkers, err := cmdutil.ExtractCheckers(false)
		if err != nil {
			return errors.Wrap(err, "cannot build checkers")
		}

		checkerNames := lo.Map(checkers, func(c hazycheck.Checker, _ int) hazycheck.CheckerID { return c.CheckerID() })
		util.PrintJson(checkerNames)
		return nil
	},
}
