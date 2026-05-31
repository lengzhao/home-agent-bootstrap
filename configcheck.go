package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type configFinding struct {
	level   string
	message string
}

func analyzeConfig(content string) []configFinding {
	findings := make([]configFinding, 0)

	if strings.TrimSpace(content) == "" {
		findings = append(findings, configFinding{"MISS", "配置文件为空"})
		return findings
	}

	if hasLegacyTopLevelProviders(content) {
		findings = append(findings, configFinding{
			"WARN",
			"检测到旧版顶层 [[providers]]。daemon 下 Claude Code 应使用 [[projects.agent.providers]]。可运行 migrate-config 自动迁移",
		})
	}
	if strings.Contains(content, "provider_refs") {
		findings = append(findings, configFinding{
			"WARN",
			"检测到 provider_refs。当前 cc-connect 项目 Provider 只需在 [projects.agent.options] 设置 provider",
		})
	}

	agentType := configValue(content, `[projects.agent]`, "type")
	if agentType == "" {
		findings = append(findings, configFinding{"WARN", "未找到 [projects.agent] type"})
	} else {
		findings = append(findings, configFinding{"OK", "agent type = " + agentType})
	}

	activeProvider := configValue(content, `[projects.agent.options]`, "provider")
	projectProviders := projectProviderNames(content)

	if agentType == "claudecode" && activeProvider != "" {
		if len(projectProviders) == 0 {
			findings = append(findings, configFinding{
				"FAIL",
				fmt.Sprintf("已设置 provider = %q，但未找到 [[projects.agent.providers]]", activeProvider),
			})
		} else if !containsString(projectProviders, activeProvider) {
			findings = append(findings, configFinding{
				"FAIL",
				fmt.Sprintf("provider = %q 未在 [[projects.agent.providers]] 中定义，当前有 %s", activeProvider, strings.Join(projectProviders, ", ")),
			})
		} else {
			findings = append(findings, configFinding{
				"OK",
				fmt.Sprintf("provider %q 已在 [[projects.agent.providers]] 中定义", activeProvider),
			})
		}
	}

	if agentType == "claudecode" && activeProvider == "" && len(projectProviders) > 0 {
		findings = append(findings, configFinding{
			"WARN",
			"已配置 [[projects.agent.providers]]，但 [projects.agent.options] 未设置 provider",
		})
	}

	if !hasLegacyTopLevelProviders(content) && activeProvider == "" && len(projectProviders) == 0 && agentType == "claudecode" {
		findings = append(findings, configFinding{
			"OK",
			"未配置第三方 Provider，Claude Code 将依赖交互式登录",
		})
	}

	return findings
}

func hasLegacyTopLevelProviders(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[[providers]]" {
			return true
		}
	}
	return false
}

func configValue(content, sectionHeader, key string) string {
	inSection := false
	header := strings.TrimSpace(sectionHeader)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[[") {
			inSection = trimmed == header
			continue
		}
		if !inSection {
			continue
		}
		prefix := key + " = "
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		value, err := strconv.Unquote(raw)
		if err != nil {
			return raw
		}
		return value
	}
	return ""
}

func projectProviderNames(content string) []string {
	names := make([]string, 0)
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "[[projects.agent.providers]]":
			inBlock = true
		case "[[providers]]", "[[projects]]", "[[projects.platforms]]", "[[hooks]]":
			if trimmed != "[[projects.agent.providers]]" {
				inBlock = false
			}
		}
		if !inBlock {
			continue
		}
		if strings.HasPrefix(trimmed, "name = ") {
			raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "name = "))
			if value, err := strconv.Unquote(raw); err == nil {
				names = append(names, value)
			}
		}
	}
	return names
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func printConfigFindings(findings []configFinding) {
	for _, finding := range findings {
		fmt.Printf("%-4s %s\n", finding.level, finding.message)
	}
}

func migrateLegacyConfig(content string) (string, bool, error) {
	if !hasLegacyTopLevelProviders(content) && !strings.Contains(content, "provider_refs") {
		return content, false, nil
	}

	legacyBlocks := extractLegacyProviderBlocks(content)
	updated := removeLegacyProviderBlocks(content)
	updated = removeProviderRefsLines(updated)
	updated = removeAgentTypesLines(updated)

	if len(legacyBlocks) > 0 {
		updated, err := insertProjectAgentProviders(updated, legacyBlocks)
		if err != nil {
			return "", false, err
		}
		return updated, true, nil
	}

	return updated, true, nil
}

type legacyProviderBlock struct {
	lines []string
}

