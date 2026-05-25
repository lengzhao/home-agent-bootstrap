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
	"io/fs"
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
	ConfigPath      string
	DataDir         string
	Workspace       string
	ProjectName     string
	AgentType       string
	AgentMode       string
	ManagementToken string
	BridgeToken     string
	WebhookToken    string
	AdminFrom       string
	ProviderName    string
	ProviderAPIKey  string
	ProviderBaseURL string
	ProviderModel   string
	Platforms       []PlatformBlock
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
	case "setup-weixin":
		return runSetupWeixin(args)
	case "start":
		return runStart()
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("未知命令 %q，运行 %s help 查看用法", cmd, appName)
	}
}

func printUsage() {
	fmt.Printf(`%s - cc-connect 家庭超级助手引导器

用法:
  %s bootstrap       全新 Mac 引导安装和生成配置，默认命令
  %s doctor          检查本机依赖和 cc-connect 状态
  %s setup-weixin N  按平台顺序扫码绑定 N 个微信个人号
  %s start           安装并启动 cc-connect daemon

环境变量:
  CONFIG_PATH        cc-connect 配置路径，默认 ~/.cc-connect/config.toml
  PROJECT_NAME       cc-connect project 名称，默认 home
  INSTALL_DEPS=0     跳过 Homebrew、Node、ffmpeg 等系统依赖安装
`, appName, appName, appName, appName, appName)
}

func runBootstrap() error {
	p := prompt{in: bufio.NewReader(os.Stdin), out: os.Stdout}

	say("开始配置个人家庭超级助手")

	if os.Getenv("INSTALL_DEPS") != "0" {
		installXcodeCLTIfNeeded(&p)
		installHomebrewIfNeeded(&p)
		installBasePackagesIfNeeded(&p)
	} else {
		warn("INSTALL_DEPS=0，跳过系统依赖安装")
	}

	installCCConnectIfNeeded(&p)

	defaultConfig := filepath.Join(homeDir(), ".cc-connect", "config.toml")
	defaultWorkspace := filepath.Join(homeDir(), "home-assistant-workspace")

	configPath := p.ask("cc-connect 配置文件路径", defaultConfig)
	workspace := p.ask("家庭助手工作目录", defaultWorkspace)
	projectName := p.ask("cc-connect project 名称", "home")

	fmt.Fprintln(os.Stdout, "选择运行时 Agent：")
	fmt.Fprintln(os.Stdout, "1) Claude Code，推荐用于家庭助手运行时")
	fmt.Fprintln(os.Stdout, "2) Cursor Agent，适合只读/规划或开发维护")
	agentChoice := p.askAllowed("请选择", "1", []string{"1", "2"})

	agentType := "claudecode"
	agentMode := "default"
	if agentChoice == "2" {
		agentType = "cursor"
		agentMode = p.askAllowed("Cursor Agent 默认权限模式", "ask", []string{"ask", "plan", "default", "force"})
	} else {
		printClaudeCodeModeHelp()
		agentMode = p.askAllowed("Claude Code 默认权限模式", "default", []string{"default", "plan", "auto", "acceptEdits"})
	}
	if err := validateAgentMode(agentType, agentMode); err != nil {
		return err
	}

	installAgentIfNeeded(&p, agentType)
	provider := configureLLM(&p, agentType)

	platforms, err := configurePlatforms(&p)
	if err != nil {
		return err
	}

	adminFrom := ""
	if hasWeixinPlatform(platforms) {
		adminFrom = p.ask("管理员微信 ilink user_id，未知可先留空，扫码后再修改", "")
		if adminFrom == "" {
			warn("admin_from 为空时，特权命令不会授予任何用户。扫码后请用 /whoami 获取 user_id 并补充配置。")
		}
	} else {
		adminFrom = p.ask("管理员 user_id（平台相关），未知可先留空", "")
		if adminFrom == "" {
			warn("admin_from 为空时，特权命令不会授予任何用户。请在各平台完成首次对话后用 /whoami 获取 id 并补充配置。")
		}
	}

	if exists(configPath) {
		if !p.askYesNo("配置文件已存在，是否备份并覆盖", false) {
			return errors.New("已取消，未覆盖现有配置")
		}
		if err := backupFile(configPath); err != nil {
			return err
		}
	}

	if err := writeWorkspaceFiles(workspace); err != nil {
		return err
	}
	if agentType == "claudecode" {
		initializeClaudeCodeWorkspace(workspace)
	}

	cfg := RenderConfigInput{
		ConfigPath:      configPath,
		DataDir:         filepath.Join(homeDir(), ".cc-connect"),
		Workspace:       workspace,
		ProjectName:     projectName,
		AgentType:       agentType,
		AgentMode:       agentMode,
		ManagementToken: mustRandomToken(),
		BridgeToken:     mustRandomToken(),
		WebhookToken:    mustRandomToken(),
		AdminFrom:       adminFrom,
		ProviderName:    provider.Name,
		ProviderAPIKey:  provider.APIKey,
		ProviderBaseURL: provider.BaseURL,
		ProviderModel:   provider.Model,
		Platforms:       platforms,
	}
	if err := writeFile(configPath, []byte(renderConfig(cfg)), 0o600); err != nil {
		return err
	}

	weixinCount := countWeixinPlatforms(platforms)
	weixinSetupDone := false
	if weixinCount > 0 && p.askYesNo("是否现在逐个扫码绑定微信个人号", true) {
		if err := runSetupWeixinWithConfig(configPath, projectName, weixinCount); err != nil {
			return err
		}
		if err := completeAdminRoleAfterWeixinSetup(&p, configPath); err != nil {
			return err
		}
		weixinSetupDone = true
	}

	printNextSteps(configPath, projectName, platforms, agentType, weixinSetupDone)
	return nil
}

