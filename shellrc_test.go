package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertShellRCBlockCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".zshrc")

	created, err := upsertShellRCBlock(path, defaultKimiClaudeCodeEnv("sk-test").shellExportLines())
	if err != nil {
		t.Fatalf("upsertShellRCBlock() error: %v", err)
	}
	if !created {
		t.Fatal("expected created=true for new file")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(got)
	for _, want := range []string{
		shellRCBlockBegin,
		"export ANTHROPIC_BASE_URL=https://api.moonshot.cn/anthropic",
		"export ANTHROPIC_AUTH_TOKEN=sk-test",
		"export ANTHROPIC_MODEL=kimi-k2.5",
		"export ENABLE_TOOL_SEARCH=false",
		shellRCBlockEnd,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in:\n%s", want, content)
		}
	}
}

func TestUpsertShellRCBlockReplacesExistingBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".zshrc")
	initial := "export PATH=/bin\n\n" + renderShellRCBlock(defaultKimiClaudeCodeEnv("old-key").shellExportLines()) + "alias ll='ls -la'\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := upsertShellRCBlock(path, defaultKimiClaudeCodeEnv("new-key").shellExportLines()); err != nil {
		t.Fatalf("upsertShellRCBlock() error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(got)
	if strings.Contains(content, "old-key") {
		t.Fatalf("expected old block replaced:\n%s", content)
	}
	if !strings.Contains(content, "new-key") {
		t.Fatalf("expected new key in block:\n%s", content)
	}
	if !strings.Contains(content, "export PATH=/bin") || !strings.Contains(content, "alias ll='ls -la'") {
		t.Fatalf("expected surrounding shell config preserved:\n%s", content)
	}
}

func TestShellExportValueQuotesSpecialChars(t *testing.T) {
	got := shellExportValue(`sk"test`)
	if got != `'sk"test'` {
		t.Fatalf("shellExportValue() = %q", got)
	}
}
