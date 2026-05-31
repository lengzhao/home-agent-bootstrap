package providers

import (
	"fmt"
	"os"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/prompt"
	"github.com/lengzhao/home-agent-bootstrap/shellenv"
)

type Config struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
}

type Preset struct {
	Name              string
	DisplayName       string
	DefaultBaseURL    string
	DefaultModel      string
	BuildShellProfile func(apiKey, model, baseURL string) shellenv.ClaudeCodeProfile
}

var presets = []Preset{
	{
		Name: "openai", DisplayName: "OpenAI",
		DefaultBaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-4.1",
		BuildShellProfile: shellenv.OpenAIProfile,
	},
	{
		Name: "openrouter", DisplayName: "OpenRouter",
		DefaultBaseURL: "https://openrouter.ai/api/v1", DefaultModel: "anthropic/claude-sonnet-4",
		BuildShellProfile: shellenv.OpenRouterProfile,
	},
	{
		Name: "kimi", DisplayName: "Kimi (Moonshot)",
		DefaultBaseURL: "https://api.moonshot.cn/anthropic", DefaultModel: "kimi-k2.5",
		BuildShellProfile: shellenv.KimiProfile,
	},
	{
		Name: "volcengine", DisplayName: "火山引擎 (豆包)",
		DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3", DefaultModel: "",
		BuildShellProfile: shellenv.VolcengineProfile,
	},
	{
		Name: "qwen", DisplayName: "通义千问 (DashScope)",
		DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", DefaultModel: "qwen-plus",
		BuildShellProfile: shellenv.QwenProfile,
	},
}

func PresetByName(name string) (Preset, bool) {
	for _, preset := range presets {
		if preset.Name == name {
			return preset, true
		}
	}
	return Preset{}, false
}

