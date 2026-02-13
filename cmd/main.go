package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"gitlab.com/logbase/logbase/config"
	"gitlab.com/logbase/logbase/internal/datastore/postgres"
	"gitlab.com/logbase/logbase/internal/pkg/cache/rediscache"
	"gitlab.com/logbase/logbase/internal/pkg/email"
	"gitlab.com/logbase/logbase/internal/pkg/email/resend"
	"gitlab.com/logbase/logbase/internal/pkg/googleauth"
	"gitlab.com/logbase/logbase/internal/pkg/jwttoken"
	watermillqueue "gitlab.com/logbase/logbase/internal/pkg/queues/watermill"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"gitlab.com/logbase/logbase/server"
	"go.uber.org/zap"
)

// @title Logbase API
// @version 0.1.0
// @description Logbase API documentation
// @host localhost:8080
// @BasePath /
// @schemes http https
func main() {
	cfg := &config.Config{}
	if err := config.InitializeConfig(cfg, ".."); err != nil {
		panic(err)
	}

	var logger *zap.Logger
	var err error

	switch cfg.LogLevel {
	case "prod", "production":
		logger, err = zap.NewProduction()
		if err != nil {
			fmt.Printf(`{"error":"%s"}`, err)
			os.Exit(1)
		}
	case "dev", "development":
		logger, err = zap.NewDevelopment()
		if err != nil {
			fmt.Printf(`{"error":"%s"}`, err)
			os.Exit(1)
		}
	default:
		logger, err = zap.NewDevelopment()
		if err != nil {
			fmt.Printf(`{"error":"%s"}`, err)
			os.Exit(1)
		}
	}

	opts, err := redis.ParseURL(cfg.Database.Redis.DSN)
	if err != nil {
		logger.Fatal("could not parse redis dsn", zap.Error(err))
	}

	redisClient := redis.NewClient(opts)

	if err := redisotel.InstrumentTracing(redisClient); err != nil {
		logger.Fatal("could not instrument tracing of redis client",
			zap.Error(err))

		if err := redisotel.InstrumentMetrics(redisClient); err != nil {
			logger.Fatal("could not instrument metrics collection of redis client",
				zap.Error(err))
		}
	}

	db, err := postgres.New(cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize database connection", zap.Error(err))
	} else {
		logger.Info("database connection established successfully")
	}

	userRepo := postgres.NewUserRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	orgRepo := postgres.NewOrganizationRepository(db)
	emailRepo := postgres.NewEmailRepository(db)
	apiKeyRepo := postgres.NewAPIKeyRepository(db)
	planRepo := postgres.NewPlanRepository(db)
	passwordRepo := postgres.NewPasswordRepository(db)
	auditLogRepo := postgres.NewAuditLogRepository(db)

	tokenManager := jwttoken.New(cfg)
	googleAuth := googleauth.NewGoogle(cfg)

	var emailClient email.Client

	emailClient, err = resend.New(cfg)
	if err != nil {
		logger.Fatal("could not set up smtp client",
			zap.Error(err))
	}

	queueHandler, err := watermillqueue.New(
		redisClient, util.DeRef(cfg),
		logger, emailClient, userRepo, orgRepo)
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
		auditLogRepo,
	)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("unexpected error with http server", zap.Error(err))
		}
	}()

	logger.Info("logbase server started successfully on port", zap.Int("port", cfg.HTTP.Port))

	<-sig

	logger.Info("shutting down logbase server")
	ctx := context.Background()
	cleanupSrv(ctx)

	if err := db.Close(); err != nil {
		logger.Error("could not close db", zap.Error(err))
	}

	_ = logger.Sync()
	logger.Info("shutdown complete")
}
