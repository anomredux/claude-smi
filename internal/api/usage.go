package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	usageEndpoint  = "https://api.anthropic.com/api/oauth/usage"
	keychainLabel  = "Claude Code-credentials"
	anthropicBeta  = "oauth-2025-04-20"
	requestTimeout = 5 * time.Second
	SessionWindow  = 5 * time.Hour
)

// UsageData holds the parsed API response.
type UsageData struct {
	FiveHour  WindowData `json:"five_hour"`
	SevenDay  WindowData `json:"seven_day"`
	FetchedAt time.Time  `json:"-"`
}

// WindowData holds utilization info for a single time window.
type WindowData struct {
	Utilization float64 `json:"utilization"` // 0-100 percentage
	ResetsAt    string  `json:"resets_at"`   // ISO 8601 timestamp
}

// ResetTime parses the resets_at string into time.Time.
func (w WindowData) ResetTime() (time.Time, error) {
	return time.Parse(time.RFC3339, w.ResetsAt)
}

// SessionStart returns the inferred session start time (resets_at - 5h).
func (u UsageData) SessionStart() (time.Time, error) {
	resetTime, err := u.FiveHour.ResetTime()
	if err != nil {
		return time.Time{}, err
	}
	return resetTime.Add(-SessionWindow), nil
}

// SessionEnd returns the session end time (resets_at).
func (u UsageData) SessionEnd() (time.Time, error) {
	return u.FiveHour.ResetTime()
}

// SessionRemaining returns the duration until the 5-hour window resets.
func (u UsageData) SessionRemaining() (time.Duration, error) {
	resetTime, err := u.FiveHour.ResetTime()
	if err != nil {
		return 0, err
	}
	remaining := time.Until(resetTime)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// FetchUsage retrieves current usage data from the Anthropic OAuth API.
func FetchUsage() (*UsageData, error) {
	token, err := getOAuthToken()
	if err != nil {
		return nil, fmt.Errorf("get oauth token: %w", err)
	}

	client := &http.Client{Timeout: requestTimeout}
	req, err := http.NewRequest("GET", usageEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", anthropicBeta)
	req.Header.Set("User-Agent", "claude-smi/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var data UsageData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	data.FetchedAt = time.Now()
	return &data, nil
}

// parseCredentialJSON extracts the OAuth access token from Claude Code's
// credential JSON stored in the system credential store.
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
