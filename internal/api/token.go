package api

import (
	"encoding/json"
	"fmt"
)

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
