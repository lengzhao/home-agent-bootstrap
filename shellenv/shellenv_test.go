package shellenv

import (
	"strings"
	"testing"
)

func TestOpenRouterProfile(t *testing.T) {
	profile := OpenRouterProfile("sk-or", "", "")
	block := BuildExportBlock(profile)

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

func TestReplaceMarkedBlockUpdatesExisting(t *testing.T) {
	content := "# old\n# >>> home-agent-bootstrap claude-code >>>\nexport OLD='1'\n# <<< home-agent-bootstrap claude-code <<<\n"
	block := "# >>> home-agent-bootstrap claude-code >>>\nexport NEW='2'\n# <<< home-agent-bootstrap claude-code <<<"
	got := replaceMarkedBlock(content, block)
	if strings.Contains(got, "export OLD='1'") {
		t.Fatalf("expected old block replaced:\n%s", got)
	}
	if !strings.Contains(got, "export NEW='2'") {
		t.Fatalf("expected new block present:\n%s", got)
	}
}

func TestProfilePathDefaultsToZshrc(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	if !strings.HasSuffix(ProfilePath(), ".zshrc") {
		t.Fatalf("ProfilePath() = %q, want zshrc", ProfilePath())
	}
}

func TestProfilePathUsesBashrc(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	if !strings.HasSuffix(ProfilePath(), ".bashrc") {
		t.Fatalf("ProfilePath() = %q, want bashrc", ProfilePath())
	}
}
