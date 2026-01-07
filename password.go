package logbase

import (
	"context"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var ErrPasswordNotFound = LogbaseError("password not found")

type Password struct {
	ID           uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	UserPassword string     `bun:"user_password,notnull"`
	UserID       uuid.UUID  `bun:"user_id,unique,notnull,type:uuid"`
	CreatedAt    time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt    *time.Time `bun:"deleted_at,soft_delete,nullzero"`

	bun.BaseModel ` bun:"table:user_passwords" json:"-"`
}

type PasswordReset struct {
	ID        uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Token     string     `json:"token"`
	Status    string     `json:"status"`
	UserID    uuid.UUID  `json:"user_id"     bun:"type:uuid,notnull"`
	CreatedAt time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt *time.Time `json:"-,omitempty" bun:"soft_delete,nullzero"`

	bun.BaseModel ` bun:"table:user_password_resets" json:"-"`
}

type ResetPassword struct {
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hashedPassword, nil
}

func ComparePasswordAndHash(password, hash string) (bool, error) {
	passwordMatch, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return passwordMatch, nil
}

type PasswordRepository interface {
	Create(context.Context, *Password) error
	Get(context.Context, uuid.UUID) (*Password, error)
}
