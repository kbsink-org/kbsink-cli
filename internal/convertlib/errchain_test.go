package convertlib

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrChainString_nil(t *testing.T) {
	if s := ErrChainString(nil); s != "" {
		t.Fatalf("got %q", s)
	}
}

func TestErrChainString_chain(t *testing.T) {
	inner := errors.New("connection reset")
	mid := fmt.Errorf("kbsinkHTTPRoundTrip: %w", inner)
	outer := fmt.Errorf("fetch article: %w", mid)
	got := ErrChainString(outer)
	for _, sub := range []string{"fetch article", "kbsinkHTTPRoundTrip", "connection reset"} {
		if !strings.Contains(got, sub) {
			t.Fatalf("expected %q in %q", sub, got)
		}
	}
}

func TestErrChainString_dedupesRedundantUnwraps(t *testing.T) {
	inner := errors.New("context deadline exceeded")
	mid := fmt.Errorf(`Get "https://example.com/x": %w`, inner)
	outer := fmt.Errorf("fetch article: DRIVER_REQUEST_FAILED: execute request: %w", mid)
	got := ErrChainString(outer)
	if strings.Count(got, "context deadline exceeded") != 1 {
		t.Fatalf("expected single deadline mention, got %q", got)
	}
	if strings.Count(got, "DRIVER_REQUEST_FAILED") != 1 {
		t.Fatalf("expected single DRIVER_REQUEST_FAILED, got %q", got)
	}
}
