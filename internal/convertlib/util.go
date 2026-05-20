package convertlib

import (
	"net/url"
	"regexp"
	"strings"
)

var firstURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

// firstURLInText returns the first http(s) URL in s, or empty if none.
func firstURLInText(s string) string {
	return firstURLPattern.FindString(s)
}

// DetectPlugin returns a plugin id inferred from raw input
// (a URL or text containing an http(s) URL). The second return is false if unknown.
func DetectPlugin(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}
	candidate := s
	if u := firstURLInText(s); u != "" {
		candidate = u
	}
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	return pluginForHost(parsed.Hostname())
}

func pluginForHost(host string) (string, bool) {
	host = strings.ToLower(host)
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
