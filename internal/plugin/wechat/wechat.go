package wechat

import (
	"net/http"

	"github.com/kbsink-org/kbsink/pkg/core"
	"github.com/kbsink-org/kbsink/pkg/driver"
	"github.com/kbsink-org/kbsink/pkg/parser"
)

// Plugin is the WeChat article plugin (parser + fetch driver) for CLI wiring.
type Plugin struct{}

// New returns a Plugin registered as "wechat".
func New() core.Plugin {
	return Plugin{}
}

func (Plugin) Name() string { return "wechat" }

func (Plugin) NewComponents(client *http.Client) (core.Parser, core.Driver, error) {
	return parser.NewWechatParser(), driver.NewWechatDriver(client), nil
}
