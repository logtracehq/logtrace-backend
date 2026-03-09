package watermillqueue

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	wotelfloss "github.com/dentech-floss/watermill-opentelemetry-go-extra/pkg/opentelemetry"
	"github.com/garsue/watermillzap"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	wotel "github.com/voi-oss/watermill-opentelemetry/pkg/opentelemetry"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/email"
	queue "gitlab.com/logtrace/logtrace/internal/pkg/queues"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("watermill")

type WatermillClient struct {
	publisher  *message.Router
	susbcriber *redisstream.Subscriber
	messager   message.Publisher
	logger     *zap.Logger

	userRepo    logtrace.UserRepository
	orgRepo     logtrace.OrganizationRepository
	cfg         config.Config
	emailClient email.Client
}

func New(redisClient *redis.Client,
	cfg config.Config,
	logger *zap.Logger,
	emailClient email.Client,
	userRepo logtrace.UserRepository,
	orgRepo logtrace.OrganizationRepository,
) (queue.QueueHandler, error) {
	p, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     redisClient,
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		watermillzap.NewLogger(logger))
	if err != nil {
		return nil, err
	}

	publisher := wotel.NewNamedPublisherDecorator("queue.Publish",
		wotelfloss.NewTracePropagatingPublisherDecorator(p))

	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:       redisClient,
			Unmarshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		watermillzap.NewLogger(logger))
	if err != nil {
		return nil, err
	}

	router, err := message.NewRouter(message.RouterConfig{},
		watermillzap.NewLogger(logger))
	if err != nil {
		return nil, err
	}

	router.AddPlugin(plugin.SignalsHandler)

	poisionQueue, err := middleware.PoisonQueue(publisher, "poision.queue")
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		middleware.CorrelationID,

		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: time.Millisecond * 100,
			Logger:          watermill.NewStdLogger(false, false),
		}.Middleware,
		poisionQueue,
		// Recoverer handles panics from handlers.
		// In this case, it passes them as errors to the Retry middleware.
		middleware.Recoverer,
		// OTEL
		wotelfloss.ExtractRemoteParentSpanContext(),
		wotel.Trace(),
	)

	t := &WatermillClient{
		cfg:         cfg,
		publisher:   router,
		logger:      logger,
		messager:    publisher,
		susbcriber:  subscriber,
		userRepo:    userRepo,
		orgRepo:     orgRepo,
		emailClient: emailClient,
	}

	t.setUpRoutes(router, subscriber)

	return t, nil
}

func (t *WatermillClient) setUpRoutes(router *message.Router,
	subscriber *redisstream.Subscriber,
) {
	router.AddNoPublisherHandler(
		queue.QueueTopicBillingTrialEnding.String(),
		queue.QueueTopicBillingTrialEnding.String(),
		subscriber,
		t.sendBillingTrialEmail)

	router.AddNoPublisherHandler(
		queue.QueueTopicSubscriptionExpired.String(),
		queue.QueueTopicSubscriptionExpired.String(),
		subscriber,
		t.sendSubExpiredEmail)

	router.AddNoPublisherHandler(
		queue.QueueTopicShareDashboard.String(),
		queue.QueueTopicShareDashboard.String(),
		subscriber,
		t.sendDashboardSharingEmail)

	router.AddNoPublisherHandler(
		queue.QueueTopicVerifyEmail.String(),
		queue.QueueTopicVerifyEmail.String(),
		subscriber,
		t.sendEmailVerification,
	)
}

func (t *WatermillClient) Add(ctx context.Context,
	topic queue.QueueTopic, data any,
) error {
	return t.messager.Publish(
		topic.String(), message.NewMessage(uuid.NewString(),
			queue.ToPayload(data)))
}

func (t *WatermillClient) Start(context.Context) {
	_ = t.publisher.Run(context.Background())
}

func (t *WatermillClient) Close() error { return t.publisher.Close() }
