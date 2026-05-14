package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/riandyrn/otelchi"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/cache"
	"gitlab.com/logtrace/logtrace/internal/pkg/googleauth"
	"gitlab.com/logtrace/logtrace/internal/pkg/jwttoken"
	queue "gitlab.com/logtrace/logtrace/internal/pkg/queues"
	_ "gitlab.com/logtrace/logtrace/swagger"
	"go.uber.org/zap"
)

func New(
	logger *zap.Logger,
	cfg config.Config,
	userRepo logtrace.UserRepository,
	orgRepo logtrace.OrganizationRepository,
	emailVerificationRepo logtrace.EmailVerificationRepository,
	jwtTokenManager jwttoken.JWTokenManager,
	queueHandler queue.QueueHandler,
	googleAuthProvider googleauth.GoogleAuthProvider,
	eventRepo logtrace.EventRepository, sessionRepo logtrace.SessionRepository,
	redisCache cache.Cache, apiKeyRepo logtrace.APIKeyRepository,
	planRepo logtrace.PlanRepository, passwordRepo logtrace.PasswordRepository,
	auditLogRepo logtrace.AuditLogRepository, organizationUserRepo logtrace.OrganizationUserRepository, invitationRepo logtrace.InvitationRepository,
	metricsHandler http.Handler,
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
			googleAuthProvider, eventRepo, sessionRepo, redisCache,
			apiKeyRepo, planRepo, passwordRepo, auditLogRepo, organizationUserRepo,
			invitationRepo, metricsHandler,
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
	userRepo logtrace.UserRepository,
	orgRepo logtrace.OrganizationRepository,
	jwtTokenManager jwttoken.JWTokenManager,
	queueHandler queue.QueueHandler,
	emailVerificationRepo logtrace.EmailVerificationRepository,
	googleAuthProvider googleauth.GoogleAuthProvider,
	eventRepo logtrace.EventRepository,
	sessionRepo logtrace.SessionRepository, _ cache.Cache,
	apiKeyRepo logtrace.APIKeyRepository, planRepo logtrace.PlanRepository,
	passwordRepo logtrace.PasswordRepository, auditLogRepo logtrace.AuditLogRepository,
	organizationUserRepo logtrace.OrganizationUserRepository, invitationRepo logtrace.InvitationRepository,
	metricsHandler http.Handler,
) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(writeRequestIDHeader)
	router.Use(middleware.RealIP)
	router.Use(httprate.Limit(100, time.Minute, httprate.WithKeyFuncs(HTTPThrottleKeyFunc)))
	router.Use(
		middleware.AllowContentType("application/json", "multipart/form-data"))
	router.Use(
		otelchi.Middleware("logtrace.server",
			otelchi.WithChiRoutes(router)))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.Frontend.AppURL, "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by browsers
	}))

	// router.Use(CSRFMiddleware([]byte(cfg.CSRFSecret), cfg))

	if cfg.Env == config.EnvTypeDevelopment {
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
		orgUserRepo:  organizationUserRepo,
		queue:        queueHandler,
	}

	event := &eventHander{
		eventRepo:   eventRepo,
		userRepo:    userRepo,
		orgRepo:     orgRepo,
		queue:       queueHandler,
		cfg:         cfg,
		orgUserRepo: organizationUserRepo,
	}

	session := &sessionHandler{
		cfg:         cfg,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		orgRepo:     orgRepo,
		orgUserRepo: organizationUserRepo,
		queue:       queueHandler,
	}

	apiKey := &apiKeyHandler{
		cfg:        cfg,
		apiKeyRepo: apiKeyRepo,
	}

	plan := &planHandler{
		cfg:      cfg,
		planRepo: planRepo,
	}

	org := &orgHandler{
		cfg:      cfg,
		orgRepo:  orgRepo,
		userRepo: userRepo,
	}

	invitation := &invitationHandler{
		cfg:            cfg,
		tokenManager:   jwtTokenManager,
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
	}

	router.Route("/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"OK","service":"All services operational"}`))
		})

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"status":"NotFound","error":"not found"}`))
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/connect/{provider}", WrapLogtraceHTTPHandler(logger, auth.login, cfg, "Auth.provider"))
			r.Post("/register", WrapLogtraceHTTPHandler(logger, auth.emailSignUp, cfg, "Auth.register"))
		})

		r.Route("/auth/account", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Get("/me", WrapLogtraceHTTPHandler(logger, auth.fetchCurrentUser, cfg, "Auth.me"))
			r.Post("/invite", WrapLogtraceHTTPHandler(logger, auth.inviteUserByEmail, cfg, "Auth.inviteUserByEmail"))
			r.Get("/users", WrapLogtraceHTTPHandler(logger, auth.listOrganizationUsers, cfg, "Auth.listOrganizationUsers"))
			r.Patch("/revoke/{reference}", WrapLogtraceHTTPHandler(logger, auth.revokeUserRole, cfg, "Auth.revokeUserRole"))
			r.Patch("/edit", WrapLogtraceHTTPHandler(logger, auth.editProfile, cfg, "Auth.editProfile"))
		})

		r.Route("/organizations", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Post("/", WrapLogtraceHTTPHandler(logger, org.createOrganization, cfg, "Organization.create"))
			r.Patch("/", WrapLogtraceHTTPHandler(logger, org.updateOrganization, cfg, "Organization.update"))
			r.Delete("/", WrapLogtraceHTTPHandler(logger, org.deleteOrganization, cfg, "Organization.delete"))
		})

		r.Route("/audit-logs", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Get("/{reference}", WrapLogtraceHTTPHandler(logger, auditLog.List, cfg, "AuditLog.listAuditLog"))
			r.Get("/", WrapLogtraceHTTPHandler(logger, auditLog.ListAll, cfg, "AuditLog.listAll"))
		})

		r.Route("/events", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Post("/", WrapLogtraceHTTPHandler(logger, event.Create, cfg, "Event.create"))
			r.Get("/{reference}", WrapLogtraceHTTPHandler(logger, event.List, cfg, "Event.list"))
			r.Get("/", WrapLogtraceHTTPHandler(logger, event.ListAll, cfg, "Event.listAll"))
		})

		r.Route("/sessions", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Get("/{reference}", WrapLogtraceHTTPHandler(logger, session.List, cfg, "Session.list"))
			r.Get("/", WrapLogtraceHTTPHandler(logger, session.ListAll, cfg, "Session.listAll"))
		})

		r.Route("/developers/keys", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Post("/", WrapLogtraceHTTPHandler(logger, apiKey.create, cfg, "APIKeys.create"))
			r.Get("/", WrapLogtraceHTTPHandler(logger, apiKey.list, cfg, "APIKeys.list"))
			r.Delete("/{reference}", WrapLogtraceHTTPHandler(logger, apiKey.revoke, cfg, "APIKeys.revoke"))
		})

		r.Route("/plans/admin", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Post("/", WrapLogtraceHTTPHandler(logger, plan.Create, cfg, "Plan.create"))
			r.Get("/{reference}", WrapLogtraceHTTPHandler(logger, plan.List, cfg, "Plan.list"))
			r.Patch("/{reference}", WrapLogtraceHTTPHandler(logger, plan.Update, cfg, "Plan.update"))
			r.Delete("/{reference}", WrapLogtraceHTTPHandler(logger, plan.Delete, cfg, "Plan.delete"))
		})

		r.Route("/plans", func(r chi.Router) {
			r.Get("/", WrapLogtraceHTTPHandler(logger, plan.ListAll, cfg, "Plan.listAll"))
		})

		r.Route("/developers/", func(r chi.Router) {
			r.Use(requireAPIKeyOnly(logger, cfg, apiKeyRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Post("/events", WrapLogtraceHTTPHandler(logger, event.Create, cfg, "Event.create"))
			r.Post("/sessions", WrapLogtraceHTTPHandler(logger, session.Create, cfg, "Session.create"))
			r.Post("/sessions/logout", WrapLogtraceHTTPHandler(logger, session.Logout, cfg, "Session.logout"))
			r.Post("/audit-logs", WrapLogtraceHTTPHandler(logger, auditLog.Create, cfg, "AuditLog.create"))
		})

		r.Route("/metrics", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Get("/sessions", WrapLogtraceHTTPHandler(logger, session.Metrics, cfg, "Metrics.sessions"))
			r.Get("/events", WrapLogtraceHTTPHandler(logger, event.Metrics, cfg, "Metrics.events"))
			r.Get("/events/top-actors", WrapLogtraceHTTPHandler(logger, event.TopActorMetrics, cfg, "Metrics.events.topActors"))
			r.Get("/auditlogs", WrapLogtraceHTTPHandler(logger, auditLog.Metrics, cfg, "Metrics.auditLogs"))
		})

		r.Route("/invitations", func(r chi.Router) {
			r.Use(requireAuthentication(logger, jwtTokenManager, cfg, userRepo, orgRepo))
			r.Use(requireOrganizationValidSubscription(cfg))
			r.Post("/", WrapLogtraceHTTPHandler(logger, invitation.Create, cfg, "Invitation.create"))
			r.Get("/", WrapLogtraceHTTPHandler(logger, invitation.List, cfg, "Invitation.list"))
			r.Delete("/{reference}", WrapLogtraceHTTPHandler(logger, invitation.Delete, cfg, "Invitation.delete"))
		})
	})
	router.Handle("/prometheus/metrics", metricsHandler)
	return cors.AllowAll().Handler(router)
}
