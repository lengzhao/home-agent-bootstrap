package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const appName = "home-agent-bootstrap"

//go:embed workspace templates/config.generated.toml.tmpl
var workspaceTemplates embed.FS

type RenderConfigInput struct {
	ConfigPath             string
	DataDir                string
	Workspace              string
	ProjectName            string
	AgentType              string
	AgentMode              string
	ManagementToken        string
	BridgeToken            string
	WebhookToken           string
	AdminFrom              string
	ProviderName           string
	ProviderAPIKey         string
	ProviderBaseURL        string
	ProviderModel          string
	Platforms              []PlatformBlock
	PermissionTemplate     string
	MemberDisabledCommands []string
}

type prompt struct {
	in  *bufio.Reader
	out io.Writer
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "\n[ERROR] %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "bootstrap"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "bootstrap":
		return runBootstrap()
	case "doctor":
		return runDoctor()
	case "migrate-config":
		return runMigrateConfig()
	case "setup-weixin":
		return runSetupWeixin(args)
	case "start":
		return runStart()
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("未知命令 %q，运行 %s help 查看用法", cmd, appName)
	}
}

func printUsage(out io.Writer) {
	defaults := defaultBootstrapSettings()
	fmt.Fprintf(out, `%s - cc-connect 家庭超级助手引导器

用法:
  %s                 交互引导安装，默认命令，大部分问题直接回车即可
  %s bootstrap       同上
  %s doctor          检查本机依赖和 cc-connect 状态
  %s migrate-config  将旧版 [[providers]] 迁移到 [[projects.agent.providers]]
  %s setup-weixin N  按平台顺序扫码绑定 N 个微信个人号
  %s start           安装并启动 cc-connect daemon
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
  WORKSPACE            家庭助手工作目录
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
`, appName,
		appName, appName, appName, appName, appName, appName, appName,
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

func runBootstrap() error {
	p := prompt{in: bufio.NewReader(os.Stdin), out: os.Stdout}

	say("开始配置个人家庭超级助手")
	if !nonInteractive() {
		printBootstrapQuickGuide(p.out)
	}

	if os.Getenv("INSTALL_DEPS") != "0" {
		if err := installXcodeCLTIfNeeded(&p); err != nil {
			return err
		}
		if err := installHomebrewIfNeeded(&p); err != nil {
			return err
		}
		if err := installBasePackagesIfNeeded(&p); err != nil {
			return err
		}
	} else {
		warn("INSTALL_DEPS=0，跳过系统依赖安装")
	}

	if err := installCCConnectIfNeeded(&p); err != nil {
		return err
	}

	settings := loadBootstrapSettings(&p)
	if err := validateAgentMode(settings.AgentType, settings.AgentMode); err != nil {
		return err
	}

	if err := installAgentIfNeeded(&p, settings.AgentType); err != nil {
		return err
	}
	provider := chooseLLM(&p, settings, settings.AgentType)

	platforms, err := choosePlatforms(&p, settings)
	if err != nil {
		return err
	}

	adminFrom := ""
	if hasWeixinPlatform(platforms) {
		if nonInteractive() {
			adminFrom = envDefault("ADMIN_FROM", "")
		} else {
			adminFrom = p.ask("管理员微信 ilink user_id，未知可先留空，扫码后再修改", "")
		}
		if adminFrom == "" {
			warn("admin_from 为空时，特权命令不会授予任何用户。扫码后请用 /whoami 获取 user_id 并补充配置。")
		}
	} else {
		if nonInteractive() {
			adminFrom = envDefault("ADMIN_FROM", "")
		} else {
			adminFrom = p.ask("管理员 user_id（平台相关），未知可先留空", "")
		}
		if adminFrom == "" {
			warn("admin_from 为空时，特权命令不会授予任何用户。请在各平台完成首次对话后用 /whoami 获取 id 并补充配置。")
		}
	}

	configPath := settings.ConfigPath
	if exists(configPath) {
		overwrite := settings.OverwriteConfig
		if !nonInteractive() {
			overwrite = p.askYesNo("配置文件已存在，是否备份并覆盖", false)
		}
		if !overwrite {
			return errors.New("已取消，未覆盖现有配置")
		}
		if err := backupFile(configPath); err != nil {
			return err
		}
	}

	report, err := syncWorkspaceFiles(settings.Workspace)
	if err != nil {
		return err
	}
	printWorkspaceSyncReport(report)

	if settings.AgentType == "claudecode" && !nonInteractive() {
		initializeClaudeCodeWorkspace(settings.Workspace)
	}

	cfg := RenderConfigInput{
		ConfigPath:             configPath,
		DataDir:                filepath.Join(homeDir(), ".cc-connect"),
		Workspace:              settings.Workspace,
		ProjectName:            settings.ProjectName,
		AgentType:              settings.AgentType,
		AgentMode:              settings.AgentMode,
		ManagementToken:        mustRandomToken(),
		BridgeToken:            mustRandomToken(),
		WebhookToken:           mustRandomToken(),
		AdminFrom:              adminFrom,
		ProviderName:           provider.Name,
		ProviderAPIKey:         provider.APIKey,
		ProviderBaseURL:        provider.BaseURL,
		ProviderModel:          provider.Model,
		Platforms:              platforms,
		PermissionTemplate:     settings.PermissionTemplate,
		MemberDisabledCommands: memberDisabledCommands(settings.PermissionTemplate),
	}
	if err := writeFile(configPath, []byte(renderConfig(cfg)), 0o600); err != nil {
		return err
	}

	weixinCount := countWeixinPlatforms(platforms)
	weixinSetupDone := false
	setupWeixin := weixinCount > 0 && !settings.SkipWeixinSetup
	if !nonInteractive() && weixinCount > 0 {
		setupWeixin = p.askYesNo("是否现在逐个扫码绑定微信个人号", true)
	}
	if setupWeixin {
		if err := runSetupWeixinWithConfig(configPath, settings.ProjectName, weixinCount); err != nil {
			return err
		}
		if err := completeAdminRoleAfterWeixinSetup(&p, configPath); err != nil {
			return err
		}
		weixinSetupDone = true
	}

	printNextSteps(configPath, settings.ProjectName, platforms, settings.AgentType, weixinSetupDone)
	return nil
}

func renderConfig(cfg RenderConfigInput) string {
	auditCmd := fmt.Sprintf(`mkdir -p %s && echo "$(date '+%%Y-%%m-%%dT%%H:%%M:%%S%%z') $CC_HOOK_EVENT $CC_HOOK_USER_ID $CC_HOOK_USER_NAME" >> %s`,
		filepath.Join(cfg.DataDir, "audit"),
		filepath.Join(cfg.DataDir, "audit", "events.log"),
	)

	data := struct {
		RenderConfigInput
		AdminUserIDs           []string
		AuditCommand           string
		MemberDisabledCommands []string
	}{
		RenderConfigInput:      cfg,
		AdminUserIDs:           optionalStringSlice(cfg.AdminFrom),
		AuditCommand:           auditCmd,
		MemberDisabledCommands: cfg.MemberDisabledCommands,
	}
	if len(data.MemberDisabledCommands) == 0 {
		data.MemberDisabledCommands = memberDisabledCommands(defaultPermissionTemplate)
	}

	tmpl := template.Must(template.New("config.generated.toml.tmpl").Funcs(template.FuncMap{
		"quote": tomlQuote,
		"array": tomlArray,
	}).ParseFS(workspaceTemplates, "templates/config.generated.toml.tmpl"))

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "config.generated.toml.tmpl", data); err != nil {
		panic(err)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		out.WriteByte('\n')
	}
	return out.String()
}

func validateAgentMode(agentType, mode string) error {
	allowed := map[string][]string{
		"claudecode": {"default", "plan", "auto", "acceptEdits"},
		"cursor":     {"ask", "plan", "default", "force"},
	}
	values, ok := allowed[agentType]
	if !ok {
		return fmt.Errorf("不支持的 agent 类型 %q", agentType)
	}
	for _, value := range values {
		if mode == value {
			return nil
		}
	}
	return fmt.Errorf("%s 不支持权限模式 %q", agentType, mode)
}

func printClaudeCodeModeHelp() {
	fmt.Fprintln(os.Stdout, "Claude Code 权限模式说明：")
	fmt.Fprintln(os.Stdout, "- auto：推荐。自动执行低风险操作，适合可信本机家庭助手。")
	fmt.Fprintln(os.Stdout, "- default：执行工具前按 Claude Code 默认策略询问。")
	fmt.Fprintln(os.Stdout, "- plan：只读规划，不执行修改。")
	fmt.Fprintln(os.Stdout, "- acceptEdits：自动接受编辑，风险更高，首次部署不建议。")
}

func installXcodeCLTIfNeeded(p *prompt) error {
	if exec.Command("xcode-select", "-p").Run() == nil {
		say("已检测到 Xcode Command Line Tools")
		return nil
	}
	warn("未检测到 Xcode Command Line Tools")
	if !p.askYesNo("是否现在安装 Xcode Command Line Tools", true) {
		return nil
	}
	if err := runCommand("xcode-select", "--install"); err != nil {
		return fmt.Errorf("安装 Xcode Command Line Tools 失败: %w", err)
	}
	fmt.Fprintln(os.Stdout, "请在弹窗中完成安装。安装完成后回到终端按回车继续。")
	_, _ = p.in.ReadString('\n')
	return nil
}

func installHomebrewIfNeeded(p *prompt) error {
	addHomebrewToPath()
	if commandExists("brew") {
		say("已检测到 Homebrew")
		return nil
	}
	warn("未检测到 Homebrew")
	if !p.askYesNo("是否现在安装 Homebrew", true) {
		return nil
	}
	if err := runCommand("/bin/bash", "-c", `$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`); err != nil {
		return fmt.Errorf("安装 Homebrew 失败: %w", err)
	}
	addHomebrewToPath()
	if !commandExists("brew") {
		return errors.New("Homebrew 安装后仍未检测到 brew 命令")
	}
	return nil
}

func installBasePackagesIfNeeded(p *prompt) error {
	if !commandExists("brew") {
		warn("未检测到 brew，跳过基础包自动安装")
		return nil
	}
	if (!commandExists("node") || !commandExists("npm")) && p.askYesNo("是否使用 brew 安装 Node.js/npm", true) {
		if err := runCommand("brew", "install", "node"); err != nil {
			return fmt.Errorf("安装 Node.js/npm 失败: %w", err)
		}
	}
	if !commandExists("ffmpeg") && p.askYesNo("是否使用 brew 安装 ffmpeg，用于微信语音转写", true) {
		if err := runCommand("brew", "install", "ffmpeg"); err != nil {
			return fmt.Errorf("安装 ffmpeg 失败: %w", err)
		}
	}
	return nil
}

func installCCConnectIfNeeded(p *prompt) error {
	if commandExists("cc-connect") {
		say("已检测到 cc-connect")
		return nil
	}
	if nonInteractive() {
		return errors.New("未检测到 cc-connect，NONINTERACTIVE 模式下请先手动安装 cc-connect")
	}
	if !p.askYesNo("是否尝试使用 npm 全局安装 cc-connect", true) {
		return nil
	}
	if !commandExists("npm") {
		return errors.New("未检测到 npm，无法自动安装 cc-connect")
	}
	if err := runCommand("npm", "install", "-g", "cc-connect"); err != nil {
		return fmt.Errorf("安装 cc-connect 失败: %w", err)
	}
	if !commandExists("cc-connect") {
		return errors.New("cc-connect 安装后仍未检测到 cc-connect 命令")
	}
	return nil
}

func installAgentIfNeeded(p *prompt, agentType string) error {
	switch agentType {
	case "claudecode":
		if commandExists("claude") {
			say("已检测到 Claude Code CLI")
			return nil
		}
		warn("未检测到 Claude Code CLI: claude")
		if !p.askYesNo("是否尝试使用 npm 全局安装 @anthropic-ai/claude-code", true) {
			return nil
		}
		if !commandExists("npm") {
			return errors.New("未检测到 npm，无法自动安装 Claude Code CLI")
		}
		if err := runCommand("npm", "install", "-g", "@anthropic-ai/claude-code"); err != nil {
			return fmt.Errorf("安装 Claude Code CLI 失败: %w", err)
		}
	case "cursor":
		if commandExists("agent") {
			say("已检测到 Cursor Agent CLI")
			return nil
		}
		warn("未检测到 Cursor Agent CLI: agent")
		if !p.askYesNo("是否尝试使用 npm 全局安装 @anthropic-ai/cursor-agent", false) {
			return nil
		}
		if !commandExists("npm") {
			return errors.New("未检测到 npm，无法自动安装 Cursor Agent CLI")
		}
		if err := runCommand("npm", "install", "-g", "@anthropic-ai/cursor-agent"); err != nil {
			return fmt.Errorf("安装 Cursor Agent CLI 失败: %w", err)
		}
	}
	return nil
}

type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
}

