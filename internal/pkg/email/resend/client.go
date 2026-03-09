package resend

import (
	"context"
	"errors"
	"fmt"

	resendclient "github.com/resend/resend-go/v2"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/email"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
)

type client struct {
	inner       *resendclient.Client
	senderName  string
	senderEmail string
}

func New(cfg *config.Config) (email.Client, error) {
	if util.IsStringEmpty(cfg.Email.Resend.APIKey) {
		return nil, errors.New("please provide your resend api key")
	}

	if util.IsStringEmpty(cfg.Email.Resend.WebhookSecret) {
		return nil, errors.New("please provide your resend webhook secret")
	}

	c := resendclient.NewClient(cfg.Email.Resend.APIKey)

	return &client{
		inner:       c,
		senderName:  cfg.Email.SenderName,
		senderEmail: cfg.Email.Sender.String(),
	}, nil
}

func (s *client) Close() error { return nil }

func (s *client) Send(ctx context.Context,
	opts email.SendOptions,
) (string, error) {
	params := &resendclient.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.senderName, s.senderEmail),
		To:      []string{opts.Recipient.String()},
		Subject: opts.Subject,
		Html:    opts.HTML,
	}

	res, err := s.inner.Emails.Send(params)
	if err != nil {
		return "", err
	}

	return res.Id, nil
}
