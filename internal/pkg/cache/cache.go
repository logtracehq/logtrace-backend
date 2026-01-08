package cache

import (
	"context"
	"time"

	"gitlab.com/logbase/logbase"
)

const (
	ErrCacheMiss = logbase.LogbaseError("cache has been missed")
)

type Cache interface {
	Add(context.Context, string, []byte, time.Duration) error
	Exists(context.Context, string) (bool, error)
	Get(context.Context, string) ([]byte, error)
}
