package logtrace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
)

type EmailVerification struct {
	Token     string     `json:"token"`
	ID        uuid.UUID  `json:"id"         bun:"type:uuid,pk,default:uuid_generate_v4(),pk"`
	UserID    uuid.UUID  `json:"user_id"`
	CreatedAt time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`

	bun.BaseModel `bun:"table:email_verifications" json:"-"`
}

func NewEmailVerification(u *User) (*EmailVerification, error) {
	val, err := util.Random(40)
	if err != nil {
		return nil, err
	}

	return &EmailVerification{
		Token:  val,
		UserID: u.ID,
	}, nil
}

type EmailVerificationRepository interface {
	Create(context.Context, *EmailVerification) error
	Delete(ctx context.Context, token string) error
	List(ctx context.Context, token string) (*EmailVerification, error)
}
