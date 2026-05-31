# 模块拆分设计

## 背景

P1 完成后代码量约 3500 行，全部在 `package main` 多文件中，不利于测试与维护。2026-05 进行模块化拆分。

## 结构

根目录仅保留 `main.go`（入口 + embed）。业务按领域拆包：

| 包 | 职责 |
|---|---|
| `bootstrap/` | CLI 命令分发、bootstrap 编排、安装、微信绑定 |
| `config/` | 配置渲染、doctor、migrate、admin 角色 |
| `platforms/` | 平台预设 |
| `providers/` | LLM Provider |
| `permissions/` | 权限模板 |
| `shellenv/` | Claude Code shell 环境变量 |
| `workspacesync/` | 工作区模板同步与版本 |
| `prompt/` | 交互问答 |
| `cmdutil/` | 通用工具 |

`workspace/`、`templates/` 仍在仓库根目录，由 `main.go` embed。

## 新增命令

- `sync-workspace` — 补全缺失工作区文件，支持 `--dry-run`
- `workspace-status` — 对比 VERSION 与缺失文件

## 测试约定

子包测试通过 `os.DirFS("..")` 读取根目录 embed 内容。后续可抽 `assets/` 包统一 embed。

## 后续

- 配置 migrate/doctor 逐步引入 TOML 结构化解析
- bootstrap 主路径集成测试
- 可选 CLI flags 补充 env 预设
