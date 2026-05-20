package wechat

import (
	"github.com/kbsink-org/kbsink/pkg/core"
	wechatlib "github.com/kbsink-org/kbsink-plugins/pkg/wechat"
)

// Plugin is the WeChat article plugin (parser + fetch driver) for CLI wiring.
type Plugin struct{}

// New returns a Plugin registered as "wechat".
func New() core.Plugin {
	return Plugin{}
}

func (Plugin) Name() string { return "wechat" }

func (Plugin) NewComponents(client core.HTTPClient) (core.Parser, core.Driver, error) {
	return wechatlib.NewParser(), wechatlib.NewDriver(client), nil
}
