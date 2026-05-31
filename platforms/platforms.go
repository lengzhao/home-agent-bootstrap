package platforms

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
	"github.com/lengzhao/home-agent-bootstrap/prompt"
)

const DefaultPlatformChoice = "7"

type Option struct {
	Key   string
	Value string
}

type Block struct {
	Type    string
	Options []Option
}

type Field struct {
	Key      string
	Label    string
	Default  string
	Secret   bool
	Required bool
}

type Preset struct {
	Type          string
	DisplayName   string
	Connection    string
	NeedsPublicIP bool
	PublicIPLabel string
	DocHint       string
	SetupCLI      string
	Fields        []Field
}

var presets = []Preset{
	{
		Type: "feishu", DisplayName: "飞书 / Lark", Connection: "WebSocket", DocHint: "cc-connect docs/feishu.md",
		SetupCLI: "feishu setup",
		Fields: []Field{
			{Key: "app_id", Label: "飞书 app_id", Required: true},
			{Key: "app_secret", Label: "飞书 app_secret", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "dingtalk", DisplayName: "钉钉", Connection: "Stream", DocHint: "cc-connect docs/dingtalk.md",
		SetupCLI: "dingtalk setup",
		Fields: []Field{
			{Key: "client_id", Label: "钉钉 client_id (AppKey)", Required: true},
			{Key: "client_secret", Label: "钉钉 client_secret (AppSecret)", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "telegram", DisplayName: "Telegram", Connection: "Long Polling", DocHint: "cc-connect docs/telegram.md",
		Fields: []Field{
			{Key: "token", Label: "Telegram Bot Token（@BotFather）", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，Telegram user id，首次可留空", Default: ""},
		},
	},
	{
		Type: "slack", DisplayName: "Slack", Connection: "Socket Mode", DocHint: "cc-connect docs/slack.md",
		SetupCLI: "slack setup",
		Fields: []Field{
			{Key: "bot_token", Label: "Slack Bot Token (xoxb-...)", Secret: true, Required: true},
			{Key: "app_token", Label: "Slack App-Level Token (xapp-...)", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "discord", DisplayName: "Discord", Connection: "Gateway", DocHint: "cc-connect docs/discord.md",
		Fields: []Field{
			{Key: "token", Label: "Discord Bot Token", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，Discord user id，首次可留空", Default: ""},
		},
	},
	{
		Type: "wecom", DisplayName: "企业微信", Connection: "WebSocket / Webhook", PublicIPLabel: "视模式", DocHint: "cc-connect docs/wecom.md",
		Fields: []Field{
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
		Fields: []Field{
			{Key: "account_id", Label: "account_id", Default: "wx-1"},
			{Key: "allow_from", Label: "allow_from，首次可留空让 setup 回填", Default: ""},
		},
	},
	{
		Type: "line", DisplayName: "LINE", Connection: "Webhook", NeedsPublicIP: true, DocHint: "cc-connect docs/line.md",
		Fields: []Field{
			{Key: "channel_secret", Label: "LINE Channel Secret", Secret: true, Required: true},
			{Key: "channel_access_token", Label: "LINE Channel Access Token", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "qq", DisplayName: "QQ (NapCat/OneBot)", Connection: "WebSocket", DocHint: "cc-connect docs/qq.md",
		Fields: []Field{
			{Key: "ws_url", Label: "OneBot WebSocket 地址", Default: "ws://127.0.0.1:3001"},
			{Key: "allow_from", Label: "allow_from，QQ 号，首次可留空", Default: ""},
		},
	},
	{
		Type: "qqbot", DisplayName: "QQ 官方机器人", Connection: "WebSocket", DocHint: "cc-connect docs/qq.md",
		SetupCLI: "qqbot setup",
		Fields: []Field{
			{Key: "app_id", Label: "QQ 机器人 app_id", Required: true},
			{Key: "client_secret", Label: "QQ 机器人 client_secret", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
	{
		Type: "weibo", DisplayName: "微博", Connection: "WebSocket", DocHint: "cc-connect docs/weibo.md",
		Fields: []Field{
			{Key: "access_token", Label: "微博 access_token", Secret: true, Required: true},
			{Key: "allow_from", Label: "allow_from，限制使用者，首次可留空", Default: ""},
		},
	},
}

func PresetByType(typ string) (Preset, bool) {
	for _, preset := range presets {
		if preset.Type == typ {
			return preset, true
		}
	}
	return Preset{}, false
}

func TypesFromPresets() []string {
	types := make([]string, len(presets))
	for i, preset := range presets {
		types[i] = preset.Type
	}
	return types
}

func PrintCatalog(out io.Writer) {
	fmt.Fprintln(out, "\n可选接入平台（与 cc-connect 支持列表对齐）：")
	for i, preset := range presets {
		publicIP := presetPublicIPLabel(preset)
		fmt.Fprintf(out, "  %2d) %-10s %s (%s，公网 %s)\n", i+1, preset.Type, preset.DisplayName, preset.Connection, publicIP)
	}
	fmt.Fprintln(out, "\n输入序号，多个用逗号分隔，例如 7 或 1,7。默认 7 为微信个人号。")
}

func presetPublicIPLabel(preset Preset) string {
	if preset.PublicIPLabel != "" {
		return preset.PublicIPLabel
	}
	if preset.NeedsPublicIP {
		return "是"
	}
	return "否"
}

func ParseChoices(raw string) ([]Preset, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "7"
	}
	parts := strings.Split(raw, ",")
	selected := make([]Preset, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(presets) {
			return nil, fmt.Errorf("无效平台序号 %q", part)
		}
		preset := presets[idx-1]
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

func Configure(p *prompt.Prompt) ([]Block, error) {
	PrintCatalog(p.Out)
	choices := p.Ask("选择要接入的平台序号", cmdutil.EnvDefault("PLATFORM_CHOICES", DefaultPlatformChoice))
	presetList, err := ParseChoices(choices)
	if err != nil {
		return nil, err
	}

	blocks := make([]Block, 0)
	for _, preset := range presetList {
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

func configureWeixinPlatforms(p *prompt.Prompt) ([]Block, error) {
	countText := p.Ask("要配置几个微信个人号", "1")
	count, err := strconv.Atoi(countText)
	if err != nil || count < 1 {
		return nil, fmt.Errorf("微信个人号数量必须是大于 0 的数字")
	}
	blocks := make([]Block, 0, count)
	for i := 1; i <= count; i++ {
		accountID := p.Ask(fmt.Sprintf("第 %d 个微信个人号 account_id", i), fmt.Sprintf("wx-%d", i))
		allowFrom := p.Ask(fmt.Sprintf("第 %d 个微信个人号 allow_from，首次可留空让 setup 回填", i), "")
		blocks = append(blocks, Block{
			Type: "weixin",
			Options: []Option{
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

func configureGenericPlatform(p *prompt.Prompt, preset Preset) (Block, error) {
	fmt.Fprintf(p.Out, "\n配置平台 %s (%s)\n", preset.DisplayName, preset.Type)
	if preset.NeedsPublicIP {
		cmdutil.Warn("该平台通常需要公网 URL 或反向代理，请参考 " + preset.DocHint)
	}
	if preset.SetupCLI != "" {
		fmt.Fprintf(p.Out, "提示：也可在生成配置后执行 cc-connect %s 完成凭证写入。\n", preset.SetupCLI)
	}
	options := make([]Option, 0, len(preset.Fields))
	for _, field := range preset.Fields {
		label := field.Label
		if field.Required {
			label += "（必填，可留空稍后手动编辑 config.toml）"
		}
		var value string
		if field.Secret {
			value = p.AskSecret(label)
		} else {
			value = p.Ask(label, field.Default)
		}
		if value == "" && field.Default != "" {
			value = field.Default
		}
		options = append(options, Option{Key: field.Key, Value: value})
	}
	return Block{Type: preset.Type, Options: options}, nil
}

func CountWeixin(platforms []Block) int {
	n := 0
	for _, block := range platforms {
		if block.Type == "weixin" {
			n++
		}
	}
	return n
}

func HasWeixin(platforms []Block) bool {
	return CountWeixin(platforms) > 0
}

func SetupHints(platforms []Block) []string {
	hints := make([]string, 0)
	seen := map[string]bool{}
	for _, block := range platforms {
		if block.Type == "weixin" {
			continue
		}
		preset, ok := PresetByType(block.Type)
		if !ok || preset.SetupCLI == "" || seen[preset.Type] {
			continue
		}
		seen[preset.Type] = true
		hints = append(hints, fmt.Sprintf("  cc-connect %s --config <config> --project <project>", preset.SetupCLI))
	}
	return hints
}

func BlocksFromPresetsNonInteractive(presetList []Preset) ([]Block, error) {
	blocks := make([]Block, 0)
	for _, preset := range presetList {
		switch preset.Type {
		case "weixin":
			count := 1
			if raw := strings.TrimSpace(os.Getenv("WEIXIN_COUNT")); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < 1 {
					return nil, fmt.Errorf("WEIXIN_COUNT 必须是大于 0 的数字")
				}
				count = n
			}
			for i := 1; i <= count; i++ {
				accountID := fmt.Sprintf("wx-%d", i)
				if i == 1 {
					if v := strings.TrimSpace(os.Getenv("WEIXIN_ACCOUNT_ID")); v != "" {
						accountID = v
					}
				}
				allowFrom := cmdutil.EnvDefault("ADMIN_FROM", "")
				blocks = append(blocks, Block{
					Type: "weixin",
					Options: []Option{
						{Key: "token", Value: ""},
						{Key: "base_url", Value: "https://ilinkai.weixin.qq.com"},
						{Key: "cdn_base_url", Value: "https://novac2c.cdn.weixin.qq.com/c2c"},
						{Key: "allow_from", Value: allowFrom},
						{Key: "account_id", Value: accountID},
						{Key: "long_poll_timeout_ms", Value: "35000"},
					},
				})
			}
		default:
			block, err := blockFromPresetNonInteractive(preset)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func blockFromPresetNonInteractive(preset Preset) (Block, error) {
	options := make([]Option, 0, len(preset.Fields))
	for _, field := range preset.Fields {
		value := cmdutil.EnvDefault(strings.ToUpper(preset.Type+"_"+field.Key), field.Default)
		options = append(options, Option{Key: field.Key, Value: value})
	}
	return Block{Type: preset.Type, Options: options}, nil
}
