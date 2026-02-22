package views

import tea "github.com/charmbracelet/bubbletea"

// KeyHandledCmd is returned by view Update methods to signal that a key
// was consumed and should not propagate to app-level scroll or global
// handlers. It is a no-op cmd: bubbletea discards nil messages.
var KeyHandledCmd tea.Cmd = func() tea.Msg { return nil }
