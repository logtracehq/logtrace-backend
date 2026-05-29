package jwttoken

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gitlab.com/logtrace/logtrace/config"
)

// ENUM(access,refresh)
type Purpose uint8

type JWTokenData struct {
	Token     string
	Purpose   Purpose
	Audience  string
	UserID    uuid.UUID
	ExpiresAt time.Time
}

type jwtokenManager struct {
	signingKey string
}

type JWTokenManager interface {
	GenerateJWToken(JWTokenData) (JWTokenData, error)
	ParseJWToken(string) (JWTokenData, error)
	GeneratePasswordResetToken(JWTokenData) (JWTokenData, error)
	ParsePasswordResetToken(string) (JWTokenData, error)
}

func New(cfg *config.Config) JWTokenManager {
	return &jwtokenManager{
		signingKey: cfg.Auth.JWT.Key,
	}
}

func (t *jwtokenManager) GenerateJWToken(data JWTokenData) (JWTokenData, error) {
	claims := jwt.MapClaims{
		"signer": "logtrace",
		"id":     data.UserID,
		"aud":    "logtrace",
		"exp":    time.Now().Add(time.Hour * 168), // 7 days
	}
	jwtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtoken.SignedString([]byte(t.signingKey))
	if err != nil {
		return JWTokenData{}, fmt.Errorf(
			"GenerateJWToken/SignedString: sign jwtoken failed: %w",
			err,
		)
	}

	data.Token = token
	return data, nil
}

func (t *jwtokenManager) ParseJWToken(JWToken string) (JWTokenData, error) {
	parsedJWToken, err := jwt.Parse(JWToken, func(JWToken *jwt.Token) (i any, e error) {
		if _, ok := JWToken.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf(
				"ParseJWToken/Parse: error: unexpected signing method: %v",
				JWToken.Header["alg"],
			)
		}
		return []byte(t.signingKey), nil
	})

	if err != nil && parsedJWToken == nil {
		return JWTokenData{}, fmt.Errorf("ParseJWToken/Parse: parse JWToken failed: %w", err)
	}

	claims, ok := parsedJWToken.Claims.(jwt.MapClaims)
	if !ok {
		return JWTokenData{}, fmt.Errorf(
			"ParseJWToken/parsedJWToken.Claims: error: JWToken wrong claims",
		)
	}

	id, ok := claims["id"].(string)
	if !ok {
		return JWTokenData{}, errors.New("user_id does not exist")
	}

	audience, ok := claims["aud"].(string)
	if audience != "logtrace" {
		return JWTokenData{}, errors.New("incorrect audience")
	}

	expiresAt, ok := claims["exp"].(string)
	if !ok {
		return JWTokenData{}, errors.New(
			"ParseJWToken/parseJWTokenClaim/exp: expiration date not found",
		)
	}

	expiresTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return JWTokenData{}, err
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		return JWTokenData{}, err
	}

	return JWTokenData{
		UserID:    userID,
		ExpiresAt: expiresTime,
	}, nil
}

func (t *jwtokenManager) GeneratePasswordResetToken(data JWTokenData) (JWTokenData, error) {
	claims := jwt.MapClaims{
		"signer":  "logtrace",
		"id":      data.UserID,
		"aud":     "logtrace",
		"purpose": "password_reset",
		"exp":     time.Now().Add(time.Minute * 60).Unix(),
	}

	jwtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtoken.SignedString([]byte(t.signingKey))
	if err != nil {
		return JWTokenData{}, fmt.Errorf("GeneratePasswordResetToken/SignedString: %w", err)
	}

	data.Token = token
	data.ExpiresAt = time.Now().Add(time.Minute * 60)
	return data, nil
}

func (t *jwtokenManager) ParsePasswordResetToken(rawToken string) (JWTokenData, error) {
	parsed, err := jwt.Parse(rawToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("ParsePasswordResetToken: unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(t.signingKey), nil
	})
	if err != nil {
		return JWTokenData{}, fmt.Errorf("ParsePasswordResetToken/Parse: %w", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return JWTokenData{}, errors.New("ParsePasswordResetToken: invalid claims")
	}

	purpose, ok := claims["purpose"].(string)
	if !ok || purpose != "password_reset" {
		return JWTokenData{}, errors.New("ParsePasswordResetToken: invalid token purpose")
	}

	id, ok := claims["id"].(string)
	if !ok {
		return JWTokenData{}, errors.New("ParsePasswordResetToken: user_id not found")
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		return JWTokenData{}, fmt.Errorf("ParsePasswordResetToken: invalid user_id: %w", err)
	}

	return JWTokenData{
		UserID: userID,
	}, nil
}
