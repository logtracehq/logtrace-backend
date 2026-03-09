package googleauth

import (
	"context"

	"gitlab.com/logtrace/logtrace"
	"golang.org/x/oauth2"
)

type User struct {
	FullName string         `json:"full_name"`
	Email    logtrace.Email `json:"email"`
}

type ValidateOptions struct {
	Code string
}

type GoogleAuthProvider interface {
	User(context.Context, *oauth2.Token) (User, error)
	Validate(context.Context, ValidateOptions) (*oauth2.Token, error)
}
