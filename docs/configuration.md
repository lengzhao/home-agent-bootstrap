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

## 接入平台

bootstrap 会列出 cc-connect 支持的平台类型（飞书、钉钉、Telegram、Slack、Discord、企业微信、微信个人号、LINE、QQ、QQ 官方机器人、微博等），按序号多选后生成配置。

详见 [接入平台选择](platforms.md)。微信多账号说明见 [多微信个人号](multi-weixin.md)。

## LLM 配置

Claude Code 运行时（`claudecode`）分两类配置方式：

### 官方方式（写入 config.toml）

1. **Claude Code 自带登录**：不写 `[[providers]]`，在家庭助手工作目录运行 `claude` 完成授权。
2. **Anthropic API Key**：写入 `[[providers]]` 并在 agent 中引用。

```toml
[[providers]]
name = "anthropic"
api_key = "sk-ant-..."
agent_types = ["claudecode"]

[projects.agent.options]
provider = "anthropic"
provider_refs = ["anthropic"]
```

### 第三方 LLM（写入 ~/.zshrc）

OpenAI、OpenRouter、Kimi、火山、通义及自定义 OpenAI-compatible **不写入** `config.toml`，而是写入 `~/.zshrc` 中带标记的环境变量块，供 Claude Code 通过 `ANTHROPIC_*` 变量路由。

bootstrap 选项 3–8 会生成类似：

```bash
# >>> home-agent-bootstrap claude-code >>>
export ANTHROPIC_BASE_URL='https://api.moonshot.cn/anthropic'
export ANTHROPIC_AUTH_TOKEN='YOUR_MOONSHOT_API_KEY'
export ANTHROPIC_MODEL='kimi-k2.5'
# <<< home-agent-bootstrap claude-code <<<
```

配置完成后执行：

```bash
source ~/.zshrc
claude
```

在 Claude Code 内用 `/status` 确认模型。Kimi 详见 [官方 Claude Code 接入说明](https://platform.kimi.com/docs/guide/agent-support)。

| 预设 | 默认 ANTHROPIC_BASE_URL | 默认模型 |
|------|-------------------------|----------|
| OpenAI | https://api.openai.com/v1 | gpt-4.1 |
| OpenRouter | https://openrouter.ai/api/v1 | anthropic/claude-sonnet-4 |
| Kimi | https://api.moonshot.cn/anthropic | kimi-k2.5 |
| 火山 | https://ark.cn-beijing.volces.com/api/v3 | 安装时指定 |
| 通义 | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen-plus |

不要把 `~/.cc-connect/config.toml` 或含 API Key 的 `~/.zshrc` 片段提交到 GitHub。

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
