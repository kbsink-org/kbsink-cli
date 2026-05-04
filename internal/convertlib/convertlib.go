// Package convertlib runs kbsink conversion (shared by the CLI and the wasm JS binding).
package convertlib

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kbsink-org/kbsink-cli/internal/plugin/douyin"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/wechat"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/xhs"
	kbsink "github.com/kbsink-org/kbsink/pkg"
	"github.com/kbsink-org/kbsink/pkg/core"
	"github.com/kbsink-org/kbsink/pkg/pluginreg"
)

var registerOnce sync.Once

// EnsurePluginsRegistered registers wechat, xhs, and douyin once per process.
func EnsurePluginsRegistered() {
	registerOnce.Do(func() {
		pluginreg.Register(wechat.New())
		pluginreg.Register(xhs.New())
		pluginreg.Register(douyin.New())
	})
}

type nopStorage struct{}

func (nopStorage) Save(ctx context.Context, article *core.ArticleResult) error {
	_ = ctx
	_ = article
	return nil
}

// Params configures a single conversion (no local filesystem writes; storage is a no-op).
type Params struct {
	URL         string
	Plugin      string
	VideoMode   string
	Timeout     time.Duration
	OutputRoot  string
	HTTPClient  *http.Client
}

// ArticleJSON is the JSON-friendly conversion result (asset bytes as base64).
type ArticleJSON struct {
	Title         string      `json:"title"`
	SafeTitle     string      `json:"safeTitle"`
	AccountName   string      `json:"accountName,omitempty"`
	SourceURL     string      `json:"sourceUrl"`
	OutputDir     string      `json:"outputDir"`
	MarkdownPath  string      `json:"markdownPath"`
	Markdown      string      `json:"markdown"`
	RawHTML       string      `json:"rawHtml,omitempty"`
	Assets        []AssetJSON `json:"assets"`
	Plugin        string      `json:"plugin"`
}

// AssetJSON is one image or video asset after download.
type AssetJSON struct {
	Type         string `json:"type"`
	SourceURL    string `json:"sourceUrl,omitempty"`
	RelativePath string `json:"relativePath"`
	FileName     string `json:"fileName"`
	ContentType  string `json:"contentType,omitempty"`
	DataBase64   string `json:"dataBase64,omitempty"`
}

// Convert runs the pipeline and returns structured data without persisting to disk.
func Convert(ctx context.Context, p Params) (*ArticleJSON, error) {
	url := strings.TrimSpace(p.URL)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	EnsurePluginsRegistered()

	pluginName := strings.TrimSpace(p.Plugin)
	if pluginName == "" {
		var ok bool
		pluginName, ok = DetectPlugin(url)
		if !ok {
			return nil, fmt.Errorf("could not infer plugin from URL; set explicitly (wechat, xhs, douyin); registered: %s",
				strings.Join(pluginreg.Names(), ", "))
		}
	}
	pluginName = strings.ToLower(pluginName)

	mode, err := resolveVideoMode(p.VideoMode)
	if err != nil {
		return nil, err
	}

	pl, ok := pluginreg.Lookup(pluginName)
	if !ok {
		return nil, fmt.Errorf("unknown plugin %q; registered: %s",
			pluginName, strings.Join(pluginreg.Names(), ", "))
	}

	client := p.HTTPClient
	if client == nil {
		client = httpClientForTimeout(p.Timeout)
	}

	parser, driver, err := pl.NewComponents(client)
	if err != nil {
		return nil, fmt.Errorf("plugin %q: %w", pluginName, err)
	}
	if parser == nil {
		return nil, fmt.Errorf("plugin %q returned nil parser", pluginName)
	}

	opts := []kbsink.Option{
		kbsink.WithHTTPClient(client),
		kbsink.WithParser(parser),
		kbsink.WithStorage(nopStorage{}),
	}
	if driver != nil {
		opts = append(opts, kbsink.WithDriver(driver))
	}
	converter := kbsink.NewConverter(opts...)

	outputRoot := strings.TrimSpace(p.OutputRoot)
	if outputRoot == "" {
		outputRoot = "output"
	}

	res, err := converter.Convert(ctx, url, core.ConvertOptions{
		OutputRoot: outputRoot,
		VideoMode:  mode,
	})
	if err != nil {
		return nil, err
	}

	return articleToJSON(res, pluginName), nil
}

func articleToJSON(res *core.ArticleResult, plugin string) *ArticleJSON {
	out := &ArticleJSON{
		Title:        res.Title,
		SafeTitle:    res.SafeTitle,
		AccountName:  res.AccountName,
		SourceURL:    res.SourceURL,
		OutputDir:    res.OutputDir,
		MarkdownPath: res.MarkdownPath,
		Markdown:     res.Markdown,
		RawHTML:      res.RawHTMLContent,
		Plugin:       plugin,
	}
	for _, a := range res.Assets {
		aj := AssetJSON{
			Type:         string(a.Type),
			SourceURL:    a.SourceURL,
			RelativePath: a.RelativePath,
			FileName:     a.FileName,
			ContentType:  a.ContentType,
		}
		if len(a.Data) > 0 {
			aj.DataBase64 = base64.StdEncoding.EncodeToString(a.Data)
		}
		out.Assets = append(out.Assets, aj)
	}
	return out
}

func httpClientForTimeout(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		return http.DefaultClient
	}
	return &http.Client{Timeout: timeout}
}

func resolveVideoMode(raw string) (core.VideoMode, error) {
	mode := strings.TrimSpace(strings.ToLower(raw))
	switch mode {
	case "", string(core.VideoModeLink):
		return core.VideoModeLink, nil
	case string(core.VideoModeEmbed):
		return core.VideoModeEmbed, nil
	default:
		return "", fmt.Errorf("unsupported video mode %q, expected link|embed", raw)
	}
}
