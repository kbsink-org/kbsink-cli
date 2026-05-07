//go:build js && wasm

// Command kbsink-wasm compiles to WebAssembly and exposes a JSON API for JavaScript.
//
// Build:
//
//	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o kbsink.wasm ./cmd/kbsink-wasm
//
// Load with GOROOT's lib/wasm/wasm_exec.js (Go 1.25+) or misc/wasm/wasm_exec.js, then call globalThis.kbsinkConvertJSON(reqJsonString).
// Optional: globalThis.kbsinkHTTPRoundTrip — host-backed net/http (bridge.go, README).
package main

import (
	"context"
	"encoding/json"
	"syscall/js"
	"time"

	"github.com/kbsink-org/kbsink-cli/internal/convertlib"
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

	timeout := 60 * time.Second
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
		Plugin:     req.Plugin,
		VideoMode:  req.VideoMode,
		Timeout:    timeout,
		OutputRoot: req.OutputRoot,
		HTTPClient: NewHostBridgedHTTPClient(timeout),
	})
	if err != nil {
		return mustJSON(wasmResponse{OK: false, Error: err.Error()})
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
