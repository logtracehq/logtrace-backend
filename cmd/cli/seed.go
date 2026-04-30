package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/datastore/postgres"
	"gitlab.com/logtrace/logtrace/internal/seed"
)

func addSeedCommand(c *cobra.Command, cfg *config.Config) {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data (10 events, 10 sessions, 10 audit logs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, err := cmd.Flags().GetString("org-id")
			if err != nil {
				return err
			}
			if orgID == "" {
				return fmt.Errorf("--org-id is required")
			}

			logger, err := getLogger(*cfg)
			if err != nil {
				return fmt.Errorf("failed to initialise logger: %w", err)
			}
			defer logger.Sync() //nolint:errcheck

			db, err := postgres.New(cfg, logger)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			eventRepo := postgres.NewEventRepository(db)
			sessionRepo := postgres.NewSessionRepository(db)
			auditLogRepo := postgres.NewAuditLogRepository(db)

			if err := seed.Run(context.Background(), orgID, eventRepo, sessionRepo, auditLogRepo); err != nil {
				return fmt.Errorf("seed failed: %w", err)
			}

			fmt.Println("seed completed successfully")
			return nil
		},
	}

	cmd.Flags().String("org-id", "", "UUID of the organization to seed data into (required)")
	c.AddCommand(cmd)
}
