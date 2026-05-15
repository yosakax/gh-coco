package main

import (
	"bufio"
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type copilotSessionToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

const (
	defaultAPIBaseURL = "https://api.individual.githubcopilot.com"
	defaultModel      = "gpt-4o"
	defaultMaxTokens  = 1024
)

type chatCompletionRequest struct {
	Model     string        `json:"model"`
	Stream    bool          `json:"stream"`
	MaxTokens int           `json:"max_tokens,omitempty"`
	Messages  []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func main() {
	prompt := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if prompt == "" {
		prompt = "自己紹介して"
	}

	oauthToken, err := loadCopilotToken()
	if err != nil {
		fatal(err)
	}

	token, err := exchangeCopilotToken(oauthToken)
	if err != nil {
		fatal(err)
	}

	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:     getEnvDefault("COPILOT_MODEL", defaultModel),
		Stream:    true,
		MaxTokens: defaultMaxTokens,
		Messages: []chatMessage{
			{Role: "user", Content: buildPrompt(prompt)},
		},
	})
	if err != nil {
		fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(getEnvDefault("COPILOT_API_BASE_URL", defaultAPIBaseURL), "/")+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tmpSoft/0.0.1")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
	req.Header.Set("X-Request-Id", requestID())
	req.Header.Set("X-Vscode-User-Agent-Library-Version", "electron-fetch")

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
		fatal(fmt.Errorf("copilot API error (%d): %s", res.StatusCode, strings.TrimSpace(string(body))))
	}

	if err := streamResponse(res.Body); err != nil {
		fatal(err)
	}
}

func exchangeCopilotToken(oauthToken string) (string, error) {
	cacheFile := filepath.Join(os.TempDir(), "gh-hello-copilot-token.json")

	if data, err := os.ReadFile(cacheFile); err == nil {
		var cached copilotSessionToken
		if json.Unmarshal(data, &cached) == nil && time.Now().Before(cached.ExpiresAt.Add(-2*time.Minute)) {
			return cached.Token, nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/copilot_internal/v2/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+oauthToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tmpSoft/0.0.1")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return "", fmt.Errorf("token exchange failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp struct {
		Token     string  `json:"token"`
		ExpiresAt float64 `json:"expires_at"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}
	if tokenResp.Token == "" {
		return "", fmt.Errorf("empty token from copilot token endpoint")
	}

	expiresAt := time.Unix(int64(tokenResp.ExpiresAt), 0)
	if tokenResp.ExpiresAt == 0 {
		expiresAt = time.Now().Add(25 * time.Minute)
	}

	if data, err := json.Marshal(copilotSessionToken{Token: tokenResp.Token, ExpiresAt: expiresAt}); err == nil {
		_ = os.WriteFile(cacheFile, data, 0o600)
	}

	return tokenResp.Token, nil
}

func loadCopilotToken() (string, error) {
	for _, key := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(key)); token != "" {
			return token, nil
		}
	}

	for _, path := range copilotTokenFiles() {
		token, err := tokenFromFile(path)
		if err == nil && token != "" {
			return token, nil
		}
	}

	return "", fmt.Errorf("no Copilot token found; set COPILOT_GITHUB_TOKEN or sign in to Copilot")
}

func copilotTokenFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "github-copilot", "apps.json"),
		filepath.Join(home, ".config", "github-copilot", "hosts.json"),
	}
}

func tokenFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var entries map[string]struct {
		OAuthToken string `json:"oauth_token"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", err
	}

	for _, entry := range entries {
		if token := strings.TrimSpace(entry.OAuthToken); token != "" {
			return token, nil
		}
	}

	return "", fmt.Errorf("no oauth token in %s", path)
}

func streamResponse(body io.Reader) error {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || strings.TrimSpace(data) == "" || data == "[DONE]" {
			continue
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
	return scanner.Err()
}

func buildPrompt(userPrompt string) string {
	return "語尾に「にゃん」をつけて質問に回答して。 User message: " + userPrompt
}

func requestID() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	parts := []string{
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	}
	return strings.Join(parts, "-")
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
