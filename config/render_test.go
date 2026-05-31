package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lengzhao/home-agent-bootstrap/permissions"
	"github.com/lengzhao/home-agent-bootstrap/platforms"
)

func testTemplates(t *testing.T) fs.FS {
	t.Helper()
	return os.DirFS(filepath.Join(".."))
}

func mustRender(t *testing.T, cfg RenderInput) string {
	t.Helper()
	got, err := Render(testTemplates(t), cfg)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	return got
}

func testWeixinPlatform(accountID, allowFrom string) platforms.Block {
	return platforms.Block{
		Type: "weixin",
		Options: []platforms.Option{
			{Key: "token", Value: ""},
			{Key: "base_url", Value: "https://ilinkai.weixin.qq.com"},
			{Key: "cdn_base_url", Value: "https://novac2c.cdn.weixin.qq.com/c2c"},
			{Key: "allow_from", Value: allowFrom},
			{Key: "account_id", Value: accountID},
			{Key: "long_poll_timeout_ms", Value: "35000"},
		},
	}
}

func TestRenderOmitsAdminRoleWhenUnknown(t *testing.T) {
	cfg := RenderInput{
		ConfigPath:      "/tmp/config.toml",
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		Platforms: []platforms.Block{
			testWeixinPlatform("wx-main", ""),
			testWeixinPlatform("wx-family", "family@im.wechat"),
		},
	}

	got := mustRender(t, cfg)

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
	if !strings.Contains(got, `tool_messages = false`) {
		t.Fatalf("config should hide tool progress messages by default:\n%s", got)
	}
	if !strings.Contains(got, `reset_on_idle_mins = 0`) {
		t.Fatalf("config should disable idle session auto-rotation by default:\n%s", got)
	}
	if strings.Count(got, `type = "weixin"`) != 2 {
		t.Fatalf("expected two weixin platform blocks:\n%s", got)
	}
	if !strings.Contains(got, `account_id = "wx-family"`) {
		t.Fatalf("expected second account id in config:\n%s", got)
	}
}

func TestRenderIncludesAdminRoleWhenAdminKnown(t *testing.T) {
	cfg := RenderInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		AdminFrom:       "admin@im.wechat",
		Platforms:       []platforms.Block{testWeixinPlatform("wx-main", "")},
	}

	got := mustRender(t, cfg)

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
	got := DaemonInstallArgs("/tmp/config.toml")
	want := []string{"daemon", "install", "--config", "/tmp/config.toml", "--force"}

	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("DaemonInstallArgs() = %#v, want %#v", got, want)
	}
}

