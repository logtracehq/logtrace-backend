package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/datastore/postgres"
	pkgemail "gitlab.com/logtrace/logtrace/internal/pkg/email"
	awsses "gitlab.com/logtrace/logtrace/internal/pkg/email/aws-ses"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"gitlab.com/logtrace/logtrace/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

const (
	defaultBatchSize         = 100
	defaultWorkerCount       = 5
	defaultMaxRetries        = 3
	defaultProcessingTimeout = 30 * time.Minute
	defaultEmailTimeout      = 10 * time.Second
	verificationEmailSubject = "Verify your account to get started with Logtrace"
)

type ProcessMetrics struct {
	TotalEmails     int64
	SentEmails      int64
	FailedEmails    int64
	LastProcessedID string
	StartTime       time.Time
	EndTime         time.Time
	mu              sync.Mutex
}

func (m *ProcessMetrics) IncrementSent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentEmails++
}

func (m *ProcessMetrics) IncrementFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedEmails++
}

type EmailJob struct {
	Recipient   *logtrace.User
	Subject     string
	Content     string
	RetryCount  int
	LastError   error
	LastAttempt time.Time
}

type EmailProcessor struct {
	db                    *bun.DB
	emailVerificationRepo logtrace.EmailVerificationRepository
	emailClient           pkgemail.Client
	logger                *zap.Logger
	tracer                trace.Tracer
	cfg                   *config.Config
	metrics               *ProcessMetrics
	rateLimiter           *rate.Limiter
	opts                  ProcessorOptions
}

type ProcessorOptions struct {
	BatchSize         int
	WorkerCount       int
	MaxRetries        int
	ProcessingTimeout time.Duration
	EmailTimeout      time.Duration
	RateLimit         int
}

func DefaultProcessorOptions() ProcessorOptions {
	return ProcessorOptions{
		BatchSize:         defaultBatchSize,
		WorkerCount:       defaultWorkerCount,
		MaxRetries:        defaultMaxRetries,
		ProcessingTimeout: defaultProcessingTimeout,
		EmailTimeout:      defaultEmailTimeout,
		RateLimit:         10,
	}
}

func NewEmailProcessor(db *bun.DB, emailClient pkgemail.Client, logger *zap.Logger, tracer trace.Tracer, cfg *config.Config, opts ProcessorOptions) *EmailProcessor {
	return &EmailProcessor{
		db:                    db,
		emailVerificationRepo: postgres.NewEmailRepository(db),
		emailClient:           emailClient,
		logger:                logger,
		tracer:                tracer,
		cfg:                   cfg,
		metrics:               &ProcessMetrics{StartTime: time.Now()},
		rateLimiter:           rate.NewLimiter(rate.Limit(opts.RateLimit), opts.RateLimit),
		opts:                  opts,
	}
}

func fetchPendingUsers(ctx context.Context, db *bun.DB, offset, limit int) ([]*logtrace.User, error) {
	users := make([]*logtrace.User, 0, limit)
	err := db.NewSelect().
		Model(&users).
		Where("deleted_at IS NULL").
		Where("email_verified_at IS NULL").
		Where("status = ?", logtrace.UserStatusPending).
		OrderExpr("created_at ASC, id ASC").
		Offset(offset).
		Limit(limit).
		Scan(ctx)

	return users, err
}

func (p *EmailProcessor) ProcessPendingUsers(ctx context.Context) error {
	ctx, span := p.tracer.Start(ctx, "processPendingUsers")
	defer span.End()

	offset := 0
	for {
		batchCtx, cancel := context.WithTimeout(ctx, p.opts.ProcessingTimeout)
		users, err := fetchPendingUsers(batchCtx, p.db, offset, p.opts.BatchSize)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to fetch users: %w", err)
		}
		if len(users) == 0 {
			break
		}

		jobs, err := p.createEmailJobs(ctx, users)
		if err != nil {
			return fmt.Errorf("failed to create email jobs: %w", err)
		}
		if len(jobs) == 0 {
			offset += len(users)
			continue
		}

		p.metrics.TotalEmails += int64(len(jobs))

		results := make(chan error, len(jobs))
		if err := p.processEmailBatch(ctx, jobs, results); err != nil {
			return fmt.Errorf("failed to process email batch: %w", err)
		}

		for i := 0; i < len(jobs); i++ {
			if err := <-results; err != nil {
				p.logger.Error("email processing failed",
					zap.Error(err),
					zap.String("user_id", jobs[i].Recipient.ID.String()),
					zap.String("recipient", jobs[i].Recipient.Email.String()))
			}
		}

		p.metrics.LastProcessedID = users[len(users)-1].ID.String()
		offset += len(users)
	}

	p.metrics.EndTime = time.Now()
	if p.metrics.FailedEmails > 0 {
		return fmt.Errorf("some emails failed to send: %d/%d", p.metrics.FailedEmails, p.metrics.TotalEmails)
	}

	return nil
}

func (p *EmailProcessor) createEmailJobs(ctx context.Context, users []*logtrace.User) ([]*EmailJob, error) {
	jobs := make([]*EmailJob, 0, len(users))
	for _, user := range users {
		job, err := p.createEmailJob(ctx, user)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (p *EmailProcessor) createEmailJob(ctx context.Context, user *logtrace.User) (*EmailJob, error) {
	verification, err := logtrace.NewEmailVerification(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token for user %s: %w", user.ID, err)
	}

	if err := p.emailVerificationRepo.Create(ctx, verification); err != nil {
		return nil, fmt.Errorf("failed to store verification token for user %s: %w", user.ID, err)
	}

	content, err := prepareVerificationEmailTemplate(user, p.cfg.Frontend.AppURL, verification.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to render verification email for user %s: %w", user.ID, err)
	}

	return &EmailJob{
		Recipient: user,
		Subject:   verificationEmailSubject,
		Content:   content,
	}, nil
}

func (p *EmailProcessor) processEmailBatch(ctx context.Context, jobs []*EmailJob, results chan<- error) error {
	workerPool := make(chan struct{}, p.opts.WorkerCount)
	var wg sync.WaitGroup

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case workerPool <- struct{}{}:
			wg.Add(1)
			go func(j *EmailJob) {
				defer wg.Done()
				defer func() { <-workerPool }()

				err := p.processEmailWithRetry(ctx, j)
				if err != nil {
					p.metrics.IncrementFailed()
					results <- err
					return
				}

				p.metrics.IncrementSent()
				results <- nil
			}(job)
		}
	}

	wg.Wait()
	return nil
}

