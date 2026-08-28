package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvironment_FromFile(t *testing.T) {
	raw := `{"name": "CI", "variables": {"BASE_URL": {"value": "https://api.example.com", "secret": false}}}`
	path := filepath.Join(t.TempDir(), "env.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	env, err := loadEnvironment(path, nil)
	if err != nil {
		t.Fatalf("loadEnvironment returned error: %v", err)
	}
	if env.Variables["BASE_URL"].Value != "https://api.example.com" {
		t.Fatalf("unexpected BASE_URL: %+v", env.Variables["BASE_URL"])
	}
}

func TestLoadEnvironment_VarOverridesWinOverFile(t *testing.T) {
	raw := `{"name": "CI", "variables": {"TOKEN": {"value": "stale", "secret": true}}}`
	path := filepath.Join(t.TempDir(), "env.json")
	os.WriteFile(path, []byte(raw), 0o644)

	env, err := loadEnvironment(path, []string{"TOKEN=fresh-from-ci-secret"})
	if err != nil {
		t.Fatalf("loadEnvironment returned error: %v", err)
	}
	if env.Variables["TOKEN"].Value != "fresh-from-ci-secret" {
		t.Fatalf("--var override did not win, got %+v", env.Variables["TOKEN"])
	}
}

func TestLoadEnvironment_NoFileUsesOverridesOnly(t *testing.T) {
	env, err := loadEnvironment("", []string{"BASE_URL=https://example.com"})
	if err != nil {
		t.Fatalf("loadEnvironment returned error: %v", err)
	}
	if env.Variables["BASE_URL"].Value != "https://example.com" {
		t.Fatal("expected override-only environment to still resolve")
	}
}

func TestLoadEnvironment_RejectsMalformedOverride(t *testing.T) {
	if _, err := loadEnvironment("", []string{"NOT-A-KEY-VALUE-PAIR"}); err == nil {
		t.Fatal("expected an error for a malformed --var override")
	}
}
