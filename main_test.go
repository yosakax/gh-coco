package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopilotCommandArgs(t *testing.T) {
	args := copilotCommandArgs("hello", "gpt-5.6-luna")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-p hello") {
		t.Fatalf("prompt arg missing: %q", joined)
	}
	if !strings.Contains(joined, "--reasoning-effort low") {
		t.Fatalf("reasoning effort missing: %q", joined)
	}
	if !strings.Contains(joined, "--model gpt-5.6-luna") {
		t.Fatalf("model arg missing: %q", joined)
	}
}

func TestCopilotCommandArgs_WithoutModel(t *testing.T) {
	args := copilotCommandArgs("hello", "  ")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--model gpt-5.6-luna") {
		t.Fatalf("default model missing: %q", joined)
	}
}

func TestRunCopilotPrompt(t *testing.T) {
	dir := t.TempDir()
	ghPath := filepath.Join(dir, "gh")
	script := "#!/bin/sh\necho result-from-gh\n"
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	got, err := runCopilotPrompt("hello", "gpt-5.6-luna")
	if err != nil {
		t.Fatal(err)
	}
	if got != "result-from-gh" {
		t.Fatalf("got %q, want %q", got, "result-from-gh")
	}
}

func TestRunCopilotPrompt_Failure(t *testing.T) {
	dir := t.TempDir()
	ghPath := filepath.Join(dir, "gh")
	script := "#!/bin/sh\necho copilot failed >&2\nexit 1\n"
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	_, err := runCopilotPrompt("hello", "gpt-5.6-luna")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh copilot failed") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestRunCopilotPrompt_EmptyResponse(t *testing.T) {
	dir := t.TempDir()
	ghPath := filepath.Join(dir, "gh")
	script := "#!/bin/sh\necho\n"
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	_, err := runCopilotPrompt("hello", "gpt-5.6-luna")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestBuildCommitPrompt(t *testing.T) {
	got := buildCommitPrompt("system", "diff --git a/a b/a")
	if !strings.Contains(got, "system") || !strings.Contains(got, "```diff") {
		t.Fatalf("unexpected prompt format: %q", got)
	}
}

func TestResolveModel(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "gpt-5.6-luna", false},
		{"gpt-5.6-luna", "gpt-5.6-luna", false},
		{"4.1", "", true},
		{"5-mini", "", true},
		{"gpt-5", "", true},
	}
	for _, tt := range tests {
		got, err := resolveModel(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("resolveModel(%q): expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("resolveModel(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("resolveModel(%q): got %q, want %q", tt.in, got, tt.want)
		}
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

func TestCommitSystemPrompt_Default(t *testing.T) {
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

func TestConfirmCommit_Default(t *testing.T) {
	input := strings.NewReader("\n")
	reader := bufio.NewReader(input)
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
