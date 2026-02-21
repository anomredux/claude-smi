package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

type Notification struct {
	Message   string
	CreatedAt time.Time
}

type NotificationManager struct {
	active *Notification
	bell   bool
}

func NewNotificationManager(bell bool) *NotificationManager {
	return &NotificationManager{bell: bell}
}

// SetMessage shows a transient informational notification.
func (nm *NotificationManager) SetMessage(msg string) {
	nm.active = &Notification{
		Message:   msg,
		CreatedAt: time.Now(),
	}
}

// Active returns the current notification if it has not expired.
func (nm *NotificationManager) Active() *Notification {
	if nm.active == nil {
		return nil
	}
	if time.Since(nm.active.CreatedAt) > 5*time.Second {
		return nil
	}
	return nm.active
}

// Expire clears expired notifications. Call from Update(), not View().
func (nm *NotificationManager) Expire() {
	if nm.active != nil && time.Since(nm.active.CreatedAt) > 5*time.Second {
		nm.active = nil
	}
}

func (nm *NotificationManager) RenderBanner(width int) string {
	n := nm.Active()
	if n == nil {
		return ""
	}

	style := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Padding(0, 1).
		Foreground(theme.ColorMauve)

	bellChar := ""
	if nm.bell {
		bellChar = "\a"
	}

	return bellChar + style.Render(n.Message)
}
