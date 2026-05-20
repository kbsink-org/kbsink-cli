//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"strings"
	"syscall/js"
	"time"

	"github.com/kbsink-org/kbsink-cli/internal/convertlib"
	clidriver "github.com/kbsink-org/kbsink-cli/internal/driver"
	"github.com/kbsink-org/kbsink-cli/internal/netclient"
)

func registerWASM() {
	convertFn := js.FuncOf(convertJSON)
	js.Global().Set("kbsinkConvertJSON", convertFn)
}

type wasmRequest struct {
	URL        string `json:"url"`
	Plugin     string `json:"plugin,omitempty"`
	VideoMode  string `json:"videoMode,omitempty"`
	TimeoutMs  int    `json:"timeoutMs,omitempty"`
	OutputRoot string `json:"outputRoot,omitempty"`
	LogLevel   string `json:"logLevel,omitempty"`
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

	log := JSLogger{}
	httpClient := netclient.FromHostHTTP(NewHostBridgedHTTPClient())
	coreHTTP := netclient.CoreHTTPClient(httpClient)

	drv, err := clidriver.ForPlugin(pluginName, coreHTTP, log)
	if err != nil {
		return mustJSON(wasmResponse{OK: false, Error: err.Error()})
	}

	res, err := convertlib.Convert(ctx, convertlib.Params{
		URL:        req.URL,
		Plugin:     pluginName,
		VideoMode:  req.VideoMode,
		Timeout:    timeout,
		OutputRoot: req.OutputRoot,
		HTTP:       httpClient,
		KbsinkLog:  log,
		LogLevel:   req.LogLevel,
		Driver:     drv,
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
