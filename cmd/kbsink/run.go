package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log/slog"

	"github.com/kbsink-org/kbsink-cli/internal/convertlib"
	clidriver "github.com/kbsink-org/kbsink-cli/internal/driver"
	"github.com/kbsink-org/kbsink-cli/internal/netclient"
	"github.com/kbsink-org/kbsink/pkg/core"
	klog "github.com/kbsink-org/kbsink/pkg/logger"
	"github.com/kbsink-org/kbsink/pkg/pluginreg"
	"github.com/kbsink-org/kbsink/pkg/storage"
)

// run parses argv: optional flags then a single URL (or share text containing a URL).
// Exit codes: 0 success, 1 error, 2 usage.
func run(args []string) int {
	if len(args) < 1 {
		return 2
	}
	prog := filepath.Base(args[0])
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	outputRootFlag := fs.String("o", "output", "output root directory (local filesystem)")
	timeout := fs.Duration("timeout", 60*time.Second, "timeout for the conversion")
	videoMode := fs.String("video-mode", "link", "video markdown mode: link|embed")
	pluginFlag := fs.String("plugin", "", "plugin id: wechat, xhs, douyin (optional if the URL host is recognized)")
	logLevel := fs.String("log-level", "info", "log level: debug|info|warn|error")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage:\n  %s [flags] <article-url-or-share-text>\n\nFlags:\n", prog)
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "\nIf --plugin is omitted, the tool picks wechat, xhs, or douyin from the URL host.\n")
		convertlib.EnsurePluginsRegistered()
		if names := pluginreg.Names(); len(names) > 0 {
			_, _ = fmt.Fprintf(fs.Output(), "Plugins in this build: %s\n", strings.Join(names, ", "))
		}
	}

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if err := setupSlog(*logLevel); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "invalid --log-level: %v\n", err)
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

	pluginName := strings.TrimSpace(*pluginFlag)
	if pluginName == "" {
		var ok bool
		pluginName, ok = convertlib.DetectPlugin(articleURL)
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "could not infer plugin from URL; use --plugin (wechat, xhs, douyin)\n")
			if names := pluginreg.Names(); len(names) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "registered: %s\n", strings.Join(names, ", "))
			}
			return 2
		}
		slog.Default().Debug("plugin detected", "plugin", pluginName)
	}
	pluginName = strings.ToLower(pluginName)

	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	outRoot := strings.TrimSpace(*outputRootFlag)
	if outRoot == "" {
		outRoot = "output"
	}

	log := klog.Slog{L: slog.Default()}
	httpClient := netclient.New(*timeout)
	coreHTTP := netclient.CoreHTTPClient(httpClient)

	drv, err := clidriver.ForPlugin(pluginName, coreHTTP, log)
	if err != nil {
		emitError(err)
		return 1
	}

	res, err := convertlib.ConvertArticle(ctx, convertlib.Params{
		URL:        articleURL,
		Plugin:     pluginName,
		VideoMode:  *videoMode,
		Timeout:    *timeout,
		OutputRoot: outRoot,
		HTTP:       httpClient,
		Storage:    storage.NewLocalStorage(outRoot, log),
		KbsinkLog:  log,
		LogLevel:   *logLevel,
		Driver:     drv,
	})
	if err != nil {
		emitError(err)
		return 1
	}

	emitSuccess(res)
	return 0
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
