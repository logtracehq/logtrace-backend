package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/google/uuid"
	"gitlab.com/logtrace/logtrace"
)

// ENUM(billing_trial_ending,billing_create_customer,
// invite_team_member,subscription_expired, verify_email,
// save_event, save_audit_log, save_session)
type QueueTopic string

type Message struct {
	ID       string
	Metadata map[string]string
	Data     []byte
}

type QueueHandler interface {
	io.Closer
	Add(context.Context, QueueTopic, any) error
	Start(context.Context)
}

func ToPayload(m any) []byte {
	b := new(bytes.Buffer)

	_ = json.NewEncoder(b).Encode(m)

	return b.Bytes()
}

type BillingCreateCustomerOptions struct {
	Organization *logtrace.Organization
	Email        logtrace.Email
}

type SendEmailOptions struct {
	Organization *logtrace.Organization
	Token        string
	Recipient    logtrace.Email
}

type SendBillingTrialEmailOptions struct {
	Organization *logtrace.Organization
	Expiration   string
	Recipient    logtrace.Email
}

type SubscriptionExpiredOptions struct {
	Organization *logtrace.Organization
	Recipient    logtrace.Email
}

type EmailVerificationOptions struct {
	UserID uuid.UUID
	Token  string
}

type InviteUserOptions struct {
	Email        logtrace.Email
	Organization uuid.UUID
	Token        string
}

type SaveEventOptions struct {
	Event *logtrace.Event
}

type SaveAuditLogOptions struct {
	AuditLog *logtrace.AuditLog
}

type SaveSessionOptions struct {
	Session *logtrace.Session
}
