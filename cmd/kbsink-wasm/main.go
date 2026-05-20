//go:build js && wasm

// Command kbsink-wasm compiles to WebAssembly and exposes a JSON API for JavaScript.
//
// Build:
//
//	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o kbsink.wasm ./cmd/kbsink-wasm
//
// Load with GOROOT's lib/wasm/wasm_exec.js (Go 1.25+) or misc/wasm/wasm_exec.js, then call globalThis.kbsinkConvertJSON(reqJsonString).
// Optional: globalThis.kbsinkLog — host log sink (jslog.go).
// HTTP uses globalThis.kbsinkHTTPRoundTrip when set (bridge.go); scripts/run-wasm.mjs installs it for Node.
package main

import (
	"context"
	"encoding/json"
	"strings"
	"syscall/js"
	"time"

	"github.com/kbsink-org/kbsink-cli/internal/convertlib"
	"github.com/kbsink-org/kbsink-cli/internal/netclient"
)

func main() {
	convertFn := js.FuncOf(convertJSON)
	js.Global().Set("kbsinkConvertJSON", convertFn)
	// Keep wasm instance alive (browser or long-lived Node host).
	select {}
}

type wasmRequest struct {
	URL        string `json:"url"`
	Plugin     string `json:"plugin,omitempty"`
	VideoMode  string `json:"videoMode,omitempty"`
	TimeoutMs  int    `json:"timeoutMs,omitempty"`
	OutputRoot string `json:"outputRoot,omitempty"`
}

type wasmResponse struct {
	OK     bool                    `json:"ok"`
	Error  string                  `json:"error,omitempty"`
	Result *convertlib.ArticleJSON `json:"result,omitempty"`
}

func convertJSON(this js.Value, args []js.Value) any {
	_ = this
	if len(args) == 0 || args[0].Type() != js.TypeString {
		return mustJSON(wasmResponse{OK: false, Error: "expected one string argument: JSON request body"})
	}
	var req wasmRequest
	if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
		return mustJSON(wasmResponse{OK: false, Error: "invalid JSON: " + err.Error()})
	}
	if req.URL == "" {
		return mustJSON(wasmResponse{OK: false, Error: "field \"url\" is required"})
	}

	convertlib.EnsurePluginsRegistered()
	pluginName := strings.TrimSpace(req.Plugin)
	if pluginName == "" {
		var ok bool
		pluginName, ok = convertlib.DetectPlugin(req.URL)
		if !ok {
			return mustJSON(wasmResponse{OK: false, Error: "field \"plugin\" is required (wechat, xhs, douyin), or use a URL with a recognized host"})
		}
	}
	pluginName = strings.ToLower(pluginName)

	timeout := 10 * time.Minute
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	res, err := convertlib.Convert(ctx, convertlib.Params{
		URL:        req.URL,
		Plugin:     pluginName,
		VideoMode:  req.VideoMode,
		Timeout:    timeout,
		OutputRoot: req.OutputRoot,
		HTTP:       netclient.FromHostHTTP(NewHostBridgedHTTPClient()),
		KbsinkLog:  JSLogger{},
	})
	if err != nil {
		return mustJSON(wasmResponse{OK: false, Error: convertlib.ErrChainString(err)})
	}
	return mustJSON(wasmResponse{OK: true, Result: res})
}

func mustJSON(v wasmResponse) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"ok":false,"error":"marshal failed"}`
	}
	return string(b)
}
