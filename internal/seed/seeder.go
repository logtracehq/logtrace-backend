package seed

import (
	"context"
	"fmt"

	"gitlab.com/logtrace/logtrace"
)

// Run inserts 10 events, 10 sessions, and 10 audit logs for the given organization.
func Run(
	ctx context.Context,
	orgID string,
	eventRepo logtrace.EventRepository,
	sessionRepo logtrace.SessionRepository,
	auditLogRepo logtrace.AuditLogRepository,
) error {
	orgUUID, err := parseOrgID(orgID)
	if err != nil {
		return fmt.Errorf("invalid organization id: %w", err)
	}

	events := EventSeedData()
	for i := range events {
		events[i].OrganizationID = orgUUID
		if err := eventRepo.Create(ctx, &events[i]); err != nil {
			return fmt.Errorf("failed to seed event %q: %w", events[i].ActionName, err)
		}
	}
	fmt.Printf("seeded %d events\n", len(events))

	sessions := SessionSeedData()
	for i := range sessions {
		sessions[i].OrganizationID = orgUUID
		if err := sessionRepo.Create(ctx, &sessions[i]); err != nil {
			return fmt.Errorf("failed to seed session for user %q: %w", sessions[i].UserName, err)
		}
	}
	fmt.Printf("seeded %d sessions\n", len(sessions))

	auditLogs := AuditLogSeedData()
	for i := range auditLogs {
		auditLogs[i].OrganizationID = orgUUID
		if err := auditLogRepo.Create(ctx, &auditLogs[i]); err != nil {
			return fmt.Errorf("failed to seed audit log %q: %w", auditLogs[i].Action, err)
		}
	}
	fmt.Printf("seeded %d audit logs\n", len(auditLogs))

	return nil
}
