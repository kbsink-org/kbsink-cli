# kbsink-cli

Command-line front-end for [kbsink](https://github.com/kbsink-org/kbsink): convert **WeChat**, **Xiaohongshu (xhs)**, and **Douyin** share/article links to local Markdown and assets.

## Install

From a clone that includes `kbsink` and `douyin-plugin` as siblings (or adjust `replace` in `go.mod`):

```bash
go install ./cmd/kbsink
```

Or build:

```bash
go build -o kbsink ./cmd/kbsink
```

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
| Douyin parser/driver | `github.com/kbsink-org/douyin-plugin` |
