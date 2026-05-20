package xhs

import (
	xhslib "github.com/kbsink-org/kbsink-plugins/pkg/xhs"
	"github.com/kbsink-org/kbsink/pkg/core"
)

const XHSUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"

// Plugin is the XHS (小红书) plugin (parser + fetch driver) for CLI wiring.
type Plugin struct{}

// New returns a Plugin registered as "xhs".
func New() core.Plugin {
	return Plugin{}
}

func (Plugin) Name() string { return "xhs" }

func (Plugin) NewComponents(client core.HTTPClient) (core.Parser, core.Driver, error) {
	return xhslib.NewParser(), xhslib.NewDriver(client), nil
}
