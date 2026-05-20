package driver

import (
	"fmt"
	"strings"

	"github.com/kbsink-org/kbsink-cli/internal/plugin/wechat"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/xhs"
	"github.com/kbsink-org/kbsink/pkg/core"
	"github.com/kbsink-org/kbsink/pkg/logger"
)

// ForPlugin returns the fetch driver for a built-in plugin id.
// For douyin, nil driver means convertlib uses the plugin default from NewComponents.
func ForPlugin(plugin string, client core.HTTPClient, log logger.Logger) (core.Driver, error) {
	switch strings.ToLower(strings.TrimSpace(plugin)) {
	case "wechat":
		return NewDriver(client, wechat.WeChatUserAgent, log), nil
	case "xhs":
		return NewDriver(client, xhs.XHSUserAgent, log), nil
	case "douyin":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown plugin %q for driver", plugin)
	}
}
