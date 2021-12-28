package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates a new instance of the root command
func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kangal",
		Short:   "Kangal is an application for creating environments for performance testing",
		Version: version,
	}

	cmd.AddCommand(NewProxyCmd())
	cmd.AddCommand(NewControllerCmd())

	return cmd
}
