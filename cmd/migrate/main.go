package main

import (
	"flag"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/viper"
	"github.com/terra-consults/logbase/config"
)

func main() {
	var (
		path    string
		action  string
		steps   int
		version int
	)

	flag.StringVar(&path, "path", "./migrations", "Path to migrations directory")
	flag.StringVar(&action, "action", "up", "Action: up | down | force | version")
	flag.IntVar(&steps, "steps", 1, "Number of steps for down action")
	flag.IntVar(&version, "version", -1, "Version for force action")
	flag.Parse()

	// Initialize configuration defaults and enable environment overrides.
	cfg := &config.Config{}
	cfg.SetDefaultValues()
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	dsn := cfg.Database.Postgres.DSN
	m, err := migrate.New("file://"+path, dsn)
	if err != nil {
		log.Fatalf("failed to initialize migrate instance: %v", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("source close error: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("db close error: %v", dbErr)
		}
	}()

	switch action {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("migration up failed: %v", err)
		}
		log.Println("migrations applied successfully")
	case "down":
		if steps < 1 {
			log.Fatalf("steps must be >= 1")
		}
		if err := m.Steps(-steps); err != nil {
			log.Fatalf("migration down failed: %v", err)
		}
		log.Printf("rolled back %d step(s)", steps)
	case "force":
		if version < 0 {
			log.Fatalf("version must be specified for force")
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("force failed: %v", err)
		}
		log.Printf("forced version to %d", version)
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			if err == migrate.ErrNilVersion {
				log.Println("no version applied yet")
			} else {
				log.Fatalf("version query failed: %v", err)
			}
		} else {
			log.Printf("current version: %d dirty=%v", v, dirty)
		}
	default:
		log.Fatalf("unknown action: %s", action)
	}
}
