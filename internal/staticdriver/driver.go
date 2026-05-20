// Package staticdriver implements core.Driver that returns a fixed FetchResult (no HTTP).
package staticdriver

import (
	"context"

	"github.com/kbsink-org/kbsink/pkg/core"
)

// Driver skips HTTP and always returns Result.
type Driver struct {
	Result *core.FetchResult
}

func (d Driver) Fetch(ctx context.Context, url string) (*core.FetchResult, error) {
	_ = ctx
	_ = url
	return d.Result, nil
}
