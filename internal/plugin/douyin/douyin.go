package douyin

import (
	douyinlib "github.com/kbsink-org/kbsink-plugins/pkg/douyin"
	"github.com/kbsink-org/kbsink/pkg/core"
)

// Plugin is the Douyin plugin (parser + driver) for CLI wiring with pluginreg.
type Plugin struct{}

// New returns a Plugin registered as "douyin".
func New() core.Plugin {
	return Plugin{}
}

func (Plugin) Name() string { return "douyin" }

func (Plugin) NewComponents(c core.HTTPClient) (core.Parser, core.Driver, error) {
	return douyinlib.NewParser(), douyinlib.NewDriver(c), nil
}
