package cmd

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/api-gateway/internal/command"
	"github.com/psds-microservice/api-gateway/internal/config"
	"github.com/psds-microservice/api-gateway/internal/database"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run database seeds (migrate up first)",
	RunE:  runSeed,
}

func runSeed(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	cfg := config.LoadConfigFromEnv()

	if err := command.MigrateUp(cfg.DatabaseURL()); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	db, err := database.Open(cfg.DSN())
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer db.Close()

	if err := command.Seed(db); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	log.Println("seed: ok")
	return nil
}
