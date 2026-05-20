package convertlib

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/kbsink-org/kbsink/pkg/core"
)

const maxMarkdownLogRunes = 800

type parsedAssetPreview struct {
	Type      string `json:"type,omitempty"`
	SourceURL string `json:"sourceUrl,omitempty"`
}

type parsedArticlePreview struct {
	Plugin          string               `json:"plugin"`
	Title           string               `json:"title"`
	AccountName     string               `json:"accountName,omitempty"`
	SourceURL       string               `json:"sourceUrl"`
	MarkdownLen     int                  `json:"markdownLen"`
	RawHTMLLen      int                  `json:"rawHtmlLen"`
	AssetCount      int                  `json:"assetCount"`
	MarkdownPreview string               `json:"markdownPreview,omitempty"`
	Assets          []parsedAssetPreview `json:"assets"`
}

func collectAssetPreviews(parsed *core.ArticleResult) []parsedAssetPreview {
	seen := make(map[string]struct{})
	var out []parsedAssetPreview
	add := func(typ core.AssetType, url string) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		key := string(typ) + "\x00" + url
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, parsedAssetPreview{
			Type:      string(typ),
			SourceURL: url,
		})
	}
	for _, a := range parsed.Assets {
		t := a.Type
		if t == "" {
			t = core.AssetTypeImage
		}
		add(t, a.SourceURL)
	}
	for _, img := range parsed.Images {
		add(core.AssetTypeImage, img.SourceURL)
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func logParsedArticle(log *slog.Logger, parsed *core.ArticleResult, plugin string) {
	if log == nil || parsed == nil {
		return
	}
	assets := collectAssetPreviews(parsed)
	preview := parsedArticlePreview{
		Plugin:          plugin,
		Title:           parsed.Title,
		AccountName:     parsed.AccountName,
		SourceURL:       parsed.SourceURL,
		MarkdownLen:     len(parsed.Markdown),
		RawHTMLLen:      len(parsed.RawHTMLContent),
		AssetCount:      len(assets),
		MarkdownPreview: truncateRunes(parsed.Markdown, maxMarkdownLogRunes),
		Assets:          assets,
	}
	log.Info("parse result",
		"plugin", plugin,
		"title", parsed.Title,
		"accountName", parsed.AccountName,
		"markdownLen", preview.MarkdownLen,
		"rawHtmlLen", preview.RawHTMLLen,
		"assetCount", preview.AssetCount,
	)
	b, err := json.Marshal(preview)
	if err != nil {
		log.Warn("parse result json", "err", err)
		return
	}
	log.Info("parse result json", "data", string(b))
}
