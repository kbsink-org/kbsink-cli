// Package netclient builds *httpclient.HTTP values for kbsink-cli.
package netclient

import (
	"time"

	"github.com/SolaTyolo/httpclient"
)

// New returns a retryable client (same defaults as httpclient.New).
func New(timeout time.Duration) *httpclient.HTTP {
	var opts []httpclient.ClientOption
	if timeout > 0 {
		opts = append(opts, httpclient.WithTimeout(timeout))
	}
	return httpclient.New(opts...)
}
