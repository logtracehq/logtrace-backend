package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/riandyrn/otelchi"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/terra-consults/logbase"
	"github.com/terra-consults/logbase/config"
	"github.com/terra-consults/logbase/internal/pkg/cache"
	"github.com/terra-consults/logbase/internal/pkg/googleauth"
	"github.com/terra-consults/logbase/internal/pkg/jwttoken"
	queue "github.com/terra-consults/logbase/internal/pkg/queues"
	_ "github.com/terra-consults/logbase/swagger"
	"go.uber.org/zap"
)

func New(
	logger *zap.Logger,
	cfg config.Config,
	userRepo logbase.UserRepository,
	orgRepo logbase.OrganizationRepository,
	emailVerificationRepo logbase.EmailVerificationRepository,
	jwtTokenManager jwttoken.JWTokenManager,
	queueHandler queue.QueueHandler,
	googleAuthProvider googleauth.GoogleAuthProvider,
	eventRepo logbase.EventRepository, sessionRepo logbase.SessionRepository,
	redisCache cache.Cache, resourceRepo logbase.ResourceRepository, apiKeyRepo logbase.APIKeyRepository,
	planRepo logbase.PlanRepository, passwordRepo logbase.PasswordRepository,
	auditLogRepo logbase.AuditLogRepository,
) (*http.Server, func(context.Context)) {
	if err := cfg.Validate(); err != nil {
		logger.Fatal("invalid configuration", zap.Error(err))
		return nil, nil
	}

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: setUpRoutes(
			logger, cfg, userRepo,
			orgRepo, jwtTokenManager,
			queueHandler, emailVerificationRepo,
			googleAuthProvider, eventRepo, sessionRepo, redisCache, resourceRepo,
			apiKeyRepo, planRepo, passwordRepo, auditLogRepo,
		),
	}

	cleanup := func(ctx context.Context) {
		logger.Info("Shutting down HTTP server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server forced to shutdown", zap.Error(err))
		} else {
			logger.Info("Server stopped cleanly")
		}
	}

	return srv, cleanup
}

