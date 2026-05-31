package bootstrap

import (
	"fmt"
	"io/fs"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/workspacesync"
)

func workspacePath() string {
	return cmdutil.EnvDefault("WORKSPACE", cmdutil.DefaultWorkspacePath())
}

func RunWorkspaceStatus(templates fs.FS) error {
	status, err := workspacesync.Analyze(templates, workspacePath())
	if err != nil {
		return err
	}
	workspacesync.PrintStatus(status)
	return nil
}

func RunSyncWorkspace(templates fs.FS, args []string) error {
	dryRun := false
	for _, arg := range args {
		switch arg {
		case "--dry-run", "-n":
			dryRun = true
		default:
			return fmt.Errorf("未知参数 %q，用法 sync-workspace [--dry-run]", arg)
		}
	}

	workspace := workspacePath()
	report, err := workspacesync.Sync(templates, workspace, workspacesync.Options{DryRun: dryRun})
	if err != nil {
		return err
	}
	workspacesync.PrintReport(report)
	if dryRun {
		cmdutil.Say("预览完成，去掉 --dry-run 后执行写入")
	} else if len(report.Added) > 0 {
		cmdutil.Say("工作区同步完成")
	} else {
		cmdutil.Say("工作区已是最新，无需补全")
	}
	return nil
}
