package xhs

import (
	"net/http"

	"github.com/kbsink-org/kbsink/pkg/core"
	"github.com/kbsink-org/kbsink/pkg/driver"
	"github.com/kbsink-org/kbsink/pkg/parser"
)

// Plugin is the XHS (小红书) plugin (parser + fetch driver) for CLI wiring.
type Plugin struct{}

// New returns a Plugin registered as "xhs".
func New() core.Plugin {
	return Plugin{}
}

func (Plugin) Name() string { return "xhs" }

func (Plugin) NewComponents(client *http.Client) (core.Parser, core.Driver, error) {
	return parser.NewXHSParser(), driver.NewXHSDriver(client), nil
}
