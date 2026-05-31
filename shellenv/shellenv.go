package shellenv

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
)

const (
	zshrcMarkerStart = "# >>> home-agent-bootstrap claude-code >>>"
	zshrcMarkerEnd   = "# <<< home-agent-bootstrap claude-code <<<"
)

type ClaudeCodeProfile struct {
	Label string
	Vars  map[string]string
}

func ProfilePath() string {
	switch filepath.Base(os.Getenv("SHELL")) {
	case "bash":
		return filepath.Join(cmdutil.HomeDir(), ".bashrc")
	case "zsh", "":
		return filepath.Join(cmdutil.HomeDir(), ".zshrc")
	default:
		return filepath.Join(cmdutil.HomeDir(), ".zshrc")
	}
}

func shellQuote(value string) string {
	value = strings.ReplaceAll(value, `'`, `'"'"'`)
	return `'` + value + `'`
}

func BuildExportBlock(profile ClaudeCodeProfile) string {
	keys := make([]string, 0, len(profile.Vars))
	for key := range profile.Vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := []string{
		zshrcMarkerStart,
		"# Managed by home-agent-bootstrap. " + profile.Label,
	}
	for _, key := range keys {
		value := profile.Vars[key]
		if value == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("export %s=%s", key, shellQuote(value)))
	}
	lines = append(lines, zshrcMarkerEnd)
	return strings.Join(lines, "\n")
}

func UpsertClaudeCodeEnv(profile ClaudeCodeProfile) (string, error) {
	path := ProfilePath()
	existing, err := os.ReadFile(path)
	content := ""
	if err == nil {
		content = string(existing)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	block := BuildExportBlock(profile)
	updated := replaceMarkedBlock(content, block)
	if !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	if err := cmdutil.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func replaceMarkedBlock(content, block string) string {
	start := strings.Index(content, zshrcMarkerStart)
	end := strings.Index(content, zshrcMarkerEnd)
	if start >= 0 && end > start {
		end += len(zshrcMarkerEnd)
		for end < len(content) && content[end] == '\n' {
			end++
		}
		return strings.TrimRight(content[:start], "\n") + "\n\n" + block + "\n"
	}
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return block + "\n"
	}
	return content + "\n\n" + block + "\n"
}

func claudeCodeEnvWithModel(apiKey, baseURL, model string, extras map[string]string) map[string]string {
	vars := map[string]string{
		"ANTHROPIC_BASE_URL":   baseURL,
		"ANTHROPIC_AUTH_TOKEN": apiKey,
		"ANTHROPIC_MODEL":      model,
	}
	for key, value := range extras {
		vars[key] = value
	}
	return vars
}

func KimiProfile(apiKey, model, baseURL string) ClaudeCodeProfile {
	if baseURL == "" {
		baseURL = "https://api.moonshot.cn/anthropic"
	}
	if model == "" {
		model = "kimi-k2.5"
	}
	extras := map[string]string{
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   model,
		"ANTHROPIC_DEFAULT_SONNET_MODEL": model,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  model,
		"CLAUDE_CODE_SUBAGENT_MODEL":     model,
		"ENABLE_TOOL_SEARCH":             "false",
	}
	return ClaudeCodeProfile{
		Label: "Kimi k2.5 via Moonshot Anthropic-compatible API",
		Vars:  claudeCodeEnvWithModel(apiKey, baseURL, model, extras),
	}
}

func OpenRouterProfile(apiKey, model, baseURL string) ClaudeCodeProfile {
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	if model == "" {
		model = "anthropic/claude-sonnet-4"
	}
	return ClaudeCodeProfile{
		Label: "OpenRouter via Anthropic-compatible API",
		Vars:  claudeCodeEnvWithModel(apiKey, baseURL, model, nil),
	}
}

func OpenAIProfile(apiKey, model, baseURL string) ClaudeCodeProfile {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4.1"
	}
	return ClaudeCodeProfile{
		Label: "OpenAI-compatible API for Claude Code",
		Vars:  claudeCodeEnvWithModel(apiKey, baseURL, model, nil),
	}
}

func VolcengineProfile(apiKey, model, baseURL string) ClaudeCodeProfile {
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}
	return ClaudeCodeProfile{
		Label: "Volcengine Ark OpenAI-compatible API for Claude Code",
		Vars:  claudeCodeEnvWithModel(apiKey, baseURL, model, nil),
	}
}

func QwenProfile(apiKey, model, baseURL string) ClaudeCodeProfile {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if model == "" {
		model = "qwen-plus"
	}
	return ClaudeCodeProfile{
		Label: "Qwen DashScope compatible API for Claude Code",
		Vars:  claudeCodeEnvWithModel(apiKey, baseURL, model, nil),
	}
}

func CustomProfile(apiKey, model, baseURL string) ClaudeCodeProfile {
	return ClaudeCodeProfile{
		Label: "Custom OpenAI-compatible API for Claude Code",
		Vars:  claudeCodeEnvWithModel(apiKey, baseURL, model, nil),
	}
}

func ConfigureClaudeCodeEnv(out io.Writer, profile ClaudeCodeProfile, docURL string) error {
	path, err := UpsertClaudeCodeEnv(profile)
	if err != nil {
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	cmdutil.Say("已写入 Claude Code 环境变量到 " + path)
	fmt.Fprintf(out, "请执行 source %s 或重新打开终端，再运行 claude。\n", path)
	fmt.Fprintln(out, "在 Claude Code 中可用 /status 确认模型。")
	if docURL != "" {
		fmt.Fprintf(out, "参考 %s\n", docURL)
	}
	return nil
}
