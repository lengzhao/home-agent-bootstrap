package bootstrap

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/config"
)

const MinCCConnectVersion = "1.3.0"

func Run(args []string, templates fs.FS) error {
	cmd := "bootstrap"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "bootstrap":
		return RunBootstrap(templates)
	case "doctor":
		return RunDoctor()
	case "migrate-config":
		return RunMigrate()
	case "setup-weixin":
		return RunSetupWeixin(args)
	case "start":
		return RunStart()
	case "sync-workspace":
		return RunSyncWorkspace(templates, args)
	case "workspace-status":
		return RunWorkspaceStatus(templates)
	case "help", "-h", "--help":
		PrintUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("未知命令 %q，运行 %s help 查看用法", cmd, cmdutil.AppName)
	}
}

func PrintUsage(out io.Writer) {
	defaults := DefaultSettings()
	fmt.Fprintf(out, `%s - cc-connect 家庭超级助手引导器

用法:
  %s                 交互引导安装，默认命令，大部分问题直接回车即可
  %s bootstrap       同上
  %s doctor          检查本机依赖和 cc-connect 状态
  %s migrate-config  将旧版 [[providers]] 迁移到 [[projects.agent.providers]]
  %s setup-weixin N  按平台顺序扫码绑定 N 个微信个人号
  %s start           安装并启动 cc-connect daemon
  %s sync-workspace  补全工作区缺失模板文件，不覆盖已有内容
  %s workspace-status 查看工作区模板版本与缺失文件
  %s help            显示本帮助

交互模式默认配置（直接回车即可）:
  配置文件         %s
  工作目录         %s
  project 名称     %s
  Agent            %s，权限模式 %s
  权限模板         %s
  平台序号         %s（微信个人号）
  LLM 选项         %s（Claude Code 自带登录）
  微信个人号数量   1
  是否现在扫码     是

通用环境变量（交互和非交互均可预设，会作为问答默认值）:
  CONFIG_PATH          cc-connect 配置路径
                       默认 %s
  WORKSPACE            家庭助手工作目录，sync-workspace / workspace-status 也使用
                       默认 %s
  PROJECT_NAME         cc-connect project 名称，默认 %s
  AGENT_TYPE           claudecode 或 cursor，默认 %s
  AGENT_MODE           Agent 权限模式
                       Claude Code 默认 auto，Cursor 默认 default
  PERMISSION_TEMPLATE  admin-only / family-readonly / family-remind
                       默认 %s
  PLATFORM_CHOICES     平台序号，如 7 或 1,7，默认 %s
  LLM_CHOICE           Claude Code LLM 选项 1-9，默认 %s
  INSTALL_DEPS=0       跳过 Homebrew、Node、ffmpeg 等系统依赖安装

非交互模式（跳过全部问答）:
  NONINTERACTIVE=1     或 BOOTSTRAP_YES=1
  OVERWRITE_CONFIG=1   覆盖已有 config.toml，默认不覆盖
  SKIP_WEIXIN_SETUP=1  跳过微信扫码
  ADMIN_FROM           管理员 user_id，可留空稍后补充
  WEIXIN_COUNT         微信个人号数量，默认 1
  WEIXIN_ACCOUNT_ID    第一个微信 account_id，默认 wx-1

LLM Provider 环境变量（NONINTERACTIVE 且 LLM_CHOICE 对应时使用）:
  LLM_CHOICE=1         Claude Code 自带登录，无需额外变量
  LLM_CHOICE=2         需要 ANTHROPIC_API_KEY
  LLM_CHOICE=3-7       需要 LLM_API_KEY，可选 LLM_BASE_URL、LLM_MODEL
  LLM_CHOICE=8         自定义 OpenAI-compatible，需要 LLM_API_KEY
  LLM_CHOICE=9         暂不配置 LLM

非交互示例（微信个人号 + Claude Code 自带登录）:
  NONINTERACTIVE=1 SKIP_WEIXIN_SETUP=1 home-agent-bootstrap bootstrap

非交互示例（指定工作目录和 OpenAI Provider）:
  NONINTERACTIVE=1 \
    WORKSPACE="$HOME/home-assistant-workspace" \
    PERMISSION_TEMPLATE=family-remind \
    PLATFORM_CHOICES=7 \
    LLM_CHOICE=3 \
    LLM_API_KEY="sk-..." \
    SKIP_WEIXIN_SETUP=1 \
    home-agent-bootstrap bootstrap

更多说明见 docs/configuration.md
`, cmdutil.AppName,
		cmdutil.AppName, cmdutil.AppName, cmdutil.AppName, cmdutil.AppName, cmdutil.AppName, cmdutil.AppName, cmdutil.AppName, cmdutil.AppName, cmdutil.AppName,
		defaults.ConfigPath,
		defaults.Workspace,
		defaults.ProjectName,
		defaults.AgentType, defaults.AgentMode,
		defaults.PermissionTemplate,
		defaults.PlatformChoices,
		defaults.LLMChoice,
		defaults.ConfigPath,
		defaults.Workspace,
		defaults.ProjectName,
		defaults.AgentType,
		defaults.PermissionTemplate,
		defaults.PlatformChoices,
		defaults.LLMChoice,
	)
}

func RunMigrate() error {
	configPath := cmdutil.EnvDefault("CONFIG_PATH", cmdutil.DefaultConfigPath())
	return config.RunMigrate(configPath)
}
