# 接入平台选择

bootstrap 会展示与 [cc-connect](https://github.com/chenhg5/cc-connect) 对齐的平台列表，按序号多选后生成对应的 `[[projects.platforms]]` 配置块。

## 支持的平台

| 序号 | type | 名称 | 连接方式 | 公网 IP |
|------|------|------|----------|---------|
| 1 | feishu | 飞书 / Lark | WebSocket | 否 |
| 2 | dingtalk | 钉钉 | Stream | 否 |
| 3 | telegram | Telegram | Long Polling | 否 |
| 4 | slack | Slack | Socket Mode | 否 |
| 5 | discord | Discord | Gateway | 否 |
| 6 | wecom | 企业微信 | WebSocket / Webhook | 视模式 |
| 7 | weixin | 微信个人号 | ilink 长轮询 | 否 |
| 8 | line | LINE | Webhook | 是 |
| 9 | qq | QQ (NapCat/OneBot) | WebSocket | 否 |
| 10 | qqbot | QQ 官方机器人 | WebSocket | 否 |
| 11 | weibo | 微博 | WebSocket | 否 |

示例：

- 只接微信：`7`
- 飞书 + 微信：`1,7`

## 微信个人号

微信仍支持同一 project 下配置多个 `weixin` 平台，并可使用 bootstrap 内扫码绑定。详见 [多微信个人号](multi-weixin.md)。

## 其他平台

非微信平台在 bootstrap 中会收集主要凭证字段（如 `app_id`、`token` 等），也可在生成配置后使用 cc-connect 自带的 setup 命令补全，例如：

```bash
cc-connect feishu setup --config ~/.cc-connect/config.toml --project home
cc-connect dingtalk setup --config ~/.cc-connect/config.toml --project home
```

其中企业微信是否需要公网 URL 取决于实际接入模式，bootstrap 只在平台列表里标注“视模式”，具体以 cc-connect 上游文档为准。

各平台完整配置说明请参考 cc-connect 上游文档（`docs/feishu.md`、`docs/telegram.md` 等）。

## 管理员 ID

- 若选择了微信，可填写管理员微信 ilink `user_id`，或在扫码后由 bootstrap 自动回填。
- 其他平台可在首次对话后发送 `/whoami` 获取 user id，再写入 `admin_from` 和 `projects.users.roles.admin`。
