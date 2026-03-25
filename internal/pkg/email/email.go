package email

import (
	"context"
	"embed"
	"errors"
	"io"

	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
)

//go:embed all:templates
var templatesFS embed.FS

var (
	DashboardSharingTemplate  = mustReadTemplate("templates/sharing/dashboard_share.html")
	BillingTrialTemplate      = mustReadTemplate("templates/billing/trial.html")
	BillingEndedTemplate      = mustReadTemplate("templates/billing/expired.html")
	EmailVerificationTemplate = mustReadTemplate("templates/auth/email_verify.html")
	InviteUserTemplate        = mustReadTemplate("templates/invitation/invite_user.html")
)

func mustReadTemplate(path string) string {
	b, err := templatesFS.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return string(b)
}

type SendOptionsBatch []SendOptions

func (s SendOptionsBatch) Validate() error {
	if len(s) > 25 {
		return errors.New("maximum of 25 allowed per batch")
	}

	for _, v := range s {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type SendOptions struct {
	HTML      string
	Sender    logtrace.Email
	Recipient logtrace.Email
	Subject   string
	DKIM      struct {
		Sign       bool
		PrivateKey []byte
	}
}

func (s SendOptions) Validate() error {
	if util.IsStringEmpty(s.HTML) {
		return errors.New("html copy of email must be provided")
	}

	if util.IsStringEmpty(s.Subject) {
		return errors.New("please provide subject")
	}

	if util.IsStringEmpty(s.Recipient.String()) {
		return errors.New("please provide recipient")
	}

	if util.IsStringEmpty(s.Sender.String()) {
		return errors.New("please provide sender")
	}

	return nil
}

type Client interface {
	io.Closer
	Send(context.Context, SendOptions) (string, error)
}
