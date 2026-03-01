package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/medievalsoftware/herald/internal/config"
	"github.com/medievalsoftware/herald/internal/format"
)

// ApplyTheme updates all themed style variables from the given config theme.
// Must be called before tui.New().
func ApplyTheme(t config.Theme) {
	paletteSelStyle = paletteSelStyle.Background(lipgloss.Color(t.BarBg))

	separatorStyle = separatorStyle.Foreground(lipgloss.Color(t.Yellow))

	linkStyle = linkStyle.Foreground(lipgloss.Color(t.Accent))

	usersTitleStyle = usersTitleStyle.
		Foreground(lipgloss.Color(t.BarFg)).
		Background(lipgloss.Color(t.BarBg))
	usersBorderStyle = usersBorderStyle.BorderForeground(lipgloss.Color(t.Border))
	usersOpStyle = usersOpStyle.Foreground(lipgloss.Color(t.Green))
	usersVoiceStyle = usersVoiceStyle.Foreground(lipgloss.Color(t.Yellow))

	channelActiveStyle = channelActiveStyle.
		Foreground(lipgloss.Color(t.Accent)).
		Background(lipgloss.Color(t.BarBg))

	statusBarStyle = statusBarStyle.
		Foreground(lipgloss.Color(t.BarFg)).
		Background(lipgloss.Color(t.BarBg))
	statusNickStyle = statusNickStyle.Foreground(lipgloss.Color(t.Green))

	format.SetNickColors(t.Nicks)
}
