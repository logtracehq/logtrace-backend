package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/logtrace/logtrace/config"
	"go.uber.org/zap"
)

var (
	// Version describes the version of the current build.
	Version = "dev"

	// Commit describes the commit of the current build.
	Commit = "none"

	// Date describes the date of the current build.
	Date = time.Now().UTC()
)

const (
	defaultConfigFilePath = "config"
	envPrefix             = "LOGTRACE_"
)

func Execute() error {
	cfg := &config.Config{}

	rootCmd := &cobra.Command{
		Use:   "logtrace",
		Short: `Audit trail logging for regulatory compliance and security`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Use == "version" || cmd.Use == "export-config" {
				return nil
			}

			confFile, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}

			if err := config.InitializeConfig(cfg, confFile); err != nil {
				return err
			}

			return cfg.Validate()
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\nCommit: %s\nBuild Date: %s\n", Version, Commit, Date.Format(time.RFC3339))
		},
	}

	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringP("config", "c", defaultConfigFilePath, "Config file. This is in YAML")

	addHTTPCommand(rootCmd, cfg)
	addCronCommand(rootCmd, cfg)
	addConfigGenerateCommand(rootCmd)

	return rootCmd.Execute()
}

func getLogger(cfg config.Config) (*zap.Logger, error) {
	switch cfg.Logging.Mode {
	case config.LogModeProd:
		return zap.NewProduction()
	case config.LogModeDev:
		return zap.NewDevelopment()
	default:
		return zap.NewDevelopment()
	}
}
