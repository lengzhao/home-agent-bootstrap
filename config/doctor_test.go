package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeDetectsLegacyProviders(t *testing.T) {
	content := `
[[providers]]
name = "openai"
api_key = "sk-openai"
agent_types = ["claudecode"]

[projects.agent]
type = "claudecode"

[projects.agent.options]
provider = "openai"
provider_refs = ["openai"]
`

	findings := Analyze(content)

	if !containsFinding(findings, "WARN", "旧版顶层 [[providers]]") {
		t.Fatalf("expected legacy provider warning, got %#v", findings)
	}
	if !containsFinding(findings, "WARN", "provider_refs") {
		t.Fatalf("expected provider_refs warning, got %#v", findings)
	}
	if !containsFinding(findings, "FAIL", "未找到 [[projects.agent.providers]]") {
		t.Fatalf("expected missing project provider failure, got %#v", findings)
	}
}

func TestAnalyzeAcceptsProjectProviders(t *testing.T) {
	content := `
[projects.agent]
type = "claudecode"

[projects.agent.options]
provider = "openai"

[[projects.agent.providers]]
name = "openai"
api_key = "sk-openai"
`

	findings := Analyze(content)

	if containsFinding(findings, "FAIL", "") {
		t.Fatalf("unexpected failure findings: %#v", findings)
	}
	if !containsFinding(findings, "OK", `provider "openai" 已在 [[projects.agent.providers]] 中定义`) {
		t.Fatalf("expected provider ok finding, got %#v", findings)
	}
}

func TestMigrateLegacyMovesProviders(t *testing.T) {
	content := `
[[providers]]
name = "openai"
api_key = "sk-openai"
base_url = "https://api.openai.com/v1"
model = "gpt-4.1"
agent_types = ["claudecode"]

[projects.agent]
type = "claudecode"

[projects.agent.options]
work_dir = "/Users/me/home-assistant-workspace"
mode = "default"
provider = "openai"
provider_refs = ["openai"]
`

	got, changed, err := MigrateLegacy(content)
	if err != nil {
		t.Fatalf("MigrateLegacy() error: %v", err)
	}
	if !changed {
		t.Fatal("expected migration to change config")
	}
	for _, want := range []string{
		`[[projects.agent.providers]]`,
		`name = "openai"`,
		`api_key = "sk-openai"`,
		`provider = "openai"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("migrated config missing %q:\n%s", want, got)
		}
	}
	for _, bad := range []string{
		"[[providers]]",
		"provider_refs",
		"agent_types",
	} {
		if strings.Contains(got, bad) {
			t.Fatalf("migrated config should remove %q:\n%s", bad, got)
		}
	}
}

func TestPlatformPresetsMatchDocs(t *testing.T) {
	docPath := filepath.Join("..", "docs", "platforms.md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read docs/platforms.md: %v", err)
	}

	docTypes, err := PlatformTypesFromDocs(string(content))
	if err != nil {
		t.Fatalf("PlatformTypesFromDocs() error: %v", err)
	}
	codeTypes := PlatformTypesFromPresets()

	if len(docTypes) != len(codeTypes) {
		t.Fatalf("platform count mismatch docs=%d code=%d", len(docTypes), len(codeTypes))
	}
	for i := range docTypes {
		if docTypes[i] != codeTypes[i] {
			t.Fatalf("platform[%d] docs=%q code=%q", i+1, docTypes[i], codeTypes[i])
		}
	}
}

func containsFinding(findings []finding, level, substr string) bool {
	for _, item := range findings {
		if item.level != level {
			continue
		}
		if substr == "" || strings.Contains(item.message, substr) {
			return true
		}
	}
	return false
}
