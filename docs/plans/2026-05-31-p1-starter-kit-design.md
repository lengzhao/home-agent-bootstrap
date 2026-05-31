# P1 Starter Kit Design

## 目标

在 P0 安装器可靠基础上，增强默认工作区、bootstrap 自动化能力和家庭权限模板。

## 范围

- workspace 模板版本号与缺失文件补全提示
- Heartbeat 启用向导
- 丰富 members/devices/tasks 示例与家庭场景技能
- bootstrap 权限模板（admin-only / family-readonly / family-remind）
- 非交互 bootstrap（NONINTERACTIVE / 环境变量预设）
- 关键依赖安装失败时明确报错
- doctor 检测 cc-connect 最低版本
- token 泄露恢复流程文档与 skill 补充

## 非交互环境变量

| 变量 | 说明 |
|------|------|
| NONINTERACTIVE=1 | 跳过交互问答，使用默认值或下方变量 |
| CONFIG_PATH | 配置文件路径 |
| WORKSPACE | 工作目录 |
| PROJECT_NAME | project 名称 |
| AGENT_TYPE | claudecode 或 cursor |
| AGENT_MODE | Agent 权限模式 |
| PERMISSION_TEMPLATE | admin-only / family-readonly / family-remind |
| PLATFORM_CHOICES | 平台序号，如 7 或 1,7 |
| LLM_CHOICE | Claude Code LLM 选项 1-9 |
| SKIP_WEIXIN_SETUP=1 | 跳过微信扫码 |
| INSTALL_DEPS=0 | 跳过系统依赖安装 |

## 权限模板

| ID | 说明 |
|----|------|
| admin-only | 仅管理员可执行工具和高危命令 |
| family-readonly | 家人只读问答，禁用 cron/provider 等管理命令 |
| family-remind | 默认可提醒，禁用 shell/restart 等高危命令 |