func (p *EmailProcessor) processEmailWithRetry(ctx context.Context, job *EmailJob) error {
	var err error
	for attempt := 0; attempt <= p.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if err := waitForRetry(ctx, backoff); err != nil {
				return err
			}
		}

		err = p.rateLimiter.Wait(ctx)
		if err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		emailCtx, cancel := context.WithTimeout(ctx, p.opts.EmailTimeout)
		err = p.sendEmail(emailCtx, job)
		cancel()

		if err == nil {
			return nil
		}

		job.RetryCount++
		job.LastError = err
		job.LastAttempt = time.Now()

		p.logger.Warn("email send failed, will retry",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.String("recipient", job.Recipient.Email.String()))
	}

	return fmt.Errorf("max retries exceeded: %w", err)
}

func (p *EmailProcessor) sendEmail(ctx context.Context, job *EmailJob) error {
	_, err := p.emailClient.Send(ctx, pkgemail.SendOptions{
		HTML:      job.Content,
		Sender:    p.cfg.Email.Sender,
		Recipient: job.Recipient.Email,
		Subject:   job.Subject,
		DKIM: struct {
			Sign       bool
			PrivateKey []byte
		}{
			Sign:       false,
			PrivateKey: []byte(""),
		},
	})

	return err
}

func sendScheduledUpdates(c *cobra.Command, cfg *config.Config) *cobra.Command {
	opts := DefaultProcessorOptions()

	cmd := &cobra.Command{
		Use:   "users",
		Short: "Send verification emails to pending users",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := setupLogger(cfg)
			if err != nil {
				return fmt.Errorf("failed to setup logger: %w", err)
			}

			cleanupOtel, _ := server.InitOTELCapabilities(util.DeRef(cfg), logger)
			defer cleanupOtel()

			emailClient, err := awsses.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to setup email client: %w", err)
			}
			defer emailClient.Close()

			db, err := postgres.New(cfg, logger)
			if err != nil {
				return fmt.Errorf("failed to setup database: %w", err)
			}
			defer db.Close()

			tracer := otel.Tracer("logtrace.cron")
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			ctx, span := tracer.Start(ctx, "users-send")
			defer span.End()

			processor := NewEmailProcessor(db, emailClient, logger, tracer, cfg, opts)
			if err := processor.ProcessPendingUsers(ctx); err != nil {
				return err
			}

			logger.Info("email processing completed",
				zap.Int64("total_emails", processor.metrics.TotalEmails),
				zap.Int64("sent_emails", processor.metrics.SentEmails),
				zap.Int64("failed_emails", processor.metrics.FailedEmails),
				zap.String("last_processed_user_id", processor.metrics.LastProcessedID),
				zap.Duration("duration", processor.metrics.EndTime.Sub(processor.metrics.StartTime)))

			return nil
		},
	}

	cmd.Flags().IntVar(&opts.BatchSize, "batch-size", opts.BatchSize, "number of users to process per batch")
	cmd.Flags().IntVar(&opts.WorkerCount, "workers", opts.WorkerCount, "number of concurrent email workers")
	cmd.Flags().IntVar(&opts.MaxRetries, "max-retries", opts.MaxRetries, "number of retries per email")
	cmd.Flags().DurationVar(&opts.ProcessingTimeout, "processing-timeout", opts.ProcessingTimeout, "maximum time allowed to fetch a batch")
	cmd.Flags().DurationVar(&opts.EmailTimeout, "email-timeout", opts.EmailTimeout, "maximum time allowed to send one email")
	cmd.Flags().IntVar(&opts.RateLimit, "rate-limit", opts.RateLimit, "maximum emails per second")

	return cmd
}

func setupLogger(cfg *config.Config) (*zap.Logger, error) {
	mode := strings.ToLower(cfg.LogLevel)
	if mode == "" {
		mode = strings.ToLower(string(cfg.Logging.Mode))
	}

	var (
		logger *zap.Logger
		err    error
	)

	switch mode {
	case "prod", "production":
		logCfg := zap.NewProductionConfig()
		logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		logger, err = logCfg.Build()
	default:
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	hostname, _ := os.Hostname()
	return logger.With(
		zap.String("host", hostname),
		zap.String("app", "logtrace"),
		zap.String("component", "cron.users-send"),
	), nil
}

func prepareVerificationEmailTemplate(user *logtrace.User, frontendAppURL, token string) (string, error) {
	tmpl, err := template.New("template").Parse(pkgemail.EmailVerificationTemplate)
	if err != nil {
		return "", err
	}

	link := strings.TrimRight(frontendAppURL, "/") + "/email-verify?token=" + token

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FullName": firstNameFromUser(user),
		"Link":     link,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func firstNameFromUser(user *logtrace.User) string {
	fullName := strings.TrimSpace(user.FullName)
	if fullName != "" {
		parts := strings.Fields(fullName)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	emailAddress := user.Email.String()
	if idx := strings.Index(emailAddress, "@"); idx > 0 {
		return emailAddress[:idx]
	}

	return "there"
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
