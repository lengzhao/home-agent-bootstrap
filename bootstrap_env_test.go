package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultBootstrapSettings(t *testing.T) {
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

	got := defaultBootstrapSettings()

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
	printUsage(&buf)
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

func TestLoadBootstrapSettingsNonInteractive(t *testing.T) {
	t.Setenv("NONINTERACTIVE", "1")
	t.Setenv("AGENT_TYPE", "claudecode")
	t.Setenv("PERMISSION_TEMPLATE", "family-readonly")
	t.Setenv("PLATFORM_CHOICES", "7")

	got := loadBootstrapSettings(&prompt{out: os.Stdout})

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

func TestPermissionTemplateChoiceDefaultFromEnv(t *testing.T) {
	t.Setenv("PERMISSION_TEMPLATE", "family-readonly")
	if got := permissionTemplateChoiceDefault(); got != "2" {
		t.Fatalf("permissionTemplateChoiceDefault() = %q, want 2", got)
	}
}
