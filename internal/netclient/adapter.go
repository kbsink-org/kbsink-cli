package netclient

import (
	"net/http"

	"github.com/SolaTyolo/httpclient"
	"github.com/SolaTyolo/httpclient/retry"
	"github.com/SolaTyolo/httpclient/transport"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/kbsink-org/kbsink/pkg/core"
)

// CoreHTTPClient adapts *httpclient.HTTP to [core.HTTPClient] (*http.Request).
func CoreHTTPClient(h *httpclient.HTTP) core.HTTPClient {
	if h == nil {
		return http.DefaultClient
	}
	return requesterAdapter{r: h.Requester()}
}

type requesterAdapter struct {
	r transport.Requester
}

func (a requesterAdapter) Do(req *http.Request) (*http.Response, error) {
	rreq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}
	return a.r.Do(rreq)
}

// FromHostHTTP wraps a host-bridged *http.Client for WASM (no automatic retries).
// The host must implement globalThis.kbsinkHTTPRoundTrip (see cmd/kbsink-wasm/bridge.go).
func FromHostHTTP(c *http.Client) *httpclient.HTTP {
	if c == nil {
		c = http.DefaultClient
	}
	rc := retryablehttp.NewClient()
	rc.HTTPClient = c
	rc.RetryMax = 0
	return httpclient.New(
		httpclient.WithRetryableClient(rc),
		httpclient.WithCheckRetry(retry.NoCheckRetryFn),
	)
}
