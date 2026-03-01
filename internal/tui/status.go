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

func (m statusModel) View(width int) string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Width(width).
		Padding(0, 1)

	nickStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	serverStyle := lipgloss.NewStyle().Faint(true)

	left := nickStyle.Render(m.nick)
	if m.mode != "" {
		left += " [+" + m.mode + "]"
	}

	right := serverStyle.Render(m.server)
	if m.users > 0 {
		right += fmt.Sprintf(" (%d users)", m.users)
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	content := left + fmt.Sprintf("%*s", gap, "") + right
	return style.Render(content)
}
