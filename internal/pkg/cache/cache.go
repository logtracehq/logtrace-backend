package cache

import (
	"context"
	"time"

	"gitlab.com/logtrace/logtrace"
)

const (
	ErrCacheMiss = logtrace.LogtraceError("cache has been missed")
)

type Cache interface {
	Add(context.Context, string, []byte, time.Duration) error
	Exists(context.Context, string) (bool, error)
	Get(context.Context, string) ([]byte, error)
}
