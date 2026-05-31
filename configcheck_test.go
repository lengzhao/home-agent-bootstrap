package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeConfigDetectsLegacyProviders(t *testing.T) {
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

	findings := analyzeConfig(content)

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

func TestAnalyzeConfigAcceptsProjectProviders(t *testing.T) {
	content := `
[projects.agent]
type = "claudecode"

[projects.agent.options]
provider = "openai"

[[projects.agent.providers]]
name = "openai"
api_key = "sk-openai"
`

	findings := analyzeConfig(content)

	if containsFinding(findings, "FAIL", "") {
		t.Fatalf("unexpected failure findings: %#v", findings)
	}
	if !containsFinding(findings, "OK", `provider "openai" 已在 [[projects.agent.providers]] 中定义`) {
		t.Fatalf("expected provider ok finding, got %#v", findings)
	}
}

func TestMigrateLegacyConfigMovesProviders(t *testing.T) {
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

	got, changed, err := migrateLegacyConfig(content)
	if err != nil {
		t.Fatalf("migrateLegacyConfig() error: %v", err)
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
	docPath := filepath.Join("docs", "platforms.md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read docs/platforms.md: %v", err)
	}

	docTypes, err := platformTypesFromDocs(string(content))
	if err != nil {
		t.Fatalf("platformTypesFromDocs() error: %v", err)
	}
	codeTypes := platformTypesFromPresets()

	if len(docTypes) != len(codeTypes) {
		t.Fatalf("platform count mismatch docs=%d code=%d", len(docTypes), len(codeTypes))
	}
	for i := range docTypes {
		if docTypes[i] != codeTypes[i] {
			t.Fatalf("platform[%d] docs=%q code=%q", i+1, docTypes[i], codeTypes[i])
		}
	}
}

func TestWriteRenderConfigGoldenOpenAI(t *testing.T) {
	if os.Getenv("WRITE_GOLDEN") != "1" {
		t.Skip("set WRITE_GOLDEN=1 to regenerate golden file")
	}

	cfg := goldenRenderConfigInput()
	got := renderConfig(cfg)
	path := filepath.Join("testdata", "render_config_openai.golden.toml")
	if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
		t.Fatalf("write golden file: %v", err)
	}
}

func TestRenderConfigGoldenOpenAI(t *testing.T) {
	got := renderConfig(goldenRenderConfigInput())
	want, err := os.ReadFile(filepath.Join("testdata", "render_config_openai.golden.toml"))
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if got != string(want) {
		t.Fatalf("rendered config differs from golden file\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func goldenRenderConfigInput() RenderConfigInput {
	return RenderConfigInput{
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
		Platforms:              []PlatformBlock{testWeixinPlatform("wx-main", "")},
		MemberDisabledCommands: memberDisabledCommands("family-remind"),
	}
}

func containsFinding(findings []configFinding, level, substr string) bool {
	for _, finding := range findings {
		if finding.level != level {
			continue
		}
		if substr == "" || strings.Contains(finding.message, substr) {
			return true
		}
	}
	return false
}
