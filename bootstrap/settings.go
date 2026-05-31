package bootstrap

import (
	"fmt"
	"io"
	"os"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/permissions"
	"github.com/lengzhao/home-agent-bootstrap/platforms"
	"github.com/lengzhao/home-agent-bootstrap/prompt"
	"github.com/lengzhao/home-agent-bootstrap/providers"
)

const (
	defaultProjectName = "home"
	defaultAgentType   = "claudecode"
	defaultLLMChoice   = "1"
)

type Settings struct {
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

func NonInteractive() bool {
	return os.Getenv("NONINTERACTIVE") == "1" || os.Getenv("BOOTSTRAP_YES") == "1"
}

func defaultAgentMode(agentType string) string {
	if agentType == "cursor" {
		return "default"
	}
	return "auto"
}

func DefaultSettings() Settings {
	agentType := cmdutil.EnvDefault("AGENT_TYPE", defaultAgentType)
	return Settings{
		ConfigPath:         cmdutil.EnvDefault("CONFIG_PATH", cmdutil.DefaultConfigPath()),
		Workspace:          cmdutil.EnvDefault("WORKSPACE", cmdutil.DefaultWorkspacePath()),
		ProjectName:        cmdutil.EnvDefault("PROJECT_NAME", defaultProjectName),
		AgentType:          agentType,
		AgentMode:          cmdutil.EnvDefault("AGENT_MODE", defaultAgentMode(agentType)),
		PermissionTemplate: cmdutil.EnvDefault("PERMISSION_TEMPLATE", permissions.DefaultTemplate),
		PlatformChoices:    cmdutil.EnvDefault("PLATFORM_CHOICES", platforms.DefaultPlatformChoice),
		LLMChoice:          cmdutil.EnvDefault("LLM_CHOICE", defaultLLMChoice),
		SkipWeixinSetup:    os.Getenv("SKIP_WEIXIN_SETUP") == "1",
		OverwriteConfig:    os.Getenv("OVERWRITE_CONFIG") == "1",
	}
}

func agentTypeChoiceDefault(agentType string) string {
	if agentType == "cursor" {
		return "2"
	}
	return "1"
}

func printQuickGuide(out io.Writer) {
	defaults := DefaultSettings()
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

func loadSettings(p *prompt.Prompt) Settings {
	settings := DefaultSettings()

	if NonInteractive() {
		if settings.AgentMode == "" {
			settings.AgentMode = defaultAgentMode(settings.AgentType)
		}
		return settings
	}

	settings.ConfigPath = p.Ask("cc-connect 配置文件路径", settings.ConfigPath)
	settings.Workspace = p.Ask("家庭助手工作目录", settings.Workspace)
	settings.ProjectName = p.Ask("cc-connect project 名称", settings.ProjectName)

	fmt.Fprintln(p.Out, "\n选择运行时 Agent：")
	fmt.Fprintln(p.Out, "1) Claude Code，推荐用于家庭助手运行时")
	fmt.Fprintln(p.Out, "2) Cursor Agent，适合只读/规划或开发维护")
	agentChoice := p.AskAllowed("请选择", agentTypeChoiceDefault(settings.AgentType), []string{"1", "2"})

	settings.AgentType = "claudecode"
	settings.AgentMode = defaultAgentMode("claudecode")
	if agentChoice == "2" {
		settings.AgentType = "cursor"
		settings.AgentMode = p.AskAllowed("Cursor Agent 默认权限模式", cmdutil.EnvDefault("AGENT_MODE", "default"), []string{"ask", "plan", "default", "force"})
	} else {
		printClaudeCodeModeHelp()
		settings.AgentMode = p.AskAllowed("Claude Code 默认权限模式", cmdutil.EnvDefault("AGENT_MODE", "auto"), []string{"default", "plan", "auto", "acceptEdits"})
	}

	permissions.PrintCatalog(p.Out)
	permissionChoice := p.Ask("选择权限模板序号", permissions.ChoiceDefaultFromEnv())
	if preset, err := permissions.ParseChoice(permissionChoice); err != nil {
		cmdutil.Warn(err.Error() + "，使用 family-remind")
		settings.PermissionTemplate = permissions.DefaultTemplate
	} else {
		settings.PermissionTemplate = preset.ID
	}

	return settings
}

func printClaudeCodeModeHelp() {
	fmt.Fprintln(os.Stdout, "Claude Code 权限模式说明：")
	fmt.Fprintln(os.Stdout, "- auto：推荐。自动执行低风险操作，适合可信本机家庭助手。")
	fmt.Fprintln(os.Stdout, "- default：执行工具前按 Claude Code 默认策略询问。")
	fmt.Fprintln(os.Stdout, "- plan：只读规划，不执行修改。")
	fmt.Fprintln(os.Stdout, "- acceptEdits：自动接受编辑，风险更高，首次部署不建议。")
}

func choosePlatforms(p *prompt.Prompt, settings Settings) ([]platforms.Block, error) {
	if NonInteractive() {
		presets, err := platforms.ParseChoices(settings.PlatformChoices)
		if err != nil {
			return nil, err
		}
		return platforms.BlocksFromPresetsNonInteractive(presets)
	}
	return platforms.Configure(p)
}

func chooseLLM(p *prompt.Prompt, settings Settings, agentType string) providers.Config {
	if NonInteractive() {
		return providers.ConfigureFromChoice(p, agentType, settings.LLMChoice)
	}
	return providers.ConfigureLLM(p, agentType)
}
