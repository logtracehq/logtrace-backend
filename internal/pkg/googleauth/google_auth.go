package googleauth

import (
	"context"

	"gitlab.com/logbase/logbase"
	"golang.org/x/oauth2"
)

type User struct {
	FullName string        `json:"full_name"`
	Email    logbase.Email `json:"email"`
}

type ValidateOptions struct {
	Code string
}

type GoogleAuthProvider interface {
	User(context.Context, *oauth2.Token) (User, error)
	Validate(context.Context, ValidateOptions) (*oauth2.Token, error)
}
