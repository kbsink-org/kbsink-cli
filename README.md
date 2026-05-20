# kbsink-cli

Command-line front-end for [kbsink](https://github.com/kbsink-org/kbsink): convert **WeChat**, **Xiaohongshu (xhs)**, and **Douyin** share/article links to local Markdown and assets.

## Install

From a clone that includes `kbsink` and `kbsink-plugins` as siblings (or adjust `replace` in `go.mod`):

```bash
go install ./cmd/kbsink
```

Or build:

```bash
go build -o kbsink ./cmd/kbsink
```

### WebAssembly (for JavaScript hosts)

Build the wasm binary (requires Go’s `js/wasm` target):

```bash
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o kbsink.wasm ./cmd/kbsink-wasm
```

Load `kbsink.wasm` together with Go’s loader script from `$(go env GOROOT)/lib/wasm/wasm_exec.js` (Go 1.25+), or `misc/wasm/wasm_exec.js` on older toolchains (see the [Go WebAssembly wiki](https://go.dev/wiki/WebAssembly)). After `go.run(instance)` has started the program, call **`globalThis.kbsinkConvertJSON`** with a **single string** argument: JSON describing the request. It returns a JSON string (synchronous; can block while fetching the article).

Request shape:

```json
{
  "url": "https://mp.weixin.qq.com/s/…",
  "plugin": "wechat",
  "videoMode": "link",
  "timeoutMs": 60000,
  "outputRoot": "output"
}
```

`plugin` is optional if the URL host is recognized (same rules as the CLI). `videoMode` is `link` or `embed` (default `link`). The wasm build uses in-memory storage only: the response includes `markdown` and each asset’s `dataBase64` instead of writing to the host filesystem.

In a **browser** tab, article and CDN requests are subject to **CORS**; **Node.js** with `wasm_exec.js` is often easier for unrestricted HTTP.

#### Build and run (Node)

```bash
./scripts/build-wasm.sh
node ./scripts/run-wasm.mjs "https://mp.weixin.qq.com/s/…"
```

`run-wasm.mjs` loads `kbsink.wasm`, installs a Node `fetch` HTTP bridge (avoids broken Go wasm DNS on some Macs), runs a conversion, and writes Markdown plus assets under `-o` (default `output`).

## Usage

```text
kbsink [flags] <article-url-or-share-text>
```

- **`--plugin`** (`wechat` | `xhs` | `douyin`): optional. If omitted, the tool infers the platform from the first `http(s)` URL in the argument (host-based).
- **`--plugin`** is required when the URL host is not recognized.

Examples:

```bash
kbsink -o output "https://mp.weixin.qq.com/s/xxxx"
kbsink -o output "https://www.xiaohongshu.com/explore/xxxx"
kbsink -o output "https://v.douyin.com/xxxx/"
kbsink --plugin douyin -video-mode=embed -o output "https://v.douyin.com/xxxx/"
```

## Libraries

| Role | Module |
|------|--------|
| Core conversion | `github.com/kbsink-org/kbsink` |
| Platform plugins | `github.com/kbsink-org/kbsink-plugins` |
