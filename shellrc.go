package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const shellRCBlockBegin = "# >>> home-agent-bootstrap kimi claude code >>>"
const shellRCBlockEnd = "# <<< home-agent-bootstrap kimi claude code <<<"

// KimiClaudeCodeEnv holds Moonshot/Kimi env vars for Claude Code per official docs.
type KimiClaudeCodeEnv struct {
	APIKey  string
	BaseURL string
	Model   string
}

func defaultKimiClaudeCodeEnv(apiKey string) KimiClaudeCodeEnv {
	model := "kimi-k2.5"
	return KimiClaudeCodeEnv{
		APIKey:  apiKey,
		BaseURL: "https://api.moonshot.cn/anthropic",
		Model:   model,
	}
}

func (e KimiClaudeCodeEnv) shellExportLines() []string {
	return []string{
		"# Kimi k2.5 via Moonshot Anthropic-compatible endpoint",
		"# https://platform.kimi.com/docs/guide/agent-support",
		fmt.Sprintf("export ANTHROPIC_BASE_URL=%s", shellExportValue(e.BaseURL)),
		fmt.Sprintf("export ANTHROPIC_AUTH_TOKEN=%s", shellExportValue(e.APIKey)),
		fmt.Sprintf("export ANTHROPIC_MODEL=%s", shellExportValue(e.Model)),
		fmt.Sprintf("export ANTHROPIC_DEFAULT_OPUS_MODEL=%s", shellExportValue(e.Model)),
		fmt.Sprintf("export ANTHROPIC_DEFAULT_SONNET_MODEL=%s", shellExportValue(e.Model)),
		fmt.Sprintf("export ANTHROPIC_DEFAULT_HAIKU_MODEL=%s", shellExportValue(e.Model)),
		fmt.Sprintf("export CLAUDE_CODE_SUBAGENT_MODEL=%s", shellExportValue(e.Model)),
		"export ENABLE_TOOL_SEARCH=false",
	}
}

func shellExportValue(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, " \t$`\"'\\") {
		return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
	}
	return value
}

func renderShellRCBlock(lines []string) string {
	var b strings.Builder
	b.WriteString(shellRCBlockBegin)
	b.WriteByte('\n')
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString(shellRCBlockEnd)
	b.WriteByte('\n')
	return b.String()
}

func upsertShellRCBlock(path string, innerLines []string) (created bool, err error) {
	block := renderShellRCBlock(innerLines)
	content := ""
	if data, readErr := os.ReadFile(path); readErr == nil {
		content = string(data)
	} else if !os.IsNotExist(readErr) {
		return false, readErr
	} else {
		created = true
	}

	begin := strings.Index(content, shellRCBlockBegin)
	end := strings.Index(content, shellRCBlockEnd)
	var updated string
	if begin >= 0 && end >= 0 && end > begin {
		end += len(shellRCBlockEnd)
		for end < len(content) && content[end] == '\n' {
			end++
		}
		updated = content[:begin] + block + content[end:]
	} else {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		updated = content + block
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return created, err
	}
	return created, os.WriteFile(path, []byte(updated), 0o600)
}

func defaultZshrcPath() string {
	return filepath.Join(homeDir(), ".zshrc")
}
