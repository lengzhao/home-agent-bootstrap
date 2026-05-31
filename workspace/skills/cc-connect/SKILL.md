---
name: cc-connect
description: 使用 cc-connect CLI 管理家庭助手运行时，包括消息回传、文件或图片发送、cron 定时任务、微信个人号绑定、daemon 状态、日志和 doctor 诊断。遇到用户要求定时提醒、周期总结、查看或修改 cron、把生成文件发回聊天、绑定微信、检查服务是否在线、排查 cc-connect 或微信回复异常时使用。涉及隐私文件、daemon 重启或现实世界影响时先确认。
---

# cc-connect CLI

把 `cc-connect` 当作家庭助手运行时的受控入口。只有任务需要消息通道、定时任务、微信绑定或服务诊断时才调用；普通问答不需要调用。

## 使用前判断

- 明确用户要做什么：发消息、发文件、创建 cron、管理微信绑定，还是诊断 daemon。
- 先确认高风险或有副作用的动作：发送家庭隐私文件、删除或修改 cron、重启 daemon、向外部系统发送数据。
- 不要通过 cron 自动执行高风险动作。高风险需求只能创建提醒确认任务。
- 排障时先用只读命令观察状态和日志，再建议会改变服务状态的操作。

## Cron 定时任务

创建或修改定时任务前先确认：

- 时间和时区。
- 任务内容。
- 触发后是否主动发消息，以及发给谁。
- 失败时怎么处理。
- 是否涉及高风险动作或家庭隐私。

常用命令：

```bash
cc-connect cron add --cron "0 6 * * *" --prompt "总结今天家庭提醒" --desc "每日家庭提醒"
cc-connect cron list
cc-connect cron edit <job-id> <field> <value>
cc-connect cron del <job-id>
```

如果用户描述的是自然语言时间，先转成明确 cron 表达式并说明含义，再创建。删除或大幅修改已有任务前先展示目标任务，避免误操作。

## 文件和消息回传

生成文件后，如果用户要求“发我”“发到微信”“回传报告”，使用：

```bash
cc-connect send --message "消息内容"
cc-connect send --file <path>
cc-connect send --image <path>
```

发送前检查路径是否存在且是用户期望的文件。发送家庭隐私、账单、证件、家庭成员信息、日志或配置片段前必须确认。不要发送 token、账号密码或完整配置文件。

## 微信个人号

扫码绑定：

```bash
cc-connect weixin setup --config ~/.cc-connect/config.toml --project home --platform-index 1
```

已有 token 时：

```bash
cc-connect weixin bind --config ~/.cc-connect/config.toml --project home --platform-index 1 --token '<token>'
```

多微信号按配置顺序选择 `--platform-index`。绑定后提醒用户发送 `/whoami` 或一条普通消息，确认平台用户 ID 和消息链路正常。生产环境不要长期使用空白或通配的 `allow_from`，建议绑定后通过 `/whoami` 获取 user id 并改成白名单。若回复里出现 `Not logged in. Please run /login`，优先检查 Claude Code 登录态或自定义 LLM Provider 配置，不要把它当成微信绑定步骤。

## Daemon 和诊断

优先使用只读诊断命令：

```bash
cc-connect daemon status
cc-connect daemon logs -f
cc-connect doctor
```

只有在用户确认或日志明确显示需要重启时再执行：

```bash
cc-connect daemon restart
```

重启前说明影响：短时间内无法接收或回复消息，正在执行的任务可能被中断。

## Token 泄露恢复

若怀疑 management、bridge、webhook 或微信 token 泄露：

1. 停止 daemon：`cc-connect daemon stop`
2. 重新生成 `config.toml` 中的 token，或重新运行 bootstrap 覆盖配置
3. 微信个人号重新执行 `cc-connect weixin setup` 或 `home-agent-bootstrap setup-weixin N`
4. 检查审计日志：`tail -n 100 ~/.cc-connect/audit/events.log`
5. 恢复服务：`home-agent-bootstrap start`