func configureLLM(p *prompt, agentType string) ProviderConfig {
	if agentType == "cursor" {
		fmt.Fprintln(os.Stdout, "Cursor Agent 通常依赖 Cursor 账号登录。")
		if p.askYesNo("是否现在运行 agent --help 验证 CLI 可用", true) && commandExists("agent") {
			_ = runCommand("agent", "--help")
		}
		return ProviderConfig{}
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
	choice := p.askAllowed("请选择", "1", []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"})
	switch choice {
	case "1":
		fmt.Fprintln(os.Stdout, "稍后会在家庭助手工作目录启动 claude，用于完成登录和信任工作目录。")
	case "2":
		key := p.askSecret("请输入 ANTHROPIC_API_KEY，本值只写入本机生成的 config.toml，不会进入仓库模板")
		if key != "" {
			return ProviderConfig{Name: "anthropic", APIKey: key}
		}
		warn("API Key 为空，跳过 Provider 写入")
	case "3":
		if preset, ok := providerPresetByName("openai"); ok {
			provider, err := configureClaudeCodeProviderFromPreset(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "4":
		if preset, ok := providerPresetByName("openrouter"); ok {
			provider, err := configureClaudeCodeProviderFromPreset(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "5":
		if preset, ok := providerPresetByName("kimi"); ok {
			provider, err := configureClaudeCodeProviderFromPreset(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "6":
		if preset, ok := providerPresetByName("volcengine"); ok {
			provider, err := configureClaudeCodeProviderFromPreset(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "7":
		if preset, ok := providerPresetByName("qwen"); ok {
			provider, err := configureClaudeCodeProviderFromPreset(p, preset)
			if err != nil {
				warn(err.Error())
			}
			return provider
		}
	case "8":
		key := p.askSecret("请输入 API Key，将写入 config.toml，并同步写入 shell 配置文件")
		if key != "" {
			baseURL := p.ask("ANTHROPIC_BASE_URL（OpenAI-compatible 接口地址）", "")
			model := p.ask("模型 ID", "")
			profile := customClaudeCodeProfile(key, model, baseURL)
			if err := configureClaudeCodeShellEnv(p, profile, ""); err != nil {
				warn(err.Error())
			}
			return ProviderConfig{Name: "custom", APIKey: key, BaseURL: baseURL, Model: model}
		} else {
			warn("API Key 为空，跳过 Provider 和环境变量配置")
		}
	case "9":
		warn("已跳过 LLM 配置。启动前请确保 claude 已登录或 provider 已配置。")
	}
	return ProviderConfig{}
}

func runDoctor() error {
	fmt.Println("== 命令检查 ==")
	for _, name := range []string{"cc-connect", "claude", "agent", "ffmpeg", "npm", "brew"} {
		if path, err := exec.LookPath(name); err == nil {
			fmt.Printf("OK   %s: %s\n", name, path)
		} else {
			fmt.Printf("MISS %s\n", name)
		}
	}

	configPath := envDefault("CONFIG_PATH", filepath.Join(homeDir(), ".cc-connect", "config.toml"))

	if commandExists("cc-connect") {
		fmt.Println("\n== cc-connect 版本 ==")
		version := ccConnectVersion()
		if version != "" {
			fmt.Printf("OK   %s\n", version)
			if !compareVersionAtLeast(version, minCCConnectVersion) {
				fmt.Printf("WARN cc-connect 版本低于建议最低版本 %s\n", minCCConnectVersion)
			}
		} else {
			fmt.Println("WARN 无法读取 cc-connect 版本")
		}
	}

	if err := runConfigDoctor(configPath); err != nil {
		return err
	}

	if commandExists("cc-connect") {
		fmt.Println("\n== daemon 状态 ==")
		_ = runCommand("cc-connect", "daemon", "status")
		fmt.Println("\n== cc-connect doctor ==")
		_ = runCommand("cc-connect", "doctor")
	}
	return nil
}

func runSetupWeixin(args []string) error {
	count := 1
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return errors.New("微信账号数量必须是大于 0 的数字")
		}
		count = n
	}
	configPath := envDefault("CONFIG_PATH", filepath.Join(homeDir(), ".cc-connect", "config.toml"))
	projectName := envDefault("PROJECT_NAME", "home")
	if !commandExists("cc-connect") {
		return errors.New("未检测到 cc-connect，请先运行 bootstrap")
	}
	return runSetupWeixinWithConfig(configPath, projectName, count)
}

func runSetupWeixinWithConfig(configPath, projectName string, count int) error {
	if !commandExists("cc-connect") {
		return errors.New("未检测到 cc-connect，请先运行 bootstrap")
	}
	for i := 1; i <= count; i++ {
		fmt.Printf("\n开始绑定第 %d 个微信个人号\n", i)
		if err := runCommand("cc-connect", "weixin", "setup", "--config", configPath, "--project", projectName, "--platform-index", strconv.Itoa(i)); err != nil {
			return err
		}
	}
	fmt.Println("\n微信绑定完成。" + weixinFirstMessageInstruction())
	return nil
}

func completeAdminRoleAfterWeixinSetup(p *prompt, configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	adminUserID := firstConfiguredAdminFrom(string(content))
	if adminUserID == "" {
		adminUserID = firstBoundWeixinAllowFrom(string(content))
	}
	if adminUserID == "" {
		adminUserID = p.ask("未能自动读取扫码用户，输入管理员微信 ilink user_id，留空则暂不写入 admin", "")
	}
	if adminUserID == "" {
		warn("未写入管理员角色。之后可用 /whoami 获取 user_id，再手动补充 projects.users.roles.admin.user_ids。")
		return nil
	}
	updated := applyAdminUserToConfig(string(content), adminUserID)
	if err := writeFile(configPath, []byte(updated), 0o600); err != nil {
		return err
	}
	say("已写入管理员角色：" + adminUserID)
	return nil
}

func firstConfiguredAdminFrom(config string) string {
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "admin_from = ") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "admin_from = "))
		value, err := strconv.Unquote(raw)
		if err != nil || value == "" || value == "*" {
			continue
		}
		return value
	}
	return ""
}

func firstBoundWeixinAllowFrom(config string) string {
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "allow_from = ") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "allow_from = "))
		value, err := strconv.Unquote(raw)
		if err != nil || value == "" || value == "*" {
			continue
		}
		return value
	}
	return ""
}

