package cli

import (
	"github.com/spf13/cobra"
	"gitlab.com/logtrace/logtrace/config"
)

func addCronCommand(c *cobra.Command, cfg *config.Config) {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: `Start the cron job scheduler to run periodic tasks.`,
	}

	cmd.AddCommand(sendScheduledUpdates(c, cfg))
	cmd.AddCommand(revokeAPIKeys(c, cfg))
	c.AddCommand(cmd)
}
