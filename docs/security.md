# 安全建议

家庭助手的目标不是“让 Agent 能做任何事”，而是“让它能安全、可审计、可回滚地做明确的事”。

## 权限模式

Claude Code 推荐：

```toml
[projects.agent.options]
mode = "default"
```

Cursor Agent 推荐：

```toml
[projects.agent.options]
mode = "ask"
```

不建议家庭场景默认使用：

- Claude Code `bypassPermissions` 或 `yolo`
- Cursor Agent `force`
- 任何自动批准全部工具调用的模式

## 管理员权限

不要设置：

```toml
admin_from = "*"
```

建议只填你自己的微信 user id：

```toml
admin_from = "your_user_id@im.wechat"
```

家人默认使用 `member` 角色，并禁用高危命令：

```toml
[projects.users.roles.member]
user_ids = ["*"]
disabled_commands = ["shell", "show", "dir", "restart", "upgrade", "commands"]
```

## Token 管理

不要提交：

- `~/.cc-connect/config.toml`
- `~/.cc-connect/weixin/`
- 微信 ilink token
- Management token
- Bridge token
- Webhook token
- Claude/Cursor 登录凭据

如果怀疑 token 泄露：

1. 停止 daemon。
2. 重新生成 token。
3. 重新扫码绑定微信。
4. 检查 `~/.cc-connect/audit/events.log`。

## 端口暴露

默认仅建议本机访问：

- Management API: `9820`
- Bridge: `9810`
- Webhook: `9111`

不要直接暴露到公网。如果必须远程访问，建议使用 VPN、Tailscale、Cloudflare Access 或其他有认证的通道。

## 高风险动作

以下动作必须二次确认：

- 开门、门锁、摄像头、门铃
- 支付、转账、下单
- 删除文件、清空数据、格式化磁盘
- 关机、重启、修改网络
- 向外部发送家庭隐私数据

## 审计

安装器默认配置了 hooks 审计日志：

```text
~/.cc-connect/audit/events.log
```

建议定期查看：

```bash
tail -n 100 ~/.cc-connect/audit/events.log
```
