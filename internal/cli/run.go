// Package cli implements the kbsink command-line entrypoint (wechat, xhs, douyin).
package cli

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

func ensurePluginsRegistered() {
	registerOnce.Do(func() {
		pluginreg.Register(wechat.New())
		pluginreg.Register(xhs.New())
		pluginreg.Register(douyin.New())
	})
}

// Run parses argv: optional flags then a single URL (or share text containing a URL).
// Exit codes: 0 success, 1 error, 2 usage.
func Run(args []string) int {
	if len(args) < 1 {
		return 2
	}
	prog := filepath.Base(args[0])
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	outputRoot := fs.String("o", "output", "output root directory (local filesystem)")
	timeout := fs.Duration("timeout", 60*time.Second, "timeout for the conversion")
	videoMode := fs.String("video-mode", "link", "video markdown mode: link|embed")
	pluginFlag := fs.String("plugin", "", "plugin id: wechat, xhs, douyin (optional if the URL host is recognized)")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage:\n  %s [flags] <article-url-or-share-text>\n\nFlags:\n", prog)
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "\nIf --plugin is omitted, the tool picks wechat, xhs, or douyin from the URL host.\n")
		ensurePluginsRegistered()
		if names := pluginreg.Names(); len(names) > 0 {
			_, _ = fmt.Fprintf(fs.Output(), "Plugins in this build: %s\n", strings.Join(names, ", "))
		}
	}

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	articleURL := strings.TrimSpace(fs.Arg(0))
	if articleURL == "" {
		fs.Usage()
		return 2
	}

	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	mode, err := resolveVideoMode(*videoMode)
	if err != nil {
		emitError(err)
		return 1
	}

	ensurePluginsRegistered()

	pluginName := strings.TrimSpace(*pluginFlag)
	if pluginName == "" {
		var ok bool
		pluginName, ok = DetectPlugin(articleURL)
		if !ok {
			emitError(fmt.Errorf("could not infer --plugin from URL; set explicitly (wechat, xhs, douyin); registered: %s",
				strings.Join(pluginreg.Names(), ", ")))
			return 1
		}
	}
	pluginName = strings.ToLower(pluginName)

	httpClient := httpClientForCLI(*timeout)

	pl, ok := pluginreg.Lookup(pluginName)
	if !ok {
		emitError(fmt.Errorf("unknown plugin %q; registered: %s",
			pluginName, strings.Join(pluginreg.Names(), ", ")))
		return 1
	}
	parser, driver, err := pl.NewComponents(httpClient)
	if err != nil {
		emitError(fmt.Errorf("plugin %q: %w", pluginName, err))
		return 1
	}
	if parser == nil {
		emitError(fmt.Errorf("plugin %q returned nil parser", pluginName))
		return 1
	}

	opts := []kbsink.Option{
		kbsink.WithHTTPClient(httpClient),
		kbsink.WithParser(parser),
	}
	if driver != nil {
		opts = append(opts, kbsink.WithDriver(driver))
	}
	converter := kbsink.NewConverter(opts...)

	res, err := converter.Convert(ctx, articleURL, core.ConvertOptions{
		OutputRoot: *outputRoot,
		VideoMode:  mode,
	})
	if err != nil {
		emitError(err)
		return 1
	}

	emitSuccess(res)
	return 0
}

func httpClientForCLI(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		return http.DefaultClient
	}
	return &http.Client{Timeout: timeout}
}

func emitError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
}

func emitSuccess(res *core.ArticleResult) {
	_, _ = fmt.Fprintf(os.Stdout, "title: %s\n", res.Title)
	_, _ = fmt.Fprintf(os.Stdout, "markdown: %s\n", res.MarkdownPath)
	_, _ = fmt.Fprintf(os.Stdout, "images: %d\n", len(res.Images))
	videoCount := 0
	for _, asset := range res.Assets {
		if asset.Type == core.AssetTypeVideo {
			videoCount++
		}
	}
	_, _ = fmt.Fprintf(os.Stdout, "videos: %d\n", videoCount)
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
