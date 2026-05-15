package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokenFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apps.json")
	if err := os.WriteFile(path, []byte(`{"github.com:Iv1.test":{"oauth_token":"abc123","user":"octo"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	token, err := tokenFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if token != "abc123" {
		t.Fatalf("got %q, want %q", token, "abc123")
	}
}

func TestTokenFromFile_NotFound(t *testing.T) {
	_, err := tokenFromFile(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestTokenFromFile_NoToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apps.json")
	if err := os.WriteFile(path, []byte(`{"github.com:Iv1.test":{"oauth_token":"","user":"octo"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := tokenFromFile(path)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestGetEnvDefault(t *testing.T) {
	t.Setenv("TEST_KEY_XYZ", "myvalue")
	if got := getEnvDefault("TEST_KEY_XYZ", "fallback"); got != "myvalue" {
		t.Fatalf("got %q, want %q", got, "myvalue")
	}
	if got := getEnvDefault("TEST_KEY_UNSET_XYZ", "fallback"); got != "fallback" {
		t.Fatalf("got %q, want %q", got, "fallback")
	}
}

func TestCollectResponse(t *testing.T) {
	lines := []string{
		`data: {"choices":[{"delta":{"content":"feat"},"finish_reason":null}]}`,
		`data: {"choices":[{"delta":{"content":": add streaming"},"finish_reason":null}]}`,
		`data: [DONE]`,
	}
	body := strings.NewReader(strings.Join(lines, "\n"))
	got, err := collectResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if want := "feat: add streaming"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCollectResponse_SkipsInvalidLines(t *testing.T) {
	lines := []string{
		`data: not-json`,
		`data: {"choices":[{"delta":{"content":"ok"},"finish_reason":null}]}`,
		``,
	}
	body := strings.NewReader(strings.Join(lines, "\n"))
	got, err := collectResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("got %q, want %q", got, "ok")
	}
}

func TestCommitSystemPrompt_Default(t *testing.T) {
	// Point UserConfigDir somewhere with no file to exercise the fallback path.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got := commitSystemPrompt()
	if got != defaultCommitSystemPrompt {
		t.Fatalf("expected default prompt, got %q", got)
	}
}

func TestCommitSystemPrompt_CustomFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	custom := "Custom prompt for testing."
	promptDir := filepath.Join(dir, "gh-hello")
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "commit-prompt.txt"), []byte(custom), 0o600); err != nil {
		t.Fatal(err)
	}

	got := commitSystemPrompt()
	if got != custom {
		t.Fatalf("got %q, want %q", got, custom)
	}
}

func TestDefaultCommitSystemPrompt_ContainsRequiredElements(t *testing.T) {
	for _, keyword := range []string{"feat", "fix", "refactor", "imperative", "72"} {
		if !strings.Contains(defaultCommitSystemPrompt, keyword) {
			t.Errorf("defaultCommitSystemPrompt missing expected keyword %q", keyword)
		}
	}
}

func TestRequestID_UniqueAndFormatted(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := requestID()
		parts := strings.Split(id, "-")
		if len(parts) != 5 {
			t.Fatalf("requestID %q: expected 5 parts, got %d", id, len(parts))
		}
		if ids[id] {
			t.Fatalf("duplicate requestID: %q", id)
		}
		ids[id] = true
	}
}
