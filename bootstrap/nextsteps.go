package bootstrap

import (
	"fmt"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/platforms"
	"github.com/lengzhao/home-agent-bootstrap/shellenv"
	"github.com/lengzhao/home-agent-bootstrap/workspacesync"
)

func printNextSteps(configPath, projectName string, platformBlocks []platforms.Block, agentType string, weixinSetupDone bool) {
	fmt.Printf("\n配置已生成：%s\n", configPath)
	fmt.Println("\n下一步：")
	if agentType == "claudecode" {
		fmt.Println("\n1. 确认 Claude Code 可登录：")
		fmt.Printf("   source %s   # 若使用第三方 LLM 环境变量\n", shellenv.ProfilePath())
		fmt.Println("   claude")
	} else {
		fmt.Println("\n1. 确认 Cursor Agent 可用：")
		fmt.Println("   agent --help")
	}

	step := 2
	weixinCount := platforms.CountWeixin(platformBlocks)
	if weixinCount > 0 {
		if weixinSetupDone {
			fmt.Printf("\n%d. 微信扫码绑定已完成。%s\n", step, weixinFirstMessageInstruction())
		} else {
			fmt.Printf("\n%d. 逐个扫码绑定微信个人号：\n", step)
			fmt.Printf("   PROJECT_NAME=%s CONFIG_PATH=%q %s setup-weixin %d\n", projectName, configPath, cmdutil.AppName, weixinCount)
		}
		step++
	}

	hints := platforms.SetupHints(platformBlocks)
	if len(hints) > 0 {
		fmt.Printf("\n%d. 其他平台可使用 cc-connect setup 命令补全凭证：\n", step)
		for _, hint := range hints {
			fmt.Println(hint)
		}
		step++
	}

	fmt.Printf("\n%d. 启动服务：\n", step)
	fmt.Printf("   %s start\n", cmdutil.AppName)
	step++
	fmt.Printf("\n%d. Web 管理后台：\n", step)
	fmt.Println("   http://localhost:9820")
	step++
	fmt.Printf("\n%d. 启用 Heartbeat（可选）：\n", step)
	fmt.Println("   " + workspacesync.HeartbeatSetupGuide())

	if weixinCount > 0 && !weixinSetupDone {
		fmt.Println("\n注意：" + weixinFirstMessageInstruction())
	}
}