func applyAdminUserToConfig(config, adminUserID string) string {
	lines := strings.Split(config, "\n")
	inAdminRole := false
	hasAdminRole := false
	insertAt := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "admin_from = "):
			lines[i] = linePrefix(line) + "admin_from = " + tomlQuote(adminUserID)
		case strings.HasPrefix(trimmed, "[") && trimmed != "[projects.users.roles.admin]":
			if insertAt == -1 && trimmed == "[projects.users.roles.member]" {
				insertAt = i
			}
			inAdminRole = false
		case trimmed == "[projects.users.roles.admin]":
			hasAdminRole = true
			inAdminRole = true
		case inAdminRole && strings.HasPrefix(trimmed, "user_ids = "):
			lines[i] = linePrefix(line) + "user_ids = " + tomlArray([]string{adminUserID})
		}
	}
	if !hasAdminRole {
		adminRole := []string{
			"[projects.users.roles.admin]",
			"user_ids = " + tomlArray([]string{adminUserID}),
			"disabled_commands = []",
			"rate_limit = { max_messages = 50, window_secs = 60 }",
			"",
		}
		if insertAt == -1 {
			lines = append(lines, adminRole...)
		} else {
			updated := make([]string, 0, len(lines)+len(adminRole))
			updated = append(updated, lines[:insertAt]...)
			updated = append(updated, adminRole...)
			updated = append(updated, lines[insertAt:]...)
			lines = updated
		}
	}
	return strings.Join(lines, "\n")
}

