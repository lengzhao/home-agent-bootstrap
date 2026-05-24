# 配置说明

## 运行时选择

推荐默认使用 `Claude Code`：

```toml
[projects.agent]
type = "claudecode"

[projects.agent.options]
work_dir = "/Users/you/home-assistant-workspace"
mode = "default"
model = "sonnet"
```

原因：

- cc-connect 对 Claude Code 的权限审批、Cron、会话和隔离支持更完整。
- 家庭助手需要明确的人类确认流程，`default` 模式更安全。
- 后续接 MCP、脚本、日历、家庭知识库时，Claude Code 的工具生态更成熟。

如果选择 `Cursor Agent`：

```toml
[projects.agent]
type = "cursor"

[projects.agent.options]
work_dir = "/Users/you/home-assistant-workspace"
mode = "ask"
cmd = "agent"
```

建议模式：

- `ask`：只读问答，最安全。
- `plan`：只读规划，不执行。
- `default`：可执行但会询问。
- `force`：自动批准所有工具调用，不建议用于家庭场景。

## 管理后台

安装器会启用：

```toml
[management]
enabled = true
port = 9820
token = "随机生成"
```

访问地址：

```text
http://localhost:9820
```

只建议绑定本机或可信内网，不要把管理后台直接暴露到公网。

## Bridge 和 Webhook

`bridge` 用于外部适配器接入，`webhook` 用于外部事件触发：

```toml
[bridge]
enabled = true
port = 9810
token = "随机生成"
path = "/bridge/ws"

[webhook]
enabled = true
port = 9111
token = "随机生成"
path = "/hook"
```

未来接入 Home Assistant 时，可以让 Home Assistant 或本地 `home-tools` 服务调用 webhook。

## 家庭工作目录

默认目录：

```text
~/home-assistant-workspace
```

建议至少包含：

- `CLAUDE.md`：Claude Code 项目级指令，要求先读家庭规则并遵守安全边界。
- `HOME.md`：长期家庭规则、风险分级、家庭偏好。
- `HEARTBEAT.md`：周期巡检规则。
- `members.md`：家庭成员和称呼，不要写敏感隐私。
- `devices.md`：设备清单和控制风险等级。
- `tasks.md`：家庭长期待办。

## LLM 配置

全新 Mac 上推荐按 bootstrap 提示选择：

1. 使用 Claude Code 自带登录。安装器会在家庭助手工作目录中启动 `claude`，按官方流程完成登录和信任工作目录。
2. 在安装器里输入 `ANTHROPIC_API_KEY`。脚本会把 `anthropic` Provider 写入本机 `~/.cc-connect/config.toml`，并在项目里引用该 Provider。
3. 在安装器里输入 `OPENAI_API_KEY`。脚本会把 `openai` Provider 写入本机配置，并允许设置 `base_url` 和 `model`。
4. 使用自定义 OpenAI-compatible Provider。脚本会询问 Provider 名称、API Key、`base_url` 和 `model`。

示例：

```toml
[[providers]]
name = "openai"
api_key = "sk-..."
base_url = "https://api.openai.com/v1"
model = "gpt-4.1"
agent_types = ["claudecode"]

[projects.agent.options]
provider = "openai"
provider_refs = ["openai"]
```

不要把生成后的 `~/.cc-connect/config.toml` 提交到 GitHub。

## 启动服务

```bash
cc-connect daemon install --config ~/.cc-connect/config.toml --force
cc-connect daemon start
cc-connect daemon logs -f
```

如果修改了配置：

```bash
cc-connect daemon restart
```
