package convertlib

import (
	"testing"

	"github.com/kbsink-org/kbsink/pkg/core"
)

func TestCollectAssetPreviews_dedupes(t *testing.T) {
	parsed := &core.ArticleResult{
		Assets: []core.Asset{{Type: core.AssetTypeImage, SourceURL: "https://a/x.jpg"}},
		Images: []core.ImageAsset{{SourceURL: "https://a/x.jpg"}},
	}
	got := collectAssetPreviews(parsed)
	if len(got) != 1 {
		t.Fatalf("got %d previews", len(got))
	}
}

func TestTruncateRunes(t *testing.T) {
	got := truncateRunes("你好世界", 2)
	if got != "你好…" {
		t.Fatalf("got %q", got)
	}
}
