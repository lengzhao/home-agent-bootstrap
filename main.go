package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/lengzhao/home-agent-bootstrap/bootstrap"
)

//go:embed workspace templates/config.generated.toml.tmpl
var embeddedTemplates embed.FS

func main() {
	if err := bootstrap.Run(os.Args[1:], embeddedTemplates); err != nil {
		fmt.Fprintf(os.Stderr, "\n[ERROR] %v\n", err)
		os.Exit(1)
	}
}
