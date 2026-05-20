//go:build js && wasm

package main

import (
	"strings"

	"github.com/kbsink-org/kbsink-cli/internal/staticdriver"
	"github.com/kbsink-org/kbsink/pkg/core"
)

// prefetchDriver wraps host-supplied article HTML (Obsidian requestUrl) for the initial fetch.
func prefetchDriver(pageURL, html string) staticdriver.Driver {
	html = strings.TrimSpace(html)
	return staticdriver.Driver{
		Result: &core.FetchResult{
			URL:  strings.TrimSpace(pageURL),
			HTML: html,
		},
	}
}
