# 全新 Mac 引导

目标：一台全新 Mac 获取本仓库或 Release 二进制后，运行 Go 引导程序，按提示完成基础依赖、cc-connect、运行时 Agent、LLM 配置和微信个人号绑定。

配置模板和工作区模板已经通过 Go `embed` 打进二进制。Release 场景只需要下载并运行一个 `cc-home` 文件。

## 安装器会处理什么

`cc-home bootstrap` 会依次引导：

1. 检查并安装 Xcode Command Line Tools。
2. 检查并安装 Homebrew。
3. 使用 Homebrew 安装 Node.js/npm 和 ffmpeg。
4. 使用 npm 安装 `cc-connect`。
5. 按选择安装 `Claude Code` 或 `Cursor Agent`。
6. 引导配置 LLM：
   - Claude Code 登录。
   - 或 Anthropic API Key 写入本机 `config.toml`。
7. 生成 `~/.cc-connect/config.toml`。
8. 创建家庭助手工作目录。
9. 写入 `HOME.md`、`HEARTBEAT.md`、`CLAUDE.md`。
10. 按数量生成多个微信个人号平台块。

## 执行

```bash
go run . bootstrap
```

或先构建本地二进制：

```bash
go build -o cc-home .
./cc-home bootstrap
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

LLM 配置有两种方式：

- 使用 `claude` 交互式登录，适合个人订阅或 Claude Code 常规使用。
- 输入 `ANTHROPIC_API_KEY`，安装器会写入本机 `~/.cc-connect/config.toml` 的 `[[providers]]`。

不要把生成后的 `config.toml` 提交到 GitHub。

## Cursor Agent 配置建议

Cursor Agent 可以作为运行时，但更推荐用于维护这个家庭助手项目。若选择 Cursor Agent，默认模式使用：

```toml
[projects.agent.options]
mode = "ask"
```

不要在家庭场景默认使用 `force`。

## 微信绑定

安装器只生成平台块，不会自动扫码。生成配置后执行：

```bash
go run . setup-weixin 2
```

其中 `2` 是微信个人号数量。

## 启动服务

```bash
go run . start
```

查看日志：

```bash
cc-connect daemon logs -f
```
