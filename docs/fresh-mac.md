# 全新 Mac 引导

目标：一台全新 Mac 获取本仓库或 Release 二进制后，运行 Go 引导程序，按提示完成基础依赖、cc-connect、运行时 Agent、LLM 配置和微信个人号绑定。

配置模板和工作区模板已经通过 Go `embed` 打进二进制。Release 场景只需要下载并运行一个 `home-agent-bootstrap` 文件。

## 安装器会处理什么

`home-agent-bootstrap bootstrap` 会依次引导：

1. 检查并安装 Xcode Command Line Tools。
2. 检查并安装 Homebrew。
3. 使用 Homebrew 安装 Node.js/npm 和 ffmpeg。
4. 使用 npm 安装 `cc-connect`。
5. 按选择安装 `Claude Code` 或 `Cursor Agent`。
6. 引导配置 LLM（Claude Code 登录或 Anthropic/OpenAI/OpenRouter/Kimi/火山/通义等预设 Provider）。
7. 从 cc-connect 支持列表中选择接入平台（可多选）。
8. 生成 `~/.cc-connect/config.toml`。
9. 创建家庭助手工作目录。
10. 写入 `HOME.md`、`HEARTBEAT.md`、`CLAUDE.md`。
11. 按选择生成各平台 `[[projects.platforms]]` 块；微信可配置多个个人号。
12. 若包含微信，默认立即逐个扫码绑定。
13. 扫码后自动把第一个已回填的 `allow_from` 写入 `projects.users.roles.admin.user_ids`。

## 执行

```bash
go run . bootstrap
```

或先构建本地二进制：

```bash
go build -o home-agent-bootstrap .
./home-agent-bootstrap bootstrap
```

如果你已经装好 Homebrew、Node.js/npm、ffmpeg 等依赖，可以跳过系统依赖安装：

```bash
INSTALL_DEPS=0 go run . bootstrap
```

## Claude Code 配置建议

推荐第一阶段使用 Claude Code 作为家庭助手运行时：

```toml
[projects.agent]
type = "claudecode"

[projects.agent.options]
mode = "default"
```

LLM 配置方式：

- 使用 `claude` 交互式登录，适合个人订阅或 Claude Code 常规使用。
- 官方 Anthropic API Key 写入 `config.toml`；其他第三方 LLM（OpenAI、OpenRouter、Kimi、火山、通义等）也会写入 `config.toml` Provider，并同步写入 `~/.zshrc` 方便直接运行 `claude`。
- 使用自定义 OpenAI-compatible Provider。

接入平台见 [接入平台选择](platforms.md)。

不要把生成后的 `config.toml` 提交到 GitHub。

## Cursor Agent 配置建议

Cursor Agent 可以作为运行时，但更推荐用于维护这个家庭助手项目。若选择 Cursor Agent，默认模式使用：

```toml
[projects.agent.options]
mode = "ask"
```

不要在家庭场景默认使用 `force`。

## 微信绑定

bootstrap 默认会在生成配置后立即逐个扫码绑定微信个人号：

```bash
go run . bootstrap
```

如果当时跳过扫码，之后仍可执行：

```bash
go run . setup-weixin 2
```

其中 `2` 是微信个人号数量。扫码完成后，安装器会优先读取配置中第一个已回填的 `allow_from`，并写入 `projects.users.roles.admin.user_ids`。如果没有读取到，会提示手动输入管理员微信 ilink `user_id`；继续留空时会省略 admin role，避免生成 `user_ids = []` 的无效配置。

扫码绑定后，请用每个已绑定微信号先给机器人发送 `/login`。完成登录后，再发送普通消息或 `/whoami`，cc-connect 才能缓存 `context_token` 并正常回复。

## 启动服务

```bash
go run . start
```

查看日志：

```bash
cc-connect daemon logs -f
```

默认生成的配置会关闭空闲自动换 session，并隐藏工具调用进度消息；如果希望恢复上游默认的 30 分钟自动换 session，可在 `config.toml` 中把 `reset_on_idle_mins` 改为 `30`。
