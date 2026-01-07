package cache

import (
	"context"
	"time"

	"github.com/terra-consults/logbase"
)

const (
	ErrCacheMiss = logbase.LogbaseError("cache has been missed")
)

type Cache interface {
	Add(context.Context, string, []byte, time.Duration) error
	Exists(context.Context, string) (bool, error)
	Get(context.Context, string) ([]byte, error)
}
