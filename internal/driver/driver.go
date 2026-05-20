package driver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kbsink-org/kbsink/pkg/core"
	"github.com/kbsink-org/kbsink/pkg/logger"
)

const defaultUserAgent = "Mozilla/5.0 (compatible; kbsink-cli/1.0)"

// Driver fetches page HTML by URL over HTTP.
type Driver struct {
	client    core.HTTPClient
	userAgent string
	log       logger.Logger
}

// New returns a fetch driver. Empty userAgent uses defaultUserAgent. log may be nil.
func NewDriver(client core.HTTPClient, userAgent string, log logger.Logger) core.Driver {
	if client == nil {
		client = http.DefaultClient
	}
	ua := strings.TrimSpace(userAgent)
	if ua == "" {
		ua = defaultUserAgent
	}
	return &Driver{
		client:    client,
		userAgent: ua,
		log:       log,
	}
}

func (d *Driver) Fetch(ctx context.Context, rawURL string) (*core.FetchResult, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, core.NewCodedError(core.ErrCodeInvalidArgument, "url is required", nil)
	}

	if d.log != nil {
		d.log.Debug("driver: fetch start", "url", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		if d.log != nil {
			d.log.Error("driver: build request failed", "url", rawURL, "err", err)
		}
		return nil, core.NewCodedError(core.ErrCodeDriverBuildRequest, "build request", err)
	}
	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		if d.log != nil {
			d.log.Error("driver: request failed", "url", rawURL, "err", err)
		}
		return nil, core.NewCodedError(core.ErrCodeDriverRequestFailed, "execute request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if d.log != nil {
			d.log.Warn("driver: unexpected status", "url", rawURL, "status", resp.Status)
		}
		return nil, core.NewCodedError(
			core.ErrCodeDriverUnexpectedHTTP,
			fmt.Sprintf("unexpected status: %s", resp.Status),
			nil,
		)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if d.log != nil {
			d.log.Error("driver: read body failed", "url", rawURL, "err", err)
		}
		return nil, core.NewCodedError(core.ErrCodeDriverReadBodyFailed, "read response body", err)
	}
	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	if d.log != nil {
		d.log.Info("driver: fetch done", "url", finalURL, "htmlLen", len(body))
	}
	return &core.FetchResult{
		URL:  finalURL,
		HTML: string(body),
	}, nil
}
