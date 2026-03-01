package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type statusModel struct {
	server string
	nick   string
	mode   string
	users  int
}

func newStatus() statusModel {
	return statusModel{}
}

var (
	statusBarStyle    = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("252")).Padding(0, 1)
	statusNickStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	statusServerStyle = lipgloss.NewStyle().Faint(true)
)

func (m statusModel) View(width int) string {
	left := statusNickStyle.Render(m.nick)
	if m.mode != "" {
		left += " [+" + m.mode + "]"
	}

	right := statusServerStyle.Render(m.server)
	if m.users > 0 {
		right += fmt.Sprintf(" (%d users)", m.users)
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	content := left + fmt.Sprintf("%*s", gap, "") + right
	return statusBarStyle.Width(width).Render(content)
}
