package util

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ResponseError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

type ResponseSuccess struct {
	Message    string `json:"message"`
	Data       any    `json:"data"`
	StatusCode int    `json:"status_code"`
}

var timeout = 20 * time.Second

func WithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

func IsStringEmpty(s string) bool { return len(strings.TrimSpace(s)) == 0 }

func Ref[T any](value T) *T {
	return &value
}

func DeRef[T any](ptr *T) T {
	var zeroValue T

	if ptr == nil {
		return zeroValue
	}

	return *ptr
}

func Random(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func UUIDGenerate() string {
	uuid := uuid.New().String()
	return uuid
}

func GenerateRandom() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._"
	const length = 20

	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(err)
		}
		b[i] = chars[n.Int64()]
	}

	return string(b)
}
