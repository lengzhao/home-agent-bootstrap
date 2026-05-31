package config

import (
	"strconv"
	"strings"

	"github.com/lengzhao/home-agent-bootstrap/cmdutil"
)

func FirstConfiguredAdminFrom(config string) string {
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "admin_from = ") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "admin_from = "))
		value, err := strconv.Unquote(raw)
		if err != nil || value == "" || value == "*" {
			continue
		}
		return value
	}
	return ""
}

func FirstBoundWeixinAllowFrom(config string) string {
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "allow_from = ") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "allow_from = "))
		value, err := strconv.Unquote(raw)
		if err != nil || value == "" || value == "*" {
			continue
		}
		return value
	}
	return ""
}

func ApplyAdminUser(config, adminUserID string) string {
	lines := strings.Split(config, "\n")
	inAdminRole := false
	hasAdminRole := false
	insertAt := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "admin_from = "):
			lines[i] = cmdutil.LinePrefix(line) + "admin_from = " + cmdutil.TomlQuote(adminUserID)
		case strings.HasPrefix(trimmed, "[") && trimmed != "[projects.users.roles.admin]":
			if insertAt == -1 && trimmed == "[projects.users.roles.member]" {
				insertAt = i
			}
			inAdminRole = false
		case trimmed == "[projects.users.roles.admin]":
			hasAdminRole = true
			inAdminRole = true
		case inAdminRole && strings.HasPrefix(trimmed, "user_ids = "):
			lines[i] = cmdutil.LinePrefix(line) + "user_ids = " + cmdutil.TomlArray([]string{adminUserID})
		}
	}
	if !hasAdminRole {
		adminRole := []string{
			"[projects.users.roles.admin]",
			"user_ids = " + cmdutil.TomlArray([]string{adminUserID}),
			"disabled_commands = []",
			"rate_limit = { max_messages = 50, window_secs = 60 }",
			"",
		}
		if insertAt == -1 {
			lines = append(lines, adminRole...)
		} else {
			updated := make([]string, 0, len(lines)+len(adminRole))
			updated = append(updated, lines[:insertAt]...)
			updated = append(updated, adminRole...)
			updated = append(updated, lines[insertAt:]...)
			lines = updated
		}
	}
	return strings.Join(lines, "\n")
}
