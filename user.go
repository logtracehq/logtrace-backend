package logtrace

import (
	"context"
	"database/sql/driver"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	ErrUserNotFound = LogtraceError("user not found")
	ErrUserExists   = LogtraceError("user with same email address already exists")
)

type RoleName string

// ENUM(active, inactive, pending)
type UserStatus string

type UserRole struct {
	ID            uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	UserID        *uuid.UUID `json:"user_id"     bun:"type:uuid,notnull"`
	RoleName      RoleName   `json:"role_name"   bun:"column:role_name,notnull"`
	AssignedAt    time.Time  `json:"assigned_at" bun:"default:current_timestamp,notnull"`
	bun.BaseModel `json:"-" bun:"table:user_roles"`
}

type Email string

func (e Email) String() string { return strings.ToLower(string(e)) }

func (e Email) Value() (driver.Value, error) { return driver.Value(e.String()), nil }

type UserMetadata struct {
	OrganizationID []uuid.UUID `json:"organization_id"`
	UserRole       RoleName    `json:"user_role"`
}

type User struct {
	ID              uuid.UUID     `json:"id"                bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Email           Email         `json:"email"             bun:"email,unique"`
	FullName        string        `json:"full_name" bun:"full_name"`
	EmailVerifiedAt *time.Time    `json:"email_verified_at" bun:"email_verified_at,nullzero"`
	Metadata        *UserMetadata `json:"metadata"   bun:"type:jsonb"`
	Phone           string        `json:"phone"`
	Status          string        `json:"status"            bun:"status,default:'active',notnull"`
	Roles           []UserRole    `json:"roles"       bun:"rel:has-many,join:id=user_id"`
	CreatedAt       time.Time     `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt       time.Time     `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt       *time.Time    `json:"-,omitempty" bun:",soft_delete,nullzero"`
	bun.BaseModel   `json:"-" bun:"table:users"`
}

type FindUserOptions struct {
	Email          Email     `json:"email,omitempty"`
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Paginator      Paginator
}

type UserRepository interface {
	Create(context.Context, *User) (*User, error)
	List(context.Context, *FindUserOptions) (*User, error)
	ListAll(context.Context, *FindUserOptions) ([]*User, int64, error)
	Update(context.Context, *User) (*User, error)
}
