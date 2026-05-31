package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/prompt"
)

func TestDefaultSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CONFIG_PATH", "")
	t.Setenv("WORKSPACE", "")
	t.Setenv("PROJECT_NAME", "")
	t.Setenv("AGENT_TYPE", "")
	t.Setenv("AGENT_MODE", "")
	t.Setenv("PERMISSION_TEMPLATE", "")
	t.Setenv("PLATFORM_CHOICES", "")
	t.Setenv("LLM_CHOICE", "")

	got := DefaultSettings()

	if got.ConfigPath != filepath.Join(home, ".cc-connect", "config.toml") {
		t.Fatalf("ConfigPath = %q", got.ConfigPath)
	}
	if got.Workspace != filepath.Join(home, "home-assistant-workspace") {
		t.Fatalf("Workspace = %q", got.Workspace)
	}
	if got.ProjectName != "home" {
		t.Fatalf("ProjectName = %q", got.ProjectName)
	}
	if got.AgentType != "claudecode" {
		t.Fatalf("AgentType = %q", got.AgentType)
	}
	if got.AgentMode != "auto" {
		t.Fatalf("AgentMode = %q", got.AgentMode)
	}
	if got.PermissionTemplate != "family-remind" {
		t.Fatalf("PermissionTemplate = %q", got.PermissionTemplate)
	}
	if got.PlatformChoices != "7" {
		t.Fatalf("PlatformChoices = %q", got.PlatformChoices)
	}
	if got.LLMChoice != "1" {
		t.Fatalf("LLMChoice = %q", got.LLMChoice)
	}
}

func TestPrintUsageDocumentsEnvAndDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var buf bytes.Buffer
	PrintUsage(&buf)
	out := buf.String()

	for _, want := range []string{
		"NONINTERACTIVE=1",
		"BOOTSTRAP_YES=1",
		"WORKSPACE",
		"LLM_API_KEY",
		"WEIXIN_COUNT",
		"family-remind",
		"直接回车即可",
		"home-assistant-workspace",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q", want)
		}
	}
}

func TestLoadSettingsNonInteractive(t *testing.T) {
	t.Setenv("NONINTERACTIVE", "1")
	t.Setenv("AGENT_TYPE", "claudecode")
	t.Setenv("PERMISSION_TEMPLATE", "family-readonly")
	t.Setenv("PLATFORM_CHOICES", "7")

	got := loadSettings(prompt.New(nil, os.Stdout, true))

	if got.AgentMode != "auto" {
		t.Fatalf("AgentMode = %q, want auto", got.AgentMode)
	}
	if got.PermissionTemplate != "family-readonly" {
		t.Fatalf("PermissionTemplate = %q, want family-readonly", got.PermissionTemplate)
	}
	if got.PlatformChoices != "7" {
		t.Fatalf("PlatformChoices = %q, want 7", got.PlatformChoices)
	}
}

func TestWeixinFirstMessageInstructionDoesNotMentionClaudeLogin(t *testing.T) {
	got := weixinFirstMessageInstruction()

	if strings.Contains(got, "/login") {
		t.Fatalf("weixin first message instruction should not confuse Claude Code /login with weixin binding:\n%s", got)
	}
	if !strings.Contains(got, "/whoami") {
		t.Fatalf("first message instruction should mention /whoami:\n%s", got)
	}
}

func TestDefaultConfigPathUsesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := cmdutil.DefaultConfigPath(); got != filepath.Join(home, ".cc-connect", "config.toml") {
		t.Fatalf("DefaultConfigPath() = %q", got)
	}
}
