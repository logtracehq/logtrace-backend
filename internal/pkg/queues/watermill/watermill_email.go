package watermillqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/ThreeDotsLabs/watermill/message"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/internal/pkg/email"
	queue "gitlab.com/logtrace/logtrace/internal/pkg/queues"
	"go.uber.org/zap"
)

func (t *WatermillClient) sendSubExpiredEmail(msg *message.Message) error {
	ctx, span := tracer.Start(context.Background(),
		"sendSubExpiredEmail")

	defer span.End()

	var opts queue.SubscriptionExpiredOptions

	if err := json.NewDecoder(bytes.NewBuffer(msg.Payload)).
		Decode(&opts); err != nil {
		return err
	}

	logger := t.logger.With(zap.String("method", "sendSubExpiredEmail"),
		zap.String("workspace_id", opts.Organization.ID.String()))

	logger.Debug("sending sub expired email")

	tmpl, err := template.New("template").Parse(email.BillingEndedTemplate)
	if err != nil {
		logger.Error("could not parse email template", zap.Error(err))
		return err
	}

	link := t.cfg.Frontend.AppURL + "/settings?tab=billing"

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"WorkspaceName": opts.Organization.Name,
		"Link":          link,
	}); err != nil {
		logger.Error("could not embed content in template", zap.Error(err))
		return err
	}

	emailOpts := email.SendOptions{
		HTML:      buf.String(),
		Sender:    logtrace.Email(t.cfg.Email.Sender),
		Recipient: opts.Recipient,
		Subject:   "Your Logtrace subscription has come to an end. Please resubscribe",
		DKIM: struct {
			Sign       bool
			PrivateKey []byte
		}{
			Sign:       false,
			PrivateKey: []byte(""),
		},
	}

	_, err = t.emailClient.Send(ctx, emailOpts)
	if err != nil {
		logger.Error("could not send email", zap.Error(err))
		return err
	}

	return nil
}

func (t *WatermillClient) sendBillingTrialEmail(msg *message.Message) error {
	ctx, span := tracer.Start(context.Background(),
		"sendBillingTrialEmail")

	defer span.End()

	var opts queue.SendBillingTrialEmailOptions

	if err := json.NewDecoder(bytes.NewBuffer(msg.Payload)).
		Decode(&opts); err != nil {
		return err
	}

	logger := t.logger.With(zap.String("method", "sendBillingTrialEmail"),
		zap.String("workspace_id", opts.Organization.ID.String()))

	logger.Debug("sending email to user for free trial")

	tmpl, err := template.New("template").Parse(email.BillingTrialTemplate)
	if err != nil {
		logger.Error("could not parse email template", zap.Error(err))
		return err
	}

	link := t.cfg.Frontend.AppURL + "/settings?tab=billing"

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"WorkspaceName": opts.Organization.Name,
		"Link":          link,
		"Expiration":    opts.Expiration,
	}); err != nil {
		logger.Error("could not embed content in template", zap.Error(err))
		return err
	}

	emailOpts := email.SendOptions{
		HTML:      buf.String(),
		Sender:    t.cfg.Email.Sender,
		Recipient: opts.Recipient,
		Subject:   "Your Logtrace trial is coming to an end",
		DKIM: struct {
			Sign       bool
			PrivateKey []byte
		}{
			Sign:       false,
			PrivateKey: []byte(""),
		},
	}

	_, err = t.emailClient.Send(ctx, emailOpts)
	if err != nil {
		logger.Error("could not send email", zap.Error(err))
		return err
	}

	return nil
}

func (t *WatermillClient) sendDashboardSharingEmail(msg *message.Message) error {
	ctx, span := tracer.Start(context.Background(),
		"sendDashboardSharingEmail")

	defer span.End()

	var opts queue.SendEmailOptions

	if err := json.NewDecoder(bytes.NewBuffer(msg.Payload)).
		Decode(&opts); err != nil {
		return err
	}

	logger := t.logger.With(zap.String("method", "sendDashboardSharingEmail"),
		zap.String("workspace_id", opts.Organization.ID.String()))

	logger.Debug("sending email to user")

	tmpl, err := template.New("template").Parse(email.DashboardSharingTemplate)
	if err != nil {
		logger.Error("could not parse email template", zap.Error(err))
		return err
	}

	link := t.cfg.Frontend.AppURL + "/shared/dashboards/" + opts.Token

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"WorkspaceName": opts.Organization.Name,
		"Link":          link,
	}); err != nil {
		logger.Error("could not embed content in template", zap.Error(err))
		return err
	}

	emailOpts := email.SendOptions{
		HTML:      buf.String(),
		Sender:    t.cfg.Email.Sender,
		Recipient: opts.Recipient,
		Subject:   "Metrics dashboard shared with you by " + opts.Organization.Name,
		DKIM: struct {
			Sign       bool
			PrivateKey []byte
		}{
			Sign:       false,
			PrivateKey: []byte(""),
		},
	}

	_, err = t.emailClient.Send(ctx, emailOpts)
	if err != nil {
		logger.Error("could not send email", zap.Error(err))
		return err
	}

	return nil
}

func (t *WatermillClient) sendEmailVerification(msg *message.Message) error {
	ctx, span := tracer.Start(context.Background(),
		"sendEmailVerification")

	defer span.End()

	var opts queue.EmailVerificationOptions

	if err := json.NewDecoder(bytes.NewBuffer(msg.Payload)).
		Decode(&opts); err != nil {
		return err
	}

	logger := t.logger.With(zap.String("method", "sendEmailVerification"))

	logger.Debug("sending email to user")

	tmpl, err := template.New("template").Parse(email.EmailVerificationTemplate)
	if err != nil {
		logger.Error("could not parse email template", zap.Error(err))
		return err
	}

	user, err := t.userRepo.List(ctx, &logtrace.FindUserOptions{
		ID: opts.UserID,
	})
	if err != nil {
		logger.Error("could not fetch user from database", zap.Error(err))
		return err
	}

	link := t.cfg.Frontend.AppURL + "/email-verify?token=" + opts.Token

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FullName": strings.Split(user.FullName, " ")[0],
		"Link":     link,
	}); err != nil {
		logger.Error("could not embed content in template", zap.Error(err))
		return err
	}

	emailOpts := email.SendOptions{
		HTML:      buf.String(),
		Sender:    t.cfg.Email.Sender,
		Recipient: user.Email,
		Subject:   "Verify your account to get started with Logtrace",
		DKIM: struct {
			Sign       bool
			PrivateKey []byte
		}{
			Sign:       false,
			PrivateKey: []byte(""),
		},
	}

	_, err = t.emailClient.Send(ctx, emailOpts)
	if err != nil {
		logger.Error("could not send email", zap.Error(err))
		return err
	}

	return nil
}
