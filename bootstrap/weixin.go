package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/config"
	"github.com/lengzhao/home-agent-bootstrap/prompt"
)

func RunSetupWeixin(args []string) error {
	count := 1
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return errors.New("微信账号数量必须是大于 0 的数字")
		}
		count = n
	}
	configPath := cmdutil.EnvDefault("CONFIG_PATH", cmdutil.DefaultConfigPath())
	projectName := cmdutil.EnvDefault("PROJECT_NAME", defaultProjectName)
	if !cmdutil.CommandExists("cc-connect") {
		return errors.New("未检测到 cc-connect，请先运行 bootstrap")
	}
	return setupWeixinWithConfig(configPath, projectName, count)
}

func setupWeixinWithConfig(configPath, projectName string, count int) error {
	if !cmdutil.CommandExists("cc-connect") {
		return errors.New("未检测到 cc-connect，请先运行 bootstrap")
	}
	for i := 1; i <= count; i++ {
		fmt.Printf("\n开始绑定第 %d 个微信个人号\n", i)
		if err := cmdutil.RunCommand("cc-connect", "weixin", "setup", "--config", configPath, "--project", projectName, "--platform-index", strconv.Itoa(i)); err != nil {
			return err
		}
	}
	fmt.Println("\n微信绑定完成。" + weixinFirstMessageInstruction())
	return nil
}

func completeAdminRoleAfterWeixinSetup(p *prompt.Prompt, configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	adminUserID := config.FirstConfiguredAdminFrom(string(content))
	if adminUserID == "" {
		adminUserID = config.FirstBoundWeixinAllowFrom(string(content))
	}
	if adminUserID == "" {
		adminUserID = p.Ask("未能自动读取扫码用户，输入管理员微信 ilink user_id，留空则暂不写入 admin", "")
	}
	if adminUserID == "" {
		cmdutil.Warn("未写入管理员角色。之后可用 /whoami 获取 user_id，再手动补充 projects.users.roles.admin.user_ids。")
		return nil
	}
	updated := config.ApplyAdminUser(string(content), adminUserID)
	if err := cmdutil.WriteFile(configPath, []byte(updated), 0o600); err != nil {
		return err
	}
	cmdutil.Say("已写入管理员角色：" + adminUserID)
	return nil
}

func weixinFirstMessageInstruction() string {
	return "请用每个已绑定微信号给机器人发送 /whoami 或一条普通消息，确认平台用户 ID 和消息链路正常。"
}
