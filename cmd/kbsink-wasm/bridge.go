//go:build js && wasm

// Host-backed HTTP for kbsink.wasm: HostTransport delegates to globalThis.kbsinkHTTPRoundTrip.
// Node hosts should install the hook before go.run (see scripts/run-wasm.mjs).
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall/js"
	"time"
)

// GlobalJSName is the property on globalThis the host must implement for bridged HTTP.
const GlobalJSName = "kbsinkHTTPRoundTrip"

type jsHTTPRequest struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	BodyB64   string            `json:"bodyB64,omitempty"`
	TimeoutMs int               `json:"timeoutMs,omitempty"`
}

func contextTimeoutMs(ctx context.Context) int {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	ms := int(time.Until(deadline).Milliseconds())
	if ms < 1 {
		return 1
	}
	return ms
}

type jsHTTPResponse struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	BodyB64    string            `json:"bodyBase64,omitempty"`
}

func flattenHeaders(h http.Header) map[string]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, vs := range h {
		if len(vs) == 0 {
			continue
		}
		out[k] = strings.Join(vs, ",")
	}
	return out
}

// HostTransport implements [http.RoundTripper] via the host JS hook when set;
// otherwise it falls back to [http.DefaultTransport].
type HostTransport struct{}

func (HostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fn := js.Global().Get(GlobalJSName)
	if fn.Type() != js.TypeFunction {
		return http.DefaultTransport.RoundTrip(req)
	}

	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	payload := jsHTTPRequest{
		Method:    req.Method,
		URL:       req.URL.String(),
		Headers:   flattenHeaders(req.Header),
		TimeoutMs: contextTimeoutMs(req.Context()),
	}
	if len(body) > 0 {
		payload.BodyB64 = base64.StdEncoding.EncodeToString(body)
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	promise := fn.Invoke(js.ValueOf(string(jsonBytes)))
	if promise.Type() != js.TypeObject || promise.Get("then").Type() != js.TypeFunction {
		return nil, fmt.Errorf("%s must return a Promise", GlobalJSName)
	}

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	var success, failure js.Func
	success = js.FuncOf(func(this js.Value, args []js.Value) any {
		success.Release()
		failure.Release()
		raw := args[0].String()
		var jr jsHTTPResponse
		if err := json.Unmarshal([]byte(raw), &jr); err != nil {
			errCh <- fmt.Errorf("%s: bad JSON: %w", GlobalJSName, err)
			return nil
		}
		var bodyReader io.ReadCloser = http.NoBody
		if jr.BodyB64 != "" {
			dec, err := base64.StdEncoding.DecodeString(jr.BodyB64)
			if err != nil {
				errCh <- fmt.Errorf("%s: body base64: %w", GlobalJSName, err)
				return nil
			}
			bodyReader = io.NopCloser(bytes.NewReader(dec))
		}
		h := make(http.Header)
		for k, v := range jr.Headers {
			if v != "" {
				h.Set(k, v)
			}
		}
		st := jr.Status
		if st == 0 {
			st = 200
		}
		stText := jr.StatusText
		if stText == "" {
			stText = http.StatusText(st)
			if stText == "" {
				stText = "status"
			}
		}
		respCh <- &http.Response{
			Status:        fmt.Sprintf("%d %s", st, stText),
			StatusCode:    st,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        h,
			Body:          bodyReader,
			ContentLength: -1,
			Request:       req,
		}
		return nil
	})
	failure = js.FuncOf(func(this js.Value, args []js.Value) any {
		success.Release()
		failure.Release()
		msg := args[0].Call("toString").String()
		errCh <- fmt.Errorf("%s: %s", GlobalJSName, msg)
		return nil
	})
	promise.Call("then", success, failure)

	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	case <-req.Context().Done():
		// Brief grace: host fetch may finish at the same instant as the Go deadline.
		grace := time.NewTimer(10 * time.Second)
		defer grace.Stop()
		select {
		case resp := <-respCh:
			return resp, nil
		case err := <-errCh:
			return nil, err
		case <-grace.C:
			return nil, req.Context().Err()
		}
	}
}

// NewHostBridgedHTTPClient returns an [http.Client] using [HostTransport].
func NewHostBridgedHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   0,
		Transport: HostTransport{},
	}
}
