package workspacesync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
)

const TemplateVersion = "2"

type Report struct {
	Added            []string
	Skipped          []string
	InstalledVersion string
	ExistingVersion  string
	DryRun           bool
}

type Status struct {
	Workspace        string
	InstalledVersion string
	TemplateVersion  string
	Outdated         bool
	Missing          []string
}

type Options struct {
	DryRun bool
}

func Analyze(templates fs.FS, workspace string) (Status, error) {
	status := Status{
		Workspace:        workspace,
		InstalledVersion: readVersion(workspace),
		TemplateVersion:  TemplateVersion,
	}
	status.Outdated = status.InstalledVersion != "" && status.InstalledVersion != TemplateVersion

	err := fs.WalkDir(templates, "workspace", func(path string, entry fs.DirEntry, err error) error {
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
		if !cmdutil.Exists(target) {
			status.Missing = append(status.Missing, rel)
		}
		return nil
	})
	if err != nil {
		return status, err
	}
	if !cmdutil.Exists(filepath.Join(workspace, "VERSION")) {
		status.Missing = append(status.Missing, "VERSION")
	}
	return status, nil
}

func PrintStatus(status Status) {
	fmt.Printf("工作目录         %s\n", status.Workspace)
	fmt.Printf("已安装模板版本   %s\n", displayVersion(status.InstalledVersion))
	fmt.Printf("当前模板版本   %s\n", status.TemplateVersion)
	if status.Outdated {
		fmt.Println("状态             模板有更新，运行 sync-workspace 补全缺失文件")
	} else if status.InstalledVersion == "" {
		fmt.Println("状态             未检测到 VERSION，可能尚未初始化工作区")
	} else {
		fmt.Println("状态             版本一致")
	}
	if len(status.Missing) > 0 {
		fmt.Println("\n缺失文件：")
		for _, rel := range status.Missing {
			fmt.Printf("  - %s\n", rel)
		}
	} else {
		fmt.Println("\n缺失文件         无")
	}
}

func displayVersion(version string) string {
	if version == "" {
		return "（未安装）"
	}
	return version
}

func Sync(templates fs.FS, workspace string, opts Options) (Report, error) {
	report := Report{
		InstalledVersion: TemplateVersion,
		ExistingVersion:  readVersion(workspace),
		DryRun:           opts.DryRun,
	}

	err := fs.WalkDir(templates, "workspace", func(path string, entry fs.DirEntry, err error) error {
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
		if cmdutil.Exists(target) {
			report.Skipped = append(report.Skipped, rel)
			return nil
		}
		report.Added = append(report.Added, rel)
		if opts.DryRun {
			return nil
		}
		content, err := fs.ReadFile(templates, path)
		if err != nil {
			return err
		}
		if err := cmdutil.WriteFile(target, content, 0o644); err != nil {
			return err
		}
		report.Added = append(report.Added, rel)
		return nil
	})
	if err != nil {
		return report, err
	}

	versionPath := filepath.Join(workspace, "VERSION")
	if !cmdutil.Exists(versionPath) {
		report.Added = append(report.Added, "VERSION")
		if !opts.DryRun {
			if err := cmdutil.WriteFile(versionPath, []byte(TemplateVersion+"\n"), 0o644); err != nil {
				return report, err
			}
		}
	}
	return report, nil
}

func readVersion(workspace string) string {
	content, err := os.ReadFile(filepath.Join(workspace, "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func PrintReport(report Report) {
	if report.DryRun {
		fmt.Println("\n（dry-run 模式，未写入任何文件）")
	}
	if report.ExistingVersion != "" && report.ExistingVersion != TemplateVersion {
		fmt.Printf("\n检测到工作区模板版本 %s，当前 bootstrap 模板版本 %s。已补全缺失文件，不会覆盖已有内容。\n",
			report.ExistingVersion, TemplateVersion)
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

func HeartbeatSetupGuide() string {
	return "完成首次对话后，在聊天里发送 /status 获取 session_key，编辑 config.toml 取消 [projects.heartbeat] 注释并填入 session_key，然后 cc-connect daemon restart。"
}

func CompareVersionAtLeast(current, minimum string) bool {
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
