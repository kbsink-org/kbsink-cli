package plugin

import (
	"net/http"
	"os"
	"testing"

	"github.com/kbsink-org/kbsink-cli/internal/plugin/douyin"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/wechat"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/xhs"
	"github.com/kbsink-org/kbsink/pkg/pluginreg"
)

func TestMain(m *testing.M) {
	registerTestPlugins()
	os.Exit(m.Run())
}

func registerTestPlugins() {
	pluginreg.Register(wechat.New())
	pluginreg.Register(xhs.New())
	pluginreg.Register(douyin.New())
}

func TestBuiltin_lookup(t *testing.T) {
	for _, name := range []string{"wechat", "WeChat", "xhs", "XHS", "douyin", "DOUYIN"} {
		pl, ok := pluginreg.Lookup(name)
		if !ok || pl == nil {
			t.Fatalf("Lookup(%q) missing", name)
		}
		p, d, err := pl.NewComponents(http.DefaultClient)
		if err != nil {
			t.Fatal(err)
		}
		if p == nil {
			t.Fatalf("parser nil for %q", name)
		}
		if d == nil {
			t.Fatalf("plugin %q should include a driver", name)
		}
	}
}

func TestWechatNewComponents(t *testing.T) {
	p, d, err := wechat.New().NewComponents(http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || d == nil {
		t.Fatal("expected non-nil parser and driver")
	}
}

func TestXHSNewComponents(t *testing.T) {
	p, d, err := xhs.New().NewComponents(http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || d == nil {
		t.Fatal("expected non-nil parser and driver")
	}
}

func TestDouyinNewComponents(t *testing.T) {
	p, d, err := douyin.New().NewComponents(http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || d == nil {
		t.Fatal("expected non-nil parser and driver")
	}
}
