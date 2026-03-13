package api

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// maxCredFileSize bounds the credential file read to 1 MB.
const maxCredFileSize = 1 << 20

// getOAuthTokenFromFile reads the OAuth token from ~/.claude/.credentials.json.
func getOAuthTokenFromFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return getOAuthTokenFromPath(filepath.Join(home, ".claude", ".credentials.json"))
}

// getOAuthTokenFromPath reads and validates a credential file at the given path.
func getOAuthTokenFromPath(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // G304: path derived from user home directory
	if err != nil {
		return "", fmt.Errorf("open credentials file: %w", err)
	}
	defer f.Close()

	if runtime.GOOS != "windows" {
		info, err := f.Stat()
		if err != nil {
			return "", fmt.Errorf("stat credentials file: %w", err)
		}
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			return "", fmt.Errorf("credentials file %s has mode %04o; want 0600", path, perm)
		}
	}

	data, err := io.ReadAll(io.LimitReader(f, maxCredFileSize))
	if err != nil {
		return "", fmt.Errorf("read credentials file: %w", err)
	}
	return parseCredentialJSON(string(data))
}

// parseCredentialJSON extracts the OAuth access token from Claude Code's
// credential JSON (sourced from a file or system credential store).
func parseCredentialJSON(raw string) (string, error) {
	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return "", fmt.Errorf("parse credentials: %w", err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}
	return creds.ClaudeAiOauth.AccessToken, nil
}
