package cli

import (
	"net/url"
	"regexp"
	"strings"
)

var firstURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

// DetectPlugin returns a plugin id (wechat, xhs, douyin) inferred from raw input
// (a URL or text containing an http(s) URL). The second return is false if unknown.
func DetectPlugin(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}
	candidate := s
	if m := firstURLPattern.FindString(s); m != "" {
		candidate = m
	}
	u, err := url.Parse(candidate)
	if err != nil || u.Host == "" {
		return "", false
	}
	host := strings.ToLower(u.Hostname())

	switch {
	case strings.Contains(host, "weixin.qq.com"):
		return "wechat", true
	case strings.Contains(host, "xiaohongshu.com"),
		strings.Contains(host, "xhslink.com"),
		strings.Contains(host, "xhs.cn"):
		return "xhs", true
	case strings.Contains(host, "douyin.com"),
		strings.Contains(host, "iesdouyin.com"):
		return "douyin", true
	default:
		return "", false
	}
}
