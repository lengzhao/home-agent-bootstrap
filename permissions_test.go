package main

import "testing"

func TestParsePermissionTemplateChoiceDefault(t *testing.T) {
	got, err := parsePermissionTemplateChoice("")
	if err != nil {
		t.Fatalf("parsePermissionTemplateChoice() error: %v", err)
	}
	if got.ID != "family-remind" {
		t.Fatalf("expected family-remind, got %q", got.ID)
	}
}

func TestMemberDisabledCommandsAdminOnlyIncludesCron(t *testing.T) {
	got := memberDisabledCommands("admin-only")
	found := false
	for _, cmd := range got {
		if cmd == "cron" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("admin-only should disable cron, got %#v", got)
	}
}
