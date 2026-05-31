package permissions

import "testing"

func TestParseChoiceDefault(t *testing.T) {
	got, err := ParseChoice("")
	if err != nil {
		t.Fatalf("ParseChoice() error: %v", err)
	}
	if got.ID != "family-remind" {
		t.Fatalf("expected family-remind, got %q", got.ID)
	}
}

func TestMemberDisabledCommandsAdminOnlyIncludesCron(t *testing.T) {
	got := MemberDisabledCommands("admin-only")
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

func TestChoiceDefaultFromEnv(t *testing.T) {
	t.Setenv("PERMISSION_TEMPLATE", "family-readonly")
	if got := ChoiceDefaultFromEnv(); got != "2" {
		t.Fatalf("ChoiceDefaultFromEnv() = %q, want 2", got)
	}
}
