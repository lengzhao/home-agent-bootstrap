package main

import (
	"strings"
	"testing"
)

func TestParsePlatformChoicesDefaultWeixin(t *testing.T) {
	got, err := parsePlatformChoices("")
	if err != nil {
		t.Fatalf("parsePlatformChoices() error: %v", err)
	}
	if len(got) != 1 || got[0].Type != "weixin" {
		t.Fatalf("expected default weixin, got %#v", got)
	}
}

func TestParsePlatformChoicesMultiple(t *testing.T) {
	got, err := parsePlatformChoices("1,7")
	if err != nil {
		t.Fatalf("parsePlatformChoices() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 platforms, got %#v", got)
	}
	if got[0].Type != "feishu" || got[1].Type != "weixin" {
		t.Fatalf("unexpected platforms: %#v", got)
	}
}

func TestParsePlatformChoicesDedupes(t *testing.T) {
	got, err := parsePlatformChoices("7,7")
	if err != nil {
		t.Fatalf("parsePlatformChoices() error: %v", err)
	}
	if len(got) != 1 || got[0].Type != "weixin" {
		t.Fatalf("expected deduped weixin, got %#v", got)
	}
}

func TestCountWeixinPlatforms(t *testing.T) {
	platforms := []PlatformBlock{
		{Type: "telegram"},
		{Type: "weixin"},
		{Type: "weixin"},
	}
	if count := countWeixinPlatforms(platforms); count != 2 {
		t.Fatalf("countWeixinPlatforms() = %d, want 2", count)
	}
}

func TestPlatformSetupHintsSkipsWeixinWhenListed(t *testing.T) {
	platforms := []PlatformBlock{
		{Type: "feishu"},
		{Type: "weixin"},
	}
	hints := platformSetupHints(platforms)
	joined := strings.Join(hints, "\n")
	if !strings.Contains(joined, "feishu setup") {
		t.Fatalf("expected feishu setup hint, got %q", joined)
	}
	if strings.Contains(joined, "weixin setup") {
		t.Fatalf("weixin setup should not appear in generic hints: %q", joined)
	}
}

func testWeixinPlatform(accountID, allowFrom string) PlatformBlock {
	return PlatformBlock{
		Type: "weixin",
		Options: []PlatformOption{
			{Key: "token", Value: ""},
			{Key: "base_url", Value: "https://ilinkai.weixin.qq.com"},
			{Key: "cdn_base_url", Value: "https://novac2c.cdn.weixin.qq.com/c2c"},
			{Key: "allow_from", Value: allowFrom},
			{Key: "account_id", Value: accountID},
			{Key: "long_poll_timeout_ms", Value: "35000"},
		},
	}
}
