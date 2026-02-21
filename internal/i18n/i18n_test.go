package i18n

import "testing"

func TestT_English(t *testing.T) {
	SetLanguage("en")

	if got := T("tab_live"); got != "Live Dashboard" {
		t.Errorf("T(tab_live) = %q, want %q", got, "Live Dashboard")
	}
	if got := T("tokens"); got != "Tokens" {
		t.Errorf("T(tokens) = %q, want %q", got, "Tokens")
	}
}

func TestT_MissingKey(t *testing.T) {
	SetLanguage("en")
	if got := T("nonexistent_key"); got != "nonexistent_key" {
		t.Errorf("T(nonexistent_key) = %q, want %q", got, "nonexistent_key")
	}
}

func TestTf(t *testing.T) {
	SetLanguage("en")
	got := Tf("current_size", 120, 40)
	want := "Current: 120x40"
	if got != want {
		t.Errorf("Tf(current_size, 120, 40) = %q, want %q", got, want)
	}
}

func TestSetLanguage_Unknown(t *testing.T) {
	SetLanguage("fr")
	if Current() != LangEN {
		t.Errorf("unknown language should default to EN, got %q", Current())
	}
}
