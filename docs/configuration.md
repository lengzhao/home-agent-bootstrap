# 配置说明

## 运行时选择

推荐默认使用 `Claude Code`：

```toml
[projects.agent]
type = "claudecode"

[projects.agent.options]
work_dir = "/Users/you/home-assistant-workspace"
mode = "auto"
```

原因：

- cc-connect 对 Claude Code 的权限审批、Cron、会话和隔离支持更完整。
- bootstrap 默认使用 `auto`，在可信本机环境下减少反复确认；更保守时可改为 `default`。
- 后续接 MCP、脚本、日历、家庭知识库时，Claude Code 的工具生态更成熟。

如果选择 `Cursor Agent`：

```toml
[projects.agent]
type = "cursor"

[projects.agent.options]
work_dir = "/Users/you/home-assistant-workspace"
mode = "default"
cmd = "agent"
```

建议模式：

- `default`：bootstrap 默认。可执行工具，但会按 Cursor Agent 策略询问。
- `ask`：只读问答，更保守。
- `plan`：只读规划，不执行。
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

1. **Claude Code 自带登录**：不写 Provider，在家庭助手工作目录运行 `claude` 完成授权。
2. **Anthropic API Key**：写入 `[[projects.agent.providers]]` 并在 agent 中引用。

```toml
[projects.agent.options]
provider = "anthropic"

[[projects.agent.providers]]
name = "anthropic"
api_key = "sk-ant-..."
```

### 第三方 LLM（写入 config.toml，并同步 shell 配置文件）

OpenAI、OpenRouter、Kimi、火山、通义及自定义 OpenAI-compatible 会写入 `config.toml` 的 `[[projects.agent.providers]]`，并在 `[projects.agent.options]` 中引用该 provider。cc-connect 启动 Claude Code 子进程时会按 Provider 配置注入环境变量；`daemon install` 本身没有单独的 env 参数，所以不要依赖交互式 shell 配置给 daemon 传密钥。

同时，bootstrap 会同步写入当前 shell 的配置文件中带标记的 `ANTHROPIC_*` 环境变量块，方便你在终端里直接运行 `claude` 调试。zsh 使用 `~/.zshrc`，bash 使用 `~/.bashrc`，其他 shell 默认写入 `~/.zshrc`。

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
source ~/.zshrc  # bash 用户使用 source ~/.bashrc
claude
```

在 Claude Code 内用 `/status` 确认模型。Kimi 详见 [官方 Claude Code 接入说明](https://platform.kimi.com/docs/guide/agent-support)。

如果聊天里返回 `Not logged in. Please run /login`，优先检查生成的配置是否包含：

```toml
[projects.agent.options]
provider = "kimi"

[[projects.agent.providers]]
name = "kimi"
api_key = "..."
base_url = "https://api.moonshot.cn/anthropic"
model = "kimi-k2.5"
```

修改配置后重新执行 `home-agent-bootstrap start` 或 `cc-connect daemon install --config ~/.cc-connect/config.toml --force && cc-connect daemon restart`。

## 排障与迁移

检查依赖、配置结构、Provider 引用、cc-connect 版本和 daemon 状态：

```bash
home-agent-bootstrap doctor
```

如果配置文件仍是旧版顶层 `[[providers]]` 或包含 `provider_refs`，可自动迁移到 `[[projects.agent.providers]]`：

```bash
home-agent-bootstrap migrate-config
home-agent-bootstrap doctor
```

迁移前会自动备份现有配置文件。

| 预设 | 默认 ANTHROPIC_BASE_URL | 默认模型 |
|------|-------------------------|----------|
| OpenAI | https://api.openai.com/v1 | gpt-4.1 |
| OpenRouter | https://openrouter.ai/api/v1 | anthropic/claude-sonnet-4 |
| Kimi | https://api.moonshot.cn/anthropic | kimi-k2.5 |
| 火山 | https://ark.cn-beijing.volces.com/api/v3 | 安装时指定 |
| 通义 | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen-plus |

不要把 `~/.cc-connect/config.toml` 或含 API Key 的 shell 配置片段提交到 GitHub。

## 会话和显示默认值

生成配置默认关闭空闲自动换 session：

```toml
[[projects]]
reset_on_idle_mins = 0
```

这样 Claude Code 运行时不会因为长时间没有用户消息而自动切到新 session。需要恢复 cc-connect 上游默认行为时，可改成 `30` 或删除该项。

聊天平台默认隐藏工具调用进度，避免每次工具调用都刷屏：

```toml
[display]
mode = "compact"
thinking_messages = false
tool_messages = false
```

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

## Heartbeat

生成的配置里会包含注释状态的 Heartbeat 示例。bootstrap 完成后会提示启用步骤：

1. 完成首次对话
2. 在聊天里发送 `/status` 获取 `session_key`
3. 编辑 `config.toml`，取消 `[projects.heartbeat]` 注释并填入 `session_key`
4. 执行 `cc-connect daemon restart`

示例配置：

```toml
[projects.heartbeat]
enabled = true
interval_mins = 60
session_key = "weixin:REPLACE_WITH_SESSION_KEY"
only_when_idle = true
silent = true
timeout_mins = 10
prompt = "读取 HEARTBEAT.md，检查今天家庭提醒、待办、异常事项，只在需要时提醒。"
```

## 权限模板

bootstrap 可选择三种 member 角色权限模板：

| 模板 | 说明 |
|------|------|
| admin-only | 仅管理员可执行工具和高危命令 |
| family-readonly | 家人只读问答，禁用 cron/provider 等管理命令 |
| family-remind | 默认可提醒，禁用 shell/restart 等高危命令 |

非交互模式可通过 `PERMISSION_TEMPLATE` 指定。

## 非交互 bootstrap

完整环境变量说明与示例请运行：

```bash
home-agent-bootstrap help
```

设置 `NONINTERACTIVE=1` 可跳过问答，配合环境变量预设：

| 变量 | 说明 |
|------|------|
| CONFIG_PATH | 配置文件路径 |
| WORKSPACE | 工作目录 |
| PROJECT_NAME | project 名称 |
| AGENT_TYPE | claudecode 或 cursor |
| AGENT_MODE | Agent 权限模式 |
| PERMISSION_TEMPLATE | admin-only / family-readonly / family-remind |
| PLATFORM_CHOICES | 平台序号，如 7 或 1,7 |
| LLM_CHOICE | Claude Code LLM 选项 1-9 |
| LLM_API_KEY | 非交互 Provider API Key |
| SKIP_WEIXIN_SETUP=1 | 跳过微信扫码 |
| OVERWRITE_CONFIG=1 | 覆盖已有配置 |

示例：

```bash
NONINTERACTIVE=1 \
  WORKSPACE="$HOME/home-assistant-workspace" \
  PERMISSION_TEMPLATE=family-remind \
  PLATFORM_CHOICES=7 \
  LLM_CHOICE=1 \
  SKIP_WEIXIN_SETUP=1 \
  home-agent-bootstrap bootstrap
```

## 工作区模板同步

bootstrap 会补全缺失的工作区文件，不会覆盖已有内容。工作区根目录的 `VERSION` 表示模板版本；若低于当前 bootstrap 版本，会提示有新模板可用。

独立命令：

```bash
# 查看版本与缺失文件
home-agent-bootstrap workspace-status

# 补全缺失文件（不覆盖已有内容）
home-agent-bootstrap sync-workspace

# 预览将新增的文件
home-agent-bootstrap sync-workspace --dry-run
```

可通过 `WORKSPACE` 指定工作目录，默认 `~/home-assistant-workspace`。
