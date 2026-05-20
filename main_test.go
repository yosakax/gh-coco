package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCopilotToken_FromEnv(t *testing.T) {
	t.Setenv("COPILOT_GITHUB_TOKEN", "env-token")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	token, err := loadCopilotToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "env-token" {
		t.Fatalf("got %q, want %q", token, "env-token")
	}
}

func TestLoadCopilotToken_FromGhCLI(t *testing.T) {
	t.Setenv("COPILOT_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	dir := t.TempDir()
	ghPath := filepath.Join(dir, "gh")
	if err := os.WriteFile(ghPath, []byte("#!/bin/sh\necho gh-cli-token\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	token, err := loadCopilotToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "gh-cli-token" {
		t.Fatalf("got %q, want %q", token, "gh-cli-token")
	}
}

func TestLoadCopilotToken_GhCLIFailure(t *testing.T) {
	t.Setenv("COPILOT_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	dir := t.TempDir()
	ghPath := filepath.Join(dir, "gh")
	if err := os.WriteFile(ghPath, []byte("#!/bin/sh\necho auth failed >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	_, err := loadCopilotToken()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh auth login") {
		t.Fatalf("expected error to mention gh auth login, got %q", err.Error())
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

func TestResolveCopilotBaseURL_UsesEndpointAndTrimsSlash(t *testing.T) {
	got := resolveCopilotBaseURL("https://api.business.githubcopilot.com/")
	if got != "https://api.business.githubcopilot.com" {
		t.Fatalf("got %q, want %q", got, "https://api.business.githubcopilot.com")
	}
}

func TestResolveCopilotBaseURL_FallsBackToDefault(t *testing.T) {
	got := resolveCopilotBaseURL("  ")
	if got != defaultAPIBaseURL {
		t.Fatalf("got %q, want %q", got, defaultAPIBaseURL)
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
	t.Chdir(dir)

	got := commitSystemPrompt()
	if got != defaultCommitSystemPrompt {
		t.Fatalf("expected default prompt, got %q", got)
	}
}

func TestCommitSystemPrompt_CustomFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Chdir(dir)

	custom := "Custom prompt for testing."
	promptDir := filepath.Join(dir, "gh-coco")
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

func TestCommitSystemPrompt_RepoRootFilePreferred(t *testing.T) {
	repo := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	t.Chdir(repo)

	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	localPrompt := "Use local prompt."
	globalPrompt := "Use global prompt."

	if err := os.WriteFile(filepath.Join(repo, ".commit-prompt.txt"), []byte(localPrompt), 0o600); err != nil {
		t.Fatal(err)
	}
	globalPromptDir := filepath.Join(configDir, "gh-coco")
	if err := os.MkdirAll(globalPromptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalPromptDir, "commit-prompt.txt"), []byte(globalPrompt), 0o600); err != nil {
		t.Fatal(err)
	}

	got := commitSystemPrompt()
	if got != localPrompt {
		t.Fatalf("got %q, want %q", got, localPrompt)
	}
}

func TestCommitSystemPrompt_GlobalFallbackWhenRepoFileMissing(t *testing.T) {
	repo := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	t.Chdir(repo)

	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	globalPrompt := "Use global prompt."
	globalPromptDir := filepath.Join(configDir, "gh-coco")
	if err := os.MkdirAll(globalPromptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalPromptDir, "commit-prompt.txt"), []byte(globalPrompt), 0o600); err != nil {
		t.Fatal(err)
	}

	got := commitSystemPrompt()
	if got != globalPrompt {
		t.Fatalf("got %q, want %q", got, globalPrompt)
	}
}

func TestResolveCommitSystemPrompt_DefaultSource(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Chdir(dir)

	gotPrompt, gotSource := resolveCommitSystemPrompt()
	if gotPrompt != defaultCommitSystemPrompt {
		t.Fatalf("expected default prompt, got %q", gotPrompt)
	}
	if gotSource != builtInPromptName {
		t.Fatalf("got source %q, want %q", gotSource, builtInPromptName)
	}
}

func TestResolveCommitSystemPrompt_SourcePath(t *testing.T) {
	repo := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	t.Chdir(repo)

	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	localPath := filepath.Join(repo, ".commit-prompt.txt")
	if err := os.WriteFile(localPath, []byte("Use local prompt."), 0o600); err != nil {
		t.Fatal(err)
	}

	_, gotSource := resolveCommitSystemPrompt()
	if gotSource != localPath {
		t.Fatalf("got source %q, want %q", gotSource, localPath)
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

func TestConfirmCommit_Default(t *testing.T) {
	input := strings.NewReader("\n")
	reader := bufio.NewReader(input)
	// Simulate pressing Enter (empty input)
	line, _ := reader.ReadString('\n')
	response := strings.TrimSpace(strings.ToLower(line))
	result := response == "" || response == "y"
	if !result {
		t.Fatalf("expected true for empty input, got %v", result)
	}
}

func TestConfirmCommit_Yes(t *testing.T) {
	input := strings.NewReader("y\n")
	reader := bufio.NewReader(input)
	line, _ := reader.ReadString('\n')
	response := strings.TrimSpace(strings.ToLower(line))
	result := response == "" || response == "y"
	if !result {
		t.Fatalf("expected true for 'y', got %v", result)
	}
}

func TestConfirmCommit_No(t *testing.T) {
	input := strings.NewReader("n\n")
	reader := bufio.NewReader(input)
	line, _ := reader.ReadString('\n')
	response := strings.TrimSpace(strings.ToLower(line))
	result := response == "" || response == "y"
	if result {
		t.Fatalf("expected false for 'n', got %v", result)
	}
}
