package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const minCCConnectVersion = "1.3.0"

const (
	defaultProjectName    = "home"
	defaultAgentType      = "claudecode"
	defaultPlatformChoice = "7"
	defaultLLMChoice      = "1"
)

func defaultConfigPath() string {
	return filepath.Join(homeDir(), ".cc-connect", "config.toml")
}

func defaultWorkspacePath() string {
	return filepath.Join(homeDir(), "home-assistant-workspace")
}

func defaultAgentMode(agentType string) string {
	if agentType == "cursor" {
		return "default"
	}
	return "auto"
}

func defaultBootstrapSettings() bootstrapSettings {
	agentType := envDefault("AGENT_TYPE", defaultAgentType)
	return bootstrapSettings{
		ConfigPath:         envDefault("CONFIG_PATH", defaultConfigPath()),
		Workspace:          envDefault("WORKSPACE", defaultWorkspacePath()),
		ProjectName:        envDefault("PROJECT_NAME", defaultProjectName),
		AgentType:          agentType,
		AgentMode:          envDefault("AGENT_MODE", defaultAgentMode(agentType)),
		PermissionTemplate: envDefault("PERMISSION_TEMPLATE", defaultPermissionTemplate),
		PlatformChoices:    envDefault("PLATFORM_CHOICES", defaultPlatformChoice),
		LLMChoice:          envDefault("LLM_CHOICE", defaultLLMChoice),
		SkipWeixinSetup:    os.Getenv("SKIP_WEIXIN_SETUP") == "1",
		OverwriteConfig:    os.Getenv("OVERWRITE_CONFIG") == "1",
	}
}

func permissionTemplateChoiceDefault() string {
	id := strings.TrimSpace(os.Getenv("PERMISSION_TEMPLATE"))
	if id == "" {
		return "3"
	}
	for i, preset := range permissionTemplates {
		if preset.ID == id {
			return strconv.Itoa(i + 1)
		}
	}
	return "3"
}

func agentTypeChoiceDefault(agentType string) string {
	if agentType == "cursor" {
		return "2"
	}
	return "1"
}

func printBootstrapQuickGuide(out io.Writer) {
	defaults := defaultBootstrapSettings()
	fmt.Fprintln(out, "\n交互模式下，大部分问题直接回车即可使用方括号内的默认值。")
	fmt.Fprintln(out, "完整环境变量与非交互示例见 home-agent-bootstrap help")
	fmt.Fprintln(out, "\n当前默认配置：")
	fmt.Fprintf(out, "  配置文件   %s\n", defaults.ConfigPath)
	fmt.Fprintf(out, "  工作目录   %s\n", defaults.Workspace)
	fmt.Fprintf(out, "  project    %s\n", defaults.ProjectName)
	fmt.Fprintf(out, "  Agent      %s (%s)\n", defaults.AgentType, defaults.AgentMode)
	fmt.Fprintf(out, "  权限模板   %s\n", defaults.PermissionTemplate)
	fmt.Fprintf(out, "  平台序号   %s (微信个人号)\n", defaults.PlatformChoices)
	fmt.Fprintf(out, "  LLM 选项   %s (Claude Code 自带登录)\n", defaults.LLMChoice)
}

type bootstrapSettings struct {
	ConfigPath         string
	Workspace          string
	ProjectName        string
	AgentType          string
	AgentMode          string
	PermissionTemplate string
	PlatformChoices    string
	LLMChoice          string
	SkipWeixinSetup    bool
	OverwriteConfig    bool
}

func nonInteractive() bool {
	return os.Getenv("NONINTERACTIVE") == "1" || os.Getenv("BOOTSTRAP_YES") == "1"
}

func loadBootstrapSettings(p *prompt) bootstrapSettings {
	settings := defaultBootstrapSettings()

	if nonInteractive() {
		if settings.AgentMode == "" {
			settings.AgentMode = defaultAgentMode(settings.AgentType)
		}
		return settings
	}

	settings.ConfigPath = p.ask("cc-connect 配置文件路径", settings.ConfigPath)
	settings.Workspace = p.ask("家庭助手工作目录", settings.Workspace)
	settings.ProjectName = p.ask("cc-connect project 名称", settings.ProjectName)

	fmt.Fprintln(p.out, "\n选择运行时 Agent：")
	fmt.Fprintln(p.out, "1) Claude Code，推荐用于家庭助手运行时")
	fmt.Fprintln(p.out, "2) Cursor Agent，适合只读/规划或开发维护")
	agentChoice := p.askAllowed("请选择", agentTypeChoiceDefault(settings.AgentType), []string{"1", "2"})

	settings.AgentType = "claudecode"
	settings.AgentMode = defaultAgentMode("claudecode")
	if agentChoice == "2" {
		settings.AgentType = "cursor"
		settings.AgentMode = p.askAllowed("Cursor Agent 默认权限模式", envDefault("AGENT_MODE", "default"), []string{"ask", "plan", "default", "force"})
	} else {
		printClaudeCodeModeHelp()
		settings.AgentMode = p.askAllowed("Claude Code 默认权限模式", envDefault("AGENT_MODE", "auto"), []string{"default", "plan", "auto", "acceptEdits"})
	}

	printPermissionTemplateCatalog(p.out)
	permissionChoice := p.ask("选择权限模板序号", permissionTemplateChoiceDefault())
	if preset, err := parsePermissionTemplateChoice(permissionChoice); err != nil {
		warn(err.Error() + "，使用 family-remind")
		settings.PermissionTemplate = defaultPermissionTemplate
	} else {
		settings.PermissionTemplate = preset.ID
	}

	return settings
}

