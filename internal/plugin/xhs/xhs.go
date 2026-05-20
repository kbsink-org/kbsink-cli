package xhs

import (
	"github.com/kbsink-org/kbsink/pkg/core"
	xhslib "github.com/kbsink-org/kbsink-plugins/pkg/xhs"
)

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
