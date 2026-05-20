package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	version            = "0.3.1"
	defaultForcedModel = "gpt-4.1"
	builtInPromptName  = "built-in default"
)

func main() {
	doCommit := false
	skipConfirm := false
	model := ""
	var args []string

	rawArgs := os.Args[1:]
	for i := 0; i < len(rawArgs); i++ {
		a := rawArgs[i]
		switch a {
		case "--commit", "-c":
			doCommit = true
		case "--yes", "-y":
			skipConfirm = true
		case "--model", "-m":
			if i+1 >= len(rawArgs) {
				fatal(fmt.Errorf("missing value for %s", a))
			}
			model = rawArgs[i+1]
			i++
		case "--help", "-h":
			printHelp()
			return
		case "--version", "-v":
			fmt.Println(version)
			return
		default:
			if strings.HasPrefix(a, "--model=") {
				model = strings.TrimPrefix(a, "--model=")
				continue
			}
			args = append(args, a)
		}
	}

	if strings.TrimSpace(model) == "" {
		model = strings.TrimSpace(os.Getenv("COPILOT_MODEL"))
	}
	var err error
	model, err = resolveModel(model)
	if err != nil {
		fatal(err)
	}

	prompt := strings.TrimSpace(strings.Join(args, " "))
	var copilotPrompt string
	var commitPromptSource string

	if prompt == "" {
		diff, err := stagedDiff()
		if err != nil {
			fatal(err)
		}
		if strings.TrimSpace(diff) == "" {
			fatal(fmt.Errorf("no staged changes found; run `git add` first"))
		}
		systemPrompt, source := resolveCommitSystemPrompt()
		commitPromptSource = source
		copilotPrompt = buildCommitPrompt(systemPrompt, diff)
	} else {
		copilotPrompt = prompt
	}

	if prompt == "" {
		fmt.Printf("****** using commit prompt: %s ******\n\n", commitPromptSource)
	}

	response, err := runCopilotPrompt(copilotPrompt, model)
	if err != nil {
		fatal(err)
	}

	if prompt == "" && doCommit {
		msg := strings.TrimSpace(response)
		fmt.Println(msg)

		if !skipConfirm {
			if !confirmCommit() {
				fmt.Fprintln(os.Stderr, "commit cancelled")
				os.Exit(0)
			}
		}

		out, err := exec.Command("git", "commit", "-m", msg).CombinedOutput()
		if err != nil {
			fatal(fmt.Errorf("git commit failed: %s", strings.TrimSpace(string(out))))
		}
		fmt.Print(string(out))
		return
	}

	fmt.Println(response)
}

func stagedDiff() (string, error) {
	out, err := exec.Command("git", "diff", "--staged").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}
	return string(out), nil
}

const defaultCommitSystemPrompt = `You are an expert at writing Git commit messages following the Conventional Commits specification.

Given a git diff, output a single commit message in English. Rules:
- Format: <type>(<optional scope>): <short description>
- For complex changes, add a blank line after the subject followed by a body
- The body must be written as a bullet list (each item starting with "- ")
- Choose the type that best fits the change:
    feat:     a new feature
    fix:      a bug fix
    docs:     documentation changes only
    style:    formatting, whitespace (no logic change)
    refactor: code change that is neither a fix nor a feature
    perf:     performance improvement
    test:     adding or updating tests
    chore:    build process, dependencies, tooling
    ci:       CI configuration or scripts
- The short description must be in the imperative mood, lowercase, no trailing period
- Keep the subject line under 72 characters
- Output ONLY the raw commit message text. Do not include any explanation, preamble, markdown fences, or any text other than the commit message itself.`

func commitSystemPrompt() string {
	prompt, _ := resolveCommitSystemPrompt()
	return prompt
}

func resolveCommitSystemPrompt() (string, string) {
	for _, path := range commitPromptCandidates() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if s := strings.TrimSpace(string(data)); s != "" {
			return s, path
		}
	}
	return defaultCommitSystemPrompt, builtInPromptName
}

func commitPromptCandidates() []string {
	paths := make([]string, 0, 2)
	if repoRoot, err := gitRepoRoot(); err == nil && repoRoot != "" {
		paths = append(paths, filepath.Join(repoRoot, ".commit-prompt.txt"))
	}
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		paths = append(paths, filepath.Join(configDir, "gh-coco", "commit-prompt.txt"))
	}
	return paths
}

func gitRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func printHelp() {
	configDir, _ := os.UserConfigDir()
	localPromptPath := ".commit-prompt.txt (repository root)"
	globalPromptPath := filepath.Join(configDir, "gh-coco", "commit-prompt.txt")
	fmt.Printf(`Usage: gh coco [options] [prompt]

A GitHub CLI extension that uses gh copilot to generate commit messages
and answer questions.

Modes:
  gh coco                      Generate a conventional commit message from
                               staged changes (git diff --staged)
  gh coco --commit             Generate a commit message and prompt for
                               confirmation before running git commit
  gh coco --commit --yes       Generate and commit without confirmation
  gh coco <prompt>             Ask gh copilot with a prompt

Options:
  -c, --commit                 Generate commit message (ask for confirmation)
  -y, --yes                    Skip confirmation and commit automatically
  -m, --model <name>           Model override for gh copilot
  -v, --version                Show version information
  -h, --help                   Show this help message

Environment variables:
  COPILOT_MODEL                Model to use (gpt-4.1 only, default: gpt-4.1)

Commit prompt customization:
  1. %s
  2. %s

  The first existing non-empty file is used as the system prompt for commit
  message generation. Falls back to the built-in prompt if none are found.
`, localPromptPath, globalPromptPath)
}

func buildCommitPrompt(systemPrompt, diff string) string {
	return strings.TrimSpace(systemPrompt) + "\n\nStaged git diff:\n```diff\n" + diff + "\n```\n"
}

func resolveModel(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "gpt-4.1", "4.1":
		return "gpt-4.1", nil
	case "":
		return defaultForcedModel, nil
	default:
		return "", fmt.Errorf("unsupported model %q: use gpt-4.1", raw)
	}
}

func copilotCommandArgs(prompt, model string) []string {
	args := []string{"copilot", "--", "-p", prompt, "-s", "--no-color", "--reasoning-effort", "none"}
	if m := strings.TrimSpace(model); m != "" {
		args = append(args, "--model", m)
	}
	return args
}

func runCopilotPrompt(prompt, model string) (string, error) {
	out, err := exec.Command("gh", copilotCommandArgs(prompt, model)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh copilot failed: %s", strings.TrimSpace(string(out)))
	}
	response := strings.TrimSpace(string(out))
	if response == "" {
		return "", fmt.Errorf("empty response from gh copilot")
	}
	return response, nil
}

func confirmCommit() bool {
	fmt.Fprint(os.Stderr, "commit with this message? (Y/n) ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response := strings.TrimSpace(strings.ToLower(input))
	return response == "" || response == "y"
}

func getEnvDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
