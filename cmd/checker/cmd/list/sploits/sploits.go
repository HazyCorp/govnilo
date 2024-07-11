package sploits

import (
	"github.com/HazyCorp/checker/internal/cmdutil"
	"github.com/HazyCorp/checker/internal/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var ListSploitsCmd = &cobra.Command{
	Use:   "sploits",
	Short: "lists all registered sploits",
	RunE: func(cmd *cobra.Command, args []string) error {
		sploits, err := cmdutil.ExtractSploits(false)
		if err != nil {
			return errors.Wrap(err, "cannot build sploits")
		}

		serviceSploits := make(map[string][]string)
		for _, s := range sploits {
			sploitID := s.SploitID()
			service := sploitID.Service
			sploit := sploitID.Name

			serviceSploits[service] = append(serviceSploits[service], sploit)
		}

		util.PrintJson(serviceSploits)
		return nil
	},
}
