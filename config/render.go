package config

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/permissions"
	"github.com/lengzhao/home-agent-bootstrap/platforms"
)

type RenderInput struct {
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
	Platforms              []platforms.Block
	PermissionTemplate     string
	MemberDisabledCommands []string
}

func Render(templates fs.FS, cfg RenderInput) (string, error) {
	auditCmd := fmt.Sprintf(`mkdir -p %s && echo "$(date '+%%Y-%%m-%%dT%%H:%%M:%%S%%z') $CC_HOOK_EVENT $CC_HOOK_USER_ID $CC_HOOK_USER_NAME" >> %s`,
		filepath.Join(cfg.DataDir, "audit"),
		filepath.Join(cfg.DataDir, "audit", "events.log"),
	)

	data := struct {
		RenderInput
		AdminUserIDs           []string
		AuditCommand           string
		MemberDisabledCommands []string
	}{
		RenderInput:            cfg,
		AdminUserIDs:           cmdutil.OptionalStringSlice(cfg.AdminFrom),
		AuditCommand:           auditCmd,
		MemberDisabledCommands: cfg.MemberDisabledCommands,
	}
	if len(data.MemberDisabledCommands) == 0 {
		data.MemberDisabledCommands = permissions.MemberDisabledCommands(permissions.DefaultTemplate)
	}

	tmpl, err := template.New("config.generated.toml.tmpl").Funcs(template.FuncMap{
		"quote": cmdutil.TomlQuote,
		"array": cmdutil.TomlArray,
	}).ParseFS(templates, "templates/config.generated.toml.tmpl")
	if err != nil {
		return "", fmt.Errorf("加载配置模板失败: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "config.generated.toml.tmpl", data); err != nil {
		return "", fmt.Errorf("渲染配置失败: %w", err)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		out.WriteByte('\n')
	}
	return out.String(), nil
}

func ValidateAgentMode(agentType, mode string) error {
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

func DaemonInstallArgs(configPath string) []string {
	return []string{"daemon", "install", "--config", configPath, "--force"}
}