func renderConfig(cfg RenderConfigInput) string {
	auditCmd := fmt.Sprintf(`mkdir -p %s && echo "$(date '+%%Y-%%m-%%dT%%H:%%M:%%S%%z') $CC_HOOK_EVENT $CC_HOOK_USER_ID $CC_HOOK_USER_NAME" >> %s`,
		filepath.Join(cfg.DataDir, "audit"),
		filepath.Join(cfg.DataDir, "audit", "events.log"),
	)

	data := struct {
		RenderConfigInput
		AdminUserIDs []string
		AuditCommand string
	}{
		RenderConfigInput: cfg,
		AdminUserIDs:      optionalStringSlice(cfg.AdminFrom),
		AuditCommand:      auditCmd,
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
	fmt.Fprintln(os.Stdout, "- default：推荐。执行工具前按 Claude Code 默认策略询问。")
	fmt.Fprintln(os.Stdout, "- plan：只读规划，不执行修改。")
	fmt.Fprintln(os.Stdout, "- auto：自动执行低风险操作，适合可信本机环境。")
	fmt.Fprintln(os.Stdout, "- acceptEdits：自动接受编辑，风险更高，首次部署不建议。")
}

func installXcodeCLTIfNeeded(p *prompt) {
	if exec.Command("xcode-select", "-p").Run() == nil {
		say("已检测到 Xcode Command Line Tools")
		return
	}
	warn("未检测到 Xcode Command Line Tools")
	if p.askYesNo("是否现在安装 Xcode Command Line Tools", true) {
		_ = runCommand("xcode-select", "--install")
		fmt.Fprintln(os.Stdout, "请在弹窗中完成安装。安装完成后回到终端按回车继续。")
		_, _ = p.in.ReadString('\n')
	}
}

func installHomebrewIfNeeded(p *prompt) {
	addHomebrewToPath()
	if commandExists("brew") {
		say("已检测到 Homebrew")
		return
	}
	warn("未检测到 Homebrew")
	if p.askYesNo("是否现在安装 Homebrew", true) {
		_ = runCommand("/bin/bash", "-c", `$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`)
		addHomebrewToPath()
	}
}

func installBasePackagesIfNeeded(p *prompt) {
	if !commandExists("brew") {
		warn("未检测到 brew，跳过基础包自动安装")
		return
	}
	if (!commandExists("node") || !commandExists("npm")) && p.askYesNo("是否使用 brew 安装 Node.js/npm", true) {
		_ = runCommand("brew", "install", "node")
	}
	if !commandExists("ffmpeg") && p.askYesNo("是否使用 brew 安装 ffmpeg，用于微信语音转写", true) {
		_ = runCommand("brew", "install", "ffmpeg")
	}
}

func installCCConnectIfNeeded(p *prompt) {
	if commandExists("cc-connect") {
		say("已检测到 cc-connect")
		return
	}
	warn("未检测到 cc-connect")
	if p.askYesNo("是否尝试使用 npm 全局安装 cc-connect", true) {
		if !commandExists("npm") {
			warn("未检测到 npm，无法自动安装 cc-connect")
			return
		}
		_ = runCommand("npm", "install", "-g", "cc-connect")
	}
}

func installAgentIfNeeded(p *prompt, agentType string) {
	switch agentType {
	case "claudecode":
		if commandExists("claude") {
			say("已检测到 Claude Code CLI")
			return
		}
		warn("未检测到 Claude Code CLI: claude")
		if p.askYesNo("是否尝试使用 npm 全局安装 @anthropic-ai/claude-code", true) && commandExists("npm") {
			_ = runCommand("npm", "install", "-g", "@anthropic-ai/claude-code")
		}
	case "cursor":
		if commandExists("agent") {
			say("已检测到 Cursor Agent CLI")
			return
		}
		warn("未检测到 Cursor Agent CLI: agent")
		if p.askYesNo("是否尝试使用 npm 全局安装 @anthropic-ai/cursor-agent", false) && commandExists("npm") {
			_ = runCommand("npm", "install", "-g", "@anthropic-ai/cursor-agent")
		}
	}
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
	fmt.Fprintln(os.Stdout, "3) OpenAI（写入 config.toml Provider，并同步 ~/.zshrc）")
	fmt.Fprintln(os.Stdout, "4) OpenRouter（写入 config.toml Provider，并同步 ~/.zshrc）")
	fmt.Fprintln(os.Stdout, "5) Kimi (Moonshot)（写入 config.toml Provider，并同步 ~/.zshrc）")
	fmt.Fprintln(os.Stdout, "6) 火山引擎 (豆包)（写入 config.toml Provider，并同步 ~/.zshrc）")
	fmt.Fprintln(os.Stdout, "7) 通义千问 (DashScope)（写入 config.toml Provider，并同步 ~/.zshrc）")
	fmt.Fprintln(os.Stdout, "8) 自定义 OpenAI-compatible（写入 config.toml Provider，并同步 ~/.zshrc）")
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
		key := p.askSecret("请输入 API Key，将写入 config.toml，并同步写入 ~/.zshrc")
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
	fmt.Println("\n== 配置检查 ==")
	if exists(configPath) {
		fmt.Printf("OK   config: %s\n", configPath)
	} else {
		fmt.Printf("MISS config: %s\n", configPath)
	}
	if commandExists("cc-connect") {
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
	return fs.WalkDir(workspaceTemplates, "workspace", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel("workspace", path)
		if err != nil {
			return err
		}
		target := filepath.Join(workspace, rel)
		if exists(target) {
			// Preserve user-edited skills and instruction files across reruns.
			return nil
		}
		content, err := workspaceTemplates.ReadFile(path)
		if err != nil {
			return err
		}
		if err := writeFile(target, content, 0o644); err != nil {
			return err
		}
		return nil
	})
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
		fmt.Println("   source ~/.zshrc   # 若使用第三方 LLM 环境变量")
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

	if weixinCount > 0 && !weixinSetupDone {
		fmt.Println("\n注意：" + weixinFirstMessageInstruction())
	}
}

func weixinFirstMessageInstruction() string {
	return "请用每个已绑定微信号先给机器人发送 /login，完成登录后再发普通消息或 /whoami，以便 cc-connect 缓存 context_token。"
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
