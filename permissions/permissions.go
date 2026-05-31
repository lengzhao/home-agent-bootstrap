package permissions

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const DefaultTemplate = "family-remind"

type Template struct {
	ID          string
	DisplayName string
	Description string
	Disabled    []string
}

var templates = []Template{
	{
		ID:          "admin-only",
		DisplayName: "仅管理员可用",
		Description: "家人可发消息，但默认禁用全部管理和高危命令",
		Disabled: []string{
			"shell", "show", "dir", "restart", "upgrade", "commands",
			"cron", "provider", "model", "mode", "new", "reset", "login", "help", "status",
		},
	},
	{
		ID:          "family-readonly",
		DisplayName: "家人只读",
		Description: "家人可问答和查看，不可改配置、定时任务或执行高危命令",
		Disabled: []string{
			"shell", "show", "dir", "restart", "upgrade", "commands",
			"cron", "provider", "model", "mode", "new", "reset",
		},
	},
	{
		ID:          "family-remind",
		DisplayName: "家人可提醒不可执行",
		Description: "家人可聊天和接收提醒，禁用 shell/restart 等高危命令",
		Disabled: []string{
			"shell", "show", "dir", "restart", "upgrade", "commands",
		},
	},
}

func TemplateByID(id string) (Template, bool) {
	for _, preset := range templates {
		if preset.ID == id {
			return preset, true
		}
	}
	return Template{}, false
}

func PrintCatalog(out io.Writer) {
	fmt.Fprintln(out, "\n选择家庭权限模板：")
	for i, preset := range templates {
		fmt.Fprintf(out, "  %d) %s (%s)\n", i+1, preset.DisplayName, preset.ID)
		fmt.Fprintf(out, "     %s\n", preset.Description)
	}
	fmt.Fprintln(out, "\n默认 3 为 family-remind。")
}

func ParseChoice(raw string) (Template, error) {
	raw = trimDefault(raw, "3")
	idx, err := strconv.Atoi(raw)
	if err != nil || idx < 1 || idx > len(templates) {
		return Template{}, fmt.Errorf("无效权限模板序号 %q", raw)
	}
	return templates[idx-1], nil
}

func MemberDisabledCommands(templateID string) []string {
	if preset, ok := TemplateByID(templateID); ok {
		return append([]string(nil), preset.Disabled...)
	}
	if preset, ok := TemplateByID(DefaultTemplate); ok {
		return append([]string(nil), preset.Disabled...)
	}
	return []string{"shell", "show", "dir", "restart", "upgrade", "commands"}
}

func ChoiceDefaultFromEnv() string {
	id := strings.TrimSpace(os.Getenv("PERMISSION_TEMPLATE"))
	if id == "" {
		return "3"
	}
	for i, preset := range templates {
		if preset.ID == id {
			return strconv.Itoa(i + 1)
		}
	}
	return "3"
}

func trimDefault(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	return raw
}
