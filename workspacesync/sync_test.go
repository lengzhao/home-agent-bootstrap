package workspacesync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersionAtLeast(t *testing.T) {
	tests := []struct {
		current string
		minimum string
		want    bool
	}{
		{"cc-connect 1.3.2", "1.3.0", true},
		{"1.2.9", "1.3.0", false},
		{"1.3.0", "1.3.0", true},
	}
	for _, tt := range tests {
		if got := CompareVersionAtLeast(tt.current, tt.minimum); got != tt.want {
			t.Fatalf("CompareVersionAtLeast(%q, %q) = %v, want %v", tt.current, tt.minimum, got, tt.want)
		}
	}
}

func TestSyncAddsMissingVersion(t *testing.T) {
	dir := t.TempDir()
	templates := os.DirFS(filepath.Join(".."))

	report, err := Sync(templates, dir, Options{})
	if err != nil {
		t.Fatalf("Sync() error: %v", err)
	}

	if report.InstalledVersion != TemplateVersion {
		t.Fatalf("InstalledVersion = %q, want %q", report.InstalledVersion, TemplateVersion)
	}
	foundVersion := false
	for _, rel := range report.Added {
		if rel == "VERSION" {
			foundVersion = true
		}
	}
	if !foundVersion {
		t.Fatalf("expected VERSION to be added, got %#v", report.Added)
	}
}

func TestSyncIncludesDefaultSkills(t *testing.T) {
	dir := t.TempDir()
	templates := os.DirFS(filepath.Join(".."))

	if _, err := Sync(templates, dir, Options{}); err != nil {
		t.Fatalf("Sync() error: %v", err)
	}

	for _, rel := range []string{
		"CLAUDE.md",
		"VERSION",
		"skills/home-routines/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected workspace file %s: %v", rel, err)
		}
	}
}

func TestAnalyzeDetectsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	templates := os.DirFS(filepath.Join(".."))

	status, err := Analyze(templates, dir)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}
	if len(status.Missing) == 0 {
		t.Fatalf("expected missing files in empty workspace, got %#v", status.Missing)
	}
	if status.TemplateVersion != TemplateVersion {
		t.Fatalf("TemplateVersion = %q", status.TemplateVersion)
	}
}

func TestSyncDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	templates := os.DirFS(filepath.Join(".."))

	report, err := Sync(templates, dir, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Sync() error: %v", err)
	}
	if len(report.Added) == 0 {
		t.Fatal("expected dry-run to list files to add")
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		t.Fatal("dry-run should not write CLAUDE.md")
	}
}
