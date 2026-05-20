//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	klog "github.com/kbsink-org/kbsink/pkg/logger"
)

// GlobalLogJSName is the globalThis property the host may implement for bridged logs.
// Signature: (jsonPayload: string) => void
// Payload: {"level":"info|debug|warn|error","msg":"...","fields":{...}}
const GlobalLogJSName = "kbsinkLog"

type jsLogPayload struct {
	Level  string         `json:"level"`
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
}

// JSLogger forwards log lines to globalThis.kbsinkLog when present; otherwise no-op.
type JSLogger struct{}

func (JSLogger) Log(level klog.Level, msg string, kv ...any) {
	fn := js.Global().Get(GlobalLogJSName)
	if fn.Type() != js.TypeFunction {
		return
	}
	payload := jsLogPayload{
		Level:  level.String(),
		Msg:    msg,
		Fields: kvToMap(kv),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		fn.Invoke(js.ValueOf(fmt.Sprintf(
			`{"level":"error","msg":"kbsinkLog marshal failed","fields":{"err":%q}}`,
			err.Error(),
		)))
		return
	}
	fn.Invoke(js.ValueOf(string(b)))
}

func (j JSLogger) Debug(msg string, kv ...any) { j.Log(klog.LevelDebug, msg, kv...) }
func (j JSLogger) Info(msg string, kv ...any)  { j.Log(klog.LevelInfo, msg, kv...) }
func (j JSLogger) Warn(msg string, kv ...any)  { j.Log(klog.LevelWarn, msg, kv...) }
func (j JSLogger) Error(msg string, kv ...any) { j.Log(klog.LevelError, msg, kv...) }

func kvToMap(kv []any) map[string]any {
	if len(kv) == 0 {
		return nil
	}
	out := make(map[string]any, (len(kv)+1)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok || key == "" {
			continue
		}
		out[key] = kv[i+1]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
