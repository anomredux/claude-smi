package i18n

import "fmt"

// Language represents a supported locale.
type Language string

const (
	LangEN Language = "en"
)

var current Language = LangEN

// SetLanguage changes the active locale.
// Unrecognized values fall back to English.
func SetLanguage(lang string) {
	switch Language(lang) {
	case LangEN:
		current = LangEN
	default:
		current = LangEN
	}
}

// Current returns the active language.
func Current() Language {
	return current
}

// T returns the translated string for the given key.
// If the key is not found, the key itself is returned.
func T(key string) string {
	if v, ok := en[key]; ok {
		return v
	}
	return key
}

// Tf returns a formatted translated string.
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}
