//go:build js && wasm

// Host-backed HTTP for kbsink.wasm: HostTransport delegates to globalThis.kbsinkHTTPRoundTrip (see kbsink-cli README).
package main

import (
	"bytes"
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
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	BodyB64 string            `json:"bodyB64,omitempty"`
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

// HostTransport implements [http.RoundTripper] by calling the host JS hook when set;
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
	}

	payload := jsHTTPRequest{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: flattenHeaders(req.Header),
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
		err := args[0]
		msg := err.Call("toString").String()
		errCh <- fmt.Errorf("%s: %s", GlobalJSName, msg)
		return nil
	})
	promise.Call("then", success, failure)

	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

// NewHostBridgedHTTPClient returns an [http.Client] whose transport is [HostTransport]
// (host hook + fallback). Timeout matches [net/http.Client.Timeout] semantics.
func NewHostBridgedHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: HostTransport{},
	}
}