func TestRenderIncludesProviderWhenAPIKeyProvided(t *testing.T) {
	cfg := RenderInput{
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
		Platforms:       []platforms.Block{testWeixinPlatform("wx-main", "")},
	}

	got := mustRender(t, cfg)

	for _, want := range []string{
		`[[projects.agent.providers]]`,
		`name = "anthropic"`,
		`api_key = "sk-test"`,
		`provider = "anthropic"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `[[providers]]`) {
		t.Fatalf("provider should be nested under projects.agent, not top-level:\n%s", got)
	}
	if strings.Contains(got, `provider_refs`) {
		t.Fatalf("provider_refs should not be rendered for cc-connect project providers:\n%s", got)
	}
}

func TestRenderIncludesTelegramPlatform(t *testing.T) {
	cfg := RenderInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		Platforms: []platforms.Block{
			{
				Type: "telegram",
				Options: []platforms.Option{
					{Key: "token", Value: "tg-token"},
					{Key: "allow_from", Value: ""},
				},
			},
		},
	}

	got := mustRender(t, cfg)

	for _, want := range []string{
		`type = "telegram"`,
		`token = "tg-token"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
}

func TestRenderOmitsProviderWhenUsingShellEnvOnly(t *testing.T) {
	cfg := RenderInput{
		DataDir:         "/Users/me/.cc-connect",
		Workspace:       "/Users/me/home-assistant-workspace",
		ProjectName:     "home",
		AgentType:       "claudecode",
		AgentMode:       "default",
		ManagementToken: "mgmt",
		BridgeToken:     "bridge",
		WebhookToken:    "hook",
		Platforms:       []platforms.Block{testWeixinPlatform("wx-main", "")},
	}

	got := mustRender(t, cfg)

	if strings.Contains(got, `[[providers]]`) {
		t.Fatalf("config should omit providers when only shell env is used:\n%s", got)
	}
}

func TestRenderIncludesOpenAICompatibleProviderOptions(t *testing.T) {
	cfg := RenderInput{
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
		Platforms:       []platforms.Block{testWeixinPlatform("wx-main", "")},
	}

	got := mustRender(t, cfg)

	for _, want := range []string{
		`[[projects.agent.providers]]`,
		`name = "openai"`,
		`api_key = "sk-openai"`,
		`base_url = "https://api.openai.com/v1"`,
		`model = "gpt-4.1"`,
		`provider = "openai"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `provider_refs`) {
		t.Fatalf("provider_refs should not be rendered for cc-connect project providers:\n%s", got)
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
		err := ValidateAgentMode(tt.agent, tt.mode)
		if tt.ok && err != nil {
			t.Fatalf("ValidateAgentMode(%q, %q) unexpected error: %v", tt.agent, tt.mode, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("ValidateAgentMode(%q, %q) expected error", tt.agent, tt.mode)
		}
	}
}

func TestApplyAdminUserUpdatesProjectAdminRole(t *testing.T) {
	configText := `
[[projects]]
name = "home"
admin_from = ""

[projects.users.roles.admin]
user_ids = []
disabled_commands = []

[projects.users.roles.member]
user_ids = ["*"]
`

	got := ApplyAdminUser(configText, "admin@im.wechat")

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

func TestApplyAdminUserInsertsMissingProjectAdminRole(t *testing.T) {
	configText := `
[[projects]]
name = "home"
admin_from = ""

[projects.users]
default_role = "member"

[projects.users.roles.member]
user_ids = ["*"]
`

	got := ApplyAdminUser(configText, "admin@im.wechat")

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

func TestFirstBoundWeixinAllowFrom(t *testing.T) {
	configText := `
[[projects.platforms]]
type = "weixin"
[projects.platforms.options]
allow_from = ""

[[projects.platforms]]
type = "weixin"
[projects.platforms.options]
allow_from = "admin@im.wechat"
`

	got := FirstBoundWeixinAllowFrom(configText)

	if got != "admin@im.wechat" {
		t.Fatalf("expected first non-empty allow_from, got %q", got)
	}
}

func TestFirstConfiguredAdminFrom(t *testing.T) {
	configText := `
[[projects]]
name = "home"
admin_from = "owner@im.wechat"

[[projects.platforms]]
type = "weixin"
[projects.platforms.options]
allow_from = "admin@im.wechat"
`

	got := FirstConfiguredAdminFrom(configText)

	if got != "owner@im.wechat" {
		t.Fatalf("expected configured admin_from, got %q", got)
	}
}

func goldenRenderInput() RenderInput {
	return RenderInput{
		DataDir:                "/Users/me/.cc-connect",
		Workspace:              "/Users/me/home-assistant-workspace",
		ProjectName:            "home",
		AgentType:              "claudecode",
		AgentMode:              "auto",
		ManagementToken:        "mgmt-token",
		BridgeToken:            "bridge-token",
		WebhookToken:           "hook-token",
		ProviderName:           "openai",
		ProviderAPIKey:         "sk-openai",
		ProviderBaseURL:        "https://api.openai.com/v1",
		ProviderModel:          "gpt-4.1",
		Platforms:              []platforms.Block{testWeixinPlatform("wx-main", "")},
		MemberDisabledCommands: permissions.MemberDisabledCommands("family-remind"),
	}
}

func TestRenderGoldenOpenAI(t *testing.T) {
	got := mustRender(t, goldenRenderInput())
	want, err := os.ReadFile(filepath.Join("..", "testdata", "render_config_openai.golden.toml"))
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if got != string(want) {
		t.Fatalf("rendered config differs from golden file\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestWriteRenderGoldenOpenAI(t *testing.T) {
	if os.Getenv("WRITE_GOLDEN") != "1" {
		t.Skip("set WRITE_GOLDEN=1 to regenerate golden file")
	}

	got := mustRender(t, goldenRenderInput())
	path := filepath.Join("..", "testdata", "render_config_openai.golden.toml")
	if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
		t.Fatalf("write golden file: %v", err)
	}
}
