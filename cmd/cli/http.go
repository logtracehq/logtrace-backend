package cli

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/datastore/postgres"
	"gitlab.com/logtrace/logtrace/internal/pkg/cache/rediscache"
	"gitlab.com/logtrace/logtrace/internal/pkg/email"
	awsses "gitlab.com/logtrace/logtrace/internal/pkg/email/aws-ses"
	"gitlab.com/logtrace/logtrace/internal/pkg/googleauth"
	"gitlab.com/logtrace/logtrace/internal/pkg/jwttoken"
	watermillqueue "gitlab.com/logtrace/logtrace/internal/pkg/queues/watermill"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"gitlab.com/logtrace/logtrace/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// @title Logtrace API
// @version 0.1.0
// @description Logtrace API documentation
// @host localhost:8080
// @BasePath /
// @schemes http https
func addHTTPCommand(c *cobra.Command, cfg *config.Config) {
	cmd := &cobra.Command{
		Use:   "http",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			logCfg := zap.NewProductionConfig()
			logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			logger, err := logCfg.Build()
			if err != nil {
				return err
			}
			defer logger.Sync()

			h, _ := os.Hostname()
			logger = logger.With(
				zap.String("host", h),
				zap.String("app", "logtrace"),
				zap.String("component", "http"),
			)

			opts, err := redis.ParseURL(cfg.Database.Redis.DSN)
			if err != nil {
				logger.Fatal("could not parse redis dsn", zap.Error(err))
			}

			redisClient := redis.NewClient(opts)

			if err := redisotel.InstrumentTracing(redisClient); err != nil {
				logger.Fatal("could not instrument tracing of redis client", zap.Error(err))
			}

			if err := redisotel.InstrumentMetrics(redisClient); err != nil {
				logger.Fatal("could not instrument metrics collection of redis client", zap.Error(err))
			}

			db, err := postgres.New(cfg, logger)
			if err != nil {
				logger.Fatal("failed to initialize database connection", zap.Error(err))
			}

			if err := runMigrations(logger, cfg); err != nil {
				logger.Fatal("could not run migrations",
					zap.Error(err))
			}

			logger.Info("database connection established successfully")

			userRepo := postgres.NewUserRepository(db)
			sessionRepo := postgres.NewSessionRepository(db)
			eventRepo := postgres.NewEventRepository(db)
			orgRepo := postgres.NewOrganizationRepository(db)
			emailRepo := postgres.NewEmailRepository(db)
			apiKeyRepo := postgres.NewAPIKeyRepository(db)
			planRepo := postgres.NewPlanRepository(db)
			passwordRepo := postgres.NewPasswordRepository(db)
			auditLogRepo := postgres.NewAuditLogRepository(db)
			organizationUserRepo := postgres.NewOrganizationUserRepository(db)
			invitationRepo := postgres.NewInvitationRepository(db)

			tokenManager := jwttoken.New(cfg)
			googleAuth := googleauth.NewGoogle(cfg)

			var emailClient email.Client

			emailClient, err = awsses.New(cfg)
			if err != nil {
				logger.Fatal("could not set up email client", zap.Error(err))
			}

			queueHandler, err := watermillqueue.New(
				redisClient, util.DeRef(cfg),
				logger, emailClient, userRepo, orgRepo, eventRepo, auditLogRepo, sessionRepo)
			if err != nil {
				logger.Fatal("could not set up watermill queue", zap.Error(err))
			}

			go func() {
				queueHandler.Start(context.Background())
			}()

			redisCache, err := rediscache.New(redisClient)
			if err != nil {
				logger.Fatal("could not set up redis cache", zap.Error(err))
			}

			srv, cleanupSrv := server.New(
				logger, util.DeRef(cfg), userRepo,
				orgRepo, emailRepo, tokenManager, queueHandler,
				googleAuth, eventRepo, sessionRepo, redisCache,
				apiKeyRepo, planRepo, passwordRepo,
				auditLogRepo, organizationUserRepo, invitationRepo,
			)

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("unexpected error with http server", zap.Error(err))
				}
			}()

			logger.Info("logtrace server started successfully on port", zap.Int("port", cfg.HTTP.Port))

			<-sig

			logger.Info("shutting down logtrace server")
			ctx := context.Background()
			cleanupSrv(ctx)

			if err := db.Close(); err != nil {
				logger.Error("could not close db", zap.Error(err))
			}

			_ = logger.Sync()
			logger.Info("shutdown complete")
			return nil
		},
	}

	c.AddCommand(cmd)
}
