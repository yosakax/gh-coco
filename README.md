# gh-coco

`gh-coco` is a GitHub CLI extension that uses GitHub Copilot to:

- generate Conventional Commit messages from staged changes
- optionally run `git commit` with the generated message
- run chat-style prompts

## Install

```bash
gh extension install yosakax/gh-coco
```

## Usage

```bash
gh coco [options] [prompt]
```

### Modes

1. `gh coco`
   Generate a Conventional Commit message from `git diff --staged` and print it.
2. `gh coco --commit` or `gh coco -c`
   Generate a commit message and prompt for confirmation before running `git commit`.
3. `gh coco --commit --yes` or `gh coco -c -y`
   Generate a commit message and run `git commit` without confirmation.
4. `gh coco "<prompt>"`
   Chat with Copilot.

## Options

- `-c, --commit` Generate commit message (ask for confirmation).
- `-y, --yes` Skip confirmation and commit automatically.
- `-h, --help` Show help.

## Environment variables

These are optional. If token env vars are not set, `gh-coco` auto-detects a token from local Copilot config files.

- `COPILOT_GITHUB_TOKEN` (preferred)
- `GH_TOKEN`
- `GITHUB_TOKEN`
- `COPILOT_MODEL` (default: `gpt-4o`)
- `COPILOT_API_BASE_URL` (override auto-detected endpoint from token response)

## Commit prompt customization

You can override the built-in commit system prompt with this priority order:

1. `.commit-prompt.txt` at the repository root
2. `{UserConfigDir}/gh-coco/commit-prompt.txt`
3. built-in default prompt

Examples for the user config path:

- Linux: `~/.config/gh-coco/commit-prompt.txt`
- macOS: `~/Library/Application Support/gh-coco/commit-prompt.txt`
- Windows: `%AppData%\gh-coco\commit-prompt.txt`

The first existing non-empty file is used.
When generating a commit message, `gh coco` prints the selected prompt source first.

## Development

```bash
go test ./...
go run . --help
```

## pre-commit

If you want to use `.pre-commit-config.yml`, run:

```bash
pre-commit install
pre-commit install --hook-type pre-push
```

- pre-commit: `go fmt ./...`, `go vet ./...`, basic hooks
- pre-push: `go test ./...`
