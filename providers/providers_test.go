package providers

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/lengzhao/home-agent-bootstrap/prompt"
)

func TestConfigureLLMReturnsProviderForClaudeCodePreset(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	input := strings.NewReader("3\nsk-openai\n\n\n")
	var out bytes.Buffer
	p := prompt.New(bufio.NewReader(input), &out, false)

	got := ConfigureLLM(p, "claudecode")

	if got.Name != "openai" {
		t.Fatalf("provider name = %q, want openai", got.Name)
	}
	if got.APIKey != "sk-openai" {
		t.Fatalf("provider api key = %q, want sk-openai", got.APIKey)
	}
	if got.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("provider base url = %q, want default OpenAI base URL", got.BaseURL)
	}
	if got.Model != "gpt-4.1" {
		t.Fatalf("provider model = %q, want gpt-4.1", got.Model)
	}
}
