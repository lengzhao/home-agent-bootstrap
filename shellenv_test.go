package main

import (
	"strings"
	"testing"
)

func TestBuildClaudeCodeExportBlockKimi(t *testing.T) {
	profile := kimiClaudeCodeProfile("sk-kimi", "", "")
	block := buildClaudeCodeExportBlock(profile)

	for _, want := range []string{
		zshrcMarkerStart,
		"export ANTHROPIC_BASE_URL='https://api.moonshot.cn/anthropic'",
		"export ANTHROPIC_AUTH_TOKEN='sk-kimi'",
		"export ANTHROPIC_MODEL='kimi-k2.5'",
		"export ENABLE_TOOL_SEARCH='false'",
		zshrcMarkerEnd,
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("block missing %q:\n%s", want, block)
		}
	}
}

func TestReplaceMarkedBlockUpdatesExisting(t *testing.T) {
	existing := "# old\n" + zshrcMarkerStart + "\nexport ANTHROPIC_MODEL='old'\n" + zshrcMarkerEnd + "\n"
	newBlock := buildClaudeCodeExportBlock(openrouterClaudeCodeProfile("sk-or", "new-model", ""))
	got := replaceMarkedBlock(existing, newBlock)

	if strings.Contains(got, "old-model") || strings.Contains(got, "ANTHROPIC_MODEL='old'") {
		t.Fatalf("old block should be replaced:\n%s", got)
	}
	if !strings.Contains(got, "new-model") {
		t.Fatalf("new block missing:\n%s", got)
	}
	if strings.Count(got, zshrcMarkerStart) != 1 {
		t.Fatalf("expected single marker block:\n%s", got)
	}
}

func TestShellProfilePathUsesBashrcForBash(t *testing.T) {
	t.Setenv("HOME", "/Users/me")
	t.Setenv("SHELL", "/bin/bash")

	got := shellProfilePath()

	if got != "/Users/me/.bashrc" {
		t.Fatalf("shellProfilePath() = %q, want /Users/me/.bashrc", got)
	}
}

func TestShellProfilePathDefaultsToZshrc(t *testing.T) {
	t.Setenv("HOME", "/Users/me")
	t.Setenv("SHELL", "")

	got := shellProfilePath()

	if got != "/Users/me/.zshrc" {
		t.Fatalf("shellProfilePath() = %q, want /Users/me/.zshrc", got)
	}
}
