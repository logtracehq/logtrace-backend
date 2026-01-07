package logbase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrAPIKeyNotFound = LogbaseError("api key not found")
	ErrAPIKeyMaxLimit = LogbaseError("you can only have a maximum of 10 active api keys")
)

type APIKey struct {
	ID             uuid.UUID  `json:"id,omitempty"         bun:"type:uuid,default:uuid_generate_v4(),pk"`
	CreatedBy      uuid.UUID  `json:"created_by,omitempty"`
	Value          string     `json:"-"`
	Name           string     `json:"name,omitempty"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" bun:",nullzero"`
	CreatedAt      time.Time  `json:"created_at"           bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt      time.Time  `json:"updated_at"           bun:",nullzero,notnull,default:current_timestamp"`
	DeletedAt      *time.Time `json:"-,omitempty"          bun:",soft_delete,nullzero"`

	bun.BaseModel `json:"-"`
}

func (a *APIKey) IsActive() bool { return a.ExpiresAt != nil }

func HashKey(secret, val string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(val))
	return hex.EncodeToString(h.Sum(nil))
}

// ENUM(immediate,day,week)
type RevocationType string

func (a *APIKey) IsRevoked() bool { return a.ExpiresAt != nil }

type RevokeAPIKeyOptions struct {
	APIKey         *APIKey
	RevocationType RevocationType
}

type FetchAPIKeyOptions struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
}

type ListAPIKeyOptions struct {
	Paginator Paginator `json:"paginator"`
}

type APIKeyRepository interface {
	Create(context.Context, *APIKey) error
	Revoke(context.Context, RevokeAPIKeyOptions) error
	List(context.Context, uuid.UUID) ([]APIKey, error)
	Fetch(context.Context, FetchAPIKeyOptions) (*APIKey, error)
	FetchByValue(context.Context, string) (*APIKey, error)
	FetchByName(context.Context, string, uuid.UUID) (*APIKey, error)
}
