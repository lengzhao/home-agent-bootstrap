package main

import "fmt"

// ProviderPreset is a built-in LLM provider choice for Claude Code runtime.
type ProviderPreset struct {
	Name              string
	DisplayName       string
	DefaultBaseURL    string
	DefaultModel      string
	ClaudeCodeShell   bool
	BuildShellProfile func(apiKey, model, baseURL string) ClaudeCodeShellProfile
}

var providerPresets = []ProviderPreset{
	{
		Name: "openai", DisplayName: "OpenAI",
		DefaultBaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-4.1",
		ClaudeCodeShell: true, BuildShellProfile: openaiClaudeCodeProfile,
	},
	{
		Name: "openrouter", DisplayName: "OpenRouter",
		DefaultBaseURL: "https://openrouter.ai/api/v1", DefaultModel: "anthropic/claude-sonnet-4",
		ClaudeCodeShell: true, BuildShellProfile: openrouterClaudeCodeProfile,
	},
	{
		Name: "kimi", DisplayName: "Kimi (Moonshot)",
		DefaultBaseURL: "https://api.moonshot.cn/anthropic", DefaultModel: "kimi-k2.5",
		ClaudeCodeShell: true, BuildShellProfile: kimiClaudeCodeProfile,
	},
	{
		Name: "volcengine", DisplayName: "火山引擎 (豆包)",
		DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3", DefaultModel: "",
		ClaudeCodeShell: true, BuildShellProfile: volcengineClaudeCodeProfile,
	},
	{
		Name: "qwen", DisplayName: "通义千问 (DashScope)",
		DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", DefaultModel: "qwen-plus",
		ClaudeCodeShell: true, BuildShellProfile: qwenClaudeCodeProfile,
	},
}

func providerPresetByName(name string) (ProviderPreset, bool) {
	for _, preset := range providerPresets {
		if preset.Name == name {
			return preset, true
		}
	}
	return ProviderPreset{}, false
}

func configureClaudeCodeShellFromPreset(p *prompt, preset ProviderPreset) error {
	_, err := configureClaudeCodeProviderFromPreset(p, preset)
	return err
}

func configureClaudeCodeProviderFromPreset(p *prompt, preset ProviderPreset) (ProviderConfig, error) {
	keyLabel := fmt.Sprintf("请输入 %s API Key，将写入 config.toml，并同步写入 ~/.zshrc 供直接运行 claude 使用", preset.DisplayName)
	key := p.askSecret(keyLabel)
	if key == "" {
		warn("API Key 为空，跳过 Provider 和环境变量配置")
		return ProviderConfig{}, nil
	}
	baseURL := preset.DefaultBaseURL
	if preset.DefaultBaseURL != "" {
		baseURL = p.ask(preset.DisplayName+" ANTHROPIC_BASE_URL", preset.DefaultBaseURL)
	}
	model := ""
	if preset.DefaultModel != "" || preset.Name == "volcengine" {
		model = p.ask(preset.DisplayName+" 模型 ID", preset.DefaultModel)
	}
	profile := preset.BuildShellProfile(key, model, baseURL)
	docURL := ""
	if preset.Name == "kimi" {
		docURL = "https://platform.kimi.com/docs/guide/agent-support"
	}
	cfg := ProviderConfig{
		Name:    preset.Name,
		APIKey:  key,
		BaseURL: baseURL,
		Model:   model,
	}
	if err := configureClaudeCodeShellEnv(p, profile, docURL); err != nil {
		return cfg, err
	}
	return cfg, nil
}
