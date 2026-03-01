package format

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/ergochat/irc-go/ircfmt"
)

// IRCToStyled converts IRC formatting codes to a lipgloss-styled string.
// For now, strips formatting and returns plain text. Full color support
// can be layered in later by parsing control codes directly.
func IRCToStyled(text string) string {
	return ircfmt.Strip(text)
}

// Strip removes all IRC formatting codes from text.
func Strip(text string) string {
	return ircfmt.Strip(text)
}

// NickColor returns a consistent lipgloss style for a nickname.
func NickColor(nick string) lipgloss.Style {
	h := 0
	for _, c := range nick {
		h = (h*31 + int(c)) & 0x7fffffff
	}
	colors := []lipgloss.Color{
		lipgloss.Color("1"),
		lipgloss.Color("2"),
		lipgloss.Color("3"),
		lipgloss.Color("4"),
		lipgloss.Color("5"),
		lipgloss.Color("6"),
		lipgloss.Color("9"),
		lipgloss.Color("10"),
		lipgloss.Color("11"),
		lipgloss.Color("12"),
		lipgloss.Color("13"),
		lipgloss.Color("14"),
	}
	return lipgloss.NewStyle().Foreground(colors[h%len(colors)])
}