func ConfigureFromPreset(p *prompt.Prompt, preset Preset) (Config, error) {
	keyLabel := fmt.Sprintf("请输入 %s API Key，将写入 config.toml，并同步写入 shell 配置文件供直接运行 claude 使用", preset.DisplayName)
	key := p.AskSecret(keyLabel)
	if key == "" {
		cmdutil.Warn("API Key 为空，跳过 Provider 和环境变量配置")
		return Config{}, nil
	}
	baseURL := preset.DefaultBaseURL
	if preset.DefaultBaseURL != "" {
		baseURL = p.Ask(preset.DisplayName+" ANTHROPIC_BASE_URL", preset.DefaultBaseURL)
	}
	model := ""
	if preset.DefaultModel != "" || preset.Name == "volcengine" {
		model = p.Ask(preset.DisplayName+" 模型 ID", preset.DefaultModel)
	}
	profile := preset.BuildShellProfile(key, model, baseURL)
	docURL := ""
	if preset.Name == "kimi" {
		docURL = "https://platform.kimi.com/docs/guide/agent-support"
	}
	cfg := Config{Name: preset.Name, APIKey: key, BaseURL: baseURL, Model: model}
	if err := shellenv.ConfigureClaudeCodeEnv(p.Out, profile, docURL); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ConfigureFromPresetNonInteractive(p *prompt.Prompt, preset Preset) (Config, error) {
	key := cmdutil.EnvDefault("LLM_API_KEY", "")
	if key == "" {
		return Config{}, fmt.Errorf("NONINTERACTIVE 模式下未设置 LLM_API_KEY")
	}
	baseURL := cmdutil.EnvDefault("LLM_BASE_URL", preset.DefaultBaseURL)
	model := cmdutil.EnvDefault("LLM_MODEL", preset.DefaultModel)
	profile := preset.BuildShellProfile(key, model, baseURL)
	cfg := Config{Name: preset.Name, APIKey: key, BaseURL: baseURL, Model: model}
	if err := shellenv.ConfigureClaudeCodeEnv(p.Out, profile, ""); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ConfigureLLM(p *prompt.Prompt, agentType string) Config {
	if agentType == "cursor" {
		fmt.Fprintln(os.Stdout, "Cursor Agent 通常依赖 Cursor 账号登录。")
		if p.AskYesNo("是否现在运行 agent --help 验证 CLI 可用", true) && cmdutil.CommandExists("agent") {
			_ = cmdutil.RunCommand("agent", "--help")
		}
		return Config{}
	}

	fmt.Fprintln(os.Stdout, "选择 Claude Code 的 LLM 配置方式：")
	fmt.Fprintln(os.Stdout, "1) 使用 Claude Code 自带登录，现在启动 claude 完成登录/授权")
	fmt.Fprintln(os.Stdout, "2) Anthropic API Key")
	fmt.Fprintln(os.Stdout, "3) OpenAI（写入 config.toml Provider，并同步 shell 配置文件）")
	fmt.Fprintln(os.Stdout, "4) OpenRouter（写入 config.toml Provider，并同步 shell 配置文件）")
	fmt.Fprintln(os.Stdout, "5) Kimi (Moonshot)（写入 config.toml Provider，并同步 shell 配置文件）")
	fmt.Fprintln(os.Stdout, "6) 火山引擎 (豆包)（写入 config.toml Provider，并同步 shell 配置文件）")
	fmt.Fprintln(os.Stdout, "7) 通义千问 (DashScope)（写入 config.toml Provider，并同步 shell 配置文件）")
	fmt.Fprintln(os.Stdout, "8) 自定义 OpenAI-compatible（写入 config.toml Provider，并同步 shell 配置文件）")
	fmt.Fprintln(os.Stdout, "9) 暂不配置")
	choice := p.AskAllowed("请选择", "1", []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"})
	switch choice {
	case "1":
		fmt.Fprintln(os.Stdout, "稍后会在家庭助手工作目录启动 claude，用于完成登录和信任工作目录。")
	case "2":
		key := p.AskSecret("请输入 ANTHROPIC_API_KEY，本值只写入本机生成的 config.toml，不会进入仓库模板")
		if key != "" {
			return Config{Name: "anthropic", APIKey: key}
		}
		cmdutil.Warn("API Key 为空，跳过 Provider 写入")
	case "3":
		if preset, ok := PresetByName("openai"); ok {
			provider, err := ConfigureFromPreset(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "4":
		if preset, ok := PresetByName("openrouter"); ok {
			provider, err := ConfigureFromPreset(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "5":
		if preset, ok := PresetByName("kimi"); ok {
			provider, err := ConfigureFromPreset(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "6":
		if preset, ok := PresetByName("volcengine"); ok {
			provider, err := ConfigureFromPreset(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "7":
		if preset, ok := PresetByName("qwen"); ok {
			provider, err := ConfigureFromPreset(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "8":
		key := p.AskSecret("请输入 API Key，将写入 config.toml，并同步写入 shell 配置文件")
		if key != "" {
			baseURL := p.Ask("ANTHROPIC_BASE_URL（OpenAI-compatible 接口地址）", "")
			model := p.Ask("模型 ID", "")
			profile := shellenv.CustomProfile(key, model, baseURL)
			if err := shellenv.ConfigureClaudeCodeEnv(p.Out, profile, ""); err != nil {
				cmdutil.Warn(err.Error())
			}
			return Config{Name: "custom", APIKey: key, BaseURL: baseURL, Model: model}
		}
		cmdutil.Warn("API Key 为空，跳过 Provider 和环境变量配置")
	case "9":
		cmdutil.Warn("已跳过 LLM 配置。启动前请确保 claude 已登录或 provider 已配置。")
	}
	return Config{}
}

func ConfigureFromChoice(p *prompt.Prompt, agentType, choice string) Config {
	if agentType == "cursor" {
		return Config{}
	}
	original := choice
	switch choice {
	case "1":
		return Config{}
	case "2":
		key := cmdutil.EnvDefault("ANTHROPIC_API_KEY", "")
		if key == "" {
			cmdutil.Warn("NONINTERACTIVE 模式下未设置 ANTHROPIC_API_KEY，跳过 Provider 写入")
			return Config{}
		}
		return Config{Name: "anthropic", APIKey: key}
	case "3":
		if preset, ok := PresetByName("openai"); ok {
			provider, err := ConfigureFromPresetNonInteractive(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "4":
		if preset, ok := PresetByName("openrouter"); ok {
			provider, err := ConfigureFromPresetNonInteractive(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "5":
		if preset, ok := PresetByName("kimi"); ok {
			provider, err := ConfigureFromPresetNonInteractive(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "6":
		if preset, ok := PresetByName("volcengine"); ok {
			provider, err := ConfigureFromPresetNonInteractive(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "7":
		if preset, ok := PresetByName("qwen"); ok {
			provider, err := ConfigureFromPresetNonInteractive(p, preset)
			if err != nil {
				cmdutil.Warn(err.Error())
			}
			return provider
		}
	case "8":
		key := cmdutil.EnvDefault("LLM_API_KEY", "")
		if key == "" {
			cmdutil.Warn("NONINTERACTIVE 模式下未设置 LLM_API_KEY，跳过 Provider 写入")
			return Config{}
		}
		baseURL := cmdutil.EnvDefault("LLM_BASE_URL", "")
		model := cmdutil.EnvDefault("LLM_MODEL", "")
		profile := shellenv.CustomProfile(key, model, baseURL)
		if err := shellenv.ConfigureClaudeCodeEnv(p.Out, profile, ""); err != nil {
			cmdutil.Warn(err.Error())
		}
		return Config{Name: "custom", APIKey: key, BaseURL: baseURL, Model: model}
	case "9":
		return Config{}
	default:
		cmdutil.Warn("无效 LLM_CHOICE=" + original + "，跳过 LLM 配置")
	}
	return Config{}
}
