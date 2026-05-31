package bootstrap

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/config"
	"github.com/lengzhao/home-agent-bootstrap/permissions"
	"github.com/lengzhao/home-agent-bootstrap/platforms"
	"github.com/lengzhao/home-agent-bootstrap/prompt"
	"github.com/lengzhao/home-agent-bootstrap/workspacesync"
)

func RunBootstrap(templates fs.FS) error {
	p := prompt.New(bufio.NewReader(os.Stdin), os.Stdout, NonInteractive())

	cmdutil.Say("开始配置个人家庭超级助手")
	if !NonInteractive() {
		printQuickGuide(p.Out)
	}

	if os.Getenv("INSTALL_DEPS") != "0" {
		if err := installXcodeCLTIfNeeded(p); err != nil {
			return err
		}
		if err := installHomebrewIfNeeded(p); err != nil {
			return err
		}
		if err := installBasePackagesIfNeeded(p); err != nil {
			return err
		}
	} else {
		cmdutil.Warn("INSTALL_DEPS=0，跳过系统依赖安装")
	}

	if err := installCCConnectIfNeeded(p); err != nil {
		return err
	}

	settings := loadSettings(p)
	if err := config.ValidateAgentMode(settings.AgentType, settings.AgentMode); err != nil {
		return err
	}

	if err := installAgentIfNeeded(p, settings.AgentType); err != nil {
		return err
	}
	provider := chooseLLM(p, settings, settings.AgentType)

	platformBlocks, err := choosePlatforms(p, settings)
	if err != nil {
		return err
	}

	adminFrom := ""
	if platforms.HasWeixin(platformBlocks) {
		if NonInteractive() {
			adminFrom = cmdutil.EnvDefault("ADMIN_FROM", "")
		} else {
			adminFrom = p.Ask("管理员微信 ilink user_id，未知可先留空，扫码后再修改", "")
		}
		if adminFrom == "" {
			cmdutil.Warn("admin_from 为空时，特权命令不会授予任何用户。扫码后请用 /whoami 获取 user_id 并补充配置。")
		}
	} else {
		if NonInteractive() {
			adminFrom = cmdutil.EnvDefault("ADMIN_FROM", "")
		} else {
			adminFrom = p.Ask("管理员 user_id（平台相关），未知可先留空", "")
		}
		if adminFrom == "" {
			cmdutil.Warn("admin_from 为空时，特权命令不会授予任何用户。请在各平台完成首次对话后用 /whoami 获取 id 并补充配置。")
		}
	}

	configPath := settings.ConfigPath
	if cmdutil.Exists(configPath) {
		overwrite := settings.OverwriteConfig
		if !NonInteractive() {
			overwrite = p.AskYesNo("配置文件已存在，是否备份并覆盖", false)
		}
		if !overwrite {
			return errors.New("已取消，未覆盖现有配置")
		}
		if err := cmdutil.BackupFile(configPath); err != nil {
			return err
		}
	}

	report, err := workspacesync.Sync(templates, settings.Workspace, workspacesync.Options{})
	if err != nil {
		return err
	}
	workspacesync.PrintReport(report)

	if settings.AgentType == "claudecode" && !NonInteractive() {
		initializeClaudeCodeWorkspace(settings.Workspace)
	}

	cfg := config.RenderInput{
		ConfigPath:             configPath,
		DataDir:                filepath.Join(cmdutil.HomeDir(), ".cc-connect"),
		Workspace:              settings.Workspace,
		ProjectName:            settings.ProjectName,
		AgentType:              settings.AgentType,
		AgentMode:              settings.AgentMode,
		ManagementToken:        cmdutil.MustRandomToken(),
		BridgeToken:            cmdutil.MustRandomToken(),
		WebhookToken:           cmdutil.MustRandomToken(),
		AdminFrom:              adminFrom,
		ProviderName:           provider.Name,
		ProviderAPIKey:         provider.APIKey,
		ProviderBaseURL:        provider.BaseURL,
		ProviderModel:          provider.Model,
		Platforms:              platformBlocks,
		PermissionTemplate:     settings.PermissionTemplate,
		MemberDisabledCommands: permissions.MemberDisabledCommands(settings.PermissionTemplate),
	}
	rendered, err := config.Render(templates, cfg)
	if err != nil {
		return err
	}
	if err := cmdutil.WriteFile(configPath, []byte(rendered), 0o600); err != nil {
		return err
	}

	weixinCount := platforms.CountWeixin(platformBlocks)
	weixinSetupDone := false
	setupWeixin := weixinCount > 0 && !settings.SkipWeixinSetup
	if !NonInteractive() && weixinCount > 0 {
		setupWeixin = p.AskYesNo("是否现在逐个扫码绑定微信个人号", true)
	}
	if setupWeixin {
		if err := setupWeixinWithConfig(configPath, settings.ProjectName, weixinCount); err != nil {
			return err
		}
		if err := completeAdminRoleAfterWeixinSetup(p, configPath); err != nil {
			return err
		}
		weixinSetupDone = true
	}

	printNextSteps(configPath, settings.ProjectName, platformBlocks, settings.AgentType, weixinSetupDone)
	return nil
}

func initializeClaudeCodeWorkspace(workspace string) {
	if !cmdutil.CommandExists("claude") {
		cmdutil.Warn("未检测到 claude，安装完成后请在家庭助手工作目录运行 claude 完成登录和信任。")
		return
	}
	fmt.Fprintf(os.Stdout, "\n将在家庭助手工作目录启动 Claude Code：%s\n", workspace)
	fmt.Fprintln(os.Stdout, "请在 Claude Code 里完成登录、信任工作目录。完成后退出 Claude Code，bootstrap 会继续。")
	if err := cmdutil.RunCommandInDir(workspace, "claude"); err != nil {
		cmdutil.Warn("Claude Code 初始化未完成。之后请在家庭助手工作目录手动运行 claude。")
	}
}
