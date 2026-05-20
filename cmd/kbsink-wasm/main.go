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

func main() {
	registerWASM()
	// Keep wasm instance alive (browser or long-lived Node host).
	select {}
}
