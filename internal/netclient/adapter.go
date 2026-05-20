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

// FromStdHTTP wraps a standard *http.Client in *httpclient.HTTP (retryable transport).
func FromStdHTTP(c *http.Client) *httpclient.HTTP {
	return wrapStdHTTP(c, false)
}

// FromHostHTTP wraps a host-bridged *http.Client for WASM (no automatic retries).
func FromHostHTTP(c *http.Client) *httpclient.HTTP {
	return wrapStdHTTP(c, true)
}

func wrapStdHTTP(c *http.Client, host bool) *httpclient.HTTP {
	if c == nil {
		c = http.DefaultClient
	}
	rc := retryablehttp.NewClient()
	rc.HTTPClient = c
	opts := []httpclient.ClientOption{httpclient.WithRetryableClient(rc)}
	if host {
		rc.RetryMax = 0
		opts = append(opts, httpclient.WithCheckRetry(retry.NoCheckRetryFn))
	}
	return httpclient.New(opts...)
}
