package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderConfigOmitsAdminRoleWhenUnknown(t *testing.T) {
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
		Platforms: []PlatformBlock{
			testWeixinPlatform("wx-main", ""),
			testWeixinPlatform("wx-family", "family@im.wechat"),
		},
	}

	got := renderConfig(cfg)

	if strings.Contains(got, "[projects.users.roles.admin]") {
		t.Fatalf("config should omit admin role when admin_from is unknown:\n%s", got)
	}
	if strings.Contains(got, "user_ids = []") {
		t.Fatalf("config should not render empty role user_ids:\n%s", got)
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

func TestRenderConfigIncludesAdminRoleWhenAdminKnown(t *testing.T) {
	cfg := RenderConfigInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		AdminFrom:       "admin@im.wechat",
		Platforms:       []PlatformBlock{testWeixinPlatform("wx-main", "")},
	}

	got := renderConfig(cfg)

	for _, want := range []string{
		`admin_from = "admin@im.wechat"`,
		`[projects.users.roles.admin]`,
		`user_ids = ["admin@im.wechat"]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
}

func TestDaemonInstallArgsUsesForce(t *testing.T) {
	got := daemonInstallArgs("/tmp/config.toml")
	want := []string{"daemon", "install", "--config", "/tmp/config.toml", "--force"}

	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("daemonInstallArgs() = %#v, want %#v", got, want)
	}
}

func TestClaudeWorkspaceInitCommandUsesWorkspace(t *testing.T) {
	name, args, dir := claudeWorkspaceInitCommand("/Users/me/home-assistant-workspace")

	if name != "claude" {
		t.Fatalf("command name = %q, want claude", name)
	}
	if len(args) != 0 {
		t.Fatalf("command args = %#v, want none", args)
	}
	if dir != "/Users/me/home-assistant-workspace" {
		t.Fatalf("command dir = %q, want workspace", dir)
	}
}

func TestWeixinFirstMessageInstructionMentionsLogin(t *testing.T) {
	got := weixinFirstMessageInstruction()

	if !strings.Contains(got, "/login") {
		t.Fatalf("first message instruction should mention /login:\n%s", got)
	}
	if !strings.Contains(got, "context_token") {
		t.Fatalf("first message instruction should mention context_token:\n%s", got)
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
		Platforms:       []PlatformBlock{testWeixinPlatform("wx-main", "")},
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

func TestRenderConfigIncludesTelegramPlatform(t *testing.T) {
	cfg := RenderConfigInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		Platforms: []PlatformBlock{
			{
				Type: "telegram",
				Options: []PlatformOption{
					{Key: "token", Value: "tg-token"},
					{Key: "allow_from", Value: ""},
				},
			},
		},
	}

	got := renderConfig(cfg)

	for _, want := range []string{
		`type = "telegram"`,
		`token = "tg-token"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
}

func TestOpenRouterClaudeCodeShellProfile(t *testing.T) {
	profile := openrouterClaudeCodeProfile("sk-or", "", "")
	block := buildClaudeCodeExportBlock(profile)

	for _, want := range []string{
		`ANTHROPIC_BASE_URL='https://openrouter.ai/api/v1'`,
		`ANTHROPIC_AUTH_TOKEN='sk-or'`,
		`ANTHROPIC_MODEL='anthropic/claude-sonnet-4'`,
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("block missing %q:\n%s", want, block)
		}
	}
}

func TestRenderConfigOmitsProviderWhenUsingShellEnvOnly(t *testing.T) {
	cfg := RenderConfigInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		Platforms:       []PlatformBlock{testWeixinPlatform("wx-main", "")},
	}

	got := renderConfig(cfg)

	if strings.Contains(got, `[[providers]]`) {
		t.Fatalf("config should omit providers when only shell env is used:\n%s", got)
	}
}

func TestRenderConfigIncludesOpenAICompatibleProviderOptions(t *testing.T) {
	cfg := RenderConfigInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		ProviderName:    "openai",
		ProviderAPIKey:  "sk-openai",
		ProviderBaseURL: "https://api.openai.com/v1",
		ProviderModel:   "gpt-4.1",
		Platforms:       []PlatformBlock{testWeixinPlatform("wx-main", "")},
	}

	got := renderConfig(cfg)

	for _, want := range []string{
		`name = "openai"`,
		`api_key = "sk-openai"`,
		`base_url = "https://api.openai.com/v1"`,
		`model = "gpt-4.1"`,
		`provider = "openai"`,
		`provider_refs = ["openai"]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
}

func TestFirstBoundWeixinAllowFrom(t *testing.T) {
	config := `
[[projects.platforms]]
type = "weixin"
[projects.platforms.options]
allow_from = ""

[[projects.platforms]]
type = "weixin"
[projects.platforms.options]
allow_from = "admin@im.wechat"
`

	got := firstBoundWeixinAllowFrom(config)

	if got != "admin@im.wechat" {
		t.Fatalf("expected first non-empty allow_from, got %q", got)
	}
}

func TestFirstConfiguredAdminFrom(t *testing.T) {
	config := `
[[projects]]
name = "home"
admin_from = "owner@im.wechat"

[[projects.platforms]]
type = "weixin"
[projects.platforms.options]
allow_from = "admin@im.wechat"
`

	got := firstConfiguredAdminFrom(config)

	if got != "owner@im.wechat" {
		t.Fatalf("expected configured admin_from, got %q", got)
	}
}

func TestApplyAdminUserToConfigUpdatesProjectAdminRole(t *testing.T) {
	config := `
[[projects]]
name = "home"
admin_from = ""

[projects.users.roles.admin]
user_ids = []
disabled_commands = []

[projects.users.roles.member]
user_ids = ["*"]
`

	got := applyAdminUserToConfig(config, "admin@im.wechat")

	for _, want := range []string{
		`admin_from = "admin@im.wechat"`,
		`user_ids = ["admin@im.wechat"]`,
		`[projects.users.roles.member]`,
		`user_ids = ["*"]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("updated config missing %q:\n%s", want, got)
		}
	}
}

func TestApplyAdminUserToConfigInsertsMissingProjectAdminRole(t *testing.T) {
	config := `
[[projects]]
name = "home"
admin_from = ""

[projects.users]
default_role = "member"

[projects.users.roles.member]
user_ids = ["*"]
`

	got := applyAdminUserToConfig(config, "admin@im.wechat")

	for _, want := range []string{
		`admin_from = "admin@im.wechat"`,
		`[projects.users.roles.admin]`,
		`user_ids = ["admin@im.wechat"]`,
		`disabled_commands = []`,
		`rate_limit = { max_messages = 50, window_secs = 60 }`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("updated config missing %q:\n%s", want, got)
		}
	}
	if strings.Index(got, `[projects.users.roles.admin]`) > strings.Index(got, `[projects.users.roles.member]`) {
		t.Fatalf("admin role should be inserted before member role:\n%s", got)
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