func setUpRoutes(
	logger *zap.Logger,
	cfg config.Config,
	userRepo logbase.UserRepository,
	orgRepo logbase.OrganizationRepository,
	jwtTokenManager jwttoken.JWTokenManager,
	queueHandler queue.QueueHandler,
	emailVerificationRepo logbase.EmailVerificationRepository,
	googleAuthProvider googleauth.GoogleAuthProvider,
	eventRepo logbase.EventRepository,
	sessionRepo logbase.SessionRepository, _ cache.Cache,
	resourceRepo logbase.ResourceRepository, apiKeyRepo logbase.APIKeyRepository,
	planRepo logbase.PlanRepository, passwordRepo logbase.PasswordRepository,
	auditLogRepo logbase.AuditLogRepository,
) http.Handler {
	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.Frontend.AppURL, "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by browsers
	}))

	router.Use(middleware.RequestID)
	router.Use(writeRequestIDHeader)
	router.Use(
		middleware.AllowContentType("application/json", "multipart/form-data"))

	router.Use(
		otelchi.Middleware("logbase.server",
			otelchi.WithChiRoutes(router)))

	if cfg.Env == "development" {
		go func() {
			r := chi.NewRouter()

			r.Get("/swagger/*", httpSwagger.Handler(
				httpSwagger.URL(
					fmt.Sprintf("http://localhost:%d/swagger/doc.json", cfg.HTTP.Swagger.Port),
				),
			))

			if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTP.Swagger.Port), r); err != nil {
				logger.Error("error with swagger server", zap.Error(err))
			}
		}()
	}

	auth := &authHandler{
		userRepo:          userRepo,
		orgRepo:           orgRepo,
		googleCfg:         googleAuthProvider,
		cfg:               cfg,
		tokenManager:      jwtTokenManager,
		queue:             queueHandler,
		emailVerification: emailVerificationRepo,
		passwordRepo:      passwordRepo,
	}

	auditLog := &auditLogHandler{
		userRepo:     userRepo,
		orgRepo:      orgRepo,
		auditLogRepo: auditLogRepo,
	}

	event := &eventHander{
		eventRepo: eventRepo,
		userRepo:  userRepo,
		orgRepo:   orgRepo,
		queue:     queueHandler,
		cfg:       cfg,
	}

	session := &sessionHandler{
		cfg:         cfg,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		orgRepo:     orgRepo,
	}

	resource := &resourceHandler{
		cfg:          cfg,
		userRepo:     userRepo,
		resourceRepo: resourceRepo,
	}

	apiKey := &apiKeyHandler{
		cfg:        cfg,
		apiKeyRepo: apiKeyRepo,
	}

	plan := &planHandler{
		cfg:      cfg,
		planRepo: planRepo,
	}

	router.Route("/v1", func(r chi.Router) {
		// Health endpoint
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"OK","service":"All services operational"}`))
		})

		// Not found route
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"status":"NotFound","error":"not found"}`))
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/connect/{provider}", WrapLogbaseHTTPHandler(logger, auth.login, cfg, "Auth.provider"))
			r.Post("/register", WrapLogbaseHTTPHandler(logger, auth.emailSignUp, cfg, "Auth.register"))
		})

		r.Route("/audit-logs", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Post("/", WrapLogbaseHTTPHandler(logger, auditLog.Create, cfg, "AuditLog.create"))
			r.Get("/{reference}", WrapLogbaseHTTPHandler(logger, auditLog.List, cfg, "AuditLog.listAuditLog"))
			r.Get("/", WrapLogbaseHTTPHandler(logger, auditLog.ListAll, cfg, "AuditLog.listAll"))
		})

		r.Route("/events", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Post("/", WrapLogbaseHTTPHandler(logger, event.Create, cfg, "Event.create"))
			r.Get("/{reference}", WrapLogbaseHTTPHandler(logger, event.List, cfg, "Event.list"))
			r.Get("/", WrapLogbaseHTTPHandler(logger, event.ListAll, cfg, "Event.listAll"))
		})

		r.Route("/sessions", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Post("/", WrapLogbaseHTTPHandler(logger, session.Create, cfg, "Session.create"))
			r.Get("/{reference}", WrapLogbaseHTTPHandler(logger, session.List, cfg, "Session.list"))
			r.Get("/", WrapLogbaseHTTPHandler(logger, session.ListAll, cfg, "Session.listAll"))
		})

		r.Route("/resources", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Post("/", WrapLogbaseHTTPHandler(logger, resource.Create, cfg, "Resource.create"))
			r.Get("/", WrapLogbaseHTTPHandler(logger, resource.ListAll, cfg, "Resource.listAll"))
			r.Get("/{reference}", WrapLogbaseHTTPHandler(logger, resource.List, cfg, "Resource.list"))
			r.Delete("/{reference}", WrapLogbaseHTTPHandler(logger, resource.Delete, cfg, "Resource.delete"))
		})

		r.Route("/api-keys", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Post("/", WrapLogbaseHTTPHandler(logger, apiKey.create, cfg, "APIKeys.create"))
			r.Get("/", WrapLogbaseHTTPHandler(logger, apiKey.list, cfg, "APIKeys.list"))
			r.Delete("/{reference}", WrapLogbaseHTTPHandler(logger, apiKey.revoke, cfg, "APIKeys.revoke"))
		})

		r.Route("/plans", func(r chi.Router) {
			r.Post("/", WrapLogbaseHTTPHandler(logger, plan.Create, cfg, "Plan.create"))
			r.Get("/{reference}", WrapLogbaseHTTPHandler(logger, plan.Get, cfg, "Plan.listAll"))
			r.Get("/", WrapLogbaseHTTPHandler(logger, plan.ListAll, cfg, "Plan.listAll"))
		})
	})

	return cors.AllowAll().Handler(router)
}
