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

- `COPILOT_GITHUB_TOKEN` (preferred)
- `GH_TOKEN`
- `GITHUB_TOKEN`
- `COPILOT_MODEL` (default: `gpt-4o`)
- `COPILOT_API_BASE_URL` (default: `https://api.individual.githubcopilot.com`)

## Commit prompt customization

You can override the built-in commit system prompt by creating:

`{UserConfigDir}/gh-coco/commit-prompt.txt`

Examples:

- Linux: `~/.config/gh-coco/commit-prompt.txt`
- macOS: `~/Library/Application Support/gh-coco/commit-prompt.txt`
- Windows: `%AppData%\gh-coco\commit-prompt.txt`

If the file does not exist (or is empty), the built-in prompt is used.

## Development

```bash
go test ./...
go run . --help
```