func linePrefix(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func runStart() error {
	configPath := envDefault("CONFIG_PATH", filepath.Join(homeDir(), ".cc-connect", "config.toml"))
	if !exists(configPath) {
		return fmt.Errorf("配置文件不存在：%s，请先运行 bootstrap", configPath)
	}
	if !commandExists("cc-connect") {
		return errors.New("未检测到 cc-connect，请先运行 bootstrap")
	}
	if err := runCommand("cc-connect", daemonInstallArgs(configPath)...); err != nil {
		return err
	}
	if err := runCommand("cc-connect", "daemon", "start"); err != nil {
		return err
	}
	return runCommand("cc-connect", "daemon", "status")
}

func daemonInstallArgs(configPath string) []string {
	return []string{"daemon", "install", "--config", configPath, "--force"}
}

func initializeClaudeCodeWorkspace(workspace string) {
	if !commandExists("claude") {
		warn("未检测到 claude，安装完成后请在家庭助手工作目录运行 claude 完成登录和信任。")
		return
	}
	name, args, dir := claudeWorkspaceInitCommand(workspace)
	fmt.Fprintf(os.Stdout, "\n将在家庭助手工作目录启动 Claude Code：%s\n", dir)
	fmt.Fprintln(os.Stdout, "请在 Claude Code 里完成登录、信任工作目录。完成后退出 Claude Code，bootstrap 会继续。")
	if err := runCommandInDir(dir, name, args...); err != nil {
		warn("Claude Code 初始化未完成。之后请在家庭助手工作目录手动运行 claude。")
	}
}

func claudeWorkspaceInitCommand(workspace string) (string, []string, string) {
	return "claude", nil, workspace
}

func writeWorkspaceFiles(workspace string) error {
	_, err := syncWorkspaceFiles(workspace)
	return err
}

func (p prompt) ask(label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	text, _ := p.in.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue
	}
	return text
}

