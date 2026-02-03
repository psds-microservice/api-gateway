package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("API Gateway\nVersion: %s\nCommit: %s\nBuild: %s\n", Version, Commit, BuildDate)
		return nil
	},
}
