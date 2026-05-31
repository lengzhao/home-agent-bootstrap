package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Prompt struct {
	In             *bufio.Reader
	Out            io.Writer
	NonInteractive bool
}

func New(in *bufio.Reader, out io.Writer, nonInteractive bool) *Prompt {
	return &Prompt{In: in, Out: out, NonInteractive: nonInteractive}
}

func (p *Prompt) Ask(label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Fprintf(p.Out, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(p.Out, "%s: ", label)
	}
	text, _ := p.In.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue
	}
	return text
}

func (p *Prompt) AskYesNo(label string, defaultValue bool) bool {
	if p.NonInteractive {
		return defaultValue
	}
	def := "n"
	if defaultValue {
		def = "y"
	}
	for {
		answer := strings.ToLower(p.Ask(label+" (y/n)", def))
		switch answer {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Fprintln(p.Out, "请输入 y 或 n")
		}
	}
}

func (p *Prompt) AskAllowed(label, defaultValue string, allowed []string) string {
	for {
		value := p.Ask(label, defaultValue)
		for _, item := range allowed {
			if value == item {
				return value
			}
		}
		fmt.Fprintf(p.Out, "无效输入：%s。可选值：%s\n", value, strings.Join(allowed, ", "))
	}
}

func (p *Prompt) AskSecret(label string) string {
	fmt.Fprintf(p.Out, "%s: ", label)
	text, _ := p.In.ReadString('\n')
	return strings.TrimSpace(text)
}