func (p prompt) askYesNo(label string, defaultValue bool) bool {
	if nonInteractive() {
		return defaultValue
	}
	def := "n"
	if defaultValue {
		def = "y"
	}
	for {
		answer := strings.ToLower(p.ask(label+" (y/n)", def))
		switch answer {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Fprintln(p.out, "请输入 y 或 n")
		}
	}
}

func (p prompt) askAllowed(label, defaultValue string, allowed []string) string {
	for {
		value := p.ask(label, defaultValue)
		for _, item := range allowed {
			if value == item {
				return value
			}
		}
		fmt.Fprintf(p.out, "无效输入：%s。可选值：%s\n", value, strings.Join(allowed, ", "))
	}
}

func (p prompt) askSecret(label string) string {
	fmt.Fprintf(p.out, "%s: ", label)
	// Avoid terminal-specific password handling so the binary works in simple
	// pipes and remote shells. Users are warned that the value is local only.
	text, _ := p.in.ReadString('\n')
	return strings.TrimSpace(text)
}

func printNextSteps(configPath, projectName string, platforms []PlatformBlock, agentType string, weixinSetupDone bool) {
	fmt.Printf("\n配置已生成：%s\n", configPath)
	fmt.Println("\n下一步：")
	if agentType == "claudecode" {
		fmt.Println("\n1. 确认 Claude Code 可登录：")
		fmt.Printf("   source %s   # 若使用第三方 LLM 环境变量\n", shellProfilePath())
		fmt.Println("   claude")
	} else {
		fmt.Println("\n1. 确认 Cursor Agent 可用：")
		fmt.Println("   agent --help")
	}

	step := 2
	weixinCount := countWeixinPlatforms(platforms)
	if weixinCount > 0 {
		if weixinSetupDone {
			fmt.Printf("\n%d. 微信扫码绑定已完成。%s\n", step, weixinFirstMessageInstruction())
		} else {
			fmt.Printf("\n%d. 逐个扫码绑定微信个人号：\n", step)
			fmt.Printf("   PROJECT_NAME=%s CONFIG_PATH=%q %s setup-weixin %d\n", projectName, configPath, appName, weixinCount)
		}
		step++
	}

	hints := platformSetupHints(platforms)
	if len(hints) > 0 {
		fmt.Printf("\n%d. 其他平台可使用 cc-connect setup 命令补全凭证：\n", step)
		for _, hint := range hints {
			fmt.Println(hint)
		}
		step++
	}

	fmt.Printf("\n%d. 启动服务：\n", step)
	fmt.Printf("   %s start\n", appName)
	step++
	fmt.Printf("\n%d. Web 管理后台：\n", step)
	fmt.Println("   http://localhost:9820")
	step++
	fmt.Printf("\n%d. 启用 Heartbeat（可选）：\n", step)
	fmt.Println("   " + heartbeatSetupGuide())

	if weixinCount > 0 && !weixinSetupDone {
		fmt.Println("\n注意：" + weixinFirstMessageInstruction())
	}
}

func weixinFirstMessageInstruction() string {
	return "请用每个已绑定微信号给机器人发送 /whoami 或一条普通消息，确认平台用户 ID 和消息链路正常。"
}

func addHomebrewToPath() {
	for _, path := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		if exists(filepath.Join(path, "brew")) {
			os.Setenv("PATH", path+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
	}
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func backupFile(path string) error {
	backup := fmt.Sprintf("%s.bak.%d", path, os.Getpid())
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := writeFile(backup, input, 0o600); err != nil {
		return err
	}
	say("已备份现有配置到 " + backup)
	return nil
}

func writeFile(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, mode)
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return home
}

func envDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func mustRandomToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func tomlQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func tomlArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = tomlQuote(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func optionalStringSlice(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func say(message string) {
	fmt.Printf("\n[%s] %s\n", appName, message)
}

func warn(message string) {
	fmt.Fprintf(os.Stderr, "\n[WARN] %s\n", message)
}
