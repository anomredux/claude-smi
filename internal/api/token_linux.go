//go:build linux

package api

import (
	"fmt"
	"os/exec"
	"strings"
)

// getOAuthToken reads the Claude Code OAuth token from Linux secret store
// via libsecret (gnome-keyring / kwallet).
// Requires: sudo apt install libsecret-tools (Debian/Ubuntu)
//
//	or: sudo dnf install libsecret (Fedora)
func getOAuthToken() (string, error) {
	out, err := exec.Command("secret-tool", "lookup",
		"service", keychainLabel).Output()
	if err != nil {
		return "", fmt.Errorf("secret-tool lookup failed (install libsecret-tools): %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return "", fmt.Errorf("empty credential from secret-tool")
	}
	return parseCredentialJSON(raw)
}
