//go:build darwin

package api

import (
	"fmt"
	"os/exec"
	"strings"
)

// getOAuthToken reads the Claude Code OAuth token from macOS Keychain.
func getOAuthToken() (string, error) {
	out, err := exec.Command("security", "find-generic-password",
		"-s", keychainLabel, "-w").Output()
	if err != nil {
		return "", fmt.Errorf("keychain lookup failed: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	return parseCredentialJSON(raw)
}
