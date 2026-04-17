package watermillqueue

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	queue "gitlab.com/logtrace/logtrace/internal/pkg/queues"
	"go.uber.org/zap"
)

func (t *WatermillClient) saveEvent(msg *message.Message) error {
	ctx, span := tracer.Start(context.Background(), "saveEvent")
	defer span.End()

	var opts queue.SaveEventOptions

	if err := json.NewDecoder(bytes.NewBuffer(msg.Payload)).Decode(&opts); err != nil {
		return err
	}

	logger := t.logger.With(
		zap.String("method", "saveEvent"),
		zap.String("organization_id", opts.Event.OrganizationID.String()),
	)

	logger.Debug("saving event to database")

	if err := t.eventRepo.Create(ctx, opts.Event); err != nil {
		logger.Error("failed to save event", zap.Error(err))
		return err
	}

	return nil
}

func (t *WatermillClient) saveAuditLog(msg *message.Message) error {
	ctx, span := tracer.Start(context.Background(), "saveAuditLog")
	defer span.End()

	var opts queue.SaveAuditLogOptions

	if err := json.NewDecoder(bytes.NewBuffer(msg.Payload)).Decode(&opts); err != nil {
		return err
	}

	logger := t.logger.With(
		zap.String("method", "saveAuditLog"),
		zap.String("organization_id", opts.AuditLog.OrganizationID.String()),
	)

	logger.Debug("saving audit log to database")

	if err := t.auditLogRepo.Create(ctx, opts.AuditLog); err != nil {
		logger.Error("failed to save audit log", zap.Error(err))
		return err
	}

	return nil
}

func (t *WatermillClient) saveSession(msg *message.Message) error {
	ctx, span := tracer.Start(context.Background(), "saveSession")
	defer span.End()

	var opts queue.SaveSessionOptions

	if err := json.NewDecoder(bytes.NewBuffer(msg.Payload)).Decode(&opts); err != nil {
		return err
	}

	logger := t.logger.With(
		zap.String("method", "saveSession"),
		zap.String("organization_id", opts.Session.OrganizationID.String()),
	)

	logger.Debug("saving session to database")

	if err := t.sessionRepo.Create(ctx, opts.Session); err != nil {
		logger.Error("failed to save session", zap.Error(err))
		return err
	}

	return nil
}
