package environments

import (
	"errors"
	"testing"
)

func TestRender_SubstitutesKnownVariables(t *testing.T) {
	env := New("Development")
	env.Set("BASE_URL", "https://api.example.com", false)
	env.Set("USER_ID", "42", false)

	got, err := Render("{{BASE_URL}}/users/{{USER_ID}}", env.Lookup(nil), true)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	want := "https://api.example.com/users/42"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRender_OverridesTakePrecedence(t *testing.T) {
	env := New("Development")
	env.Set("TOKEN", "stale-token", true)

	got, err := Render("{{TOKEN}}", env.Lookup(map[string]string{"TOKEN": "fresh-token"}), true)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if got != "fresh-token" {
		t.Fatalf("Render() = %q, want fresh-token (override should win)", got)
	}
}

func TestRender_StrictModeErrorsOnMissingVariable(t *testing.T) {
	env := New("Development")

	_, err := Render("{{MISSING}}", env.Lookup(nil), true)
	if err == nil {
		t.Fatal("expected an error for an unresolved variable in strict mode")
	}
	var unresolved *UnresolvedVariableError
	if !errors.As(err, &unresolved) {
		t.Fatalf("error = %v, want *UnresolvedVariableError", err)
	}
	if len(unresolved.Names) != 1 || unresolved.Names[0] != "MISSING" {
		t.Fatalf("unresolved.Names = %v, want [MISSING]", unresolved.Names)
	}
}

func TestRender_LenientModeLeavesPlaceholder(t *testing.T) {
	env := New("Development")

	got, err := Render("{{MISSING}}", env.Lookup(nil), false)
	if err != nil {
		t.Fatalf("Render returned error in lenient mode: %v", err)
	}
	if got != "{{MISSING}}" {
		t.Fatalf("Render() = %q, want placeholder left intact", got)
	}
}

func TestRender_NoPlaceholdersIsNoOp(t *testing.T) {
	env := New("Development")
	got, err := Render("https://api.example.com/health", env.Lookup(nil), true)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if got != "https://api.example.com/health" {
		t.Fatalf("Render() = %q, want input unchanged", got)
	}
}

func TestNames_ReturnsUniqueInOrder(t *testing.T) {
	got := Names("{{BASE_URL}}/users/{{USER_ID}}?token={{API_KEY}}&again={{USER_ID}}")
	want := []string{"BASE_URL", "USER_ID", "API_KEY"}
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEnvironment_RedactedMasksSecretsOnly(t *testing.T) {
	env := New("Production")
	env.Set("API_KEY", "sk-super-secret", true)
	env.Set("BASE_URL", "https://api.example.com", false)

	redacted := env.Redacted()

	if redacted.Variables["API_KEY"].Value == "sk-super-secret" {
		t.Fatal("Redacted() leaked the secret value")
	}
	if redacted.Variables["BASE_URL"].Value != "https://api.example.com" {
		t.Fatal("Redacted() should not touch non-secret values")
	}
}

func TestEnvironment_ExportSafeDropsSecretsEntirely(t *testing.T) {
	env := New("Production")
	env.Set("API_KEY", "sk-super-secret", true)
	env.Set("BASE_URL", "https://api.example.com", false)

	exported := env.ExportSafe()

	if _, ok := exported.Variables["API_KEY"]; ok {
		t.Fatal("ExportSafe() should omit secret variables entirely")
	}
	if exported.Variables["BASE_URL"].Value != "https://api.example.com" {
		t.Fatal("ExportSafe() should keep non-secret values")
	}
}
