package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/medievalsoftware/herald/internal/config"
	"github.com/medievalsoftware/herald/internal/format"
)

// ApplyTheme updates all themed style variables from the given config theme.
// Must be called before tui.New().
func ApplyTheme(t config.Theme) {
	paletteBg = lipgloss.Color(t.BarBg)
	paletteSelBg := lipgloss.Color(t.SelBg)
	if t.SelBg == "" {
		paletteSelBg = paletteBg
	}
	paletteSelStyle = paletteSelStyle.Foreground(lipgloss.Color(t.Accent)).Background(paletteSelBg)
	paletteNormalStyle = paletteNormalStyle.Background(paletteBg)
	palettePadStyle = palettePadStyle.Background(paletteBg)
	paletteDescStyle = paletteDescStyle.BorderForeground(lipgloss.Color(t.Border)).Background(paletteBg).BorderBackground(paletteBg)

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

	inputNickStyle = inputNickStyle.Foreground(lipgloss.Color(t.Green))

	notifyStyle = notifyStyle.Foreground(lipgloss.Color(t.Accent))

	format.SetNickColors(t.Nicks)
}
