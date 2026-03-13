package api

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetOAuthTokenFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		perm     os.FileMode
		wantErr  bool
		unixOnly bool
	}{
		{
			name:    "valid credentials",
			content: `{"claudeAiOauth":{"accessToken":"tok-123"}}`,
			perm:    0600,
		},
		{
			name:    "malformed JSON",
			content: `{not json}`,
			perm:    0600,
			wantErr: true,
		},
		{
			name:    "empty file",
			content: "",
			perm:    0600,
			wantErr: true,
		},
		{
			name:    "empty access token",
			content: `{"claudeAiOauth":{"accessToken":""}}`,
			perm:    0600,
			wantErr: true,
		},
		{
			name:     "world-readable file rejected",
			content:  `{"claudeAiOauth":{"accessToken":"tok-123"}}`,
			perm:     0644,
			wantErr:  true,
			unixOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("Unix-only permission check")
			}
			path := filepath.Join(t.TempDir(), "creds.json")
			if err := os.WriteFile(path, []byte(tt.content), tt.perm); err != nil {
				t.Fatalf("write test file: %v", err)
			}
			got, err := getOAuthTokenFromPath(path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == "" {
				t.Error("expected non-empty token")
			}
		})
	}
}

func TestGetOAuthTokenFromPath_Missing(t *testing.T) {
	t.Parallel()
	_, err := getOAuthTokenFromPath(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}
