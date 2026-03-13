//go:build !darwin

package api

import "context"

// getOAuthToken reads the Claude Code OAuth token from ~/.claude/.credentials.json.
func getOAuthToken(_ context.Context) (string, error) {
	return getOAuthTokenFromFile()
}