func extractLegacyProviderBlocks(content string) []legacyProviderBlock {
	lines := strings.Split(content, "\n")
	blocks := make([]legacyProviderBlock, 0)
	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) != "[[providers]]" {
			i++
			continue
		}
		blockLines := []string{"[[projects.agent.providers]]"}
		i++
		for i < len(lines) {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "[[providers]]" {
				break
			}
			if strings.HasPrefix(trimmed, "[[") {
				break
			}
			if strings.HasPrefix(trimmed, "[") {
				break
			}
			if strings.HasPrefix(trimmed, "agent_types") {
				i++
				continue
			}
			blockLines = append(blockLines, lines[i])
			i++
		}
		blocks = append(blocks, legacyProviderBlock{lines: blockLines})
	}
	return blocks
}

func removeLegacyProviderBlocks(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) != "[[providers]]" {
			out = append(out, lines[i])
			i++
			continue
		}
		i++
		for i < len(lines) {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "[[providers]]" {
				break
			}
			if strings.HasPrefix(trimmed, "[[") || (strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "#")) {
				break
			}
			i++
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

func removeProviderRefsLines(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "provider_refs = ") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func removeAgentTypesLines(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "agent_types = ") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func insertProjectAgentProviders(content string, blocks []legacyProviderBlock) (string, error) {
	if strings.Contains(content, "[[projects.agent.providers]]") {
		return content, nil
	}

	lines := strings.Split(content, "\n")
	insertAt := -1
	inOptions := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[projects.agent.options]" {
			inOptions = true
			continue
		}
		if !inOptions {
			continue
		}
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "#") {
			insertAt = i
			break
		}
	}
	if inOptions && insertAt == -1 {
		insertAt = len(lines)
	}
	if insertAt == -1 {
		return "", fmt.Errorf("未找到 [projects.agent.options] 结束位置，无法插入 [[projects.agent.providers]]")
	}

	newLines := make([]string, 0, len(lines)+8)
	newLines = append(newLines, lines[:insertAt]...)
	for _, block := range blocks {
		if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
			newLines = append(newLines, "")
		}
		newLines = append(newLines, block.lines...)
	}
	newLines = append(newLines, lines[insertAt:]...)
	return strings.Join(newLines, "\n"), nil
}

func runMigrateConfig() error {
	configPath := envDefault("CONFIG_PATH", filepath.Join(homeDir(), ".cc-connect", "config.toml"))
	if !exists(configPath) {
		return fmt.Errorf("配置文件不存在：%s", configPath)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	updated, changed, err := migrateLegacyConfig(string(content))
	if err != nil {
		return err
	}
	if !changed {
		say("未发现需要迁移的旧版 Provider 配置")
		return nil
	}

	if err := backupFile(configPath); err != nil {
		return err
	}
	if err := writeFile(configPath, []byte(updated), 0o600); err != nil {
		return err
	}

	say("已将旧版 [[providers]] 迁移到 [[projects.agent.providers]]，并移除 provider_refs")
	fmt.Println("建议执行：")
	fmt.Printf("  %s doctor\n", appName)
	fmt.Printf("  %s start\n", appName)
	return nil
}

func ccConnectVersion() string {
	if !commandExists("cc-connect") {
		return ""
	}
	out, err := exec.Command("cc-connect", "--version").CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

func runConfigDoctor(configPath string) error {
	fmt.Println("\n== 配置结构检查 ==")
	if !exists(configPath) {
		fmt.Printf("MISS config: %s\n", configPath)
		return nil
	}

	fmt.Printf("OK   config: %s\n", configPath)
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	findings := analyzeConfig(string(content))
	printConfigFindings(findings)

	for _, finding := range findings {
		if finding.level == "FAIL" {
			fmt.Println("\n修复建议：")
			fmt.Printf("  %s migrate-config   # 若存在旧版 [[providers]]\n", appName)
			fmt.Printf("  或重新运行 %s bootstrap 生成新配置\n", appName)
			break
		}
	}
	return nil
}

var platformDocRowPattern = regexp.MustCompile(`^\|\s*(\d+)\s*\|\s*([a-z0-9_]+)\s*\|`)

func platformTypesFromDocs(content string) ([]string, error) {
	types := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		match := platformDocRowPattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 3 {
			continue
		}
		types = append(types, match[2])
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("未在 docs/platforms.md 中解析到平台表格")
	}
	return types, nil
}

func platformTypesFromPresets() []string {
	types := make([]string, len(platformPresets))
	for i, preset := range platformPresets {
		types[i] = preset.Type
	}
	return types
}
