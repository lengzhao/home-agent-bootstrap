package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/config"
	"github.com/lengzhao/home-agent-bootstrap/prompt"
	"github.com/lengzhao/home-agent-bootstrap/workspacesync"
)

func installXcodeCLTIfNeeded(p *prompt.Prompt) error {
	if exec.Command("xcode-select", "-p").Run() == nil {
		cmdutil.Say("已检测到 Xcode Command Line Tools")
		return nil
	}
	cmdutil.Warn("未检测到 Xcode Command Line Tools")
	if !p.AskYesNo("是否现在安装 Xcode Command Line Tools", true) {
		return nil
	}
	if err := cmdutil.RunCommand("xcode-select", "--install"); err != nil {
		return fmt.Errorf("安装 Xcode Command Line Tools 失败: %w", err)
	}
	fmt.Fprintln(os.Stdout, "请在弹窗中完成安装。安装完成后回到终端按回车继续。")
	_, _ = p.In.ReadString('\n')
	return nil
}

func installHomebrewIfNeeded(p *prompt.Prompt) error {
	cmdutil.AddHomebrewToPath()
	if cmdutil.CommandExists("brew") {
		cmdutil.Say("已检测到 Homebrew")
		return nil
	}
	cmdutil.Warn("未检测到 Homebrew")
	if !p.AskYesNo("是否现在安装 Homebrew", true) {
		return nil
	}
	if err := cmdutil.RunCommand("/bin/bash", "-c", `$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`); err != nil {
		return fmt.Errorf("安装 Homebrew 失败: %w", err)
	}
	cmdutil.AddHomebrewToPath()
	if !cmdutil.CommandExists("brew") {
		return errors.New("Homebrew 安装后仍未检测到 brew 命令")
	}
	return nil
}

func installBasePackagesIfNeeded(p *prompt.Prompt) error {
	if !cmdutil.CommandExists("brew") {
		cmdutil.Warn("未检测到 brew，跳过基础包自动安装")
		return nil
	}
	if (!cmdutil.CommandExists("node") || !cmdutil.CommandExists("npm")) && p.AskYesNo("是否使用 brew 安装 Node.js/npm", true) {
		if err := cmdutil.RunCommand("brew", "install", "node"); err != nil {
			return fmt.Errorf("安装 Node.js/npm 失败: %w", err)
		}
	}
	if !cmdutil.CommandExists("ffmpeg") && p.AskYesNo("是否使用 brew 安装 ffmpeg，用于微信语音转写", true) {
		if err := cmdutil.RunCommand("brew", "install", "ffmpeg"); err != nil {
			return fmt.Errorf("安装 ffmpeg 失败: %w", err)
		}
	}
	return nil
}

func installCCConnectIfNeeded(p *prompt.Prompt) error {
	if cmdutil.CommandExists("cc-connect") {
		cmdutil.Say("已检测到 cc-connect")
		return nil
	}
	if NonInteractive() {
		return errors.New("未检测到 cc-connect，NONINTERACTIVE 模式下请先手动安装 cc-connect")
	}
	if !p.AskYesNo("是否尝试使用 npm 全局安装 cc-connect", true) {
		return nil
	}
	if !cmdutil.CommandExists("npm") {
		return errors.New("未检测到 npm，无法自动安装 cc-connect")
	}
	if err := cmdutil.RunCommand("npm", "install", "-g", "cc-connect"); err != nil {
		return fmt.Errorf("安装 cc-connect 失败: %w", err)
	}
	if !cmdutil.CommandExists("cc-connect") {
		return errors.New("cc-connect 安装后仍未检测到 cc-connect 命令")
	}
	return nil
}

func installAgentIfNeeded(p *prompt.Prompt, agentType string) error {
	switch agentType {
	case "claudecode":
		if cmdutil.CommandExists("claude") {
			cmdutil.Say("已检测到 Claude Code CLI")
			return nil
		}
		cmdutil.Warn("未检测到 Claude Code CLI: claude")
		if !p.AskYesNo("是否尝试使用 npm 全局安装 @anthropic-ai/claude-code", true) {
			return nil
		}
		if !cmdutil.CommandExists("npm") {
			return errors.New("未检测到 npm，无法自动安装 Claude Code CLI")
		}
		if err := cmdutil.RunCommand("npm", "install", "-g", "@anthropic-ai/claude-code"); err != nil {
			return fmt.Errorf("安装 Claude Code CLI 失败: %w", err)
		}
	case "cursor":
		if cmdutil.CommandExists("agent") {
			cmdutil.Say("已检测到 Cursor Agent CLI")
			return nil
		}
		cmdutil.Warn("未检测到 Cursor Agent CLI: agent")
		if !p.AskYesNo("是否尝试使用 npm 全局安装 @anthropic-ai/cursor-agent", false) {
			return nil
		}
		if !cmdutil.CommandExists("npm") {
			return errors.New("未检测到 npm，无法自动安装 Cursor Agent CLI")
		}
		if err := cmdutil.RunCommand("npm", "install", "-g", "@anthropic-ai/cursor-agent"); err != nil {
			return fmt.Errorf("安装 Cursor Agent CLI 失败: %w", err)
		}
	}
	return nil
}

func RunDoctor() error {
	fmt.Println("== 命令检查 ==")
	for _, name := range []string{"cc-connect", "claude", "agent", "ffmpeg", "npm", "brew"} {
		if path, err := exec.LookPath(name); err == nil {
			fmt.Printf("OK   %s: %s\n", name, path)
		} else {
			fmt.Printf("MISS %s\n", name)
		}
	}

	configPath := cmdutil.EnvDefault("CONFIG_PATH", cmdutil.DefaultConfigPath())

	if cmdutil.CommandExists("cc-connect") {
		fmt.Println("\n== cc-connect 版本 ==")
		version := config.CCConnectVersion()
		if version != "" {
			fmt.Printf("OK   %s\n", version)
			if !workspacesync.CompareVersionAtLeast(version, MinCCConnectVersion) {
				fmt.Printf("WARN cc-connect 版本低于建议最低版本 %s\n", MinCCConnectVersion)
			}
		} else {
			fmt.Println("WARN 无法读取 cc-connect 版本")
		}
	}

	if err := config.RunConfigDoctor(configPath); err != nil {
		return err
	}

	if cmdutil.CommandExists("cc-connect") {
		fmt.Println("\n== daemon 状态 ==")
		_ = cmdutil.RunCommand("cc-connect", "daemon", "status")
		fmt.Println("\n== cc-connect doctor ==")
		_ = cmdutil.RunCommand("cc-connect", "doctor")
	}
	return nil
}

func RunStart() error {
	configPath := cmdutil.EnvDefault("CONFIG_PATH", cmdutil.DefaultConfigPath())
	if !cmdutil.Exists(configPath) {
		return fmt.Errorf("配置文件不存在：%s，请先运行 bootstrap", configPath)
	}
	if !cmdutil.CommandExists("cc-connect") {
		return errors.New("未检测到 cc-connect，请先运行 bootstrap")
	}
	if err := cmdutil.RunCommand("cc-connect", config.DaemonInstallArgs(configPath)...); err != nil {
		return err
	}
	if err := cmdutil.RunCommand("cc-connect", "daemon", "start"); err != nil {
		return err
	}
	return cmdutil.RunCommand("cc-connect", "daemon", "status")
}
