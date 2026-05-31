package main

import "testing"

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
		if got := compareVersionAtLeast(tt.current, tt.minimum); got != tt.want {
			t.Fatalf("compareVersionAtLeast(%q, %q) = %v, want %v", tt.current, tt.minimum, got, tt.want)
		}
	}
}

func TestSyncWorkspaceFilesAddsMissingVersion(t *testing.T) {
	dir := t.TempDir()

	report, err := syncWorkspaceFiles(dir)
	if err != nil {
		t.Fatalf("syncWorkspaceFiles() error: %v", err)
	}

	if report.InstalledVersion != workspaceTemplateVersion {
		t.Fatalf("InstalledVersion = %q, want %q", report.InstalledVersion, workspaceTemplateVersion)
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
