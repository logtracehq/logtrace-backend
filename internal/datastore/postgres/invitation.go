package postgres

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
)

type invitationRepo struct {
	inner *bun.DB
}

func NewInvitationRepository(db *bun.DB) logtrace.InvitationRepository {
	return &invitationRepo{
		inner: db,
	}
}

func (i *invitationRepo) Create(ctx context.Context, invitation *logtrace.Invitation) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := i.inner.NewInsert().Model(invitation).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (i *invitationRepo) List(ctx context.Context, opts logtrace.ListInvitationOptions) ([]*logtrace.Invitation, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var invitations []*logtrace.Invitation
	err := i.inner.NewSelect().Model(&invitations).Where("organization_id = ?", opts.OrganizationID.String()).
		Where("status = ?", "PENDING").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return invitations, nil
}
