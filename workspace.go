package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const workspaceTemplateVersion = "2"

type workspaceSyncReport struct {
	Added            []string
	Skipped          []string
	InstalledVersion string
	ExistingVersion  string
}

func readWorkspaceVersion(workspace string) string {
	content, err := os.ReadFile(filepath.Join(workspace, "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func syncWorkspaceFiles(workspace string) (workspaceSyncReport, error) {
	report := workspaceSyncReport{InstalledVersion: workspaceTemplateVersion}
	report.ExistingVersion = readWorkspaceVersion(workspace)

	err := fs.WalkDir(workspaceTemplates, "workspace", func(path string, entry fs.DirEntry, err error) error {
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
			report.Skipped = append(report.Skipped, rel)
			return nil
		}
		content, err := workspaceTemplates.ReadFile(path)
		if err != nil {
			return err
		}
		if err := writeFile(target, content, 0o644); err != nil {
			return err
		}
		report.Added = append(report.Added, rel)
		return nil
	})
	if err != nil {
		return report, err
	}

	versionPath := filepath.Join(workspace, "VERSION")
	if !exists(versionPath) {
		if err := writeFile(versionPath, []byte(workspaceTemplateVersion+"\n"), 0o644); err != nil {
			return report, err
		}
		report.Added = append(report.Added, "VERSION")
	}
	return report, nil
}

func printWorkspaceSyncReport(report workspaceSyncReport) {
	if report.ExistingVersion != "" && report.ExistingVersion != workspaceTemplateVersion {
		fmt.Printf("\n检测到工作区模板版本 %s，当前 bootstrap 模板版本 %s。已补全缺失文件，不会覆盖已有内容。\n",
			report.ExistingVersion, workspaceTemplateVersion)
	}
	if len(report.Added) > 0 {
		fmt.Println("\n已新增工作区文件：")
		for _, rel := range report.Added {
			fmt.Printf("  + %s\n", rel)
		}
	}
	if len(report.Skipped) > 0 && len(report.Added) == 0 {
		fmt.Println("\n工作区文件已存在，未覆盖用户内容。")
	}
}

func heartbeatSetupGuide() string {
	return "完成首次对话后，在聊天里发送 /status 获取 session_key，编辑 config.toml 取消 [projects.heartbeat] 注释并填入 session_key，然后 cc-connect daemon restart。"
}

func compareVersionAtLeast(current, minimum string) bool {
	currentParts := parseVersionParts(current)
	minimumParts := parseVersionParts(minimum)
	length := len(currentParts)
	if len(minimumParts) > length {
		length = len(minimumParts)
	}
	for i := 0; i < length; i++ {
		cur := 0
		min := 0
		if i < len(currentParts) {
			cur = currentParts[i]
		}
		if i < len(minimumParts) {
			min = minimumParts[i]
		}
		if cur > min {
			return true
		}
		if cur < min {
			return false
		}
	}
	return true
}

func parseVersionParts(version string) []int {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if idx := strings.IndexFunc(version, func(r rune) bool { return r >= '0' && r <= '9' }); idx >= 0 {
		version = version[idx:]
	}
	if version == "" {
		return nil
	}
	base := strings.FieldsFunc(version, func(r rune) bool {
		return r == '.' || r == '-' || r == '+'
	})
	parts := make([]int, 0, len(base))
	for _, piece := range base {
		if piece == "" {
			continue
		}
		n, err := strconv.Atoi(piece)
		if err != nil {
			break
		}
		parts = append(parts, n)
	}
	return parts
}
