package cmdutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const AppName = "home-agent-bootstrap"

func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return home
}

func EnvDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func WriteFile(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, mode)
}

func BackupFile(path string) error {
	backup := fmt.Sprintf("%s.bak.%d", path, os.Getpid())
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := WriteFile(backup, input, 0o600); err != nil {
		return err
	}
	Say("已备份现有配置到 " + backup)
	return nil
}

func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func MustRandomToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func TomlQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func TomlArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = TomlQuote(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func OptionalStringSlice(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func LinePrefix(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func Say(message string) {
	fmt.Printf("\n[%s] %s\n", AppName, message)
}

func Warn(message string) {
	fmt.Fprintf(os.Stderr, "\n[WARN] %s\n", message)
}

func AddHomebrewToPath() {
	for _, path := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		if Exists(filepath.Join(path, "brew")) {
			os.Setenv("PATH", path+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
	}
}

func DefaultConfigPath() string {
	return filepath.Join(HomeDir(), ".cc-connect", "config.toml")
}

func DefaultWorkspacePath() string {
	return filepath.Join(HomeDir(), "home-assistant-workspace")
}
