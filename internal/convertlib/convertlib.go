// Package convertlib runs kbsink conversion (shared by the CLI and the wasm JS binding).
package convertlib

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/SolaTyolo/httpclient"
	"github.com/kbsink-org/kbsink-cli/internal/netclient"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/douyin"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/wechat"
	"github.com/kbsink-org/kbsink-cli/internal/plugin/xhs"
	kbsink "github.com/kbsink-org/kbsink/pkg"
	"github.com/kbsink-org/kbsink/pkg/core"
	klog "github.com/kbsink-org/kbsink/pkg/logger"
	"github.com/kbsink-org/kbsink/pkg/pluginreg"
)

var registerOnce sync.Once

// EnsurePluginsRegistered registers built-in platform plugins once per process.
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

// Params configures a single conversion.
//
// Storage selects persistence: nil means nop storage (no writes; used by WASM JSON export).
// CLI should pass storage.NewLocalStorage(outputRoot) from github.com/kbsink-org/kbsink/pkg/storage.
type Params struct {
	URL        string
	Plugin     string // required: wechat, xhs, douyin (callers resolve via DetectPlugin when needed)
	VideoMode  string
	Timeout    time.Duration
	OutputRoot string
	// HTTP overrides the default retryable client. Nil uses netclient.New(Timeout).
	HTTP *httpclient.HTTP
	// Driver overrides the plugin HTML driver for article fetch. Nil uses the plugin default.
	Driver core.Driver
	// Storage receives Save after conversion. Nil uses nop storage.
	Storage core.Storage
	// KbsinkLog enables logging inside kbsink.Converter (nil disables).
	KbsinkLog klog.Logger
	// LogLevel filters kbsink logs (debug|info|warn|error). Empty means no extra filter.
	LogLevel string
}

// ArticleJSON is the JSON-friendly conversion result (asset bytes as base64).
type ArticleJSON struct {
	Title        string      `json:"title"`
	SafeTitle    string      `json:"safeTitle"`
	AccountName  string      `json:"accountName,omitempty"`
	SourceURL    string      `json:"sourceUrl"`
	OutputDir    string      `json:"outputDir"`
	MarkdownPath string      `json:"markdownPath"`
	Markdown     string      `json:"markdown"`
	RawHTML      string      `json:"rawHtml,omitempty"`
	Assets       []AssetJSON `json:"assets"`
	Plugin       string      `json:"plugin"`
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

// ConvertArticle runs the pipeline and returns the article result (same wiring as Convert).
func ConvertArticle(ctx context.Context, p Params) (*core.ArticleResult, error) {
	res, _, err := convertPipeline(ctx, p)
	return res, err
}

// Convert runs the pipeline and returns JSON-friendly structured data (nop storage unless Params.Storage is set).
func Convert(ctx context.Context, p Params) (*ArticleJSON, error) {
	res, pluginName, err := convertPipeline(ctx, p)
	if err != nil {
		return nil, err
	}
	return articleToJSON(res, pluginName), nil
}

func convertPipeline(ctx context.Context, p Params) (*core.ArticleResult, string, error) {
	log := slog.Default()
	url := strings.TrimSpace(p.URL)
	if url == "" {
		return nil, "", fmt.Errorf("url is required")
	}

	EnsurePluginsRegistered()
	log.Info("convert start", "url", url, "plugin", p.Plugin)

	pluginName := strings.ToLower(strings.TrimSpace(p.Plugin))
	if pluginName == "" {
		return nil, "", fmt.Errorf("plugin is required")
	}

	mode, err := resolveVideoMode(p.VideoMode)
	if err != nil {
		return nil, "", err
	}

	pl, ok := pluginreg.Lookup(pluginName)
	if !ok {
		return nil, "", fmt.Errorf("unknown plugin %q; registered: %s",
			pluginName, strings.Join(pluginreg.Names(), ", "))
	}

	httpClient := p.HTTP
	if httpClient == nil {
		httpClient = netclient.New(p.Timeout)
	}
	coreHTTP := netclient.CoreHTTPClient(httpClient)

	parser, driver, err := pl.NewComponents(coreHTTP)
	if err != nil {
		return nil, "", fmt.Errorf("plugin %q: %w", pluginName, err)
	}
	if parser == nil {
		return nil, "", fmt.Errorf("plugin %q returned nil parser", pluginName)
	}
	if p.Driver != nil {
		driver = p.Driver
	}

	store := core.Storage(nopStorage{})
	if p.Storage != nil {
		store = p.Storage
	}

	outputRoot := strings.TrimSpace(p.OutputRoot)
	if outputRoot == "" {
		outputRoot = "output"
	}

	downloadCtx := ctx
	if p.Timeout > 0 {
		var cancelDownload context.CancelFunc
		downloadCtx, cancelDownload = context.WithTimeout(ctx, p.Timeout)
		defer cancelDownload()
	}

	dlOpts := []kbsink.Option{
		kbsink.WithHTTPClient(coreHTTP),
		kbsink.WithDriver(driver),
		kbsink.WithParser(parser),
		kbsink.WithStorage(store),
	}
	if p.KbsinkLog != nil {
		dlOpts = append(dlOpts, kbsink.WithLogger(p.KbsinkLog))
	}
	if lvl := strings.TrimSpace(p.LogLevel); lvl != "" {
		min, err := klog.ParseLevel(lvl)
		if err != nil {
			return nil, "", err
		}
		dlOpts = append(dlOpts, kbsink.WithMinLevel(min))
	}
	dlConverter := kbsink.NewConverter(dlOpts...)

	convertOpts := core.ConvertOptions{
		OutputRoot: outputRoot,
		VideoMode:  mode,
	}

	log.Info("convert running", "plugin", pluginName, "videoMode", mode, "outputRoot", outputRoot)
	res, err := dlConverter.Convert(downloadCtx, url, convertOpts)
	if err != nil {
		log.Error("convert failed", "plugin", pluginName, "err", err)
		return nil, "", err
	}
	logParsedArticle(log, res, pluginName)

	log.Info("convert done", "plugin", pluginName, "title", res.Title, "assets", len(res.Assets))
	return res, pluginName, nil
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
