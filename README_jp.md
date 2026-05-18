# gh-coco

`gh-coco` は GitHub Copilot を使って次のことを行う GitHub CLI 拡張です。

- ステージ済み変更から Conventional Commit メッセージを生成
- 生成したメッセージで `git commit` を実行（任意）
- チャット形式のプロンプトを実行

## インストール

```bash
gh extension install yosakax/gh-coco
```

## 使い方

```bash
gh coco [options] [prompt]
```

### モード

1. `gh coco`
   `git diff --staged` から Conventional Commit メッセージを生成して表示します。
2. `gh coco --commit` または `gh coco -c`
   コミットメッセージを生成し、確認後に `git commit` を実行します。
3. `gh coco --commit --yes` または `gh coco -c -y`
   確認なしでコミットメッセージを生成して `git commit` を実行します。
4. `gh coco "<prompt>"`
   Copilot とチャットします。

## オプション

- `-c, --commit` コミットメッセージを生成（確認あり）
- `-y, --yes` 確認をスキップして自動コミット
- `-v, --version` バージョン情報を表示
- `-h, --help` ヘルプを表示

## 環境変数

以下は任意です。トークン系の環境変数を設定しない場合、`gh-coco` はローカルの Copilot 設定ファイルからトークンを自動検出します。

- `COPILOT_GITHUB_TOKEN`（推奨）
- `GH_TOKEN`
- `GITHUB_TOKEN`
- `COPILOT_MODEL`（デフォルト: `gpt-4o`）
- `COPILOT_API_BASE_URL`（トークン応答から自動検出されるエンドポイントを上書き）

## コミットプロンプトのカスタマイズ

組み込みのコミット用 system prompt は、次の優先順位で上書きできます。

1. リポジトリルートの `.commit-prompt.txt`
2. `{UserConfigDir}/gh-coco/commit-prompt.txt`
3. プログラム内のデフォルトプロンプト

ユーザー設定パスの例:

- Linux: `~/.config/gh-coco/commit-prompt.txt`
- macOS: `~/Library/Application Support/gh-coco/commit-prompt.txt`
- Windows: `%AppData%\gh-coco\commit-prompt.txt`

存在し、かつ空でない最初のファイルが使われます。
コミットメッセージ生成時には、先に使用中のプロンプトソースが表示されます。

## 開発

```bash
go test ./...
go run . --help
```

## pre-commit

`.pre-commit-config.yml` を使う場合は、以下を実行してください。

```bash
pre-commit install
pre-commit install --hook-type pre-push
```

- pre-commit: `go fmt ./...`, `go vet ./...`, basic hooks
- pre-push: `go test ./...`
