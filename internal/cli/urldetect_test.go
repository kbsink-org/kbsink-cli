package cli

import "testing"

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
