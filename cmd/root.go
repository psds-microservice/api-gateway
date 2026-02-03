package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "api-gateway",
	Short: "API Gateway: video streaming, client info, HTTP + gRPC",
	RunE:  runAPI, // по умолчанию — запуск API
}

// Execute запускает корневую команду (Cobra CLI)
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
	rootCmd.AddCommand(versionCmd)
}
