# 多微信个人号配置

本文档仅说明微信个人号（`weixin`）多账号场景。其他接入平台见 [接入平台选择](platforms.md)。

cc-connect 支持在同一个 `[[projects]]` 下配置多个 `weixin` 平台。每个微信个人号需要独立的：

- `token`
- `account_id`
- 本地状态目录

## 配置示例

```toml
[[projects.platforms]]
type = "weixin"

[projects.platforms.options]
token = ""
base_url = "https://ilinkai.weixin.qq.com"
cdn_base_url = "https://novac2c.cdn.weixin.qq.com/c2c"
allow_from = ""
account_id = "wx-main"
long_poll_timeout_ms = 35000

[[projects.platforms]]
type = "weixin"

[projects.platforms.options]
token = ""
base_url = "https://ilinkai.weixin.qq.com"
cdn_base_url = "https://novac2c.cdn.weixin.qq.com/c2c"
allow_from = ""
account_id = "wx-family"
long_poll_timeout_ms = 35000
```

## 扫码绑定

bootstrap 默认会在生成配置后按平台顺序扫码绑定。第一个 `weixin` 块是 `--platform-index 1`，第二个是 `--platform-index 2`。

如果 bootstrap 时跳过扫码，可之后手动执行：

```bash
cc-connect weixin setup --config ~/.cc-connect/config.toml --project home --platform-index 1
cc-connect weixin setup --config ~/.cc-connect/config.toml --project home --platform-index 2
```

或者使用本项目脚本：

```bash
go run . setup-weixin 2
```

扫码完成后，bootstrap 会优先读取第一个已回填的 `allow_from`，并写入 `projects.users.roles.admin.user_ids`。如果配置里没有回填，会提示输入管理员微信 ilink `user_id`；继续留空时会省略 admin role，避免生成 `user_ids = []` 的无效配置。

## 已有 Token

如果你已有 ilink Bearer Token：

```bash
cc-connect weixin bind --config ~/.cc-connect/config.toml --project home --platform-index 1 --token '<token1>'
cc-connect weixin bind --config ~/.cc-connect/config.toml --project home --platform-index 2 --token '<token2>'
```

## account_id

`account_id` 用来隔离状态目录：

```text
<data_dir>/weixin/<project>/<account_id>/
```

例如：

```text
~/.cc-connect/weixin/home/wx-main/
~/.cc-connect/weixin/home/wx-family/
```

不要让多个微信号共用同一个 `account_id`。

## allow_from

生产环境不要长期使用空值或 `"*"`。

建议流程：

1. 首次配置时可先留空，让 `weixin setup` 尝试回填扫码用户。
2. 启动后在微信里发送 `/whoami` 获取 user id。
3. 将 `allow_from` 改为明确白名单。
4. 重启 daemon。

## 首次对话

每个微信号扫码成功后，建议先给机器人发送：

```text
/whoami
```

或者发送一条普通消息，确认平台用户 ID 和消息链路正常。

如果回复里出现 `Not logged in. Please run /login`，这通常是 Claude Code 运行时没有拿到登录态或自定义 LLM 环境变量，不是微信个人号绑定步骤。请优先检查 `config.toml` 里的 `[projects.agent.options] provider` 和 `[[projects.agent.providers]]` 是否正确生成，并重装或重启 daemon。
