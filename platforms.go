package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PlatformOption is a single TOML key under [projects.platforms.options].
type PlatformOption struct {
	Key   string
	Value string
}

// PlatformBlock is one [[projects.platforms]] entry.
type PlatformBlock struct {
	Type    string
	Options []PlatformOption
}

// PlatformField describes one option field to collect during bootstrap.
type PlatformField struct {
	Key      string
	Label    string
	Default  string
	Secret   bool
	Required bool
}

// PlatformPreset describes a cc-connect supported platform type.
type PlatformPreset struct {
	Type          string
	DisplayName   string
	Connection    string
	NeedsPublicIP bool
	PublicIPLabel string
	DocHint       string
	SetupCLI      string
	Fields        []PlatformField
}

var platformPresets = []PlatformPreset{
	{
		Type: "feishu", DisplayName: "飞书 / Lark", Connection: "WebSocket", DocHint: "cc-connect docs/feishu.md",
		SetupCLI: "feishu setup",
		Fields: []PlatformField{
			{Key: "app_id", Label: "飞书 app_id", Required: true},
			{Key: "app_secret", Label: "飞书 app_secret", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "dingtalk", DisplayName: "钉钉", Connection: "Stream", DocHint: "cc-connect docs/dingtalk.md",
		SetupCLI: "dingtalk setup",
		Fields: []PlatformField{
			{Key: "client_id", Label: "钉钉 client_id (AppKey)", Required: true},
			{Key: "client_secret", Label: "钉钉 client_secret (AppSecret)", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "telegram", DisplayName: "Telegram", Connection: "Long Polling", DocHint: "cc-connect docs/telegram.md",
		Fields: []PlatformField{
			{Key: "token", Label: "Telegram Bot Token（@BotFather）", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，Telegram user id，首次可留空", Default: ""},
		},
	},
	{
		Type: "slack", DisplayName: "Slack", Connection: "Socket Mode", DocHint: "cc-connect docs/slack.md",
		SetupCLI: "slack setup",
		Fields: []PlatformField{
			{Key: "bot_token", Label: "Slack Bot Token (xoxb-...)", Secret: true, Required: true},
			{Key: "app_token", Label: "Slack App-Level Token (xapp-...)", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "discord", DisplayName: "Discord", Connection: "Gateway", DocHint: "cc-connect docs/discord.md",
		Fields: []PlatformField{
			{Key: "token", Label: "Discord Bot Token", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，Discord user id，首次可留空", Default: ""},
		},
	},
	{
		Type: "wecom", DisplayName: "企业微信", Connection: "WebSocket / Webhook", PublicIPLabel: "视模式", DocHint: "cc-connect docs/wecom.md",
		Fields: []PlatformField{
			{Key: "corp_id", Label: "企业微信 corp_id", Required: true},
			{Key: "corp_secret", Label: "企业微信 corp_secret", Secret: true, Required: true},
			{Key: "token", Label: "回调 Token", Default: ""},
			{Key: "encoding_aes_key", Label: "EncodingAESKey", Secret: true, Default: ""},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "weixin", DisplayName: "微信个人号", Connection: "ilink 长轮询", DocHint: "cc-connect docs/weixin.md",
		SetupCLI: "weixin setup",
		Fields: []PlatformField{
			{Key: "account_id", Label: "account_id", Default: "wx-1"},
			{Key: "allow_from", Label: "allow_from，首次可留空让 setup 回填", Default: ""},
		},
	},
	{
		Type: "line", DisplayName: "LINE", Connection: "Webhook", NeedsPublicIP: true, DocHint: "cc-connect docs/line.md",
		Fields: []PlatformField{
			{Key: "channel_secret", Label: "LINE Channel Secret", Secret: true, Required: true},
			{Key: "channel_access_token", Label: "LINE Channel Access Token", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "qq", DisplayName: "QQ (NapCat/OneBot)", Connection: "WebSocket", DocHint: "cc-connect docs/qq.md",
		Fields: []PlatformField{
			{Key: "ws_url", Label: "OneBot WebSocket 地址", Default: "ws://127.0.0.1:3001"},
			{Key: "allow_from", Label: "allow_from，QQ 号，首次可留空", Default: ""},
		},
	},
	{
		Type: "qqbot", DisplayName: "QQ 官方机器人", Connection: "WebSocket", DocHint: "cc-connect docs/qq.md",
		SetupCLI: "qqbot setup",
		Fields: []PlatformField{
			{Key: "app_id", Label: "QQ 机器人 app_id", Required: true},
			{Key: "client_secret", Label: "QQ 机器人 client_secret", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "weibo", DisplayName: "微博", Connection: "WebSocket", DocHint: "cc-connect docs/weibo.md",
		Fields: []PlatformField{
			{Key: "access_token", Label: "微博 access_token", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
}

func platformPresetByType(typ string) (PlatformPreset, bool) {
	for _, preset := range platformPresets {
		if preset.Type == typ {
			return preset, true
		}
	}
	return PlatformPreset{}, false
}

func printPlatformCatalog(out io.Writer) {
	fmt.Fprintln(out, "\n可选接入平台（与 cc-connect 支持列表对齐）：")
	for i, preset := range platformPresets {
		publicIP := presetPublicIPLabel(preset)
		fmt.Fprintf(out, "  %2d) %-10s %s (%s，公网 %s)\n", i+1, preset.Type, preset.DisplayName, preset.Connection, publicIP)
	}
	fmt.Fprintln(out, "\n输入序号，多个用逗号分隔，例如 7 或 1,7。默认 7 为微信个人号。")
}

func presetPublicIPLabel(preset PlatformPreset) string {
	if preset.PublicIPLabel != "" {
		return preset.PublicIPLabel
	}
	if preset.NeedsPublicIP {
		return "是"
	}
	return "否"
}

func parsePlatformChoices(raw string) ([]PlatformPreset, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "7"
	}
	parts := strings.Split(raw, ",")
	selected := make([]PlatformPreset, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(platformPresets) {
			return nil, fmt.Errorf("无效平台序号 %q", part)
		}
		preset := platformPresets[idx-1]
		if seen[preset.Type] {
			continue
		}
		seen[preset.Type] = true
		selected = append(selected, preset)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("至少选择一个接入平台")
	}
	return selected, nil
}

func configurePlatforms(p *prompt) ([]PlatformBlock, error) {
	printPlatformCatalog(p.out)
	choices := p.ask("选择要接入的平台序号", envDefault("PLATFORM_CHOICES", defaultPlatformChoice))
	presets, err := parsePlatformChoices(choices)
	if err != nil {
		return nil, err
	}

	blocks := make([]PlatformBlock, 0)
	for _, preset := range presets {
		switch preset.Type {
		case "weixin":
			weixinBlocks, err := configureWeixinPlatforms(p)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, weixinBlocks...)
		default:
			block, err := configureGenericPlatform(p, preset)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func configureWeixinPlatforms(p *prompt) ([]PlatformBlock, error) {
	countText := p.ask("要配置几个微信个人号", "1")
	count, err := strconv.Atoi(countText)
	if err != nil || count < 1 {
		return nil, fmt.Errorf("微信个人号数量必须是大于 0 的数字")
	}
	blocks := make([]PlatformBlock, 0, count)
	for i := 1; i <= count; i++ {
		accountID := p.ask(fmt.Sprintf("第 %d 个微信个人号 account_id", i), fmt.Sprintf("wx-%d", i))
		allowFrom := p.ask(fmt.Sprintf("第 %d 个微信个人号 allow_from，首次可留空让 setup 回填", i), "")
		blocks = append(blocks, PlatformBlock{
			Type: "weixin",
			Options: []PlatformOption{
				{Key: "token", Value: ""},
				{Key: "base_url", Value: "https://ilinkai.weixin.qq.com"},
				{Key: "cdn_base_url", Value: "https://novac2c.cdn.weixin.qq.com/c2c"},
				{Key: "allow_from", Value: allowFrom},
				{Key: "account_id", Value: accountID},
				{Key: "long_poll_timeout_ms", Value: "35000"},
			},
		})
	}
	return blocks, nil
}

func configureGenericPlatform(p *prompt, preset PlatformPreset) (PlatformBlock, error) {
	fmt.Fprintf(p.out, "\n配置平台 %s (%s)\n", preset.DisplayName, preset.Type)
	if preset.NeedsPublicIP {
		warn("该平台通常需要公网 URL 或反向代理，请参考 " + preset.DocHint)
	}
	if preset.SetupCLI != "" {
		fmt.Fprintf(p.out, "提示：也可在生成配置后执行 cc-connect %s 完成凭证写入。\n", preset.SetupCLI)
	}
	options := make([]PlatformOption, 0, len(preset.Fields))
	for _, field := range preset.Fields {
		label := field.Label
		if field.Required {
			label += "（必填，可留空稍后手动编辑 config.toml）"
		}
		var value string
		if field.Secret {
			value = p.askSecret(label)
		} else {
			value = p.ask(label, field.Default)
		}
		if value == "" && field.Default != "" {
			value = field.Default
		}
		options = append(options, PlatformOption{Key: field.Key, Value: value})
	}
	return PlatformBlock{Type: preset.Type, Options: options}, nil
}

func countWeixinPlatforms(platforms []PlatformBlock) int {
	n := 0
	for _, block := range platforms {
		if block.Type == "weixin" {
			n++
		}
	}
	return n
}

func hasWeixinPlatform(platforms []PlatformBlock) bool {
	return countWeixinPlatforms(platforms) > 0
}

func platformSetupHints(platforms []PlatformBlock) []string {
	hints := make([]string, 0)
	seen := map[string]bool{}
	for _, block := range platforms {
		if block.Type == "weixin" {
			continue
		}
		preset, ok := platformPresetByType(block.Type)
		if !ok || preset.SetupCLI == "" || seen[preset.Type] {
			continue
		}
		seen[preset.Type] = true
		hints = append(hints, fmt.Sprintf("  cc-connect %s --config <config> --project <project>", preset.SetupCLI))
	}
	return hints
}