func choosePlatforms(p *prompt, settings bootstrapSettings) ([]PlatformBlock, error) {
	if nonInteractive() {
		presets, err := parsePlatformChoices(settings.PlatformChoices)
		if err != nil {
			return nil, err
		}
		return configurePlatformsFromPresetsNonInteractive(presets)
	}
	return configurePlatforms(p)
}

func configurePlatformsFromPresetsNonInteractive(presets []PlatformPreset) ([]PlatformBlock, error) {
	blocks := make([]PlatformBlock, 0)
	for _, preset := range presets {
		switch preset.Type {
		case "weixin":
			count := 1
			if raw := strings.TrimSpace(os.Getenv("WEIXIN_COUNT")); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < 1 {
					return nil, fmt.Errorf("WEIXIN_COUNT 必须是大于 0 的数字")
				}
				count = n
			}
			for i := 1; i <= count; i++ {
				accountID := fmt.Sprintf("wx-%d", i)
				if i == 1 {
					if v := strings.TrimSpace(os.Getenv("WEIXIN_ACCOUNT_ID")); v != "" {
						accountID = v
					}
				}
				allowFrom := envDefault("ADMIN_FROM", "")
				blocks = append(blocks, PlatformBlock{
					Type: "weixin",
					Options: []PlatformOption{
						{Key: "token", Value: ""},
						{Key: "base_url", Value: "https://ilinkai.weixin.qq.com"},
						{Key: "cdn_base_url", Value: "https://novac2c.cdn.weixin.qq.com/c2c"},
						{Key: "allow_from", Value: allowFrom},
						{Key: "account_id", Value: accountID},
						{Key: "long_poll_timeout_ms", Value: "35000"},
					},
				})
			}
		default:
			block, err := configureGenericPlatformNonInteractive(preset)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func configureGenericPlatformNonInteractive(preset PlatformPreset) (PlatformBlock, error) {
	options := make([]PlatformOption, 0, len(preset.Fields))
	for _, field := range preset.Fields {
		value := envDefault(strings.ToUpper(preset.Type+"_"+field.Key), field.Default)
		options = append(options, PlatformOption{Key: field.Key, Value: value})
	}
	return PlatformBlock{Type: preset.Type, Options: options}, nil
}

func chooseLLM(p *prompt, settings bootstrapSettings, agentType string) ProviderConfig {
	if nonInteractive() {
		return configureLLMFromChoice(p, agentType, settings.LLMChoice)
	}
	return configureLLM(p, agentType)
}

func configureLLMFromChoice(p *prompt, agentType, choice string) ProviderConfig {
	if agentType == "cursor" {
		return ProviderConfig{}
	}
	original := choice
	switch choice {
	case "1":
		return ProviderConfig{}
	case "2":
		key := envDefault("ANTHROPIC_API_KEY", "")
		if key == "" {
			warn("NONINTERACTIVE 模式下未设置 ANTHROPIC_API_KEY，跳过 Provider 写入")
			return ProviderConfig{}
		}
		return ProviderConfig{Name: "anthropic", APIKey: key}
	case "3":
		if preset, ok := providerPresetByName("openai"); ok {
			provider, err := configureClaudeCodeProviderFromPresetNonInteractive(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "4":
		if preset, ok := providerPresetByName("openrouter"); ok {
			provider, err := configureClaudeCodeProviderFromPresetNonInteractive(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "5":
		if preset, ok := providerPresetByName("kimi"); ok {
			provider, err := configureClaudeCodeProviderFromPresetNonInteractive(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "6":
		if preset, ok := providerPresetByName("volcengine"); ok {
			provider, err := configureClaudeCodeProviderFromPresetNonInteractive(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "7":
		if preset, ok := providerPresetByName("qwen"); ok {
			provider, err := configureClaudeCodeProviderFromPresetNonInteractive(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "8":
		key := envDefault("LLM_API_KEY", "")
		if key == "" {
			warn("NONINTERACTIVE 模式下未设置 LLM_API_KEY，跳过 Provider 写入")
			return ProviderConfig{}
		}
		baseURL := envDefault("LLM_BASE_URL", "")
		model := envDefault("LLM_MODEL", "")
		profile := customClaudeCodeProfile(key, model, baseURL)
		if err := configureClaudeCodeShellEnv(p, profile, ""); err != nil {
			warn(err.Error())
		}
		return ProviderConfig{Name: "custom", APIKey: key, BaseURL: baseURL, Model: model}
	case "9":
		return ProviderConfig{}
	default:
		warn("无效 LLM_CHOICE=" + original + "，跳过 LLM 配置")
	}
	return ProviderConfig{}
}

func configureClaudeCodeProviderFromPresetNonInteractive(p *prompt, preset ProviderPreset) (ProviderConfig, error) {
	key := envDefault("LLM_API_KEY", "")
	if key == "" {
		return ProviderConfig{}, fmt.Errorf("NONINTERACTIVE 模式下未设置 LLM_API_KEY")
	}
	baseURL := envDefault("LLM_BASE_URL", preset.DefaultBaseURL)
	model := envDefault("LLM_MODEL", preset.DefaultModel)
	profile := preset.BuildShellProfile(key, model, baseURL)
	cfg := ProviderConfig{Name: preset.Name, APIKey: key, BaseURL: baseURL, Model: model}
	if err := configureClaudeCodeShellEnv(p, profile, ""); err != nil {
		return cfg, err
	}
	return cfg, nil
}
