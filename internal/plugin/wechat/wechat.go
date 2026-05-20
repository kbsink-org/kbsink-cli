package wechat

import (
	wechatlib "github.com/kbsink-org/kbsink-plugins/pkg/wechat"
	"github.com/kbsink-org/kbsink/pkg/core"
)

// Plugin is the WeChat article plugin (parser + fetch driver) for CLI wiring.
type Plugin struct{}

// New returns a Plugin registered as "wechat".
func New() core.Plugin {
	return Plugin{}
}

const WeChatUserAgent = "Mozilla/5.0 (compatible; wechatmd/1.0)"

func (Plugin) Name() string { return "wechat" }

func (Plugin) NewComponents(client core.HTTPClient) (core.Parser, core.Driver, error) {
	return wechatlib.NewParser(), wechatlib.NewDriver(client), nil
}
