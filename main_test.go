package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderConfigOmitsAdminUserWhenUnknown(t *testing.T) {
	cfg := RenderConfigInput{
		ConfigPath:      "/tmp/config.toml",
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		WeixinAccounts: []WeixinAccount{
			{AccountID: "wx-main"},
			{AccountID: "wx-family", AllowFrom: "family@im.wechat"},
		},
	}

	got := renderConfig(cfg)

	if !strings.Contains(got, "user_ids = []") {
		t.Fatalf("config should use an empty admin user list when admin_from is unknown:\n%s", got)
	}
	if strings.Contains(got, `disabled_commands = ["/shell"`) {
		t.Fatalf("disabled_commands must use command ids without slash:\n%s", got)
	}
	if !strings.Contains(got, `disabled_commands = ["shell", "show", "dir", "restart", "upgrade", "commands"]`) {
		t.Fatalf("config should disable high-risk command ids for members:\n%s", got)
	}
	if strings.Count(got, `type = "weixin"`) != 2 {
		t.Fatalf("expected two weixin platform blocks:\n%s", got)
	}
	if !strings.Contains(got, `account_id = "wx-family"`) {
		t.Fatalf("expected second account id in config:\n%s", got)
	}
}

func TestRenderConfigIncludesProviderWhenAPIKeyProvided(t *testing.T) {
	cfg := RenderConfigInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		ProviderName:    "anthropic",
		ProviderAPIKey:  "sk-test",
		WeixinAccounts:  []WeixinAccount{{AccountID: "wx-main"}},
	}

	got := renderConfig(cfg)

	for _, want := range []string{
		`[[providers]]`,
		`name = "anthropic"`,
		`api_key = "sk-test"`,
		`provider = "anthropic"`,
		`provider_refs = ["anthropic"]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
}

func TestValidateAgentMode(t *testing.T) {
	tests := []struct {
		agent string
		mode  string
		ok    bool
	}{
		{"claudecode", "default", true},
		{"claudecode", "plan", true},
		{"claudecode", "force", false},
		{"cursor", "ask", true},
		{"cursor", "force", true},
		{"cursor", "acceptEdits", false},
	}

	for _, tt := range tests {
		err := validateAgentMode(tt.agent, tt.mode)
		if tt.ok && err != nil {
			t.Fatalf("validateAgentMode(%q, %q) unexpected error: %v", tt.agent, tt.mode, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("validateAgentMode(%q, %q) expected error", tt.agent, tt.mode)
		}
	}
}

func TestWriteWorkspaceFilesIncludesDefaultSkills(t *testing.T) {
	dir := t.TempDir()

	if err := writeWorkspaceFiles(dir); err != nil {
		t.Fatalf("writeWorkspaceFiles() error: %v", err)
	}

	for _, rel := range []string{
		"CLAUDE.md",
		"HOME.md",
		"HEARTBEAT.md",
		"skills/cc-connect/SKILL.md",
		"skills/skill-creator/SKILL.md",
		"skills/skill-maintenance/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected workspace file %s: %v", rel, err)
		}
	}
}
