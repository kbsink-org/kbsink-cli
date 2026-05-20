package convertlib

import "testing"

func TestFirstURLInText(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"https://mp.weixin.qq.com/s/abc", "https://mp.weixin.qq.com/s/abc"},
		{"check this https://v.douyin.com/xx/ ok", "https://v.douyin.com/xx/"},
		{"not a url", ""},
	}
	for _, tt := range tests {
		if got := firstURLInText(tt.raw); got != tt.want {
			t.Errorf("firstURLInText(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestPluginForHost(t *testing.T) {
	tests := []struct {
		host string
		want string
		ok   bool
	}{
		{"mp.weixin.qq.com", "wechat", true},
		{"www.xiaohongshu.com", "xhs", true},
		{"xhslink.com", "xhs", true},
		{"v.douyin.com", "douyin", true},
		{"www.bilibili.com", "", false},
		{"b23.tv", "", false},
		{"zhuanlan.zhihu.com", "", false},
		{"example.com", "", false},
	}
	for _, tt := range tests {
		got, ok := pluginForHost(tt.host)
		if ok != tt.ok || got != tt.want {
			t.Errorf("pluginForHost(%q) = (%q, %v), want (%q, %v)", tt.host, got, ok, tt.want, tt.ok)
		}
	}
}

func TestDetectPlugin(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"https://mp.weixin.qq.com/s/abc", "wechat", true},
		{"  https://mp.weixin.qq.com/s/x  ", "wechat", true},
		{"https://www.xiaohongshu.com/explore/xyz", "xhs", true},
		{"https://xhslink.com/a/b", "xhs", true},
		{"https://v.douyin.com/AbCdEf/", "douyin", true},
		{"https://www.douyin.com/video/123", "douyin", true},
		{"https://www.iesdouyin.com/share/video/1", "douyin", true},
		{"check this https://v.douyin.com/xx/ ok", "douyin", true},
		{"https://www.bilibili.com/video/BV1X1Lj66ENe/", "", false},
		{"https://b23.tv/BV1X1Lj66ENe", "", false},
		{"https://zhuanlan.zhihu.com/p/26954669801", "", false},
		{"https://example.com/", "", false},
		{"not a url", "", false},
	}
	for _, tt := range tests {
		got, ok := DetectPlugin(tt.raw)
		if ok != tt.ok || got != tt.want {
			t.Errorf("DetectPlugin(%q) = (%q, %v), want (%q, %v)", tt.raw, got, ok, tt.want, tt.ok)
		}
	}
}
